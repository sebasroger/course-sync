package eightfold

import (
	"context"
	"net/http"
	"net/http/httptest"
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
