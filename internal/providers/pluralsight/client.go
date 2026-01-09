package pluralsight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			Timeout: 30 * time.Second,
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
	ID               string   `json:"id"`
	IDNum            int64    `json:"idNum"`
	Slug             string   `json:"slug"`
	URL              string   `json:"url"`
	Title            string   `json:"title"`
	Level            string   `json:"level"`
	Description      string   `json:"description"`
	ShortDescription string   `json:"shortDescription"`
	CourseSeconds    float64  `json:"courseSeconds"`
	Authors          []string `json:"authors"`
	Free             bool     `json:"free"`
	ReleasedDate     string   `json:"releasedDate"`
	DisplayDate      string   `json:"displayDate"`
	PublishedDate    string   `json:"publishedDate"`
	AverageRating    float64  `json:"averageRating"`
	NumberOfRatings  int      `json:"numberOfRatings"`
	Language         string   `json:"language"`
}

// Query con variables: first + after
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
      authors
      free
      releasedDate
      displayDate
      publishedDate
      averageRating
      numberOfRatings
      language
    }
  }
}`

func (c *Client) ListCoursesPage(ctx context.Context, first int, after *string) (CourseCatalogGQLResponse, error) {
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
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight: marshal gql request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(r)
	if err != nil {
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight: read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out CourseCatalogGQLResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return CourseCatalogGQLResponse{}, fmt.Errorf("json parse error: %w body=%s", err, string(body))
	}
	if len(out.Errors) > 0 {
		return CourseCatalogGQLResponse{}, fmt.Errorf("pluralsight gql errors: %+v", out.Errors)
	}

	return out, nil
}
