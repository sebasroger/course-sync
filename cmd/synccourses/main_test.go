package main

import (
	"course-sync/internal/domain"
	syncx "course-sync/internal/sync"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Constants for test data
const (
	udemyCourse1        = "Udemy Course 1"
	pluralSightCourse1  = "PluralSight Course 1"
	udemyJSONFile       = "udemy.json"
	pluralSightJSONFile = "pluralsight.json"
	eightfoldJSONFile   = "eightfold.json"
)

func TestSplitCSV(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"IC1,IC2,IC3,IC4", []string{"IC1", "IC2", "IC3", "IC4"}},
		{"IC1, IC2, IC3, IC4", []string{"IC1", "IC2", "IC3", "IC4"}},
		{"IC1,,IC3,IC4", []string{"IC1", "IC3", "IC4"}},
		{"IC1, ,IC3,IC4", []string{"IC1", "IC3", "IC4"}},
		{"", []string{}},
		{" , , ", []string{}},
	}

	for _, tc := range testCases {
		result := splitCSV(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("splitCSV(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestLoadFromMocks(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create test data
	udemyCourses := []domain.UnifiedCourse{
		{
			Source:   "udemy",
			SourceID: "1",
			Title:    "Udemy Course 1",
		},
		{
			Source:   "udemy",
			SourceID: "2",
			Title:    "Udemy Course 2",
		},
	}

	pluralSightCourses := []domain.UnifiedCourse{
		{
			Source:   "pluralsight",
			SourceID: "3",
			Title:    pluralSightCourse1,
		},
		{
			Source:   "pluralsight",
			SourceID: "4",
			Title:    "PluralSight Course 2",
		},
	}

	efCourses := []syncx.EFCourse{
		{
			SystemID: "UDM+1",
			Title:    "Udemy Course 1",
		},
		{
			SystemID: "PLS+3",
			Title:    pluralSightCourse1,
		},
	}

	// Write test files
	writeTestJSON(t, filepath.Join(tempDir, udemyJSONFile), udemyCourses)
	writeTestJSON(t, filepath.Join(tempDir, pluralSightJSONFile), pluralSightCourses)
	writeTestJSON(t, filepath.Join(tempDir, eightfoldJSONFile), efCourses)

	// Test loadFromMocks
	providerCourses, efCoursesResult, err := loadFromMocks(tempDir)
	if err != nil {
		t.Fatalf("loadFromMocks returned error: %v", err)
	}

	// Check provider courses
	expectedProviderCount := len(udemyCourses) + len(pluralSightCourses)
	if len(providerCourses) != expectedProviderCount {
		t.Errorf("Expected %d provider courses, got %d", expectedProviderCount, len(providerCourses))
	}

	// Check Eightfold courses
	if len(efCoursesResult) != len(efCourses) {
		t.Errorf("Expected %d Eightfold courses, got %d", len(efCourses), len(efCoursesResult))
	}

	// Test error case - non-existent directory
	_, _, err = loadFromMocks("/non/existent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestWriteSnapshots(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create test data
	providerCourses := []domain.UnifiedCourse{
		{
			Source:   "udemy",
			SourceID: "1",
			Title:    "Udemy Course 1",
		},
		{
			Source:   "pluralsight",
			SourceID: "2",
			Title:    pluralSightCourse1,
		},
		{
			Source:   "unknown",
			SourceID: "3",
			Title:    "Unknown Course",
		},
	}

	efCourses := []syncx.EFCourse{
		{
			SystemID: "UDM+1",
			Title:    "Udemy Course 1",
		},
	}

	// Test writeSnapshots
	err := writeSnapshots(tempDir, providerCourses, efCourses)
	if err != nil {
		t.Fatalf("writeSnapshots returned error: %v", err)
	}

	// Verify files were created
	files := []string{"udemy.json", "pluralsight.json", "eightfold.json"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", path)
		}
	}

	// Verify file contents
	var udemyCourses []domain.UnifiedCourse
	readTestJSON(t, filepath.Join(tempDir, udemyJSONFile), &udemyCourses)
	if len(udemyCourses) != 1 || udemyCourses[0].SourceID != "1" {
		t.Errorf("Unexpected udemy.json content: %+v", udemyCourses)
	}

	var psCourses []domain.UnifiedCourse
	readTestJSON(t, filepath.Join(tempDir, pluralSightJSONFile), &psCourses)
	if len(psCourses) != 1 || psCourses[0].SourceID != "2" {
		t.Errorf("Unexpected pluralsight.json content: %+v", psCourses)
	}

	var efCoursesResult []syncx.EFCourse
	readTestJSON(t, filepath.Join(tempDir, eightfoldJSONFile), &efCoursesResult)
	if len(efCoursesResult) != 1 || efCoursesResult[0].SystemID != "UDM+1" {
		t.Errorf("Unexpected eightfold.json content: %+v", efCoursesResult)
	}
}

// TestFetchProviders is commented out because it requires mocking the Provider interfaces
// which would be complex. In a real-world scenario, you would use a mocking framework
// or create mock implementations of the Provider interfaces.
//
// func TestFetchProviders(t *testing.T) {
// 	// This would require proper mocking of the Provider interfaces
// }

// Helper functions
func writeTestJSON(t *testing.T, path string, data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}
}

func readTestJSON(t *testing.T, path string, v interface{}) {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}
}
