package eightfold

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"course-sync/internal/httpx"
)

const (
	contentTypeJSON = "application/json"
	acceptJSON      = contentTypeJSON
)

type Client struct {
	BaseURL     string
	HTTP        *http.Client
	BearerToken string
}

type CourseUpsertRequest struct {
	Status        string   `json:"status,omitempty"`
	ImageUrl      string   `json:"imageUrl,omitempty"`
	LmsCourseId   string   `json:"lmsCourseId,omitempty"`
	Language      string   `json:"language,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	SystemId      string   `json:"systemId,omitempty"`
	DurationHours float64  `json:"durationHours,omitempty"`
	CourseType    string   `json:"courseType,omitempty"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	Difficulty    string   `json:"difficulty,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	CourseUrl     string   `json:"courseUrl,omitempty"`
	Description   string   `json:"description,omitempty"`
	Title         string   `json:"title,omitempty"`
	Category      string   `json:"category,omitempty"`
}

func (c *Client) UpsertCourse(ctx context.Context, course CourseUpsertRequest) error {
	if c.BearerToken == "" {
		return errors.New("eightfold: missing bearer token (call Authenticate first)")
	}

	b, err := json.Marshal(course)
	if err != nil {
		return err
	}

	_, _, err = httpx.DoWithRetry(
		ctx,
		c.HTTP,
		func(ctx context.Context) (*http.Request, error) {
			r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v2/core/courses", bytes.NewReader(b))
			if err != nil {
				return nil, err
			}
			r.Header.Set("Content-Type", contentTypeJSON)
			r.Header.Set("Accept", acceptJSON)
			r.Header.Set("Authorization", "Bearer "+c.BearerToken)
			return r, nil
		},
		httpx.DefaultRetryConfig(),
	)
	if err != nil {
		return fmt.Errorf("eightfold: upsert course failed: %w", err)
	}
	return nil
}

func New(baseURL string) *Client {
	tr := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout:   2 * time.Minute,
			Transport: tr,
		},
	}
}

type AuthRequest struct {
	GrantType string `json:"grantType"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type AuthResponse struct {
	Data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	} `json:"data"`
}

func (c *Client) Authenticate(ctx context.Context, basicBase64 string, req AuthRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	var ar AuthResponse
	err = httpx.DoJSON(
		ctx,
		c.HTTP,
		func(ctx context.Context) (*http.Request, error) {
			r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/oauth/v1/authenticate", bytes.NewReader(b))
			if err != nil {
				return nil, err
			}
			r.Header.Set("Content-Type", contentTypeJSON)
			r.Header.Set("Accept", acceptJSON)
			r.Header.Set("Authorization", "Basic "+basicBase64)
			return r, nil
		},
		&ar,
		httpx.DefaultRetryConfig(),
	)
	if err != nil {
		return fmt.Errorf("eightfold auth failed: %w", err)
	}

	token := ar.Data.AccessToken
	if token == "" {
		return fmt.Errorf("eightfold auth: token not found")
	}
	c.BearerToken = token
	return nil

}

type ListCoursesResponse struct {
	Data []map[string]any `json:"data"`
	Meta struct {
		PageStartIndex int `json:"pageStartIndex"`
		PageTotalCount int `json:"pageTotalCount"`
		TotalCount     int `json:"totalCount"`
	} `json:"meta"`
}

func (c *Client) ListCourses(ctx context.Context, limit int) ([]map[string]any, error) {
	if c.BearerToken == "" {
		return nil, errors.New("eightfold: missing bearer token (call Authenticate first)")
	}

	u, err := url.Parse(c.BaseURL + "/api/v2/core/courses")
	if err != nil {
		return nil, fmt.Errorf("eightfold: invalid base url: %w", err)
	}
	q := u.Query()
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	var out ListCoursesResponse
	err = httpx.DoJSON(
		ctx,
		c.HTTP,
		func(ctx context.Context) (*http.Request, error) {
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				return nil, err
			}
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Authorization", "Bearer "+c.BearerToken)
			return r, nil
		},
		&out,
		httpx.DefaultRetryConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("eightfold: list courses failed: %w", err)
	}

	return out.Data, nil

}
