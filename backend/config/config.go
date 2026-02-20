// ABOUTME: Configuration loader for backend service
// ABOUTME: Loads settings from environment variables with defaults

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	Port               string
	CacheTTL           int      // seconds, default for general cache
	DashboardTTL       int      // seconds, for BOSH/CF data (default 30s)
	AuthMode           string   // disabled, optional, required (default: optional)
	CORSAllowedOrigins []string // allowed CORS origins (empty = block all cross-origin)
	CookieSecure       bool     // Set Secure flag on session cookies (default: true)

	// OAuth Client (for UAA password/refresh grants)
	OAuthClientID     string
	OAuthClientSecret string

	// Rate Limiting
	RateLimitEnabled bool // Enable rate limiting (default: true)
	RateLimitAuth    int  // Requests per minute for auth endpoints (default: 5)
	RateLimitRefresh int  // Requests per minute for refresh endpoint (default: 10)
	RateLimitWrite   int  // Requests per minute for write endpoints (default: 10)
	RateLimitDefault int  // Requests per minute for all other endpoints (default: 100)

	// CF API
	CFAPIUrl            string
	CFUsername          string
	CFPassword          string
	CFSkipSSLValidation bool // explicit opt-in for insecure connections

	// BOSH API (optional)
	BOSHEnvironment       string
	BOSHClient            string
	BOSHSecret            string
	BOSHCACert            string
	BOSHDeployment        string
	BOSHSkipSSLValidation bool // explicit opt-in for insecure connections (only if no CA cert)

	// CredHub (optional)
	CredHubURL    string
	CredHubClient string
	CredHubSecret string

	// vSphere (optional)
	VSphereHost       string
	VSphereUsername   string
	VSpherePassword   string
	VSphereDatacenter string
	VSphereInsecure   bool
	VSphereCacheTTL   int // seconds, default 300 (5 min)
}

// VSphereConfigured returns true if vSphere credentials are set
func (c *Config) VSphereConfigured() bool {
	return c.VSphereHost != "" && c.VSphereUsername != "" && c.VSpherePassword != "" && c.VSphereDatacenter != ""
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		CacheTTL:           getEnvInt("CACHE_TTL", 300),
		DashboardTTL:       getEnvInt("DASHBOARD_CACHE_TTL", 30),
		AuthMode:           getEnv("AUTH_MODE", "optional"),
		CORSAllowedOrigins: getEnvStringList("CORS_ALLOWED_ORIGINS"),
		CookieSecure:       getEnvBool("COOKIE_SECURE", true),

		OAuthClientID:     getEnv("OAUTH_CLIENT_ID", "cf"),
		OAuthClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),

		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitAuth:    getEnvInt("RATE_LIMIT_AUTH", 5),
		RateLimitRefresh: getEnvInt("RATE_LIMIT_REFRESH", 10),
		RateLimitWrite:   getEnvInt("RATE_LIMIT_WRITE", 10),
		RateLimitDefault: getEnvInt("RATE_LIMIT_DEFAULT", 100),

		CFAPIUrl:            ensureScheme(os.Getenv("CF_API_URL")),
		CFUsername:          os.Getenv("CF_USERNAME"),
		CFPassword:          os.Getenv("CF_PASSWORD"),
		CFSkipSSLValidation: getEnvBool("CF_SKIP_SSL_VALIDATION", false),

		BOSHEnvironment:       ensureScheme(os.Getenv("BOSH_ENVIRONMENT")),
		BOSHClient:            os.Getenv("BOSH_CLIENT"),
		BOSHSecret:            os.Getenv("BOSH_CLIENT_SECRET"),
		BOSHCACert:            os.Getenv("BOSH_CA_CERT"),
		BOSHDeployment:        os.Getenv("BOSH_DEPLOYMENT"),
		BOSHSkipSSLValidation: getEnvBool("BOSH_SKIP_SSL_VALIDATION", false),

		CredHubURL:    ensureScheme(os.Getenv("CREDHUB_URL")),
		CredHubClient: os.Getenv("CREDHUB_CLIENT"),
		CredHubSecret: os.Getenv("CREDHUB_SECRET"),

		VSphereHost:       os.Getenv("VSPHERE_HOST"),
		VSphereUsername:   os.Getenv("VSPHERE_USERNAME"),
		VSpherePassword:   os.Getenv("VSPHERE_PASSWORD"),
		VSphereDatacenter: os.Getenv("VSPHERE_DATACENTER"),
		VSphereInsecure:   getEnvBool("VSPHERE_INSECURE", false),
		VSphereCacheTTL:   getEnvInt("VSPHERE_CACHE_TTL", 300),
	}

	// Validate required fields
	if cfg.CFAPIUrl == "" {
		return nil, fmt.Errorf("CF_API_URL is required")
	}
	if cfg.CFUsername == "" {
		return nil, fmt.Errorf("CF_USERNAME is required")
	}
	if cfg.CFPassword == "" {
		return nil, fmt.Errorf("CF_PASSWORD is required")
	}

	// Validate rate limit values
	for _, rl := range []struct {
		name  string
		value int
	}{
		{"RATE_LIMIT_AUTH", cfg.RateLimitAuth},
		{"RATE_LIMIT_REFRESH", cfg.RateLimitRefresh},
		{"RATE_LIMIT_WRITE", cfg.RateLimitWrite},
		{"RATE_LIMIT_DEFAULT", cfg.RateLimitDefault},
	} {
		if rl.value < 1 || rl.value > 10000 {
			return nil, fmt.Errorf("%s must be between 1 and 10000, got %d", rl.name, rl.value)
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvStringList(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ensureScheme adds https:// prefix if the URL has no scheme
func ensureScheme(url string) string {
	if url == "" {
		return url
	}
	if !strings.Contains(url, "://") {
		return "https://" + url
	}
	return url
}
