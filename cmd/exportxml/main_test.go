package main

import (
	"course-sync/internal/domain"
	"reflect"
	"testing"
)

func TestFilterCoursesByLang(t *testing.T) {
	// Define test courses
	courses := []domain.UnifiedCourse{
		{
			Source:   "udemy",
			SourceID: "1",
			Title:    "Spanish Course",
			Language: "es",
		},
		{
			Source:   "udemy",
			SourceID: "2",
			Title:    "English Course",
			Language: "en",
		},
		{
			Source:   "pluralsight",
			SourceID: "3",
			Title:    "Portuguese Course",
			Language: "pt",
		},
		{
			Source:   "pluralsight",
			SourceID: "4",
			Title:    "French Course",
			Language: "fr",
		},
	}

	// Define allowed languages
	allowed := map[string]bool{
		"es": true,
		"en": true,
	}

	// Filter courses
	filtered := filterCoursesByLang(courses, allowed)

	// Check results
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
		{"en_US", "en"},
		{"english", "en"},
		{"English", "en"},
		{"es", "es"},
		{"ES", "es"},
		{"es-MX", "es"},
		{"es_MX", "es"},
		{"spanish", "es"},
		{"español", "es"},
		{"espanol", "es"},
		{"pt", "pt"},
		{"PT", "pt"},
		{"pt-BR", "pt"},
		{"pt_BR", "pt"},
		{"portuguese", "pt"},
		{"português", "pt"},
		{"portugues", "pt"},
		{"fr", "fr"},
		{"FR", "fr"},
		{"fr-FR", "fr-fr"},
		{"", ""},
		{"  ", ""},
	}

	for _, tc := range testCases {
		result := normalizeLang(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeLang(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"a,,c", []string{"a", "c"}},
		{"a, ,c", []string{"a", "c"}},
		{"", []string{}},
		{" , , ", []string{}},
		{"IC1,IC2,IC3,IC4", []string{"IC1", "IC2", "IC3", "IC4"}},
	}

	for _, tc := range testCases {
		result := splitCSV(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("splitCSV(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}
