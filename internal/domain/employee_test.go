package domain

import (
	"reflect"
	"testing"
)

func TestUnifiedEmployee(t *testing.T) {
	// Create a test employee
	employee := UnifiedEmployee{
		EmployeeID: "EMP123",
		UserID:     "USER456",
		Level:      "Senior",
		Emails:     []string{"employee@example.com", "personal@example.com"},
	}

	// Test field values
	if employee.EmployeeID != "EMP123" {
		t.Errorf("Expected EmployeeID to be 'EMP123', got '%s'", employee.EmployeeID)
	}

	if employee.UserID != "USER456" {
		t.Errorf("Expected UserID to be 'USER456', got '%s'", employee.UserID)
	}

	if employee.Level != "Senior" {
		t.Errorf("Expected Level to be 'Senior', got '%s'", employee.Level)
	}

	expectedEmails := []string{"employee@example.com", "personal@example.com"}
	if !reflect.DeepEqual(employee.Emails, expectedEmails) {
		t.Errorf("Expected Emails to be %v, got %v", expectedEmails, employee.Emails)
	}
}
