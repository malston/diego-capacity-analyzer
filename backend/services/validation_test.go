// ABOUTME: Tests for input validation functions
// ABOUTME: Verifies GUID and deployment name validation prevents URL injection

package services

import (
	"strings"
	"testing"
)

// Issue #76: URL Parameter Injection in API Clients
// Defense-in-depth validation for GUIDs and deployment names

func TestValidateGUID_ValidGUIDs(t *testing.T) {
	validGUIDs := []string{
		"12345678-1234-1234-1234-123456789abc",
		"abcdef12-3456-7890-abcd-ef1234567890",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}

	for _, guid := range validGUIDs {
		t.Run(guid, func(t *testing.T) {
			if err := ValidateGUID(guid); err != nil {
				t.Errorf("ValidateGUID(%q) returned error: %v, expected nil", guid, err)
			}
		})
	}
}

func TestValidateGUID_InvalidGUIDs(t *testing.T) {
	tests := []struct {
		name string
		guid string
	}{
		{"path traversal", "../../../admin/users"},
		{"too short", "12345678-1234-1234"},
		{"too long", "12345678-1234-1234-1234-123456789abcdef"},
		{"invalid chars", "12345678-1234-1234-1234-12345678ZZZZ"},
		{"uppercase not allowed", "12345678-1234-1234-1234-123456789ABC"},
		{"missing dashes", "123456781234123412341234567890ab"},
		{"extra dashes", "1234-5678-1234-1234-1234-123456789abc"},
		{"newline injection", "12345678-1234-1234-1234-123456789abc\nmalicious"},
		{"null byte", "12345678-1234-1234-1234-123456789abc\x00"},
		{"empty string", ""},
		{"spaces", "12345678-1234-1234-1234-123456789ab "},
		{"url encoded", "12345678-1234-1234-1234-123456789abc%2F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGUID(tt.guid); err == nil {
				t.Errorf("ValidateGUID(%q) returned nil, expected error", tt.guid)
			}
		})
	}
}

func TestValidateDeploymentName_ValidNames(t *testing.T) {
	validNames := []string{
		"cf-1234567890abcdef",
		"p-isolation-segment-xyz",
		"my-deployment",
		"deployment_name",
		"simple",
		"cf-01234567890123456789",
		"test-123-abc",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if err := ValidateDeploymentName(name); err != nil {
				t.Errorf("ValidateDeploymentName(%q) returned error: %v, expected nil", name, err)
			}
		})
	}
}

func TestValidateDeploymentName_InvalidNames(t *testing.T) {
	tests := []struct {
		name       string
		deployment string
	}{
		{"path traversal", "../../../etc/passwd"},
		{"url path traversal", "deployment/../../../admin"},
		{"newline injection", "deployment\nmalicious"},
		{"carriage return", "deployment\rmalicious"},
		{"null byte", "deployment\x00malicious"},
		{"spaces", "deployment name"},
		{"forward slash", "deployment/name"},
		{"backslash", "deployment\\name"},
		{"query string", "deployment?param=value"},
		{"hash", "deployment#anchor"},
		{"percent encoded", "deployment%2Fname"},
		{"empty string", ""},
		{"dots only", ".."},
		{"special chars", "deployment@name"},
		{"semicolon", "deployment;name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateDeploymentName(tt.deployment); err == nil {
				t.Errorf("ValidateDeploymentName(%q) returned nil, expected error", tt.deployment)
			}
		})
	}
}

// containsControlChar checks if a string contains any ASCII control characters
func containsControlChar(s string) bool {
	for _, r := range s {
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}

// Issue #83 PR feedback: Error messages should not contain control characters
// that could lead to log injection when the error is logged.

func TestValidateGUID_ErrorMessageSanitized(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"newline injection", "bad\nFAKE LOG: attack"},
		{"carriage return", "bad\rFAKE LOG: attack"},
		{"null byte", "bad\x00hidden"},
		{"tab character", "bad\tattack"},
		{"multiple control chars", "bad\n\r\t\x00attack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGUID(tt.input)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			errMsg := err.Error()
			if containsControlChar(errMsg) {
				t.Errorf("Error message contains control characters: %q", errMsg)
			}
			// Verify the sanitized input is still present (without control chars)
			if !strings.Contains(errMsg, "bad") {
				t.Errorf("Error message should contain sanitized input, got: %q", errMsg)
			}
		})
	}
}

func TestValidateDeploymentName_ErrorMessageSanitized(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"newline injection", "bad\nFAKE LOG: attack"},
		{"carriage return", "bad\rFAKE LOG: attack"},
		{"null byte", "bad\x00hidden"},
		{"tab character", "bad\tattack"},
		{"multiple control chars", "bad\n\r\t\x00attack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeploymentName(tt.input)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			errMsg := err.Error()
			if containsControlChar(errMsg) {
				t.Errorf("Error message contains control characters: %q", errMsg)
			}
			// Verify the sanitized input is still present (without control chars)
			if !strings.Contains(errMsg, "bad") {
				t.Errorf("Error message should contain sanitized input, got: %q", errMsg)
			}
		})
	}
}
