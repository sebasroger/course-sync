package mappers

import "course-sync/internal/providers/eightfold"

type UnifiedCourse struct {
	ID          string
	Title       string
	Description string
	Language    string
	Provider    string
	URL         string
	DurationHrs float64
}

func FromUnifiedCourse(c UnifiedCourse) eightfold.CourseUpsertRequest {
	return eightfold.CourseUpsertRequest{
		LmsCourseId:   c.ID,
		Title:         c.Title,
		Description:   c.Description,
		Language:      c.Language,
		Provider:      c.Provider,
		CourseUrl:     c.URL,
		DurationHours: c.DurationHrs,
		Status:        "active",
	}
}
