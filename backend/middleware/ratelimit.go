// ABOUTME: Rate limiting middleware with fixed-window counters
// ABOUTME: Provides per-endpoint rate limits keyed by IP, session, or user

package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// counter tracks requests within a fixed time window.
type counter struct {
	count     int
	expiresAt time.Time
}

// RateLimiter enforces a maximum number of requests per time window.
// Each unique key (IP, user, session) gets an independent counter.
type RateLimiter struct {
	mu           sync.Mutex
	windows      map[string]*counter
	limit        int
	window       time.Duration
	sweepCounter int // tracks new windows created; triggers sweep every 100
}

// NewRateLimiter creates a rate limiter that allows limit requests per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]*counter),
		limit:   limit,
		window:  window,
	}
}

// Allow checks whether a request for the given key should be permitted.
// Returns true if within limits, or false with the duration until the window resets.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, exists := rl.windows[key]

	// Start a new window if none exists or the current one expired.
	// Use !now.Before (>=) so the boundary instant starts a new window
	// rather than returning retryAfter==0 while still denying the request.
	if !exists || !now.Before(c.expiresAt) {
		// Delete expired entry to prevent unbounded map growth
		if exists {
			delete(rl.windows, key)
		}
		rl.windows[key] = &counter{
			count:     1,
			expiresAt: now.Add(rl.window),
		}

		// Periodic sweep: clean up all expired entries every 100 new windows.
		// This bounds memory to at most active keys + 100 stale entries.
		rl.sweepCounter++
		if rl.sweepCounter >= 100 {
			rl.sweep(now)
			rl.sweepCounter = 0
		}

		return true, 0
	}

	// Within current window -- counter only accessed while holding rl.mu
	if c.count < rl.limit {
		c.count++
		return true, 0
	}

	// Over limit -- return time until window resets
	retryAfter := c.expiresAt.Sub(now)
	return false, retryAfter
}

// sweep removes all expired entries from the windows map.
// Must be called while holding rl.mu.
func (rl *RateLimiter) sweep(now time.Time) {
	for k, c := range rl.windows {
		if !now.Before(c.expiresAt) {
			delete(rl.windows, k)
		}
	}
}

// ClientIP extracts the client IP from X-Forwarded-For (leftmost) or RemoteAddr.
// This trusts the X-Forwarded-For header, which is safe when the application runs
// behind a trusted reverse proxy (e.g., CF gorouter, ALB) that sets the header.
// If exposed directly to the internet without a proxy, attackers could spoof this
// header to bypass IP-based rate limits.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the leftmost (client-facing) IP. CF gorouter appends the real
		// client IP, so leftmost is correct for deployments behind gorouter.
		// Validate with net.ParseIP to reject garbage values from spoofed headers.
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" && net.ParseIP(ip) != nil {
			return "ip:" + ip
		}
	}

	// Fall back to RemoteAddr, stripping port
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return "ip:" + host
}

// SessionKey extracts the session cookie value as the rate limit key.
// Falls back to ClientIP if no session cookie is present.
func SessionKey(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		return "session:" + cookie.Value
	}
	return ClientIP(r)
}

// UserOrIP extracts the user ID from request context (set by auth middleware).
// Falls back to ClientIP if no user claims are present.
func UserOrIP(r *http.Request) string {
	claims := GetUserClaims(r)
	if claims != nil && claims.UserID != "" {
		return "user:" + claims.UserID
	}
	return ClientIP(r)
}

// RateLimit returns middleware that enforces rate limits using the given limiter and key function.
// If limiter is nil, the middleware is a no-op (disabled mode).
// If keyFunc returns an empty string, the request passes through (unidentifiable client).
func RateLimit(limiter *RateLimiter, keyFunc func(*http.Request) string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Disabled mode: nil limiter or nil keyFunc
			if limiter == nil || keyFunc == nil {
				next(w, r)
				return
			}

			key := keyFunc(r)
			if key == "" {
				next(w, r)
				return
			}

			allowed, retryAfter := limiter.Allow(key)
			if allowed {
				next(w, r)
				return
			}

			// Rate limited
			retrySeconds := int(math.Ceil(retryAfter.Seconds()))
			slog.Warn("Rate limit exceeded", "key", key, "path", r.URL.Path, "retry_after", retrySeconds)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySeconds))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "Rate limit exceeded",
				"retry_after": retrySeconds,
			})
		}
	}
}
