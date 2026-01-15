package mappers

import (
	"testing"
)

func TestFromUnifiedCourse(t *testing.T) {
	// Create a test unified course
	unified := UnifiedCourse{
		ID:          "course123",
		Title:       "Test Course",
		Description: "This is a test course",
		Language:    "es",
		Provider:    "udemy",
		URL:         "https://example.com/course",
		DurationHrs: 2.5,
	}

	// Convert to Eightfold course
	eightfoldCourse := FromUnifiedCourse(unified)

	// Verify field mappings
	if eightfoldCourse.LmsCourseId != unified.ID {
		t.Errorf("Expected LmsCourseId to be %q, got %q", unified.ID, eightfoldCourse.LmsCourseId)
	}

	if eightfoldCourse.Title != unified.Title {
		t.Errorf("Expected Title to be %q, got %q", unified.Title, eightfoldCourse.Title)
	}

	if eightfoldCourse.Description != unified.Description {
		t.Errorf("Expected Description to be %q, got %q", unified.Description, eightfoldCourse.Description)
	}

	if eightfoldCourse.Language != unified.Language {
		t.Errorf("Expected Language to be %q, got %q", unified.Language, eightfoldCourse.Language)
	}

	if eightfoldCourse.Provider != unified.Provider {
		t.Errorf("Expected Provider to be %q, got %q", unified.Provider, eightfoldCourse.Provider)
	}

	if eightfoldCourse.CourseUrl != unified.URL {
		t.Errorf("Expected CourseUrl to be %q, got %q", unified.URL, eightfoldCourse.CourseUrl)
	}

	if eightfoldCourse.DurationHours != unified.DurationHrs {
		t.Errorf("Expected DurationHours to be %f, got %f", unified.DurationHrs, eightfoldCourse.DurationHours)
	}

	// Check default values
	if eightfoldCourse.Status != "active" {
		t.Errorf("Expected Status to be 'active', got %q", eightfoldCourse.Status)
	}
}
