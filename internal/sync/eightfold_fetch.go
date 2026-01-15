package sync

import (
	"context"
	"fmt"
	"strings"

	"course-sync/internal/providers/eightfold"
)

// FetchEightfoldCourses fetches courses from Eightfold and maps them into EFCourse.
// This implementation is "best effort": it uses the existing Eightfold client's ListCourses()
// and supports pagination if the backend honors pageStartIndex.
//
// limit: request page size (recommended 200-500)
// maxPages: 0 means iterate until backend stops returning data.
func FetchEightfoldCourses(ctx context.Context, ef *eightfold.Client, limit int, maxPages int) ([]EFCourse, error) {
	if limit <= 0 {
		limit = 200
	}
	if maxPages < 0 {
		maxPages = 0
	}

	// Try pageStartIndex pagination if supported.
	startIndex := 0
	page := 0
	out := make([]EFCourse, 0, 1024)

	for {
		page++
		if maxPages > 0 && page > maxPages {
			break
		}

		rows, meta, err := ef.ListCoursesPage(ctx, startIndex, limit)
		if err != nil {
			// Fall back to single-shot ListCourses if paging isn't supported.
			if page == 1 {
				raw, err2 := ef.ListCourses(ctx, limit)
				if err2 != nil {
					return nil, err
				}
				mapped := mapEightfoldRows(raw)
				return filterManagedEightfold(mapped), nil
			}
			return nil, err
		}

		mapped := mapEightfoldRows(rows)
		mapped = filterManagedEightfold(mapped)
		out = append(out, mapped...)

		if len(rows) == 0 {
			break
		}

		// Advance
		if meta.PageTotalCount <= 0 {
			// Unknown paging; stop.
			break
		}
		startIndex = meta.PageStartIndex + meta.PageTotalCount
		if meta.TotalCount > 0 && startIndex >= meta.TotalCount {
			break
		}
	}

	return out, nil
}

func filterManagedEightfold(in []EFCourse) []EFCourse {
	out := make([]EFCourse, 0, len(in))
	for _, c := range in {
		id := strings.TrimSpace(firstNonEmpty(c.LMSCourseID, c.SystemID))
		// Only manage Udemy/Pluralsight by default.
		if strings.HasPrefix(id, "UDM+") || strings.HasPrefix(id, "PLS+") {
			out = append(out, c)
		}
	}
	return out
}

func mapEightfoldRows(rows []map[string]any) []EFCourse {
	out := make([]EFCourse, 0, len(rows))
	for _, r := range rows {
		c := EFCourse{
			SystemID:      getString(r, "systemId", "system_id"),
			LMSCourseID:   getString(r, "lmsCourseId", "lms_course_id"),
			Title:         getString(r, "title"),
			Description:   getString(r, "description"),
			CourseURL:     getString(r, "courseUrl", "course_url"),
			Language:      getString(r, "language"),
			Category:      getString(r, "category"),
			Difficulty:    getString(r, "difficulty"),
			Status:        getString(r, "status"),
			PublishedDate: getString(r, "publishedDate", "published_ts"),
			ImageURL:      getString(r, "imageUrl", "image_url"),
		}
		if v, ok := getFloat(r, "durationHours", "duration_hours"); ok {
			c.DurationHours = v
		}
		out = append(out, c)
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func getString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch t := v.(type) {
			case string:
				return t
			default:
				return fmt.Sprintf("%v", t)
			}
		}
	}
	return ""
}

func getFloat(m map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return t, true
			case float32:
				return float64(t), true
			case int:
				return float64(t), true
			case int64:
				return float64(t), true
			case jsonNumber:
				f, err := t.Float64()
				if err == nil {
					return f, true
				}
			}
		}
	}
	return 0, false
}

// jsonNumber is a tiny interface to avoid importing encoding/json in this file.
type jsonNumber interface {
	Float64() (float64, error)
}
