// ABOUTME: End-to-end tests for rate limiting middleware
// ABOUTME: Tests full request flows with rate limit enforcement, exemptions, and disable mode

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// TestRateLimit_E2E_LoginEndpoint tests that the login endpoint is rate limited.
// 5 requests should succeed, the 6th should return 429.
func TestRateLimit_E2E_LoginEndpoint(t *testing.T) {
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false,
	}
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	// Create rate-limited login handler with auth tier (5/min)
	rl := middleware.NewRateLimiter(5, time.Minute)
	loginHandler := middleware.Chain(h.Login, middleware.RateLimit(rl, middleware.ClientIP))

	loginBody := `{"username":"admin","password":"secret"}`

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.1:12345"
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Request %d should succeed, got %d: %s", i+1, rr.Code, rr.Body.String())
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.1:12345"
	rr := httptest.NewRecorder()
	loginHandler(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request should return 429, got %d", rr.Code)
	}

	// Verify Retry-After header
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header on 429 response")
	}

	// Verify JSON error body
	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode 429 response body: %v", err)
	}
	if body["error"] != "Rate limit exceeded" {
		t.Errorf("Expected error 'Rate limit exceeded', got %q", body["error"])
	}
	if _, ok := body["retry_after"]; !ok {
		t.Error("Expected retry_after field in 429 response body")
	}
}

// TestRateLimit_E2E_ExemptEndpoints tests that health and openapi endpoints
// are not rate limited when using "none" tier (no rate limit middleware applied).
func TestRateLimit_E2E_ExemptEndpoints(t *testing.T) {
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false,
	}
	c := cache.New(5 * time.Minute)
	h := handlers.NewHandler(cfg, c)

	// Health handler without rate limiting (simulates "none" tier)
	healthHandler := h.Health

	// Send 200 requests (well above any tier limit)
	for i := 0; i < 200; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		rr := httptest.NewRecorder()
		healthHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Exempt health request %d should succeed, got %d", i+1, rr.Code)
		}
	}
}

// TestRateLimit_E2E_DisabledMode tests that nil limiter passes all requests through.
func TestRateLimit_E2E_DisabledMode(t *testing.T) {
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false,
	}
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	// Create handler with nil limiter (disabled mode)
	loginHandler := middleware.Chain(h.Login, middleware.RateLimit(nil, middleware.ClientIP))

	loginBody := `{"username":"admin","password":"secret"}`

	// Send 20 requests (4x the normal auth limit) - all should succeed
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.1:12345"
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Disabled mode request %d should succeed, got %d", i+1, rr.Code)
		}
	}
}

// TestRateLimit_E2E_SeparateIPQuotas tests that different IPs get separate quotas.
func TestRateLimit_E2E_SeparateIPQuotas(t *testing.T) {
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false,
	}
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	rl := middleware.NewRateLimiter(2, time.Minute)
	loginHandler := middleware.Chain(h.Login, middleware.RateLimit(rl, middleware.ClientIP))

	loginBody := `{"username":"admin","password":"secret"}`

	// Exhaust quota for IP 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("IP1 request %d should succeed, got %d", i+1, rr.Code)
		}
	}

	// IP 1 should now be rate limited
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rr := httptest.NewRecorder()
	loginHandler(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("IP1 3rd request should be 429, got %d", rr.Code)
	}

	// IP 2 should still have its full quota
	req = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	rr = httptest.NewRecorder()
	loginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("IP2 first request should succeed (separate quota), got %d", rr.Code)
	}
}

// TestRateLimit_E2E_WriteEndpoint tests that write endpoints use user-based rate limiting.
func TestRateLimit_E2E_WriteEndpoint(t *testing.T) {
	rl := middleware.NewRateLimiter(2, time.Minute)

	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		},
		middleware.RateLimit(rl, middleware.UserOrIP),
	)

	// Without user claims, falls back to IP
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:5555"
		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Request %d should succeed, got %d", i+1, rr.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.1:5555"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request should be 429, got %d", rr.Code)
	}
}
