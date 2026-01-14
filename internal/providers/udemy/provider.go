package udemy

import (
	"context"
	"course-sync/internal/domain"
	"fmt"
	"net/url"
	"strings"
)

// Provider adapts the Udemy client into the internal providers.CourseProvider interface.
type Provider struct {
	C        *Client
	PageSize int
	MaxPages int // <=0 means all
}

func (p Provider) Name() string { return "udemy" }

func (p Provider) ListCourses(ctx context.Context) ([]domain.UnifiedCourse, error) {
	if p.PageSize <= 0 {
		p.PageSize = 100
	}

	courses, err := p.C.ListCourses(ctx, p.PageSize, p.MaxPages)
	if err != nil {
		return nil, err
	}

	baseHost := deriveUdemyHost(p.C.BaseURL)

	out := make([]domain.UnifiedCourse, 0, len(courses))
	for _, c := range courses {
		out = append(out, domain.UnifiedCourse{
			Source:        "udemy",
			SourceID:      fmt.Sprintf("UDM+%d", c.ID),
			Title:         c.Title,
			Description:   c.Description,
			CourseURL:     absolutizeURL(baseHost, c.URL),
			Language:      firstNonEmpty(string(c.Locale), c.Language),
			Category:      joinCategoryTitles(c.Categories),
			Difficulty:    c.Level,
			DurationHours: durationHoursFromSeconds(c.EstimatedContentLength),
			Status:        "active",
			PublishedDate: c.LastUpdateDate,
			ImageURL:      pickUdemyImageURL(c.Images),
		})
	}
	return out, nil
}

func durationHoursFromSeconds(sec int64) float64 {
	if sec <= 0 {
		return 0
	}
	return float64(sec) / 3600.0
}

func joinCategoryTitles(cats Categories) string {
	if len(cats) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cats))
	for _, c := range cats {
		t := strings.TrimSpace(c.Title)
		if t == "" {
			t = strings.TrimSpace(c.Name)
		}
		if t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " | ")
}

func deriveUdemyHost(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "https://femsa.udemy.com"
	}
	return u.Scheme + "://" + u.Host
}

func absolutizeURL(host, in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	if strings.HasPrefix(in, "http://") || strings.HasPrefix(in, "https://") {
		return in
	}
	if strings.HasPrefix(in, "/") {
		return host + in
	}
	return host + "/" + in
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
