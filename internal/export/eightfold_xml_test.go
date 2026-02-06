package export

import (
	"course-sync/internal/domain"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteEFCourseXML(t *testing.T) {
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
			Language:      "en",
			PublishedDate: "2023-01-01",
			Difficulty:    "Beginner",
			Status:        "active",
			Skills:        []string{"Skill1", "Skill2"},
		},
		{
			Source:        "pluralsight",
			SourceID:      "67890",
			Title:         "Test Course 2",
			Description:   "Description 2",
			CourseURL:     "https://example.com/course/2",
			DurationHours: 3.0,
			Category:      "Development",
			Language:      "es",
			PublishedDate: "2023-02-01",
			Difficulty:    "Intermediate",
			Status:        "",
		},
	}

	// Create temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_courses.xml")

	// Create tag configuration with operation
	tagCfg := CourseTagConfig{
		Operation:                "upsert",
		SystemID:                 "successfactors",
		EligibilityTagsFieldName: "eligibility_tags",
		TagsBySource: map[string][]string{
			"udemy":       {"IC1", "IC2"},
			"pluralsight": {"IC5", "IC6"},
		},
	}

	// Write XML
	err := WriteEFCourseXML(tempFile, courses, tagCfg)
	if err != nil {
		t.Fatalf("WriteEFCourseXML() error = %v", err)
	}

	// Read the generated XML file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Check if the XML contains expected content
	xmlContent := string(content)

	// Check XML header
	if !strings.Contains(xmlContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML header is incorrect")
	}

	// Check course attributes
	if !strings.Contains(xmlContent, "operation=\"upsert\"") {
		t.Error("Operation attribute is missing or incorrect")
	}

	// Check course elements
	if !strings.Contains(xmlContent, "<title>Test Course 1</title>") {
		t.Error("Course title is missing or incorrect")
	}

	if !strings.Contains(xmlContent, "<lms_course_id>12345</lms_course_id>") {
		t.Error("LMS course ID is missing or incorrect")
	}

	// Check skills list
	if !strings.Contains(xmlContent, "<skills_list>") ||
		!strings.Contains(xmlContent, "<skill>Skill1</skill>") {
		t.Error("Skills list is missing or incorrect")
	}

	// Check eligibility tags
	if !strings.Contains(xmlContent, "<field_name>eligibility_tags</field_name>") {
		t.Error("Eligibility tags field is missing or incorrect")
	}

	// Test with single tag (should use custom_info instead of custom_multi_value_list)
	singleTagCfg := CourseTagConfig{
		EligibilityTagsFieldName: "eligibility_tags",
		TagsBySource: map[string][]string{
			"udemy": {"IC1"},
		},
	}

	singleTagFile := filepath.Join(tempDir, "single_tag.xml")
	err = WriteEFCourseXML(singleTagFile, courses[:1], singleTagCfg)
	if err != nil {
		t.Fatalf("WriteEFCourseXML() with single tag error = %v", err)
	}

	singleTagContent, err := os.ReadFile(singleTagFile)
	if err != nil {
		t.Fatalf("Failed to read single tag XML file: %v", err)
	}

	if !strings.Contains(string(singleTagContent), "<custom_info>") {
		t.Error("Single tag should use custom_info element")
	}
}

func TestCompactStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "No duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With empty strings",
			input:    []string{"a", "", "b", " ", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With whitespace",
			input:    []string{" a ", "b", " c "},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := compactStrings(tc.input)
			if !equalStringSlices(result, tc.expected) {
				t.Errorf("compactStrings(%v) = %v, want %v", tc.input, result, tc.expected)
			}
		})
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
		{"de_DE", "de-de"},
	}

	for _, tc := range testCases {
		result := normalizeLang(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeLang(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestWriteEFCourseDeleteXML(t *testing.T) {
	// Create test courses for deletion
	courses := []DeleteCourse{
		{
			Title:       "Test Course 1",
			LMSCourseID: "UDM+12345",
		},
		{
			Title:       "Test Course 2",
			LMSCourseID: "PLS+67890",
		},
	}

	// Create temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_delete.xml")

	// Write XML
	err := WriteEFCourseDeleteXML(tempFile, courses)
	if err != nil {
		t.Fatalf("WriteEFCourseDeleteXML() error = %v", err)
	}

	// Read the generated XML file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Check if the XML contains expected content
	xmlContent := string(content)

	// Check XML header
	if !strings.Contains(xmlContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML header is incorrect")
	}

	// Check course elements
	if !strings.Contains(xmlContent, "<lms_course_id>UDM+12345</lms_course_id>") {
		t.Error("First lms_course_id is missing or incorrect")
	}

	if !strings.Contains(xmlContent, "<lms_course_id>PLS+67890</lms_course_id>") {
		t.Error("Second lms_course_id is missing or incorrect")
	}

	// Test with empty slice
	emptyFile := filepath.Join(tempDir, "empty_delete.xml")
	err = WriteEFCourseDeleteXML(emptyFile, []DeleteCourse{})
	if err != nil {
		t.Fatalf("WriteEFCourseDeleteXML() with empty slice error = %v", err)
	}

	emptyContent, err := os.ReadFile(emptyFile)
	if err != nil {
		t.Fatalf("Failed to read empty XML file: %v", err)
	}

	// Empty file should still have valid XML structure
	var emptyList struct{}
	err = xml.Unmarshal(emptyContent, &emptyList)
	if err != nil {
		t.Errorf("Empty XML file is not valid XML: %v", err)
	}
}

// Helper function to compare string slices regardless of order
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]int)
	for _, s := range a {
		aMap[s]++
	}

	for _, s := range b {
		aMap[s]--
		if aMap[s] < 0 {
			return false
		}
	}

	return true
}
