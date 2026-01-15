package pluralsight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testBaseURL = "https://api.pluralsight.com/graphql"
const testToken = "test-token"

func TestNew(t *testing.T) {
	client := New(testBaseURL, testToken)

	if client.BaseURL != testBaseURL {
		t.Errorf("Expected BaseURL to be '%s', got '%s'", testBaseURL, client.BaseURL)
	}

	if client.Token != testToken {
		t.Errorf("Expected Token to be '%s', got '%s'", testToken, client.Token)
	}

	if client.HTTP == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.HTTP.Timeout != 2*time.Minute {
		t.Errorf("Expected HTTP timeout to be 2 minutes, got %v", client.HTTP.Timeout)
	}
}

func TestListCoursesPageWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer "+testToken {
			t.Errorf("Expected Authorization header 'Bearer %s', got '%s'", testToken, r.Header.Get("Authorization"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"courseCatalog": {
					"totalCount": 2,
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": "cursor123"
					},
					"nodes": [
						{
							"id": "course1",
							"idNum": 1001,
							"slug": "course-one",
							"url": "https://app.pluralsight.com/course1",
							"title": "Course One",
							"level": "Beginner",
							"description": "Description for Course One",
							"shortDescription": "Short description",
							"courseSeconds": 3600,
							"releasedDate": "2023-01-01",
							"displayDate": "2023-01-01",
							"publishedDate": "2023-01-01",
							"language": "en"
						},
						{
							"id": "course2",
							"idNum": 1002,
							"slug": "course-two",
							"url": "https://app.pluralsight.com/course2",
							"title": "Course Two",
							"level": "Intermediate",
							"description": "Description for Course Two",
							"shortDescription": "Short description",
							"courseSeconds": 7200,
							"releasedDate": "2023-02-01",
							"displayDate": "2023-02-01",
							"publishedDate": "2023-02-01",
							"language": "en"
						}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL, testToken)

	// Test listing courses
	ctx := context.Background()
	first := 10
	var after *string = nil

	response, err := client.ListCoursesPage(ctx, first, after)

	// Verify response
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Data.CourseCatalog.TotalCount != 2 {
		t.Errorf("Expected TotalCount to be 2, got %d", response.Data.CourseCatalog.TotalCount)
	}

	if len(response.Data.CourseCatalog.Nodes) != 2 {
		t.Fatalf("Expected 2 courses, got %d", len(response.Data.CourseCatalog.Nodes))
	}

	// Check first course
	course1 := response.Data.CourseCatalog.Nodes[0]
	if course1.ID != "course1" {
		t.Errorf("Expected course ID to be 'course1', got '%s'", course1.ID)
	}

	if course1.Title != "Course One" {
		t.Errorf("Expected course title to be 'Course One', got '%s'", course1.Title)
	}

	if course1.Level != "Beginner" {
		t.Errorf("Expected course level to be 'Beginner', got '%s'", course1.Level)
	}

	// Check second course
	course2 := response.Data.CourseCatalog.Nodes[1]
	if course2.ID != "course2" {
		t.Errorf("Expected course ID to be 'course2', got '%s'", course2.ID)
	}

	if course2.Title != "Course Two" {
		t.Errorf("Expected course title to be 'Course Two', got '%s'", course2.Title)
	}

	if course2.Level != "Intermediate" {
		t.Errorf("Expected course level to be 'Intermediate', got '%s'", course2.Level)
	}
}

func TestListCoursesPageWithErrorResponse(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"courseCatalog": null
			},
			"errors": [
				{
					"message": "Test error message"
				}
			]
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL, testToken)

	// Test listing courses
	ctx := context.Background()
	first := 10
	var after *string = nil

	_, err := client.ListCoursesPage(ctx, first, after)

	// Verify error
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedErrSubstr := "pluralsight gql errors"
	if !contains(err.Error(), expectedErrSubstr) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrSubstr, err.Error())
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
