package udemy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
			Timeout: 30 * time.Second,
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

	// Optional fields (if the API returns them / you request them)
	EstimatedContentLength int64      `json:"estimated_content_length"` // seconds
	Locale                 LocaleValue `json:"locale"`
	LastUpdateDate         string     `json:"last_update_date"`
	Level                  string     `json:"level"`
	Categories             Categories `json:"categories"`
	Images                 struct {
		Image480x270 string `json:"image_480x270"`
		Image240x135 string `json:"image_240x135"`
		Image125H    string `json:"image_125_H"`
	} `json:"images"`
}

/* -------- API -------- */

func (c *Client) ListCourses(
	ctx context.Context,
	pageSize int,
	maxPages int, // if <= 0, fetch all pages
) ([]Course, error) {
	var all []Course

	u, err := url.Parse(c.BaseURL + "/courses/list/")
	if err != nil {
		return nil, fmt.Errorf("udemy: invalid base url: %w", err)
	}
	q := u.Query()
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	u.RawQuery = q.Encode()

	next := u.String()

	for page := 1; next != ""; page++ {
		if maxPages > 0 && page > maxPages {
			break
		}
		resp, err := c.fetchPage(ctx, next)
		if err != nil {
			return nil, err
		}

		fmt.Printf(
			"udemy page %d: results=%d total=%d\n",
			page,
			len(resp.Results),
			resp.Count,
		)

		all = append(all, resp.Results...)
		next = resp.Next
	}

	return all, nil
}

func (c *Client) fetchPage(
	ctx context.Context,
	pageURL string,
) (*ListCoursesResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		pageURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("udemy: build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(
		c.ClientId,
		c.ClientSecret,
	)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("udemy: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("udemy: read response body: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(
			"udemy list failed: status=%d body=%s",
			resp.StatusCode,
			string(body),
		)
	}

	var out ListCoursesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("udemy: json parse error: %w", err)
	}

	return &out, nil
}
