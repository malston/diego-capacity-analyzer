package config

import (
	"os"
	"testing"
)

func TestLoadConfig_RequiredFields(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

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
	t.Cleanup(withCleanCFEnv(t))

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

func TestEnsureScheme(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"with https scheme", "https://api.example.com", "https://api.example.com"},
		{"with http scheme", "http://api.example.com", "http://api.example.com"},
		{"without scheme", "api.example.com", "https://api.example.com"},
		{"without scheme with path", "api.example.com/v3/info", "https://api.example.com/v3/info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureScheme(tt.input)
			if result != tt.expected {
				t.Errorf("ensureScheme(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadConfig_AuthModeDefault(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Default should be "optional"
	if cfg.AuthMode != "optional" {
		t.Errorf("Expected default AuthMode 'optional', got %q", cfg.AuthMode)
	}
}

func TestLoadConfig_AuthModeFromEnv(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"AUTH_MODE": "required",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AuthMode != "required" {
		t.Errorf("Expected AuthMode 'required', got %q", cfg.AuthMode)
	}
}

func TestLoadConfig_URLSchemePrefixing(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"CF_API_URL":       "api.sys.test.com", // Override to test scheme prefixing
		"BOSH_ENVIRONMENT": "10.0.0.6:25555",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.CFAPIUrl != "https://api.sys.test.com" {
		t.Errorf("Expected CFAPIUrl to have https:// prefix, got %s", cfg.CFAPIUrl)
	}

	if cfg.BOSHEnvironment != "https://10.0.0.6:25555" {
		t.Errorf("Expected BOSHEnvironment to have https:// prefix, got %s", cfg.BOSHEnvironment)
	}
}
