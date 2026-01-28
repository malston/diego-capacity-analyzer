// ABOUTME: Input validation functions for API parameters
// ABOUTME: Prevents URL injection attacks via GUID and deployment name validation

package services

import (
	"fmt"
	"regexp"
	"strings"
)

// guidPattern matches valid Cloud Foundry GUIDs (36 chars: 8-4-4-4-12 hex with lowercase)
var guidPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// deploymentNamePattern matches valid BOSH deployment names (alphanumeric, hyphens, underscores)
var deploymentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// sanitizeForLog removes control characters from strings to prevent log injection
// when including user input in error messages
func sanitizeForLog(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1 // Remove control characters
		}
		return r
	}, s)
}

// ValidateGUID validates that a GUID has the correct format.
// This prevents URL path traversal if upstream APIs were compromised.
func ValidateGUID(guid string) error {
	if !guidPattern.MatchString(guid) {
		return fmt.Errorf("invalid GUID format: %s", sanitizeForLog(guid))
	}
	return nil
}

// ValidateDeploymentName validates that a BOSH deployment name has a safe format.
// This prevents URL path traversal if upstream APIs were compromised.
func ValidateDeploymentName(name string) error {
	if name == "" {
		return fmt.Errorf("deployment name cannot be empty")
	}
	if !deploymentNamePattern.MatchString(name) {
		return fmt.Errorf("invalid deployment name format: %s", sanitizeForLog(name))
	}
	return nil
}
