package udemy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Campos mínimos para reducir payload y parseo.
// OJO: si en tu tenant esto te rompe algo, comentá esta línea en el query.
const udemyCourseFieldsForXML = "id,title,description,url,estimated_content_length,categories,images,locale,last_update_date,level"

type Client struct {
	BaseURL      string
	ClientId     string
	ClientSecret string
	HTTP         *http.Client
}

func New(baseURL, clientId string, clientSecret string) *Client {
	tr := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &Client{
		BaseURL:      baseURL,
		ClientId:     clientId,
		ClientSecret: clientSecret,
		HTTP: &http.Client{
			Timeout:   2 * time.Minute, // por-request
			Transport: tr,
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

	EstimatedContentLength int64           `json:"estimated_content_length"`
	Locale                 LocaleValue     `json:"locale"`
	LastUpdateDate         string          `json:"last_update_date"`
	Level                  string          `json:"level"`
	Categories             Categories      `json:"categories"`
	Images                 json.RawMessage `json:"images"`
}

/* -------- API -------- */

func (c *Client) ListCourses(ctx context.Context, pageSize int, maxPages int) ([]Course, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Udemy limita a 100
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

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
	q.Set("fields[course]", udemyCourseFieldsForXML)
	u.RawQuery = q.Encode()

	baseURL := u.String() // ya trae ?page_size=100&fields[course]=...

	// 1) Page 1 para saber Count y pageSizeReal
	firstResp, err := c.fetchPageWithRetry(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	pageSizeReal := len(firstResp.Results) // normalmente 100
	if pageSizeReal == 0 {
		return nil, fmt.Errorf("udemy: empty results on first page")
	}

	totalPages := int(math.Ceil(float64(firstResp.Count) / float64(pageSizeReal)))
	if maxPages > 0 && maxPages < totalPages {
		totalPages = maxPages
	}

	fmt.Printf("udemy page 1: results=%d total=%d\n", len(firstResp.Results), firstResp.Count)

	all := make([]Course, 0, minInt(firstResp.Count, totalPages*pageSizeReal))
	all = append(all, firstResp.Results...)

	if totalPages <= 1 {
		return all, nil
	}

	workers := envInt("UDEMY_WORKERS", 8) // 6-12 recomendado
	rps := envInt("UDEMY_RPS", 8)         // global, para evitar 429
	if workers < 1 {
		workers = 1
	}
	if rps < 1 {
		rps = 1
	}

	tick := time.NewTicker(time.Second / time.Duration(rps))
	defer tick.Stop()

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var firstErr error
	var once sync.Once

	// If one page fails, cancel the rest early.
loop:
	for page := 2; page <= totalPages; page++ {
		select {
		case <-ctx.Done():
			break loop
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			// rate limit global
			select {
			case <-tick.C:
			case <-ctx.Done():
				once.Do(func() {
					firstErr = ctx.Err()
					cancel()
				})
				return
			}

			pageURL := baseURL + fmt.Sprintf("&page=%d", p)
			resp, err := c.fetchPageWithRetry(ctx, pageURL)
			if err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}

			fmt.Printf("udemy page %d: results=%d total=%d\n", p, len(resp.Results), resp.Count)

			mu.Lock()
			all = append(all, resp.Results...)
			mu.Unlock()
		}(page)
	}

	wg.Wait()

	if firstErr != nil {
		// devolvemos lo que juntamos + error (para debug)
		return all, firstErr
	}

	return all, nil
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		if retryAfter < 0 {
			return nil, err
		}

		sleep := retryAfter
		if sleep == 0 {
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
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
			return nil, parseRetryAfter(resp), fmt.Errorf("udemy list failed: status=%d body=%s", resp.StatusCode, string(body))
		}
		return nil, -1, fmt.Errorf("udemy list failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out ListCoursesResponse
	if err := json.Unmarshal(body, &out); err != nil {
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
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
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
	return errors.Is(err, context.DeadlineExceeded)
}

func pickUdemyImageURL(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	// prioridad: 480x270
	keys := []string{
		"size_480x270", "image_480x270",
		"size_240x135", "image_240x135",
		"size_125_H", "image_125_H",
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}
