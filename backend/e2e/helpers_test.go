// ABOUTME: Test helpers for e2e tests
// ABOUTME: Provides utilities for environment variable management in tests

package e2e

import (
	"os"
	"testing"
)

// withTestCFEnv sets CF environment variables plus additional vars,
// returning a cleanup function that restores all original values.
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    t.Cleanup(withTestCFEnv(t, map[string]string{
//	        "CORS_ALLOWED_ORIGINS": "https://example.com",
//	    }))
//	}
func withTestCFEnv(t *testing.T, extra map[string]string) func() {
	t.Helper()

	// Save original values for CF vars
	originals := map[string]string{
		"CF_API_URL":  os.Getenv("CF_API_URL"),
		"CF_USERNAME": os.Getenv("CF_USERNAME"),
		"CF_PASSWORD": os.Getenv("CF_PASSWORD"),
	}

	// Save original values for extra vars
	for key := range extra {
		originals[key] = os.Getenv(key)
	}

	// Set test values
	os.Setenv("CF_API_URL", "https://api.example.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	// Set extra values
	for key, value := range extra {
		os.Setenv(key, value)
	}

	// Return cleanup function
	return func() {
		for key, value := range originals {
			os.Setenv(key, value)
		}
	}
}
