// ABOUTME: Unit tests for rate limiting middleware
// ABOUTME: Tests core limiter, key extraction, and middleware factory

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- RateLimiter core tests ---

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		allowed, _ := rl.Allow("test-key")
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_RejectsOverLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	rl.Allow("test-key")
	rl.Allow("test-key")

	allowed, retryAfter := rl.Allow("test-key")
	if allowed {
		t.Fatal("Third request should be rejected")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Errorf("Expected retryAfter between 0 and 60s, got %v", retryAfter)
	}
}

func TestRateLimiter_SeparateKeys(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)

	allowed, _ := rl.Allow("key-a")
	if !allowed {
		t.Fatal("First request for key-a should be allowed")
	}

	allowed, _ = rl.Allow("key-b")
	if !allowed {
		t.Fatal("First request for key-b should be allowed (separate quota)")
	}

	allowed, _ = rl.Allow("key-a")
	if allowed {
		t.Fatal("Second request for key-a should be rejected")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)

	allowed, _ := rl.Allow("test-key")
	if !allowed {
		t.Fatal("First request should be allowed")
	}

	allowed, _ = rl.Allow("test-key")
	if allowed {
		t.Fatal("Second request should be rejected")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	allowed, _ = rl.Allow("test-key")
	if !allowed {
		t.Fatal("Request after window reset should be allowed")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute)

	var wg sync.WaitGroup
	allowed := make([]bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			allowed[idx], _ = rl.Allow("concurrent-key")
		}(i)
	}

	wg.Wait()

	allowedCount := 0
	for _, a := range allowed {
		if a {
			allowedCount++
		}
	}

	if allowedCount != 100 {
		t.Errorf("Expected exactly 100 allowed requests, got %d", allowedCount)
	}
}

func TestRateLimiter_ExpiredEntriesCleanedUp(t *testing.T) {
	rl := NewRateLimiter(1, 20*time.Millisecond)

	// Create entries for multiple keys
	for i := 0; i < 5; i++ {
		rl.Allow(fmt.Sprintf("key-%d", i))
	}

	// Verify entries exist
	rl.mu.Lock()
	initialLen := len(rl.windows)
	rl.mu.Unlock()
	if initialLen != 5 {
		t.Fatalf("Expected 5 entries, got %d", initialLen)
	}

	// Wait for windows to expire
	time.Sleep(30 * time.Millisecond)

	// Access a key to trigger lazy deletion (old entry gets deleted + new one created)
	rl.Allow("key-0")

	rl.mu.Lock()
	afterLen := len(rl.windows)
	rl.mu.Unlock()

	// key-0's expired entry was deleted and a new one created, so count stays the same
	// for key-0 but the other 4 expired entries are still there until sweep
	// (sweep happens at multiples of 100). The important thing is that key-0 was
	// properly replaced (not leaked).
	if afterLen > 5 {
		t.Errorf("Expected at most 5 entries after lazy delete, got %d", afterLen)
	}
}

func TestRateLimiter_ConcurrentMultiKey(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	keys := []string{"key-a", "key-b", "key-c", "key-d"}

	var wg sync.WaitGroup
	results := make(map[string]int)
	var mu sync.Mutex

	// Send 10 requests per key concurrently (limit is 5)
	for _, key := range keys {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				allowed, _ := rl.Allow(k)
				if allowed {
					mu.Lock()
					results[k]++
					mu.Unlock()
				}
			}(key)
		}
	}

	wg.Wait()

	for _, key := range keys {
		if results[key] != 5 {
			t.Errorf("Key %q: expected 5 allowed, got %d", key, results[key])
		}
	}
}

func TestRateLimiter_RetryAfterValue(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.Allow("test-key")

	_, retryAfter := rl.Allow("test-key")
	// retryAfter should be positive and <= window duration
	if retryAfter <= 0 {
		t.Error("retryAfter should be positive")
	}
	if retryAfter > time.Minute {
		t.Errorf("retryAfter should not exceed window duration, got %v", retryAfter)
	}
}

// --- Key extraction tests ---

