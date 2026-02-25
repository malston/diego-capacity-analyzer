package config

import (
	"os"
	"strings"
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

func TestLoadConfig_RateLimitDefaults(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !cfg.RateLimitEnabled {
		t.Error("Expected RateLimitEnabled default true, got false")
	}
	if cfg.RateLimitAuth != 5 {
		t.Errorf("Expected RateLimitAuth default 5, got %d", cfg.RateLimitAuth)
	}
	if cfg.RateLimitRefresh != 10 {
		t.Errorf("Expected RateLimitRefresh default 10, got %d", cfg.RateLimitRefresh)
	}
	if cfg.RateLimitWrite != 10 {
		t.Errorf("Expected RateLimitWrite default 10, got %d", cfg.RateLimitWrite)
	}
	if cfg.RateLimitDefault != 100 {
		t.Errorf("Expected RateLimitDefault default 100, got %d", cfg.RateLimitDefault)
	}
}

func TestLoadConfig_RateLimitFromEnv(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"RATE_LIMIT_ENABLED": "false",
		"RATE_LIMIT_AUTH":    "20",
		"RATE_LIMIT_REFRESH": "30",
		"RATE_LIMIT_WRITE":   "40",
		"RATE_LIMIT_DEFAULT": "200",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.RateLimitEnabled {
		t.Error("Expected RateLimitEnabled false, got true")
	}
	if cfg.RateLimitAuth != 20 {
		t.Errorf("Expected RateLimitAuth 20, got %d", cfg.RateLimitAuth)
	}
	if cfg.RateLimitRefresh != 30 {
		t.Errorf("Expected RateLimitRefresh 30, got %d", cfg.RateLimitRefresh)
	}
	if cfg.RateLimitWrite != 40 {
		t.Errorf("Expected RateLimitWrite 40, got %d", cfg.RateLimitWrite)
	}
	if cfg.RateLimitDefault != 200 {
		t.Errorf("Expected RateLimitDefault 200, got %d", cfg.RateLimitDefault)
	}
}

func TestLoadConfig_RateLimitInvalidValue(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{"zero value", "RATE_LIMIT_AUTH", "0"},
		{"negative value", "RATE_LIMIT_REFRESH", "-1"},
		{"exceeds max", "RATE_LIMIT_DEFAULT", "10001"},
		{"chat zero", "RATE_LIMIT_CHAT", "0"},
		{"chat exceeds max", "RATE_LIMIT_CHAT", "10001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
				tt.env: tt.value,
			}))

			_, err := Load()
			if err == nil {
				t.Errorf("Expected error for %s=%s, got nil", tt.env, tt.value)
			}
		})
	}
}

func TestLoad_OAuthClientDefaults(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.OAuthClientID != "cf" {
		t.Errorf("OAuthClientID = %q, want %q", cfg.OAuthClientID, "cf")
	}
	if cfg.OAuthClientSecret != "" {
		t.Errorf("OAuthClientSecret = %q, want %q", cfg.OAuthClientSecret, "")
	}
}

