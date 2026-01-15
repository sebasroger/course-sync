package providers

import (
	"context"
	"course-sync/internal/domain"
	"testing"
)

// MockProvider is a mock implementation of the CourseProvider interface for testing
type MockProvider struct {
	NameFunc        func() string
	ListCoursesFunc func(ctx context.Context) ([]domain.UnifiedCourse, error)
}

func (m *MockProvider) Name() string {
	return m.NameFunc()
}

func (m *MockProvider) ListCourses(ctx context.Context) ([]domain.UnifiedCourse, error) {
	return m.ListCoursesFunc(ctx)
}

func TestProviders(t *testing.T) {
	// Test with a mock provider
	mockProvider := &MockProvider{
		NameFunc: func() string {
			return "mock-provider"
		},
		ListCoursesFunc: func(ctx context.Context) ([]domain.UnifiedCourse, error) {
			return []domain.UnifiedCourse{
				{
					Source:        "mock",
					SourceID:      "123",
					Title:         "Mock Course",
					Description:   "This is a mock course",
					CourseURL:     "https://example.com/mock",
					DurationHours: 1.5,
				},
			}, nil
		},
	}

	// Verify the mock provider implements the CourseProvider interface
	var _ CourseProvider = (*MockProvider)(nil)

	// Test the mock provider
	ctx := context.Background()

	// Test Name method
	name := mockProvider.Name()
	if name != "mock-provider" {
		t.Errorf("Expected name to be 'mock-provider', got %q", name)
	}

	// Test ListCourses method
	courses, err := mockProvider.ListCourses(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(courses) != 1 {
		t.Fatalf("Expected 1 course, got %d", len(courses))
	}

	course := courses[0]
	if course.Source != "mock" {
		t.Errorf("Expected Source to be 'mock', got %q", course.Source)
	}

	if course.SourceID != "123" {
		t.Errorf("Expected SourceID to be '123', got %q", course.SourceID)
	}

	if course.Title != "Mock Course" {
		t.Errorf("Expected Title to be 'Mock Course', got %q", course.Title)
	}
}
