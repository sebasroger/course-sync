package udemy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Client struct {
	BaseURL      string
	ClientId     string
	ClientSecret string
	HTTP         *http.Client
}

func New(baseURL, clientId string, clientSecret string) *Client {
	return &Client{
		BaseURL:      baseURL,
		ClientId:     clientId,
		ClientSecret: clientSecret,
		HTTP: &http.Client{
			Timeout: 2 * time.Minute, // por-request
		},
	}
}

/* -------- Response -------- */

type ListCoursesResponse struct {
	Results []Course `json:"results"`
	Next    string   `json:"next"`
	Count   int      `json:"count"`
}

type Course struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Language    string `json:"language"`

	EstimatedContentLength int64       `json:"estimated_content_length"`
	Locale                 LocaleValue `json:"locale"`
	LastUpdateDate         string      `json:"last_update_date"`
	Level                  string      `json:"level"`
	Categories             Categories  `json:"categories"`
	Images                 struct {
		Image480x270 string `json:"image_480x270"`
		Image240x135 string `json:"image_240x135"`
		Image125H    string `json:"image_125_H"`
	} `json:"images"`
}

/* -------- API -------- */

func (c *Client) ListCourses(ctx context.Context, pageSize int, maxPages int) ([]Course, error) {
	var all []Course

	orgID := os.Getenv("UDEMY_ORG_ID")
	if orgID == "" {
		return nil, fmt.Errorf("udemy: missing env UDEMY_ORG_ID")
	}

	u, err := url.Parse(fmt.Sprintf("%s/organizations/%s/courses/list/", c.BaseURL, orgID))
	if err != nil {
		return nil, fmt.Errorf("udemy: invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	q.Set("fields[course]", "@all")
	u.RawQuery = q.Encode()

	next := u.String()

	for page := 1; next != ""; page++ {
		if maxPages > 0 && page > maxPages {
			break
		}

		resp, err := c.fetchPageWithRetry(ctx, next)
		if err != nil {
			// devolvemos lo que juntamos, para no perder todo el run
			return all, fmt.Errorf("udemy list failed after page=%d url=%s: %w", page, next, err)
		}

		fmt.Printf("udemy page %d: results=%d total=%d\n", page, len(resp.Results), resp.Count)
		all = append(all, resp.Results...)

		// pequeño “rate limit” para evitar que te tire 504/429
		time.Sleep(200 * time.Millisecond)

		next = resp.Next
	}

	return all, nil
}

func (c *Client) fetchPageWithRetry(ctx context.Context, pageURL string) (*ListCoursesResponse, error) {
	const maxAttempts = 8

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		out, retryAfter, err := c.fetchPageOnce(ctx, pageURL)
		if err == nil {
			return out, nil
		}

		lastErr = err
		// si NO es retryable, cortamos
		if retryAfter < 0 {
			return nil, err
		}

		// backoff con jitter
		sleep := retryAfter
		if sleep == 0 {
			// exponencial suave (0.7s, 1.4s, 2.8s, ...)
			base := 700 * time.Millisecond
			sleep = base * time.Duration(1<<(attempt-1))
			if sleep > 30*time.Second {
				sleep = 30 * time.Second
			}
			sleep += time.Duration(rand.Intn(500)) * time.Millisecond
		}

		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return nil, fmt.Errorf("udemy: context canceled while retrying: %w", ctx.Err())
		}
	}

	return nil, lastErr
}

// retryAfter:
//   - <0 => no retry
//   - 0  => retry con backoff
//   - >0 => retry usando ese sleep (Retry-After)
func (c *Client) fetchPageOnce(ctx context.Context, pageURL string) (*ListCoursesResponse, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, -1, fmt.Errorf("udemy: build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.ClientId, c.ClientSecret)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if isNetRetryable(err) {
			return nil, 0, fmt.Errorf("udemy: request failed (retryable): %w", err)
		}
		return nil, -1, fmt.Errorf("udemy: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if isNetRetryable(err) {
			return nil, 0, fmt.Errorf("udemy: read body failed (retryable): %w", err)
		}
		return nil, -1, fmt.Errorf("udemy: read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		// 429 o 5xx => retry
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
			return nil, parseRetryAfter(resp), fmt.Errorf("udemy list failed: status=%d body=%s", resp.StatusCode, string(body))
		}
		// 4xx => no retry (salvo 429 arriba)
		return nil, -1, fmt.Errorf("udemy list failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out ListCoursesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		// parse error rara => normalmente no retry, pero si vino HTML (cloudflare/nginx) suele ser retryable
		if looksLikeHTML(body) {
			return nil, 0, fmt.Errorf("udemy: json parse error but looks like HTML (retryable): %w body=%s", err, string(body))
		}
		return nil, -1, fmt.Errorf("udemy: json parse error: %w", err)
	}

	return &out, -1, nil
}

func looksLikeHTML(b []byte) bool {
	s := string(b)
	if len(s) == 0 {
		return false
	}
	return (len(s) >= 6 && (s[0:6] == "<html>" || s[0:5] == "<!DOC" || s[0:4] == "<htm"))
}

func parseRetryAfter(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	// segundos
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	// fecha HTTP
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func isNetRetryable(err error) bool {
	var nerr net.Error
	if errors.As(err, &nerr) {
		return nerr.Timeout() || nerr.Temporary()
	}
	// algunos timeouts vienen envueltos
	return errors.Is(err, context.DeadlineExceeded)
}
