// ABOUTME: End-to-end tests for CSRF token protection
// ABOUTME: Tests full login flow with CSRF validation using double-submit cookie pattern

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

// createMockCFUAAServer creates a mock UAA/CF server for testing authentication.
// It handles /v3/info (CF API discovery) and /oauth/token (UAA auth).
func createMockCFUAAServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v3/info":
			// CF API discovery - returns UAA URL (which is the same server in tests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"links": map[string]interface{}{
					"login": map[string]string{"href": "http://" + r.Host},
				},
			})
		case "/oauth/token":
			// UAA token endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"expires_in":    3600,
				"token_type":    "bearer",
				"user_id":       "test-user-id",
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestCSRF_E2E_FullLoginFlow tests the complete CSRF protection flow:
// 1. Login and receive both session and CSRF cookies
// 2. POST with valid CSRF token succeeds
// 3. POST without CSRF token fails with 403
func TestCSRF_E2E_FullLoginFlow(t *testing.T) {
	// Setup mock UAA server
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	// Create handler with config pointing to mock UAA
	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false, // Allow non-HTTPS in tests
	}
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	// Step 1: Login and get cookies
	loginBody := `{"username":"admin","password":"secret"}`
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	h.Login(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("Login failed: status=%d body=%s", loginRR.Code, loginRR.Body.String())
	}

	// Extract cookies
	cookies := loginRR.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case "DIEGO_SESSION":
			sessionCookie = c
		case "DIEGO_CSRF":
			csrfCookie = c
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected DIEGO_SESSION cookie after login")
	}
	if csrfCookie == nil {
		t.Fatal("Expected DIEGO_CSRF cookie after login")
	}

	// Verify CSRF cookie properties
	if csrfCookie.HttpOnly {
		t.Error("CSRF cookie should NOT be HttpOnly (JavaScript needs to read it)")
	}
	if csrfCookie.Value == "" {
		t.Error("CSRF cookie should have a value")
	}

	// Step 2: POST with valid CSRF token should succeed
	csrfHandler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		},
		middleware.CSRF(),
	)

	validReq := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-CSRF-Token", csrfCookie.Value)
	validReq.AddCookie(sessionCookie)
	validReq.AddCookie(csrfCookie)
	validRR := httptest.NewRecorder()
	csrfHandler(validRR, validReq)

	if validRR.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid CSRF token, got %d: %s", validRR.Code, validRR.Body.String())
	}

	// Step 3: POST without CSRF header should fail
	invalidReq := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.AddCookie(sessionCookie)
	invalidReq.AddCookie(csrfCookie)
	// Missing X-CSRF-Token header
	invalidRR := httptest.NewRecorder()
	csrfHandler(invalidRR, invalidReq)

	if invalidRR.Code != http.StatusForbidden {
		t.Errorf("Expected 403 without CSRF header, got %d", invalidRR.Code)
	}

	// Verify error response format
	var errResp map[string]string
	if err := json.NewDecoder(invalidRR.Body).Decode(&errResp); err != nil {
		t.Errorf("Failed to decode error response: %v", err)
	} else if errResp["error"] == "" {
		t.Error("Expected error message in response")
	}
}

// TestCSRF_E2E_TokenMismatchRejected verifies that mismatched CSRF tokens are rejected.
func TestCSRF_E2E_TokenMismatchRejected(t *testing.T) {
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		middleware.CSRF(),
	)

	req := httptest.NewRequest("POST", "/api/v1/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "token-from-header")
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "valid-session"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "different-token-in-cookie"})
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for token mismatch, got %d", rr.Code)
	}
}

// TestCSRF_E2E_BearerTokenBypassesCSRF verifies that Bearer token auth
// bypasses CSRF validation (as CSRF protection is only needed for cookie-based auth).
func TestCSRF_E2E_BearerTokenBypassesCSRF(t *testing.T) {
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		middleware.CSRF(),
	)

	req := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer some-jwt-token")
	// No CSRF token - should still work because Bearer auth is used
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for Bearer auth bypass, got %d", rr.Code)
	}
}

// TestCSRF_E2E_NoSessionCookieAllowsRequest verifies that requests without
// a session cookie are allowed through (they're not session-authenticated).
func TestCSRF_E2E_NoSessionCookieAllowsRequest(t *testing.T) {
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		middleware.CSRF(),
	)

	// POST without any cookies (anonymous request)
	req := httptest.NewRequest("POST", "/api/v1/infrastructure/manual", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for anonymous request, got %d", rr.Code)
	}
}

// TestCSRF_E2E_GETRequestsSkipValidation verifies that safe HTTP methods
// (GET, HEAD, OPTIONS) don't require CSRF validation.
func TestCSRF_E2E_GETRequestsSkipValidation(t *testing.T) {
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		middleware.CSRF(),
	)

	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/dashboard", nil)
			// Add session cookie but no CSRF - should still work for safe methods
			req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
			req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "csrf-token"})
			rr := httptest.NewRecorder()
			handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s request, got %d", method, rr.Code)
			}
		})
	}
}

// TestCSRF_E2E_StateChangingMethodsRequireValidation verifies that state-changing
// HTTP methods (POST, PUT, DELETE, PATCH) require CSRF validation when session-authenticated.
func TestCSRF_E2E_StateChangingMethodsRequireValidation(t *testing.T) {
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		middleware.CSRF(),
	)

	unsafeMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range unsafeMethods {
		t.Run(method+"_without_token", func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/test", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
			req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "csrf-token"})
			// Missing X-CSRF-Token header
			rr := httptest.NewRecorder()
			handler(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Errorf("Expected 403 for %s without CSRF token, got %d", method, rr.Code)
			}
		})

		t.Run(method+"_with_valid_token", func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/test", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", "csrf-token")
			req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
			req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "csrf-token"})
			rr := httptest.NewRecorder()
			handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s with valid CSRF token, got %d", method, rr.Code)
			}
		})
	}
}

// TestCSRF_E2E_LogoutClearsCookies verifies that logout clears both session and CSRF cookies.
func TestCSRF_E2E_LogoutClearsCookies(t *testing.T) {
	// Setup mock UAA server
	uaaServer := createMockCFUAAServer(t)
	defer uaaServer.Close()

	// Create handler with session service
	cfg := &config.Config{
		CFAPIUrl:     uaaServer.URL,
		CookieSecure: false,
	}
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	// First, login to get cookies
	loginBody := `{"username":"admin","password":"secret"}`
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	h.Login(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("Login failed: %d", loginRR.Code)
	}

	// Extract session cookie for logout request
	var sessionCookie *http.Cookie
	for _, c := range loginRR.Result().Cookies() {
		if c.Name == "DIEGO_SESSION" {
			sessionCookie = c
			break
		}
	}

	// Now logout
	logoutReq := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRR := httptest.NewRecorder()
	h.Logout(logoutRR, logoutReq)

	if logoutRR.Code != http.StatusOK {
		t.Errorf("Logout failed: %d", logoutRR.Code)
	}

	// Check that cookies are cleared (MaxAge = -1)
	for _, c := range logoutRR.Result().Cookies() {
		switch c.Name {
		case "DIEGO_SESSION", "DIEGO_CSRF":
			if c.MaxAge != -1 {
				t.Errorf("Cookie %s should be cleared (MaxAge=-1), got MaxAge=%d", c.Name, c.MaxAge)
			}
		}
	}
}