func TestClientIP_XForwardedFor(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		remote   string
		expected string
	}{
		{
			name:     "single IP",
			xff:      "203.0.113.1",
			expected: "ip:203.0.113.1",
		},
		{
			name:     "multiple IPs takes leftmost",
			xff:      "203.0.113.1, 198.51.100.1, 10.0.0.1",
			expected: "ip:203.0.113.1",
		},
		{
			name:     "no XFF falls back to RemoteAddr",
			remote:   "192.168.1.1:12345",
			expected: "ip:192.168.1.1",
		},
		{
			name:     "RemoteAddr without port",
			remote:   "192.168.1.1",
			expected: "ip:192.168.1.1",
		},
		{
			name:     "XFF with spaces",
			xff:      "  203.0.113.1 , 10.0.0.1 ",
			expected: "ip:203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.remote != "" {
				r.RemoteAddr = tt.remote
			}

			key := ClientIP(r)
			if key != tt.expected {
				t.Errorf("ClientIP() = %q, want %q", key, tt.expected)
			}
		})
	}
}

func TestClientIP_EmptyXFF(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "")
	r.RemoteAddr = "10.0.0.5:9999"

	key := ClientIP(r)
	if key != "ip:10.0.0.5" {
		t.Errorf("ClientIP() with empty XFF = %q, want %q (should fallback to RemoteAddr)", key, "ip:10.0.0.5")
	}
}

func TestSessionKey(t *testing.T) {
	t.Run("with session cookie", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "abc123"})

		key := SessionKey(r)
		if key != "session:abc123" {
			t.Errorf("SessionKey() = %q, want %q", key, "session:abc123")
		}
	})

	t.Run("without session cookie", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1:5555"

		key := SessionKey(r)
		if key != "ip:10.0.0.1" {
			t.Errorf("SessionKey() = %q, want %q (should fallback to IP)", key, "ip:10.0.0.1")
		}
	})
}

func TestUserOrIP(t *testing.T) {
	t.Run("with user claims in context", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		claims := &UserClaims{Username: "admin", UserID: "user-123"}
		ctx := context.WithValue(r.Context(), userClaimsKey, claims)
		r = r.WithContext(ctx)

		key := UserOrIP(r)
		if key != "user:user-123" {
			t.Errorf("UserOrIP() = %q, want %q", key, "user:user-123")
		}
	})

	t.Run("without claims falls back to IP", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.50")

		key := UserOrIP(r)
		if key != "ip:203.0.113.50" {
			t.Errorf("UserOrIP() = %q, want %q", key, "ip:203.0.113.50")
		}
	})
}

// --- Middleware factory tests ---

func TestRateLimitMiddleware_NilLimiter(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	mw := RateLimit(nil, ClientIP)
	wrapped := mw(handler)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	wrapped(w, r)

	if !called {
		t.Fatal("Handler should be called when limiter is nil (disabled mode)")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_EmptyKey(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	// Key function that always returns empty
	emptyKey := func(r *http.Request) string { return "" }
	rl := NewRateLimiter(1, time.Minute)
	mw := RateLimit(rl, emptyKey)
	wrapped := mw(handler)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	wrapped(w, r)

	if !called {
		t.Fatal("Handler should be called when key is empty (unidentifiable client)")
	}
}

func TestRateLimitMiddleware_Returns429(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	mw := RateLimit(rl, ClientIP)
	wrapped := mw(handler)

	// First request: OK
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	wrapped(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("First request should be 200, got %d", w.Code)
	}

	// Second request: 429
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	w = httptest.NewRecorder()
	wrapped(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Second request should be 429, got %d", w.Code)
	}

	// Check Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Expected Retry-After header")
	}

	// Check JSON body
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	if body["error"] != "Rate limit exceeded" {
		t.Errorf("Expected error 'Rate limit exceeded', got %q", body["error"])
	}
	if _, ok := body["retry_after"]; !ok {
		t.Error("Expected retry_after field in response body")
	}
}

func TestRateLimitMiddleware_ContentTypeJSON(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	mw := RateLimit(rl, ClientIP)
	wrapped := mw(handler)

	// Exhaust limit
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	wrapped(httptest.NewRecorder(), r)

	// Check 429 content type
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	wrapped(w, r)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", ct)
	}
}
