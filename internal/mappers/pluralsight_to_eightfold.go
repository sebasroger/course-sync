package mappers

import (
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/providers/pluralsight"
)

func PluralsightToEightfold(n pluralsight.CourseNode) eightfold.CourseUpsertRequest {
	return eightfold.CourseUpsertRequest{
		Status:        "active",
		LmsCourseId:   n.ID, // o fmt.Sprint(n.IDNum)
		Language:      n.Language,
		SystemId:      "pluralsight",
		DurationHours: n.CourseSeconds / 3600.0,
		Difficulty:    n.Level,
		Provider:      "PLURALSIGHT",
		CourseUrl:     n.URL,
		Description:   pickDesc(n.Description, n.ShortDescription),
		Title:         n.Title,
		PublishedDate: pickDate(n.PublishedDate, n.ReleasedDate, n.DisplayDate),
	}
}

func pickDesc(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func pickDate(a, b, c string) string {
	if a != "" {
		return a
	}
	if b != "" {
		return b
	}
	return c
}
