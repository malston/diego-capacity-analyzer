// ABOUTME: Integration tests for CORS security features
// ABOUTME: Verifies full request flow through middleware chain with CORS headers

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
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
			t.Cleanup(withTestCFEnv(t, map[string]string{
				"CORS_ALLOWED_ORIGINS": tt.envValue,
			}))

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
