// ABOUTME: HTTP request logging middleware with correlation IDs.
// ABOUTME: Logs request start/end with method, path, status, and latency.

package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LogRequest logs HTTP requests with timing and correlation ID.
func LogRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		slog.Info("Request started",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		slog.Info("Request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

// generateRequestID creates a short random hex ID.
func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
