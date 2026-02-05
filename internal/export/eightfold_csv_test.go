package export

import (
	"course-sync/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemID(t *testing.T) {
	testCases := []struct {
		source   string
		sourceID string
		expected string
	}{
		{"udemy", "12345", "UDM+12345"},
		{"UDEMY", "12345", "UDM+12345"},
		{"pluralsight", "abc123", "PLS+abc123"},
		{"PLURALSIGHT", "abc123", "PLS+abc123"},
		{"coursera", "xyz789", "COURSERA+xyz789"},
		{"", "12345", "SRC+12345"},
	}

	for _, tc := range testCases {
		result := buildSystemID(tc.source, tc.sourceID)
		if result != tc.expected {
			t.Errorf("buildSystemID(%q, %q) = %q, want %q", tc.source, tc.sourceID, result, tc.expected)
		}
	}
}

func TestFloatToString(t *testing.T) {
	testCases := []struct {
		input    float64
		expected string
	}{
		{1.5, "1.5"},
		{2.0, "2"},
		{0.0, "0"},
		{3.14159, "3.14159"},
	}

	for _, tc := range testCases {
		result := floatToString(tc.input)
		if result != tc.expected {
			t.Errorf("floatToString(%f) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"test", "test"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := firstNonEmpty(tc.input)
		if result != tc.expected {
			t.Errorf("firstNonEmpty(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestWriteEightfoldCourseCSV(t *testing.T) {
	// Create test courses
	courses := []domain.UnifiedCourse{
		{
			Source:        "udemy",
			SourceID:      "12345",
			Title:         "Test Course 1",
			Description:   "Description 1",
			CourseURL:     "https://example.com/course/1",
			DurationHours: 2.5,
			Category:      "Technology",
			ImageURL:      "https://example.com/image1.jpg",
			Language:      "en",
			PublishedDate: "2023-01-01",
			Difficulty:    "Beginner",
			Status:        "active",
		},
		{
			Source:        "pluralsight",
			SourceID:      "67890",
			Title:         "Test Course 2",
			Description:   "Description 2",
			CourseURL:     "https://example.com/course/2",
			DurationHours: 3.0,
			Category:      "Development",
			ImageURL:      "https://example.com/image2.jpg",
			Language:      "es",
			PublishedDate: "2023-02-01",
			Difficulty:    "Intermediate",
			Status:        "",
		},
	}

	// Create temporary file for testing
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "test_courses.csv")
	defer os.Remove(tempFile)

	// Create tag configuration
	tagCfg := CourseTagConfig{
		EligibilityTagsFieldName: "eligibility_tags",
		TagsBySource: map[string][]string{
			"udemy":       {"IC1", "IC2"},
			"pluralsight": {"IC5", "IC6"},
		},
	}

	// Write CSV
	err := WriteEightfoldCourseCSV(tempFile, courses, tagCfg)
	if err != nil {
		t.Fatalf("WriteEightfoldCourseCSV() error = %v", err)
	}

	// Read the generated CSV file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read test CSV file: %v", err)
	}

	// Check if the CSV contains expected content
	csvContent := string(content)

	// Check header
	if !strings.Contains(csvContent, "systemId,title,description,courseUrl,durationHours,category,imageUrl,language,publishedDate,lmsCourseId,difficulty,provider,status") {
		t.Error("CSV header is incorrect")
	}

	// Check first course
	if !strings.Contains(csvContent, "UDM+12345,Test Course 1,Description 1,https://example.com/course/1,2.5,Technology,https://example.com/image1.jpg,en,2023-01-01,12345,Beginner,udemy,active") {
		t.Error("First course data is incorrect")
	}

	// Check second course (with default status)
	if !strings.Contains(csvContent, "PLS+67890,Test Course 2,Description 2,https://example.com/course/2,3,Development,https://example.com/image2.jpg,es,2023-02-01,67890,Intermediate,pluralsight,active") {
		t.Error("Second course data is incorrect")
	}
}
