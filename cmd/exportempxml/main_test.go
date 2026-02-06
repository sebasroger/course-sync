package main

import (
	"testing"
)

func TestPickString(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		keys     []string
		expected string
	}{
		{
			name: "First key exists",
			input: map[string]any{
				"employee_id": "EMP123",
				"user_id":     "USER456",
			},
			keys:     []string{"employee_id", "user_id"},
			expected: "EMP123",
		},
		{
			name: "Second key exists",
			input: map[string]any{
				"user_id": "USER456",
			},
			keys:     []string{"employee_id", "user_id"},
			expected: "USER456",
		},
		{
			name: "No keys exist",
			input: map[string]any{
				"name": "John Doe",
			},
			keys:     []string{"employee_id", "user_id"},
			expected: "",
		},
		{
			name: "Value is nil",
			input: map[string]any{
				"employee_id": nil,
				"user_id":     "USER456",
			},
			keys:     []string{"employee_id", "user_id"},
			expected: "USER456",
		},
		{
			name: "Value needs trimming",
			input: map[string]any{
				"employee_id": "  EMP123  ",
			},
			keys:     []string{"employee_id"},
			expected: "EMP123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pickString(tc.input, tc.keys...)
			if result != tc.expected {
				t.Errorf("pickString(%v, %v) = %q, want %q", tc.input, tc.keys, result, tc.expected)
			}
		})
	}
}

func TestPickEmails(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{
			name: "Single email as string",
			input: map[string]any{
				"email": "user@example.com",
			},
			expected: []string{"user@example.com"},
		},
		{
			name: "Multiple emails as slice",
			input: map[string]any{
				"emails": []any{"user1@example.com", "user2@example.com"},
			},
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "Email in map",
			input: map[string]any{
				"emails": []any{
					map[string]any{"email": "user@example.com"},
				},
			},
			expected: []string{"user@example.com"},
		},
		{
			name: "No emails",
			input: map[string]any{
				"name": "John Doe",
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pickEmails(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("pickEmails(%v) returned %d emails, want %d", tc.input, len(result), len(tc.expected))
				return
			}

			// Check each email
			for i, email := range result {
				if i >= len(tc.expected) || email != tc.expected[i] {
					t.Errorf("pickEmails(%v) = %v, want %v", tc.input, result, tc.expected)
					return
				}
			}
		})
	}
}

func TestAnyToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "String input",
			input:    "test",
			expected: "test",
		},
		{
			name:     "Integer input",
			input:    123,
			expected: "123",
		},
		{
			name:     "Boolean input",
			input:    true,
			expected: "true",
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: "<nil>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := anyToString(tc.input)
			if result != tc.expected {
				t.Errorf("anyToString(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestAnyToStringSlice(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "Single string",
			input:    "test",
			expected: []string{"test"},
		},
		{
			name:     "Array of strings",
			input:    []any{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "Array with empty strings",
			input:    []any{"test1", "", "  ", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name: "Map with email",
			input: map[string]any{
				"email": "user@example.com",
			},
			expected: []string{"user@example.com"},
		},
		{
			name: "Array with maps",
			input: []any{
				map[string]any{"email": "user1@example.com"},
				map[string]any{"email": "user2@example.com"},
			},
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "Duplicate emails",
			input:    []any{"user@example.com", "user@example.com"},
			expected: []string{"user@example.com"},
		},
		{
			name: "Map with data array",
			input: map[string]any{
				"data": []any{"user1@example.com", "user2@example.com"},
			},
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "Mixed types in array",
			input: []any{
				"user1@example.com",
				nil,
				123,
				map[string]any{"email": "user2@example.com"},
				map[string]any{"name": "John Doe"},
			},
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace string",
			input:    "   ",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := anyToStringSlice(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("anyToStringSlice(%v) returned %d items, want %d", tc.input, len(result), len(tc.expected))
				return
			}

			// Check each item
			for i, item := range result {
				if i >= len(tc.expected) || item != tc.expected[i] {
					t.Errorf("anyToStringSlice(%v) = %v, want %v", tc.input, result, tc.expected)
					return
				}
			}
		})
	}
}