func TestLoad_OAuthClientFromEnv(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"OAUTH_CLIENT_ID":     "diego-analyzer",
		"OAUTH_CLIENT_SECRET": "my-secret",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.OAuthClientID != "diego-analyzer" {
		t.Errorf("OAuthClientID = %q, want %q", cfg.OAuthClientID, "diego-analyzer")
	}
	if cfg.OAuthClientSecret != "my-secret" {
		t.Errorf("OAuthClientSecret = %q, want %q", cfg.OAuthClientSecret, "my-secret")
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

func TestLoadConfig_AIProviderDefaults(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AIProvider != "" {
		t.Errorf("Expected AIProvider empty by default, got %q", cfg.AIProvider)
	}
	if cfg.AIAPIKey != "" {
		t.Errorf("Expected AIAPIKey empty by default, got %q", cfg.AIAPIKey)
	}
}

func TestConfig_AIModelDefault(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AIModel != "claude-sonnet-4-5-20250514" {
		t.Errorf("Expected default AIModel 'claude-sonnet-4-5-20250514', got %q", cfg.AIModel)
	}
}

func TestConfig_AIModelFromEnv(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"AI_MODEL": "custom-model",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AIModel != "custom-model" {
		t.Errorf("Expected AIModel 'custom-model', got %q", cfg.AIModel)
	}
}

func TestLoadConfig_AIProviderFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		env            map[string]string
		wantProvider   string
		wantKey        string
		wantConfigured bool
	}{
		{
			name:           "both set",
			env:            map[string]string{"AI_PROVIDER": "anthropic", "AI_API_KEY": "test-key"},
			wantProvider:   "anthropic",
			wantKey:        "test-key",
			wantConfigured: true,
		},
		{
			name:           "key only",
			env:            map[string]string{"AI_API_KEY": "test-key"},
			wantProvider:   "",
			wantKey:        "test-key",
			wantConfigured: false,
		},
		{
			name:           "neither set",
			env:            nil,
			wantProvider:   "",
			wantKey:        "",
			wantConfigured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(withCleanCFEnvAndExtra(t, tt.env))

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if cfg.AIProvider != tt.wantProvider {
				t.Errorf("AIProvider = %q, want %q", cfg.AIProvider, tt.wantProvider)
			}
			if cfg.AIAPIKey != tt.wantKey {
				t.Errorf("AIAPIKey = %q, want %q", cfg.AIAPIKey, tt.wantKey)
			}
			if cfg.AIConfigured() != tt.wantConfigured {
				t.Errorf("AIConfigured() = %v, want %v", cfg.AIConfigured(), tt.wantConfigured)
			}
		})
	}
}

func TestLoadConfig_AIProviderWithoutKey(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"AI_PROVIDER": "anthropic",
	}))

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for AI_PROVIDER set without AI_API_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "AI_API_KEY") {
		t.Errorf("Expected error mentioning AI_API_KEY, got: %v", err)
	}
}

func TestLoadConfig_AIProviderUnknown(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"AI_PROVIDER": "openai",
		"AI_API_KEY":  "test-key",
	}))

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for unknown AI_PROVIDER, got nil")
	}
	if !strings.Contains(err.Error(), "unknown AI_PROVIDER") {
		t.Errorf("Expected error mentioning 'unknown AI_PROVIDER', got: %v", err)
	}
}

func TestLoadConfig_AITimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "zero idle timeout",
			env: map[string]string{
				"AI_PROVIDER":          "anthropic",
				"AI_API_KEY":           "test-key",
				"AI_IDLE_TIMEOUT_SECS": "0",
			},
			wantErr: "AI_IDLE_TIMEOUT_SECS",
		},
		{
			name: "negative idle timeout",
			env: map[string]string{
				"AI_PROVIDER":          "anthropic",
				"AI_API_KEY":           "test-key",
				"AI_IDLE_TIMEOUT_SECS": "-5",
			},
			wantErr: "AI_IDLE_TIMEOUT_SECS",
		},
		{
			name: "zero max duration",
			env: map[string]string{
				"AI_PROVIDER":          "anthropic",
				"AI_API_KEY":           "test-key",
				"AI_MAX_DURATION_SECS": "0",
			},
			wantErr: "AI_MAX_DURATION_SECS",
		},
		{
			name: "negative max duration",
			env: map[string]string{
				"AI_PROVIDER":          "anthropic",
				"AI_API_KEY":           "test-key",
				"AI_MAX_DURATION_SECS": "-10",
			},
			wantErr: "AI_MAX_DURATION_SECS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(withCleanCFEnvAndExtra(t, tt.env))

			_, err := Load()
			if err == nil {
				t.Fatalf("Expected error for %s, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error mentioning %s, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoadConfig_AITimeoutDefaults(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"AI_PROVIDER": "anthropic",
		"AI_API_KEY":  "test-key",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AIIdleTimeoutSecs != 30 {
		t.Errorf("Expected default AIIdleTimeoutSecs 30, got %d", cfg.AIIdleTimeoutSecs)
	}
	if cfg.AIMaxDurationSecs != 300 {
		t.Errorf("Expected default AIMaxDurationSecs 300, got %d", cfg.AIMaxDurationSecs)
	}
}
