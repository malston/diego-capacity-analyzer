// ABOUTME: Test helpers for config tests
// ABOUTME: Provides utilities for environment variable management

package config

import (
	"os"
	"testing"
)

// withCleanCFEnv clears the environment, sets required CF env vars to test
// values, and returns a cleanup function that restores the original env.
// Use with t.Cleanup().
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    t.Cleanup(withCleanCFEnv(t))
//	    // Environment is cleared, CF_API_URL, CF_USERNAME, CF_PASSWORD are set
//	}
func withCleanCFEnv(t *testing.T) func() {
	t.Helper()
	return withCleanCFEnvAndExtra(t, nil)
}

// withCleanCFEnvAndExtra clears the environment, sets required CF env vars
// plus additional vars, and returns a cleanup function that restores the
// original env. Use with t.Cleanup().
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
//	        "AUTH_MODE": "required",
//	    }))
//	}
func withCleanCFEnvAndExtra(t *testing.T, extra map[string]string) func() {
	t.Helper()

	// Save entire environment
	originalEnv := os.Environ()

	// Clear environment for clean slate
	os.Clearenv()

	// Set required CF test values
	os.Setenv("CF_API_URL", "https://api.sys.test.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	// Set extra values
	for key, value := range extra {
		os.Setenv(key, value)
	}

	// Return cleanup function that restores original environment
	return func() {
		os.Clearenv()
		for _, env := range originalEnv {
			for i := 0; i < len(env); i++ {
				if env[i] == '=' {
					os.Setenv(env[:i], env[i+1:])
					break
				}
			}
		}
	}
}
