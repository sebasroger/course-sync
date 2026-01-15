package udemy

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	client := New("https://api.udemy.com", "test-id", "test-secret")

	if client.BaseURL != "https://api.udemy.com" {
		t.Errorf("Expected BaseURL to be 'https://api.udemy.com', got '%s'", client.BaseURL)
	}

	if client.ClientId != "test-id" {
		t.Errorf("Expected ClientId to be 'test-id', got '%s'", client.ClientId)
	}

	if client.ClientSecret != "test-secret" {
		t.Errorf("Expected ClientSecret to be 'test-secret', got '%s'", client.ClientSecret)
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
