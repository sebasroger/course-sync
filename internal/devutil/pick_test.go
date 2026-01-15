package devutil

import (
	"reflect"
	"testing"
)

const (
	testName = "John Doe"
)

func TestPick(t *testing.T) {
	type testStruct struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Email   string `json:"email"`
		Address string `json:"address"`
	}

	testCases := []struct {
		name     string
		input    any
		keys     []string
		expected map[string]any
	}{
		{
			name: "Pick from struct",
			input: testStruct{
				Name:    testName,
				Age:     30,
				Email:   "john@example.com",
				Address: "123 Main St",
			},
			keys: []string{"name", "email"},
			expected: map[string]any{
				"name":  testName,
				"email": "john@example.com",
			},
		},
		{
			name: "Pick from map",
			input: map[string]any{
				"name":    "Jane Smith",
				"age":     25,
				"email":   "jane@example.com",
				"address": "456 Oak Ave",
			},
			keys: []string{"name", "age"},
			expected: map[string]any{
				"name": "Jane Smith",
				"age":  float64(25), // JSON unmarshaling converts numbers to float64
			},
		},
		{
			name:     "Pick from nil",
			input:    nil,
			keys:     []string{"name"},
			expected: map[string]any{},
		},
		{
			name:     "Pick with no keys",
			input:    testStruct{Name: testName},
			keys:     []string{},
			expected: map[string]any{},
		},
		{
			name:     "Pick non-existent keys",
			input:    testStruct{Name: testName},
			keys:     []string{"nonexistent"},
			expected: map[string]any{
				// Empty because key doesn't exist
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Pick(tc.input, tc.keys...)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Pick() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestPickPrivate(t *testing.T) {
	// Test the private pick function directly
	result := pick(map[string]any{"a": 1, "b": 2}, "a")
	expected := map[string]any{"a": float64(1)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("pick() = %v, want %v", result, expected)
	}
}
