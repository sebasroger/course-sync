package main

import (
	"course-sync/internal/domain"
	"testing"
)

func TestFilterCoursesByLang(t *testing.T) {
	courses := []domain.UnifiedCourse{
		{
			Source:        "udemy",
			SourceID:      "1",
			Title:         "Spanish Course",
			Language:      "es",
			DurationHours: 1.5,
		},
		{
			Source:        "udemy",
			SourceID:      "2",
			Title:         "English Course",
			Language:      "en",
			DurationHours: 2.0,
		},
		{
			Source:        "pluralsight",
			SourceID:      "3",
			Title:         "Portuguese Course",
			Language:      "pt",
			DurationHours: 3.0,
		},
		{
			Source:        "pluralsight",
			SourceID:      "4",
			Title:         "French Course",
			Language:      "fr",
			DurationHours: 4.0,
		},
	}

	allowed := map[string]bool{
		"es": true,
		"en": true,
	}

	filtered := filterCoursesByLang(courses, allowed)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 courses, got %d", len(filtered))
	}

	for _, c := range filtered {
		if c.Language != "es" && c.Language != "en" {
			t.Errorf("Unexpected language %s in filtered courses", c.Language)
		}
	}
}

func TestNormalizeLang(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"en-US", "en"},
		{"english", "en"},
		{"English", "en"},
		{"es", "es"},
		{"ES", "es"},
		{"es-MX", "es"},
		{"spanish", "es"},
		{"español", "es"},
		{"espanol", "es"},
		{"pt", "pt"},
		{"PT", "pt"},
		{"pt-BR", "pt"},
		{"portuguese", "pt"},
		{"português", "pt"},
		{"portugues", "pt"},
		{"fr", "fr"},
		{"FR", "fr"},
		{"fr-FR", "fr"},
		{"french", "fr"},
		{"", ""},
		{"  ", ""},
		{"de_DE", "de"},
	}

	for _, tc := range testCases {
		result := normalizeLang(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeLang(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
