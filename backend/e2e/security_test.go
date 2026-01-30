// ABOUTME: Integration tests for CORS and TLS security features
// ABOUTME: Verifies full request flow through middleware chain with security headers

package e2e

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// TestCORSIntegration_AllowedOriginThroughHandlerChain verifies that requests
// from allowed origins receive proper CORS headers when processed through
// the full middleware chain (CORS -> Auth -> Logging -> Handler).
func TestCORSIntegration_AllowedOriginThroughHandlerChain(t *testing.T) {
	allowedOrigins := []string{"https://example.com", "http://localhost:5173"}
	corsMiddleware := middleware.CORSWithConfig(allowedOrigins)

	handler := handlers.NewHandler(nil, nil)

	// Build full middleware chain as in main.go (without auth for simplicity)
	healthHandler := middleware.Chain(handler.Health, corsMiddleware, middleware.LogRequest)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name           string
		origin         string
		expectHeaders  bool
		expectedOrigin string
	}{
		{
			name:           "allowed origin gets CORS headers",
			origin:         "https://example.com",
			expectHeaders:  true,
			expectedOrigin: "https://example.com",
		},
		{
			name:           "localhost dev origin gets CORS headers",
			origin:         "http://localhost:5173",
			expectHeaders:  true,
			expectedOrigin: "http://localhost:5173",
		},
		{
			name:          "disallowed origin gets no CORS headers",
			origin:        "https://evil.com",
			expectHeaders: false,
		},
		{
			name:          "different port is not allowed",
			origin:        "http://localhost:3000",
			expectHeaders: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/health", nil)
			req.Header.Set("Origin", tt.origin)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Request should succeed regardless of origin
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			gotOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			gotCredentials := resp.Header.Get("Access-Control-Allow-Credentials")

			if tt.expectHeaders {
				if gotOrigin != tt.expectedOrigin {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tt.expectedOrigin)
				}
				if gotCredentials != "true" {
					t.Errorf("Access-Control-Allow-Credentials = %q, want %q", gotCredentials, "true")
				}
			} else {
				if gotOrigin != "" {
					t.Errorf("Access-Control-Allow-Origin should be empty for disallowed origin, got %q", gotOrigin)
				}
			}
		})
	}
}

// TestCORSIntegration_PreflightOptions verifies that OPTIONS preflight requests
// return 204 with correct CORS headers for allowed origins.
func TestCORSIntegration_PreflightOptions(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	corsMiddleware := middleware.CORSWithConfig(allowedOrigins)

	// Preflight handler as configured in main.go
	preflightHandler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Response is handled by CORS middleware
	})

	mux := http.NewServeMux()
	mux.HandleFunc("OPTIONS /api/", preflightHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name          string
		origin        string
		expectHeaders bool
	}{
		{
			name:          "allowed origin preflight succeeds",
			origin:        "https://example.com",
			expectHeaders: true,
		},
		{
			name:          "disallowed origin preflight has no CORS headers",
			origin:        "https://evil.com",
			expectHeaders: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodOptions, server.URL+"/api/v1/test", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Preflight should return 204 No Content
			if resp.StatusCode != http.StatusNoContent {
				t.Errorf("Expected status 204, got %d", resp.StatusCode)
			}

			gotOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			gotMethods := resp.Header.Get("Access-Control-Allow-Methods")
			gotHeaders := resp.Header.Get("Access-Control-Allow-Headers")

			if tt.expectHeaders {
				if gotOrigin != tt.origin {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tt.origin)
				}
				if gotMethods == "" {
					t.Error("Access-Control-Allow-Methods should be set")
				}
				if gotHeaders == "" {
					t.Error("Access-Control-Allow-Headers should be set")
				}
			} else {
				if gotOrigin != "" {
					t.Errorf("Access-Control-Allow-Origin should be empty, got %q", gotOrigin)
				}
			}
		})
	}
}

// TestCORSIntegration_EmptyOriginsBlocksAll verifies that with no allowed origins,
// all cross-origin requests are blocked (no CORS headers).
func TestCORSIntegration_EmptyOriginsBlocksAll(t *testing.T) {
	corsMiddleware := middleware.CORSWithConfig(nil) // No allowed origins

	handler := handlers.NewHandler(nil, nil)
	healthHandler := middleware.Chain(handler.Health, corsMiddleware, middleware.LogRequest)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/health", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Request succeeds but no CORS headers
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should be empty with no allowed origins, got %q", got)
	}
}

