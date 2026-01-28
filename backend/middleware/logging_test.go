// ABOUTME: Tests for request logging middleware
// ABOUTME: Verifies path sanitization prevents log injection attacks

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test that paths are sanitized to prevent log injection attacks.
// Issue #77: Log Injection via Unsanitized Request Paths

func TestSanitizePath_RemovesNewlines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "path with newline injection",
			input: "/api/v1/dashboard\nAdmin access granted for user attacker",
			want:  "/api/v1/dashboardAdmin access granted for user attacker",
		},
		{
			name:  "path with carriage return",
			input: "/api/test\rmalicious",
			want:  "/api/testmalicious",
		},
		{
			name:  "path with CRLF",
			input: "/api/test\r\ninjected line",
			want:  "/api/testinjected line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizePath_RemovesControlCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "path with tab",
			input: "/api/test\tvalue",
			want:  "/api/testvalue",
		},
		{
			name:  "path with null byte",
			input: "/api/test\x00value",
			want:  "/api/testvalue",
		},
		{
			name:  "path with bell character",
			input: "/api/test\x07value",
			want:  "/api/testvalue",
		},
		{
			name:  "path with escape sequence",
			input: "/api/test\x1b[31mred\x1b[0m",
			want:  "/api/test[31mred[0m",
		},
		{
			name:  "path with DEL character",
			input: "/api/test\x7fvalue",
			want:  "/api/testvalue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizePath_PreservesValidCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal path",
			input: "/api/v1/dashboard",
			want:  "/api/v1/dashboard",
		},
		{
			name:  "path with query string chars",
			input: "/api/v1/apps?limit=10&offset=0",
			want:  "/api/v1/apps?limit=10&offset=0",
		},
		{
			name:  "path with URL encoded chars",
			input: "/api/v1/apps%2Ftest",
			want:  "/api/v1/apps%2Ftest",
		},
		{
			name:  "path with hyphens and underscores",
			input: "/api/v1/diego-cells/cell_01",
			want:  "/api/v1/diego-cells/cell_01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogRequest_SetsRequestIDHeader(t *testing.T) {
	handler := LogRequest(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID header should be set")
	}
	if len(requestID) != 16 { // 8 bytes = 16 hex chars
		t.Errorf("X-Request-ID length = %d, want 16", len(requestID))
	}
}

func TestLogRequest_CapturesStatusCode(t *testing.T) {
	handler := LogRequest(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusCreated)
	}
}
