// ABOUTME: CSRF protection middleware using double-submit cookie pattern
// ABOUTME: Validates X-CSRF-Token header matches DIEGO_CSRF cookie for session requests

package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

const (
	csrfCookieName    = "DIEGO_CSRF"
	csrfHeaderName    = "X-CSRF-Token"
	sessionCookieName = "DIEGO_SESSION"

	// base64url encoding of 32 bytes produces 44 characters (with padding)
	csrfTokenLength = 44
)

// CSRF returns middleware that validates CSRF tokens for state-changing requests.
// Validation is skipped for:
//   - GET, HEAD, OPTIONS requests (safe methods)
//   - Requests with Authorization header (Bearer token auth)
//   - Requests without session cookie (not session-authenticated)
//   - Login endpoint (creates a new session, must work with stale cookies)
func CSRF() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip safe methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next(w, r)
				return
			}

			// Skip login endpoint -- it creates a new session and must work
			// even when the browser has a stale session cookie with no CSRF cookie
			if strings.HasSuffix(r.URL.Path, "/auth/login") {
				next(w, r)
				return
			}

			// Skip if using Bearer token auth (CSRF not applicable)
			if r.Header.Get("Authorization") != "" {
				next(w, r)
				return
			}

			// Skip if no session cookie (not session-authenticated)
			sessionCookie, err := r.Cookie(sessionCookieName)
			if err != nil || sessionCookie.Value == "" {
				next(w, r)
				return
			}

			// Session-authenticated request - validate CSRF token
			csrfCookie, err := r.Cookie(csrfCookieName)
			if err != nil || csrfCookie.Value == "" {
				slog.Debug("CSRF rejected: missing cookie", "path", r.URL.Path)
				writeCSRFError(w)
				return
			}

			csrfHeader := r.Header.Get(csrfHeaderName)
			if csrfHeader == "" {
				slog.Debug("CSRF rejected: missing header", "path", r.URL.Path)
				writeCSRFError(w)
				return
			}

			// Validate token lengths before comparison
			if len(csrfCookie.Value) != csrfTokenLength || len(csrfHeader) != csrfTokenLength {
				slog.Debug("CSRF rejected: invalid token length", "path", r.URL.Path)
				writeCSRFError(w)
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) != 1 {
				slog.Debug("CSRF rejected: token mismatch", "path", r.URL.Path)
				writeCSRFError(w)
				return
			}

			slog.Debug("CSRF validated", "path", r.URL.Path)
			next(w, r)
		}
	}
}

func writeCSRFError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "CSRF token missing or invalid",
	})
}
