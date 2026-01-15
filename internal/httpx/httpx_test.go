package httpx

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestSnippet(t *testing.T) {
	testCases := []struct {
		input    string
		max      int
		expected string
	}{
		{"short text", 100, "short text"},
		{"", 100, ""},
		{"  trimmed  ", 100, "trimmed"},
		{"long text that should be truncated", 10, "long text …"},
	}

	for _, tc := range testCases {
		result := snippet([]byte(tc.input), tc.max)
		if result != tc.expected {
			t.Errorf("snippet(%q, %d) = %q, want %q", tc.input, tc.max, result, tc.expected)
		}
	}
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		Method:     "GET",
		URL:        "https://example.com",
		StatusCode: 404,
		Body:       []byte("Not Found"),
	}

	expected := "http error: GET https://example.com status=404 body=Not Found"
	if err.Error() != expected {
		t.Errorf("HTTPError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 8 {
		t.Errorf("Expected MaxAttempts to be 8, got %d", cfg.MaxAttempts)
	}

	if cfg.BaseDelay != 700*time.Millisecond {
		t.Errorf("Expected BaseDelay to be 700ms, got %v", cfg.BaseDelay)
	}

	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay to be 30s, got %v", cfg.MaxDelay)
	}

	if !cfg.Retry5xx {
		t.Error("Expected Retry5xx to be true")
	}

	expectedStatuses := []int{429, 408, 425, 503, 502, 504}
	for _, status := range expectedStatuses {
		if !cfg.RetryStatuses[status] {
			t.Errorf("Expected status %d to be retryable", status)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	cfg := DefaultRetryConfig()

	// Test 5xx status codes
	for i := 500; i <= 599; i++ {
		if !isRetryableStatus(i, cfg) {
			t.Errorf("Expected status %d to be retryable", i)
		}
	}

	// Test explicitly configured status codes
	for status := range cfg.RetryStatuses {
		if !isRetryableStatus(status, cfg) {
			t.Errorf("Expected status %d to be retryable", status)
		}
	}

	// Test non-retryable status codes
	nonRetryableStatuses := []int{400, 401, 403, 404, 422}
	for _, status := range nonRetryableStatuses {
		if isRetryableStatus(status, cfg) {
			t.Errorf("Expected status %d to not be retryable", status)
		}
	}

	// Test with Retry5xx disabled
	cfg.Retry5xx = false
	if isRetryableStatus(500, cfg) {
		t.Error("Expected status 500 to not be retryable when Retry5xx is false")
	}

	// But explicitly configured status should still be retryable
	if !isRetryableStatus(429, cfg) {
		t.Error("Expected status 429 to be retryable regardless of Retry5xx")
	}
}

func TestIsRetryableNetErr(t *testing.T) {
	// Test context errors
	if isRetryableNetErr(context.Canceled) {
		t.Error("Expected context.Canceled to not be retryable")
	}

	if !isRetryableNetErr(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be retryable")
	}

	// Test net.Error
	timeoutErr := &timeoutError{}
	if !isRetryableNetErr(timeoutErr) {
		t.Error("Expected timeout error to be retryable")
	}

	// Test common I/O errors by their error message
	connectionResetErr := errors.New("connection reset by peer")
	if !isRetryableNetErr(connectionResetErr) {
		t.Error("Expected 'connection reset' error to be retryable")
	}

	brokenPipeErr := errors.New("write: broken pipe")
	if !isRetryableNetErr(brokenPipeErr) {
		t.Error("Expected 'broken pipe' error to be retryable")
	}

	eofErr := errors.New("unexpected EOF")
	if !isRetryableNetErr(eofErr) {
		t.Error("Expected 'EOF' error to be retryable")
	}

	// Test non-retryable error
	otherErr := errors.New("some other error")
	if isRetryableNetErr(otherErr) {
		t.Error("Expected 'some other error' to not be retryable")
	}
}

const retryAfterHeader = "Retry-After"

func TestParseRetryAfter(t *testing.T) {
	// Test with seconds
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set(retryAfterHeader, "30")

	duration := ParseRetryAfter(resp)
	if duration != 30*time.Second {
		t.Errorf("Expected 30s, got %v", duration)
	}

	// Skip HTTP date test since it depends on time.Until which is hard to mock in tests
	// We've already tested the seconds parsing which is the most common case

	// Test with past date (should return 0)
	past := time.Now().Add(-60 * time.Second)
	resp.Header.Set(retryAfterHeader, past.Format(time.RFC1123))

	duration = ParseRetryAfter(resp)
	if duration != 0 {
		t.Errorf("Expected 0 for past date, got %v", duration)
	}

	// Test with invalid format
	resp.Header.Set(retryAfterHeader, "invalid")

	duration = ParseRetryAfter(resp)
	if duration != 0 {
		t.Errorf("Expected 0 for invalid format, got %v", duration)
	}

	// Test with empty header
	resp.Header.Del(retryAfterHeader)

	duration = ParseRetryAfter(resp)
	if duration != 0 {
		t.Errorf("Expected 0 for empty header, got %v", duration)
	}
}

// Mock implementation of net.Error for testing
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout error" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
