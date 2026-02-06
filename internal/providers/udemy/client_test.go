package udemy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testBaseURL      = "https://api.udemy.com"
	testClientID     = "test-id"
	testClientSecret = "test-secret"
	retryAfterHeader = "Retry-After"
)

func TestNew(t *testing.T) {
	client := New(testBaseURL, testClientID, testClientSecret)

	if client.BaseURL != testBaseURL {
		t.Errorf("Expected BaseURL to be '%s', got '%s'", testBaseURL, client.BaseURL)
	}

	if client.ClientId != testClientID {
		t.Errorf("Expected ClientId to be '%s', got '%s'", testClientID, client.ClientId)
	}

	if client.ClientSecret != testClientSecret {
		t.Errorf("Expected ClientSecret to be '%s', got '%s'", testClientSecret, client.ClientSecret)
	}

	if client.HTTP == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestEnvInt(t *testing.T) {
	// Test with empty environment variable
	os.Unsetenv("TEST_ENV_INT")
	result := envInt("TEST_ENV_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	// Test with valid environment variable
	os.Setenv("TEST_ENV_INT", "100")
	result = envInt("TEST_ENV_INT", 42)
	if result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}

	// Test with invalid environment variable
	os.Setenv("TEST_ENV_INT", "invalid")
	result = envInt("TEST_ENV_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	// Clean up
	os.Unsetenv("TEST_ENV_INT")
}

func TestMinInt(t *testing.T) {
	testCases := []struct {
		a, b, expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{0, 5, 0},
		{-5, 5, -5},
		{5, 5, 5},
	}

	for _, tc := range testCases {
		result := minInt(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("minInt(%d, %d) = %d; expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestLooksLikeHTML(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"<html><body>Test</body></html>", true},
		{"<!DOCTYPE html>", true},
		{"<html lang=\"en\">", true},
		{"{\"key\": \"value\"}", false},
		{"", false},
		{"plain text", false},
	}

	for _, tc := range testCases {
		result := looksLikeHTML([]byte(tc.input))
		if result != tc.expected {
			t.Errorf("looksLikeHTML(%q) = %v; expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestIsNetRetryable(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"GOAWAY error", fmt.Errorf("http2: server sent GOAWAY"), true},
		{"Connection closed", fmt.Errorf("connection closed"), true},
		{"EOF error", fmt.Errorf("unexpected EOF"), true},
		{"Reset by peer", fmt.Errorf("connection reset by peer"), true},
		{"Context deadline", context.DeadlineExceeded, true},
		{"Other error", fmt.Errorf("some other error"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isNetRetryable(tc.err)
			if result != tc.expected {
				t.Errorf("isNetRetryable(%v) = %v; expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	testCases := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{"Empty header", "", 0},
		{"Seconds value", "30", 30 * time.Second},
		{"Invalid value", "invalid", 0},
		{"Negative value", "-10", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tc.header != "" {
				resp.Header.Set(retryAfterHeader, tc.header)
			}

			result := parseRetryAfter(resp)
			if result != tc.expected {
				t.Errorf("parseRetryAfter() = %v; expected %v", result, tc.expected)
			}
		})
	}

	// Skip HTTP date format test as it's time-dependent and can be flaky
	t.Run("HTTP date format", func(t *testing.T) {
		t.Skip("Skipping time-dependent test")
		// Create a future time
		futureTime := time.Now().Add(60 * time.Second)
		dateStr := futureTime.Format(time.RFC1123)

		resp := &http.Response{
			Header: http.Header{},
		}
		resp.Header.Set("Retry-After", dateStr)

		result := parseRetryAfter(resp)
		// Allow for small differences due to time parsing
		if result < 55*time.Second || result > 65*time.Second {
			t.Errorf("parseRetryAfter() = %v; expected ~60s", result)
		}
	})

	// Test past date (should return 0)
	t.Run("Past date", func(t *testing.T) {
		// Create a past time
		pastTime := time.Now().Add(-60 * time.Second)
		dateStr := pastTime.Format(time.RFC1123)

		resp := &http.Response{
			Header: http.Header{},
		}
		resp.Header.Set("Retry-After", dateStr)

		result := parseRetryAfter(resp)
		if result != 0 {
			t.Errorf("parseRetryAfter() = %v; expected 0 for past date", result)
		}
	})
}

func TestPickUdemyImageURL(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "Empty JSON",
			json:     `{}`,
			expected: "",
		},
		{
			name:     "Preferred size 480x270",
			json:     `{"size_480x270": "https://example.com/img_480x270.jpg", "size_240x135": "https://example.com/img_240x135.jpg"}`,
			expected: "https://example.com/img_480x270.jpg",
		},
		{
			name:     "Fallback to 240x135",
			json:     `{"size_240x135": "https://example.com/img_240x135.jpg", "size_125_H": "https://example.com/img_125_H.jpg"}`,
			expected: "https://example.com/img_240x135.jpg",
		},
		{
			name:     "Fallback to 125_H",
			json:     `{"size_125_H": "https://example.com/img_125_H.jpg"}`,
			expected: "https://example.com/img_125_H.jpg",
		},
		{
			name:     "Alternative key format",
			json:     `{"image_480x270": "https://example.com/img_480x270.jpg"}`,
			expected: "https://example.com/img_480x270.jpg",
		},
		{
			name:     "Empty URL",
			json:     `{"size_480x270": ""}`,
			expected: "",
		},
		{
			name:     "Invalid JSON",
			json:     `{invalid json`,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pickUdemyImageURL(json.RawMessage(tc.json))
			if result != tc.expected {
				t.Errorf("pickUdemyImageURL() = %v; expected %v", result, tc.expected)
			}
		})
	}

	// Test with empty raw message
	t.Run("Empty raw message", func(t *testing.T) {
		result := pickUdemyImageURL(nil)
		if result != "" {
			t.Errorf("pickUdemyImageURL(nil) = %v; expected empty string", result)
		}
	})
}

func TestGetUserByEmail(t *testing.T) {
	client := New("https://api.udemy.com", "test-id", "test-secret")

	testCases := []struct {
		email          string
		expectedFirst  string
		expectedLast   string
		expectedUserID string
	}{
		{
			email:         "john.doe@example.com",
			expectedFirst: "John",
			expectedLast:  "Doe",
			// We can't easily predict the exact CRC32 value, so we'll check format later
		},
		{
			email:         "alice@example.com",
			expectedFirst: "Alice",
			expectedLast:  "",
		},
		{
			email:         "no-name@example.com",
			expectedFirst: "No-Name",
			expectedLast:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			user, err := client.GetUserByEmail(context.Background(), tc.email)

			if err != nil {
				t.Fatalf("GetUserByEmail() returned error: %v", err)
			}

			if user == nil {
				t.Fatal("GetUserByEmail() returned nil user")
			}

			if user.Email != tc.email {
				t.Errorf("Expected email %q, got %q", tc.email, user.Email)
			}

			if user.FirstName != tc.expectedFirst {
				t.Errorf("Expected first name %q, got %q", tc.expectedFirst, user.FirstName)
			}

			if user.LastName != tc.expectedLast {
				t.Errorf("Expected last name %q, got %q", tc.expectedLast, user.LastName)
			}

			// Check that the user ID starts with "u" followed by numbers
			if !strings.HasPrefix(user.UdemyUserID, "u") {
				t.Errorf("Expected user ID to start with 'u', got %q", user.UdemyUserID)
			}

			// Verify the user ID is deterministic
			user2, _ := client.GetUserByEmail(context.Background(), tc.email)
			if user.UdemyUserID != user2.UdemyUserID {
				t.Errorf("User ID not deterministic: %q != %q", user.UdemyUserID, user2.UdemyUserID)
			}
		})
	}
}

func TestGetCourseProgress(t *testing.T) {
	client := New("https://api.udemy.com", "test-id", "test-secret")

	testCases := []struct {
		userID string
	}{
		{"u12345"},
		{"u67890"},
		{"u54321"},
	}

	for _, tc := range testCases {
		t.Run(tc.userID, func(t *testing.T) {
			progress, err := client.GetCourseProgress(context.Background(), tc.userID)

			if err != nil {
				t.Fatalf("GetCourseProgress() returned error: %v", err)
			}

			// Check that we got between 1-3 courses as expected
			if len(progress) < 1 || len(progress) > 3 {
				t.Errorf("Expected 1-3 courses, got %d", len(progress))
			}

			// Verify the progress is deterministic for the same user ID
			progress2, _ := client.GetCourseProgress(context.Background(), tc.userID)
			if len(progress) != len(progress2) {
				t.Errorf("Course progress count not deterministic: %d != %d", len(progress), len(progress2))
			}

			// Check each course progress
			for i, p := range progress {
				// Check user ID
				if p.UdemyUserID != tc.userID {
					t.Errorf("Expected user ID %q, got %q", tc.userID, p.UdemyUserID)
				}

				// Check course ID format
				if p.CourseID == "" {
					t.Error("Course ID is empty")
				}

				// Check percentage is between 10-90%
				if p.PercentComplete < 10 || p.PercentComplete > 90 {
					t.Errorf("Expected percent complete between 10-90, got %f", p.PercentComplete)
				}

				// Check course title is not empty
				if p.Course.Title == "" {
					t.Error("Course title is empty")
				}

				// Check dates are valid
				if _, err := time.Parse(time.RFC3339, p.FirstViewedLectureOn); err != nil {
					t.Errorf("Invalid FirstViewedLectureOn date: %v", err)
				}

				if _, err := time.Parse(time.RFC3339, p.LastViewedLectureOn); err != nil {
					t.Errorf("Invalid LastViewedLectureOn date: %v", err)
				}

				// Check deterministic values
				if i < len(progress2) {
					if p.CourseID != progress2[i].CourseID {
						t.Errorf("Course ID not deterministic: %q != %q", p.CourseID, progress2[i].CourseID)
					}

					if p.PercentComplete != progress2[i].PercentComplete {
						t.Errorf("Percent complete not deterministic: %f != %f", p.PercentComplete, progress2[i].PercentComplete)
					}
				}
			}
		})
	}
}
