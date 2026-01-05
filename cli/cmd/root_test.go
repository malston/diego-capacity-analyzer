// ABOUTME: Tests for the root command and global flag handling
// ABOUTME: Verifies environment variable and flag configuration

package cmd

import (
	"os"
	"testing"
)

func TestGetAPIURL_Default(t *testing.T) {
	os.Unsetenv("DIEGO_CAPACITY_API_URL")
	apiURL = "" // Reset flag

	url := GetAPIURL()
	if url != "http://localhost:8080" {
		t.Errorf("expected default URL http://localhost:8080, got %s", url)
	}
}

func TestGetAPIURL_FromEnv(t *testing.T) {
	os.Setenv("DIEGO_CAPACITY_API_URL", "http://backend.example.com")
	defer os.Unsetenv("DIEGO_CAPACITY_API_URL")
	apiURL = "" // Reset flag

	url := GetAPIURL()
	if url != "http://backend.example.com" {
		t.Errorf("expected http://backend.example.com, got %s", url)
	}
}

func TestGetAPIURL_FlagOverridesEnv(t *testing.T) {
	os.Setenv("DIEGO_CAPACITY_API_URL", "http://backend.example.com")
	defer os.Unsetenv("DIEGO_CAPACITY_API_URL")
	apiURL = "http://flag-override.example.com"
	defer func() { apiURL = "" }()

	url := GetAPIURL()
	if url != "http://flag-override.example.com" {
		t.Errorf("expected flag to override env, got %s", url)
	}
}

func TestJSONOutput(t *testing.T) {
	jsonOutput = true
	defer func() { jsonOutput = false }()

	if !IsJSONOutput() {
		t.Error("expected IsJSONOutput to return true")
	}
}