// TestCORSAllowedOrigins_EnvParsing verifies the CORS_ALLOWED_ORIGINS environment
// variable is correctly parsed into a string slice.
func TestCORSAllowedOrigins_EnvParsing(t *testing.T) {
	// Save original env and restore after test
	originalCF := os.Getenv("CF_API_URL")
	originalUser := os.Getenv("CF_USERNAME")
	originalPass := os.Getenv("CF_PASSWORD")
	originalCORS := os.Getenv("CORS_ALLOWED_ORIGINS")
	defer func() {
		os.Setenv("CF_API_URL", originalCF)
		os.Setenv("CF_USERNAME", originalUser)
		os.Setenv("CF_PASSWORD", originalPass)
		os.Setenv("CORS_ALLOWED_ORIGINS", originalCORS)
	}()

	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single origin",
			envValue: "https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "multiple origins comma separated",
			envValue: "https://example.com,http://localhost:5173,https://staging.example.com",
			expected: []string{"https://example.com", "http://localhost:5173", "https://staging.example.com"},
		},
		{
			name:     "handles whitespace",
			envValue: "https://example.com , http://localhost:5173 , https://staging.example.com",
			expected: []string{"https://example.com", "http://localhost:5173", "https://staging.example.com"},
		},
		{
			name:     "empty value returns nil",
			envValue: "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set required env vars
			os.Setenv("CF_API_URL", "https://api.example.com")
			os.Setenv("CF_USERNAME", "admin")
			os.Setenv("CF_PASSWORD", "secret")
			os.Setenv("CORS_ALLOWED_ORIGINS", tt.envValue)

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if len(cfg.CORSAllowedOrigins) != len(tt.expected) {
				t.Errorf("CORSAllowedOrigins length = %d, want %d", len(cfg.CORSAllowedOrigins), len(tt.expected))
				return
			}

			for i, origin := range tt.expected {
				if cfg.CORSAllowedOrigins[i] != origin {
					t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], origin)
				}
			}
		})
	}
}

// TestCORSIntegration_VaryHeaderSet verifies the Vary header is set for allowed
// origins to ensure proper caching behavior with CDNs.
func TestCORSIntegration_VaryHeaderSet(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	corsMiddleware := middleware.CORSWithConfig(allowedOrigins)

	handler := handlers.NewHandler(nil, nil)
	healthHandler := middleware.Chain(handler.Health, corsMiddleware, middleware.LogRequest)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/health", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Vary header should include Origin for proper CDN caching
	if got := resp.Header.Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

// =============================================================================
// TLS Configuration Integration Tests
// =============================================================================

// TestTLS_BOSHSkipSSLValidationTrue verifies that BOSH_SKIP_SSL_VALIDATION=true
// allows the BOSH client to be created without error. The actual TLS behavior
// is tested in services/boshapi_test.go with a mock TLS server.
func TestTLS_BOSHSkipSSLValidationTrue(t *testing.T) {
	// With skipSSLValidation=true, client creation should succeed
	client, err := services.NewBOSHClient("https://bosh.example.com:25555", "test-client", "test-secret", "", "cf-test", true)
	if err != nil {
		t.Fatalf("NewBOSHClient with skipSSLValidation=true should not fail: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil with skipSSLValidation=true")
	}
}

// TestTLS_BOSHSkipSSLValidationFalse verifies that BOSH_SKIP_SSL_VALIDATION=false
// (default, secure mode) allows client creation - TLS validation happens at runtime.
func TestTLS_BOSHSkipSSLValidationFalse(t *testing.T) {
	// With skipSSLValidation=false and no CA cert, client should still be created
	// (TLS validation uses system CA pool at connection time)
	client, err := services.NewBOSHClient("https://bosh.example.com:25555", "test-client", "test-secret", "", "cf-test", false)
	if err != nil {
		t.Fatalf("NewBOSHClient with skipSSLValidation=false should not fail at creation: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil with skipSSLValidation=false")
	}
}

// TestTLS_BOSHCACertMalformed verifies that malformed CA certificates are properly
// rejected when BOSH_SKIP_SSL_VALIDATION=false.
func TestTLS_BOSHCACertMalformed(t *testing.T) {
	malformedCert := "not-a-valid-certificate"

	// With skipSSLValidation=false and malformed cert, should fail
	_, err := services.NewBOSHClient("https://bosh.example.com", "test-client", "test-secret", malformedCert, "cf-test", false)
	if err == nil {
		t.Error("NewBOSHClient should fail with malformed CA cert when skipSSLValidation=false")
	}

	// Error message should indicate the CA cert issue
	if err != nil && !strings.Contains(err.Error(), "BOSH_CA_CERT") {
		t.Errorf("Error should mention BOSH_CA_CERT, got: %v", err)
	}
}

// TestTLS_BOSHCACertMalformedWithSkipFallback verifies that malformed CA certs
// fall back to insecure mode when BOSH_SKIP_SSL_VALIDATION=true.
func TestTLS_BOSHCACertMalformedWithSkipFallback(t *testing.T) {
	malformedCert := "not-a-valid-certificate"

	// With skipSSLValidation=true, should fall back to insecure mode
	client, err := services.NewBOSHClient("https://bosh.example.com", "test-client", "test-secret", malformedCert, "cf-test", true)
	if err != nil {
		t.Errorf("NewBOSHClient should fall back to insecure mode with malformed CA cert when skipSSLValidation=true: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil when falling back to insecure mode")
	}
}

// TestTLS_CFSkipSSLValidation_EnvParsing verifies CF_SKIP_SSL_VALIDATION
// environment variable is correctly parsed.
func TestTLS_CFSkipSSLValidation_EnvParsing(t *testing.T) {
	// Save and restore env
	originalCF := os.Getenv("CF_API_URL")
	originalUser := os.Getenv("CF_USERNAME")
	originalPass := os.Getenv("CF_PASSWORD")
	originalSkip := os.Getenv("CF_SKIP_SSL_VALIDATION")
	defer func() {
		os.Setenv("CF_API_URL", originalCF)
		os.Setenv("CF_USERNAME", originalUser)
		os.Setenv("CF_PASSWORD", originalPass)
		os.Setenv("CF_SKIP_SSL_VALIDATION", originalSkip)
	}()

	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "true string",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false string",
			envValue: "false",
			expected: false,
		},
		{
			name:     "1 is truthy",
			envValue: "1",
			expected: true,
		},
		{
			name:     "0 is falsy",
			envValue: "0",
			expected: false,
		},
		{
			name:     "empty defaults to false",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CF_API_URL", "https://api.example.com")
			os.Setenv("CF_USERNAME", "admin")
			os.Setenv("CF_PASSWORD", "secret")
			os.Setenv("CF_SKIP_SSL_VALIDATION", tt.envValue)

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.CFSkipSSLValidation != tt.expected {
				t.Errorf("CFSkipSSLValidation = %v, want %v", cfg.CFSkipSSLValidation, tt.expected)
			}
		})
	}
}

