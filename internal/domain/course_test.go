package domain

import (
	"testing"
)

func TestUnifiedCourse(t *testing.T) {
	// Create a test course
	course := UnifiedCourse{
		Source:        "udemy",
		SourceID:      "12345",
		Title:         "Test Course",
		Description:   "This is a test course",
		CourseURL:     "https://example.com/course/12345",
		Language:      "es",
		Category:      "Technology",
		Difficulty:    "Intermediate",
		DurationHours: 2.5,
		Status:        "active",
		PublishedDate: "2023-01-01",
		ImageURL:      "https://example.com/image.jpg",
		Skills:        []string{"Go", "Testing"},
	}

	// Test field values
	if course.Source != "udemy" {
		t.Errorf("Expected Source to be 'udemy', got '%s'", course.Source)
	}

	if course.SourceID != "12345" {
		t.Errorf("Expected SourceID to be '12345', got '%s'", course.SourceID)
	}

	if course.Title != "Test Course" {
		t.Errorf("Expected Title to be 'Test Course', got '%s'", course.Title)
	}

	if course.DurationHours != 2.5 {
		t.Errorf("Expected DurationHours to be 2.5, got '%f'", course.DurationHours)
	}

	if len(course.Skills) != 2 {
		t.Errorf("Expected Skills to have 2 items, got %d", len(course.Skills))
	}
}
