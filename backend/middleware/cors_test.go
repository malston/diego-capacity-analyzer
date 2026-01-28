// ABOUTME: Tests for CORS middleware functionality
// ABOUTME: Verifies headers are set and OPTIONS preflight is handled

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AddsHeaders(t *testing.T) {
	handler := CORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, "Content-Type, Authorization")
	}
}

func TestCORS_HandlesPreflight(t *testing.T) {
	handlerCalled := false
	handler := CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight")
	}
}

func TestCORS_PassesThroughNonOptions(t *testing.T) {
	handlerCalled := false
	handler := CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !handlerCalled {
		t.Error("Handler should be called for POST")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestChain_AppliesMiddlewareInOrder(t *testing.T) {
	var order []string

	first := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first-before")
			next(w, r)
			order = append(order, "first-after")
		}
	}

	second := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second-before")
			next(w, r)
			order = append(order, "second-after")
		}
	}

	handler := Chain(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}, first, second)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	expected := []string{"first-before", "second-before", "handler", "second-after", "first-after"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChain_EmptyMiddlewares(t *testing.T) {
	handlerCalled := false
	handler := Chain(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !handlerCalled {
		t.Error("Handler should be called with empty middleware chain")
	}
}

// Tests for CORSWithConfig - origin whitelist functionality

func TestCORSWithConfig_AllowedOriginEchoed(t *testing.T) {
	allowedOrigins := []string{"https://example.com", "http://localhost:5173"}
	handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
}

func TestCORSWithConfig_DisallowedOriginNoHeaders(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should be empty for disallowed origin, got %q", got)
	}
}

func TestCORSWithConfig_SameOriginNoHeader(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Same-origin requests don't include Origin header
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	// Should still work but no CORS headers needed
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCORSWithConfig_PreflightAllowedOrigin(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	handlerCalled := false
	handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
}

func TestCORSWithConfig_PreflightDisallowedOrigin(t *testing.T) {
	allowedOrigins := []string{"https://example.com"}
	handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler(rec, req)

	// Preflight should complete but without CORS headers
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should be empty for disallowed origin, got %q", got)
	}
}

func TestCORSWithConfig_MultipleAllowedOrigins(t *testing.T) {
	allowedOrigins := []string{"https://prod.example.com", "http://localhost:5173", "https://staging.example.com"}

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"https://prod.example.com", true},
		{"http://localhost:5173", true},
		{"https://staging.example.com", true},
		{"https://evil.com", false},
		{"http://localhost:3000", false}, // Different port
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			handler := CORSWithConfig(allowedOrigins)(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler(rec, req)

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.allowed && got != tt.origin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.origin)
			}
			if !tt.allowed && got != "" {
				t.Errorf("Access-Control-Allow-Origin should be empty, got %q", got)
			}
		})
	}
}

func TestCORSWithConfig_EmptyAllowedOrigins(t *testing.T) {
	// With no allowed origins, all cross-origin requests should be rejected
	handler := CORSWithConfig(nil)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should be empty with no allowed origins, got %q", got)
	}
}
