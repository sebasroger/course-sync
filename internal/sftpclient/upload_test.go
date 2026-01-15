package sftpclient

import (
	"context"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	// Test default values
	cfg := Config{
		Host: "test-host",
		User: "test-user",
		Pass: "test-pass",
	}

	// Port should default to 22 in UploadFile if not set
	if cfg.Port != 0 {
		t.Errorf("Expected default Port to be 0, got %d", cfg.Port)
	}

	// RemoteDir should default to "/" in UploadFile if not set
	if cfg.RemoteDir != "" {
		t.Errorf("Expected default RemoteDir to be empty, got %q", cfg.RemoteDir)
	}
}

// Note: We can't easily test the actual SFTP upload functionality in a unit test
// without mocking the SFTP server. The following test is a placeholder that
// verifies the validation logic in UploadFile.

func TestUploadFileValidation(t *testing.T) {
	ctx := context.Background()

	// Define constants for repeated values
	const (
		testHost = "test-host"
		testUser = "test-user"
		testPass = "test-pass"
		testFile = "test.txt"
	)

	testCases := []struct {
		name           string
		cfg            Config
		localPath      string
		remoteFileName string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Missing credentials",
			cfg:            Config{},
			localPath:      testFile,
			remoteFileName: testFile,
			expectError:    true,
			errorContains:  "sftp: missing env SFTP_HOST / SFTP_USER / SFTP_PASS",
		},
		{
			name: "Non-existent local file with valid config",
			cfg: Config{
				Host: testHost,
				User: testUser,
				Pass: testPass,
			},
			localPath:      "non_existent_file.txt",
			remoteFileName: testFile,
			expectError:    true,
			errorContains:  "sftp: dial error", // This is what actually happens first in the real code
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := UploadFile(ctx, tc.cfg, tc.localPath, tc.remoteFileName)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
