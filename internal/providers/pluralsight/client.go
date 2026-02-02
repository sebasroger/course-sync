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

	"course-sync/internal/httpx"
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
	var lastRetryAfter time.Duration

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		out, retryable, retryAfter, err := c.listCoursesPageOnce(ctx, first, after)
		if err == nil {
			return out, nil
		}
		lastErr = err
		lastRetryAfter = retryAfter
		if !retryable {
			return CourseCatalogGQLResponse{}, err
		}

		sleep := lastRetryAfter
		if sleep <= 0 {
			sleep = 700*time.Millisecond*time.Duration(1<<(attempt-1)) + time.Duration(rand.Intn(500))*time.Millisecond
		}
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

func (c *Client) listCoursesPageOnce(ctx context.Context, first int, after *string) (CourseCatalogGQLResponse, bool, time.Duration, error) {
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
		return CourseCatalogGQLResponse{}, false, 0, fmt.Errorf("pluralsight: marshal gql request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return CourseCatalogGQLResponse{}, false, 0, fmt.Errorf("pluralsight: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(r)
	if err != nil {
		// red/timeouts -> retryable
		return CourseCatalogGQLResponse{}, true, 0, fmt.Errorf("pluralsight: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CourseCatalogGQLResponse{}, true, 0, fmt.Errorf("pluralsight: read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 429/5xx => retryable
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
			return CourseCatalogGQLResponse{}, true, httpx.ParseRetryAfter(resp), fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
		}
		return CourseCatalogGQLResponse{}, false, 0, fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out CourseCatalogGQLResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return CourseCatalogGQLResponse{}, false, 0, fmt.Errorf("json parse error: %w body=%s", err, string(body))
	}
	if len(out.Errors) > 0 {
		// a veces son temporales
		return CourseCatalogGQLResponse{}, true, 0, fmt.Errorf("pluralsight gql errors: %+v", out.Errors)
	}

	return out, false, 0, nil
}

type UserNode struct {
	PsUserID  string `json:"psUserId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type getUserResponse struct {
	Data struct {
		Users struct {
			Nodes []UserNode `json:"nodes"`
		} `json:"users"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

const getUserByEmailQuery = `
query GetUserByEmail($emails: [String]) {
  users(filter: { emails: $emails }) {
    nodes {
      psUserId
      email
      firstName
      lastName
    }
  }
}`

func (c *Client) GetUserByEmail(ctx context.Context, email string) (*UserNode, error) {
	reqBody := graphQLRequest{
		Query: getUserByEmailQuery,
		Variables: map[string]any{
			"emails": []string{email},
		},
	}
	var resp getUserResponse
	if err := c.doGraphQL(ctx, reqBody, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("pluralsight gql errors: %+v", resp.Errors)
	}
	if len(resp.Data.Users.Nodes) == 0 {
		return nil, nil // Not found
	}
	return &resp.Data.Users.Nodes[0], nil
}

type CourseProgressNode struct {
	PsUserID            string  `json:"psUserId"`
	CourseID            string  `json:"courseId"`
	CourseIDNum         int64   `json:"courseIdNum"`
	PercentComplete     float64 `json:"percentComplete"`
	IsCourseCompleted   bool    `json:"isCourseCompleted"`
	CompletedOn         string  `json:"completedOn"`
	CourseSeconds       float64 `json:"courseSeconds"`
	TotalWatchedSeconds float64 `json:"totalWatchedSeconds"`
	TotalClipsWatched   int     `json:"totalClipsWatched"`
	FirstViewedClipOn   string  `json:"firstViewedClipOn"`
	LastViewedClipOn    string  `json:"lastViewedClipOn"`
	PlanID              string  `json:"planId"`
	UpdatedOn           string  `json:"updatedOn"`
	Course              struct {
		Title string `json:"title"`
	} `json:"course"`
}

type courseProgressResponse struct {
	Data struct {
		CourseProgress struct {
			Nodes []CourseProgressNode `json:"nodes"`
		} `json:"courseProgress"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

const courseProgressQuery = `
query courseProgress($ids:[ID]) {
  courseProgress (filter: { psUserIds: $ids }) {
    nodes {
      psUserId
      courseId
      courseIdNum
      percentComplete
      isCourseCompleted
      completedOn
      courseSeconds
      totalWatchedSeconds
      totalClipsWatched
      firstViewedClipOn
      lastViewedClipOn
      planId
      updatedOn
      course {
        title
      }
    }
  }
}`

func (c *Client) GetCourseProgress(ctx context.Context, psUserID string) ([]CourseProgressNode, error) {
	reqBody := graphQLRequest{
		Query: courseProgressQuery,
		Variables: map[string]any{
			"ids": []string{psUserID},
		},
	}
	var resp courseProgressResponse
	if err := c.doGraphQL(ctx, reqBody, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("pluralsight gql errors: %+v", resp.Errors)
	}
	return resp.Data.CourseProgress.Nodes, nil
}

func (c *Client) doGraphQL(ctx context.Context, reqBody graphQLRequest, out any) error {
	b, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal gql request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(r)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pluralsight gql failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("json parse error: %w body=%s", err, string(body))
	}
	return nil
}
