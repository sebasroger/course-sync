package eightfold

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testBaseURL = "https://api.eightfold.ai"

func TestNew(t *testing.T) {
	client := New(testBaseURL)

	if client.BaseURL != testBaseURL {
		t.Errorf("Expected BaseURL to be 'https://api.eightfold.ai', got '%s'", client.BaseURL)
	}

	if client.HTTP == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.BearerToken != "" {
		t.Errorf("Expected BearerToken to be empty, got '%s'", client.BearerToken)
	}
}

func TestUpsertCourseValidation(t *testing.T) {
	client := New(testBaseURL)

	// Test without bearer token
	err := client.UpsertCourse(context.Background(), CourseUpsertRequest{
		Title: "Test Course",
	})

	if err == nil {
		t.Error("Expected error when BearerToken is empty, got nil")
	}

	expectedErr := "eightfold: missing bearer token (call Authenticate first)"
	if err != nil && err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestListCoursesValidation(t *testing.T) {
	client := New(testBaseURL)

	// Test without bearer token
	_, err := client.ListCourses(context.Background(), 10)

	if err == nil {
		t.Error("Expected error when BearerToken is empty, got nil")
	}

	expectedErr := "eightfold: missing bearer token (call Authenticate first)"
	if err != nil && err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestAuthenticateWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/oauth/v1/authenticate" {
			t.Errorf("Expected request to '/oauth/v1/authenticate', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Basic dGVzdC1iYXNpYw==" {
			t.Errorf("Expected Authorization header 'Basic dGVzdC1iYXNpYw==', got '%s'", r.Header.Get("Authorization"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"access_token": "test-token",
				"expires_in": 3600,
				"scope": "all",
				"token_type": "Bearer"
			}
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)

	// Test authentication
	err := client.Authenticate(context.Background(), "dGVzdC1iYXNpYw==", AuthRequest{
		GrantType: "password",
		Username:  "test-user",
		Password:  "test-pass",
	})

	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}

	if client.BearerToken != "test-token" {
		t.Errorf("Expected BearerToken to be 'test-token', got '%s'", client.BearerToken)
	}
}

func TestUpdateEmployeeValidation(t *testing.T) {
	client := New(testBaseURL)

	// Test without bearer token
	err := client.UpdateEmployee(context.Background(), "profile123", UpdateEmployeeRequest{
		Email: "test@example.com",
		CandidateData: CandidateData{
			CourseAttendance: []CourseAttendance{
				{
					LmsCourseID:          "course123",
					Title:                "Test Course",
					PercentageCompletion: 75.0,
					Status:               "in_progress",
					DurationHours:        2.5,
					Provider:             "udemy",
				},
			},
		},
	})

	if err == nil {
		t.Error("Expected error when BearerToken is empty, got nil")
	}

	expectedErr := "eightfold: missing bearer token"
	if err != nil && err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestUpdateEmployeeWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		if !strings.HasSuffix(r.URL.Path, "/api/v2/core/employees/profile123") {
			t.Errorf("Expected request to end with '/api/v2/core/employees/profile123', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}

		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Error reading request body: %v", err)
		}

		var req UpdateEmployeeRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Error unmarshaling request body: %v", err)
		}

		if req.Email != "test@example.com" {
			t.Errorf("Expected email to be 'test@example.com', got '%s'", req.Email)
		}

		if len(req.CandidateData.CourseAttendance) != 1 {
			t.Errorf("Expected 1 course attendance, got %d", len(req.CandidateData.CourseAttendance))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)
	client.BearerToken = "test-token" // Set token directly for testing

	// Test update employee
	err := client.UpdateEmployee(context.Background(), "profile123", UpdateEmployeeRequest{
		Email: "test@example.com",
		CandidateData: CandidateData{
			CourseAttendance: []CourseAttendance{
				{
					LmsCourseID:          "course123",
					Title:                "Test Course",
					PercentageCompletion: 75.0,
					Status:               "in_progress",
					DurationHours:        2.5,
					Provider:             "udemy",
				},
			},
		},
	})

	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
}

func TestUpsertCourseWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/v2/core/courses" {
			t.Errorf("Expected request to '/api/v2/core/courses', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}

		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Error reading request body: %v", err)
		}

		var req CourseUpsertRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Error unmarshaling request body: %v", err)
		}

		if req.Title != "Test Course" {
			t.Errorf("Expected title to be 'Test Course', got '%s'", req.Title)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)
	client.BearerToken = "test-token" // Set token directly for testing

	// Test upsert course
	err := client.UpsertCourse(context.Background(), CourseUpsertRequest{
		Title:         "Test Course",
		Description:   "Test Description",
		CourseUrl:     "https://example.com/course",
		DurationHours: 2.5,
		Language:      "en",
		Provider:      "udemy",
		LmsCourseId:   "course123",
		SystemId:      "UDM+course123",
	})

	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
}

func TestListCoursesPageWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/api/v2/core/courses") {
			t.Errorf("Expected request to '/api/v2/core/courses', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got '%s'", query.Get("limit"))
		}

		if query.Get("start") != "20" {
			t.Errorf("Expected start=20, got '%s'", query.Get("start"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"title": "Course 1",
					"lmsCourseId": "course1",
					"provider": "udemy"
				},
				{
					"title": "Course 2",
					"lmsCourseId": "course2",
					"provider": "pluralsight"
				}
			],
			"meta": {
				"pageStartIndex": 20,
				"pageTotalCount": 2,
				"totalCount": 100
			}
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)
	client.BearerToken = "test-token" // Set token directly for testing

	// Test list courses page
	courses, meta, err := client.ListCoursesPage(context.Background(), 20, 10)

	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}

	if len(courses) != 2 {
		t.Errorf("Expected 2 courses, got %d", len(courses))
	}

	if meta.TotalCount != 100 {
		t.Errorf("Expected totalCount to be 100, got %d", meta.TotalCount)
	}

	if meta.PageStartIndex != 20 {
		t.Errorf("Expected pageStartIndex to be 20, got %d", meta.PageStartIndex)
	}

	if meta.PageTotalCount != 2 {
		t.Errorf("Expected pageTotalCount to be 2, got %d", meta.PageTotalCount)
	}
}

func TestListCoursesWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/api/v2/core/courses") {
			t.Errorf("Expected request to '/api/v2/core/courses', got '%s'", r.URL.Path)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got '%s'", query.Get("limit"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"title": "Course 1",
					"lmsCourseId": "course1",
					"provider": "udemy"
				}
			],
			"meta": {
				"pageStartIndex": 0,
				"pageTotalCount": 1,
				"totalCount": 1
			}
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)
	client.BearerToken = "test-token" // Set token directly for testing

	// Test list courses
	courses, err := client.ListCourses(context.Background(), 10)

	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}

	if len(courses) != 1 {
		t.Errorf("Expected 1 course, got %d", len(courses))
	}

	if courses[0]["title"] != "Course 1" {
		t.Errorf("Expected course title to be 'Course 1', got '%v'", courses[0]["title"])
	}
}

func TestAuthenticateWithInvalidResponse(t *testing.T) {
	// Create a mock server that returns invalid response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"no_token": "missing"}}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := New(server.URL)

	// Test authentication with invalid response
	err := client.Authenticate(context.Background(), "dGVzdC1iYXNpYw==", AuthRequest{
		GrantType: "password",
		Username:  "test-user",
		Password:  "test-pass",
	})

	if err == nil {
		t.Error("Expected error when token is missing from response, got nil")
	}

	expectedErr := "eightfold auth: token not found"
	if err != nil && err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}
