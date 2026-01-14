package pluralsight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type CourseCatalogGQLResponse struct {
	Data struct {
		CourseCatalog struct {
			TotalCount int `json:"totalCount"`
			PageInfo   struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []CourseNode `json:"nodes"`
		} `json:"courseCatalog"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type CourseNode struct {
	ID               string  `json:"id"`
	IDNum            int64   `json:"idNum"`
	Slug             string  `json:"slug"`
	URL              string  `json:"url"`
	Title            string  `json:"title"`
	Level            string  `json:"level"`
	Description      string  `json:"description"`
	ShortDescription string  `json:"shortDescription"`
	CourseSeconds    float64 `json:"courseSeconds"`
	ReleasedDate     string  `json:"releasedDate"`
	DisplayDate      string  `json:"displayDate"`
	PublishedDate    string  `json:"publishedDate"`
	Language         string  `json:"language"`
}

const courseCatalogQuery = `
query CourseCatalog($first: Int!, $after: String) {
  courseCatalog(first: $first, after: $after) {
    totalCount
    pageInfo { hasNextPage endCursor }
    nodes {
      id
      idNum
      slug
      url
      title
      level
      description
      shortDescription
      courseSeconds
      releasedDate
      displayDate
      publishedDate
      language
    }
  }
}`

func (c *Client) ListCoursesPage(ctx context.Context, first int, after *string) (CourseCatalogGQLResponse, error) {
	const maxAttempts = 8
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		out, retryable, err := c.listCoursesPageOnce(ctx, first, after)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retryable {
			return CourseCatalogGQLResponse{}, err
		}

		sleep := 700*time.Millisecond*time.Duration(1<<(attempt-1)) + time.Duration(rand.Intn(500))*time.Millisecond
		if sleep > 30*time.Second {
			sleep = 30 * time.Second
		}

		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight: context canceled while retrying: %w", ctx.Err())
		}
	}

	return CourseCatalogGQLResponse{}, lastErr
}

func (c *Client) listCoursesPageOnce(ctx context.Context, first int, after *string) (CourseCatalogGQLResponse, bool, error) {
	reqBody := graphQLRequest{
		Query: courseCatalogQuery,
		Variables: map[string]any{
			"first": first,
			"after": func() any {
				if after == nil || *after == "" {
					return nil
				}
				return *after
			}(),
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return CourseCatalogGQLResponse{}, false, fmt.Errorf("pluralsight: marshal gql request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return CourseCatalogGQLResponse{}, false, fmt.Errorf("pluralsight: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(r)
	if err != nil {
		// red/timeouts -> retryable
		return CourseCatalogGQLResponse{}, true, fmt.Errorf("pluralsight: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CourseCatalogGQLResponse{}, true, fmt.Errorf("pluralsight: read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 429/5xx => retryable
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
			return CourseCatalogGQLResponse{}, true, fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
		}
		return CourseCatalogGQLResponse{}, false, fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out CourseCatalogGQLResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return CourseCatalogGQLResponse{}, false, fmt.Errorf("json parse error: %w body=%s", err, string(body))
	}
	if len(out.Errors) > 0 {
		// a veces son temporales
		return CourseCatalogGQLResponse{}, true, fmt.Errorf("pluralsight gql errors: %+v", out.Errors)
	}

	return out, false, nil
}
