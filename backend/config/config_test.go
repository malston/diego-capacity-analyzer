package config

import (
	"os"
	"testing"
)

func TestLoadConfig_RequiredFields(t *testing.T) {
	// Clear environment
	os.Clearenv()

	// Set required fields
	os.Setenv("CF_API_URL", "https://api.sys.test.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.CFAPIUrl != "https://api.sys.test.com" {
		t.Errorf("Expected CFAPIUrl https://api.sys.test.com, got %s", cfg.CFAPIUrl)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Error("Expected error for missing required fields, got nil")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("CF_API_URL", "https://api.sys.test.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}

	if cfg.CacheTTL != 300 {
		t.Errorf("Expected default cache TTL 300, got %d", cfg.CacheTTL)
	}
}