// TestTLS_BOSHSkipSSLValidation_EnvParsing verifies BOSH_SKIP_SSL_VALIDATION
// environment variable is correctly parsed.
func TestTLS_BOSHSkipSSLValidation_EnvParsing(t *testing.T) {
	// Save and restore env
	originalCF := os.Getenv("CF_API_URL")
	originalUser := os.Getenv("CF_USERNAME")
	originalPass := os.Getenv("CF_PASSWORD")
	originalSkip := os.Getenv("BOSH_SKIP_SSL_VALIDATION")
	defer func() {
		os.Setenv("CF_API_URL", originalCF)
		os.Setenv("CF_USERNAME", originalUser)
		os.Setenv("CF_PASSWORD", originalPass)
		os.Setenv("BOSH_SKIP_SSL_VALIDATION", originalSkip)
	}()

	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "true enables skip",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false disables skip",
			envValue: "false",
			expected: false,
		},
		{
			name:     "empty defaults to false (secure by default)",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CF_API_URL", "https://api.example.com")
			os.Setenv("CF_USERNAME", "admin")
			os.Setenv("CF_PASSWORD", "secret")
			os.Setenv("BOSH_SKIP_SSL_VALIDATION", tt.envValue)

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.BOSHSkipSSLValidation != tt.expected {
				t.Errorf("BOSHSkipSSLValidation = %v, want %v", cfg.BOSHSkipSSLValidation, tt.expected)
			}
		})
	}
}

// TestTLS_BOSHCACert_EnvParsing verifies BOSH_CA_CERT environment variable
// is correctly loaded into config.
func TestTLS_BOSHCACert_EnvParsing(t *testing.T) {
	// Save and restore env
	originalCF := os.Getenv("CF_API_URL")
	originalUser := os.Getenv("CF_USERNAME")
	originalPass := os.Getenv("CF_PASSWORD")
	originalCACert := os.Getenv("BOSH_CA_CERT")
	defer func() {
		os.Setenv("CF_API_URL", originalCF)
		os.Setenv("CF_USERNAME", originalUser)
		os.Setenv("CF_PASSWORD", originalPass)
		os.Setenv("BOSH_CA_CERT", originalCACert)
	}()

	// Sample PEM-formatted cert (not a real cert, just testing parsing)
	sampleCert := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpj2S5JNMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96HFLqGzOHHY+QLqJZT
-----END CERTIFICATE-----`

	os.Setenv("CF_API_URL", "https://api.example.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")
	os.Setenv("BOSH_CA_CERT", sampleCert)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.BOSHCACert != sampleCert {
		t.Errorf("BOSHCACert not loaded correctly, got length %d, want %d", len(cfg.BOSHCACert), len(sampleCert))
	}
}
