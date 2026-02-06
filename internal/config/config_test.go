package config

import (
	"os"
	"testing"
)

func TestGetenv(t *testing.T) {
	// Test with empty environment variable
	os.Unsetenv("TEST_GETENV")
	result := getenv("TEST_GETENV", "default")
	if result != "default" {
		t.Errorf("Expected default value 'default', got '%s'", result)
	}

	// Test with set environment variable
	os.Setenv("TEST_GETENV", "test-value")
	result = getenv("TEST_GETENV", "default")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}

	// Clean up
	os.Unsetenv("TEST_GETENV")
}

func TestGetenvInt(t *testing.T) {
	// Test with empty environment variable
	os.Unsetenv("TEST_GETENV_INT")
	result := getenvInt("TEST_GETENV_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	// Test with valid integer
	os.Setenv("TEST_GETENV_INT", "100")
	result = getenvInt("TEST_GETENV_INT", 42)
	if result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("TEST_GETENV_INT", "not-an-int")
	result = getenvInt("TEST_GETENV_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	// Clean up
	os.Unsetenv("TEST_GETENV_INT")
}

func TestGetenvBool(t *testing.T) {
	// Test with empty environment variable
	os.Unsetenv("TEST_GETENV_BOOL")
	result := getenvBool("TEST_GETENV_BOOL", true)
	if result != true {
		t.Errorf("Expected default value true, got %v", result)
	}

	// Test with valid boolean (true)
	os.Setenv("TEST_GETENV_BOOL", "true")
	result = getenvBool("TEST_GETENV_BOOL", false)
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	// Test with valid boolean (false)
	os.Setenv("TEST_GETENV_BOOL", "false")
	result = getenvBool("TEST_GETENV_BOOL", true)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}

	// Test with invalid boolean
	os.Setenv("TEST_GETENV_BOOL", "not-a-bool")
	result = getenvBool("TEST_GETENV_BOOL", true)
	if result != true {
		t.Errorf("Expected default value true, got %v", result)
	}

	// Clean up
	os.Unsetenv("TEST_GETENV_BOOL")
}

func TestLoad(t *testing.T) {
	// Save original environment
	origEnv := make(map[string]string)
	envVars := []string{
		"EIGHTFOLD_BASE_URL", "EIGHTFOLD_BASIC_AUTH", "EIGHTFOLD_USERNAME",
		"EIGHTFOLD_PASSWORD", "EIGHTFOLD_BEARER_TOKEN", "UDEMY_BASE_URL",
		"UDEMY_CLIENT_ID", "UDEMY_CLIENT_SECRET", "PLURALSIGHT_GQL_URL",
		"PLURALSIGHT_TOKEN", "SFTP_HOST", "SFTP_PORT", "SFTP_USER",
		"SFTP_PASS", "SFTP_DIR", "SFTP_INSECURE_IGNORE_HOSTKEY",
	}

	for _, env := range envVars {
		origEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	// Set test environment variables
	os.Setenv("EIGHTFOLD_BASE_URL", "https://eightfold.test")
	os.Setenv("EIGHTFOLD_BASIC_AUTH", "basic-auth")
	os.Setenv("EIGHTFOLD_USERNAME", "user")
	os.Setenv("EIGHTFOLD_PASSWORD", "pass")
	os.Setenv("EIGHTFOLD_BEARER_TOKEN", "bearer-token")
	os.Setenv("UDEMY_BASE_URL", "https://udemy.test")
	os.Setenv("UDEMY_CLIENT_ID", "client-id")
	os.Setenv("UDEMY_CLIENT_SECRET", "client-secret")
	os.Setenv("PLURALSIGHT_GQL_URL", "https://pluralsight.test")
	os.Setenv("PLURALSIGHT_TOKEN", "token")
	os.Setenv("SFTP_HOST", "sftp.test")
	os.Setenv("SFTP_PORT", "2222")
	os.Setenv("SFTP_USER", "sftp-user")
	os.Setenv("SFTP_PASS", "sftp-pass")
	os.Setenv("SFTP_DIR", "/test-upload")
	os.Setenv("SFTP_INSECURE_IGNORE_HOSTKEY", "false")

	// Test Load function
	cfg := Load()

	// Verify loaded values
	if cfg.EightfoldBaseURL != "https://eightfold.test" {
		t.Errorf("Expected EightfoldBaseURL to be 'https://eightfold.test', got '%s'", cfg.EightfoldBaseURL)
	}
	if cfg.UdemyBaseURL != "https://udemy.test" {
		t.Errorf("Expected UdemyBaseURL to be 'https://udemy.test', got '%s'", cfg.UdemyBaseURL)
	}
	if cfg.SFTPPort != 2222 {
		t.Errorf("Expected SFTPPort to be 2222, got %d", cfg.SFTPPort)
	}
	if cfg.SFTPInsecureIgnoreHostKey != false {
		t.Errorf("Expected SFTPInsecureIgnoreHostKey to be false, got %v", cfg.SFTPInsecureIgnoreHostKey)
	}

	// Test default values
	os.Unsetenv("SFTP_PORT")
	os.Unsetenv("SFTP_DIR")
	os.Unsetenv("SFTP_INSECURE_IGNORE_HOSTKEY")

	cfg = Load()
	if cfg.SFTPPort != 22 {
		t.Errorf("Expected default SFTPPort to be 22, got %d", cfg.SFTPPort)
	}
	if cfg.SFTPDir != "/inbound" {
		t.Errorf("Expected default SFTPDir to be '/inbound', got '%s'", cfg.SFTPDir)
	}
	if cfg.SFTPInsecureIgnoreHostKey != true {
		t.Errorf("Expected default SFTPInsecureIgnoreHostKey to be true, got %v", cfg.SFTPInsecureIgnoreHostKey)
	}

	// Restore original environment
	for env, val := range origEnv {
		if val != "" {
			os.Setenv(env, val)
		} else {
			os.Unsetenv(env)
		}
	}
}
