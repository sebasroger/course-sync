package pluralsight

import (
	"context"
	"course-sync/internal/domain"
	"strconv"
	"strings"
)

// Provider adapts the Pluralsight GraphQL client into the internal providers.CourseProvider interface.
type Provider struct {
	C        *Client
	First    int
	MaxPages int // <=0 means all
}

// Pluralsight recomienda pedir 1000 registros por request para evitar límites
// (y menciona que resultados sobre 10000 se truncan).
// Fuente: documentación de paginación / troubleshooting de Pluralsight Developer.
const pluralsightMaxFirst = 8000

func (p Provider) Name() string { return "pluralsight" }

func (p Provider) ListCourses(ctx context.Context) ([]domain.UnifiedCourse, error) {
	first := p.First
	if first <= 0 {
		first = pluralsightMaxFirst
	}
	if first > pluralsightMaxFirst {
		first = pluralsightMaxFirst
	}

	var cursor *string
	out := make([]domain.UnifiedCourse, 0, 2048)

	for page := 1; ; page++ {
		if p.MaxPages > 0 && page > p.MaxPages {
			break
		}

		res, err := p.C.ListCoursesPage(ctx, first, cursor)
		if err != nil {
			return nil, err
		}

		for _, n := range res.Data.CourseCatalog.Nodes {
			out = append(out, domain.UnifiedCourse{
				Source:        "pluralsight",
				SourceID:      stablePSID(n),
				Title:         n.Title,
				Description:   firstNonEmpty(n.Description, n.ShortDescription),
				CourseURL:     absolutizePSURL(n.URL),
				Language:      strings.TrimSpace(n.Language),
				Difficulty:    strings.TrimSpace(n.Level),
				DurationHours: n.CourseSeconds / 3600.0,
				Status:        "active",
				PublishedDate: firstNonEmpty(n.PublishedDate, n.DisplayDate, n.ReleasedDate),
				// Category / ImageURL / Skills not in current query (leave empty)
			})
		}

		if !res.Data.CourseCatalog.PageInfo.HasNextPage {
			break
		}
		c := res.Data.CourseCatalog.PageInfo.EndCursor
		cursor = &c
	}

	return out, nil
}

func stablePSID(n CourseNode) string {
	if n.IDNum > 0 {
		return strconv.FormatInt(n.IDNum, 10)
	}
	if strings.TrimSpace(n.ID) != "" {
		return strings.TrimSpace(n.ID)
	}
	return strings.TrimSpace(n.Slug)
}

func absolutizePSURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	if strings.HasPrefix(u, "/") {
		return "https://app.pluralsight.com" + u
	}
	return "https://app.pluralsight.com/" + u
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
