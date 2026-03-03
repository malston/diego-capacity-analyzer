// ABOUTME: Tests for chat feedback endpoint
// ABOUTME: Covers auth, validation, slog logging, and server-side truncation

package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// feedbackTestClaims reuses the shared test claims from chat_test.go.
var feedbackTestClaims = testClaims

// captureLogHandler is a slog.Handler that records log records for test assertions.
type captureLogHandler struct {
	records []slog.Record
}

func (h *captureLogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureLogHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *captureLogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureLogHandler) WithGroup(_ string) slog.Handler      { return h }

func TestChatFeedback(t *testing.T) {
	tests := []struct {
		name           string
		authenticated  bool
		body           string
		wantStatus     int
		wantError      string
		wantLogMessage string
	}{
		{
			name:          "valid feedback with rating up",
			authenticated: true,
			body:          `{"message_index":0,"rating":"up","truncated_question":"How is capacity?"}`,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "valid feedback with rating down",
			authenticated: true,
			body:          `{"message_index":2,"rating":"down","truncated_question":"Tell me about cells"}`,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "valid feedback with rating none",
			authenticated: true,
			body:          `{"message_index":1,"rating":"none","truncated_question":""}`,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "unauthenticated request returns 401",
			authenticated: false,
			body:          `{"message_index":0,"rating":"up","truncated_question":"test"}`,
			wantStatus:    http.StatusUnauthorized,
			wantError:     "authentication required",
		},
		{
			name:          "invalid rating returns 400",
			authenticated: true,
			body:          `{"message_index":0,"rating":"maybe","truncated_question":"test"}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     "rating",
		},
		{
			name:          "empty rating returns 400",
			authenticated: true,
			body:          `{"message_index":0,"rating":"","truncated_question":"test"}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     "rating",
		},
		{
			name:          "negative message_index returns 400",
			authenticated: true,
			body:          `{"message_index":-1,"rating":"up","truncated_question":"test"}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     "message_index",
		},
		{
			name:          "empty body returns 400",
			authenticated: true,
			body:          ``,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "malformed JSON returns 400",
			authenticated: true,
			body:          `{not valid`,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "long truncated_question is server-side truncated",
			authenticated: true,
			body:          `{"message_index":0,"rating":"up","truncated_question":"` + strings.Repeat("a", 200) + `"}`,
			wantStatus:    http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newChatTestHandler(&mockChatProvider{})

			// Set up log capture
			logCapture := &captureLogHandler{}
			origLogger := slog.Default()
			slog.SetDefault(slog.New(logCapture))
			defer slog.SetDefault(origLogger)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/feedback", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.authenticated {
				req = middleware.WithUserClaims(req, feedbackTestClaims)
			}
			w := httptest.NewRecorder()

			h.ChatFeedback(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d; body: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantError != "" {
				var resp models.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if !strings.Contains(strings.ToLower(resp.Error), strings.ToLower(tt.wantError)) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, resp.Error)
				}
			}

			// For successful requests, verify slog was called
			if tt.wantStatus == http.StatusNoContent {
				found := false
				for _, rec := range logCapture.records {
					if rec.Message == "chat feedback" {
						found = true
						// Verify expected attributes
						var hasUsername, hasRating, hasIndex, hasQuestion bool
						rec.Attrs(func(a slog.Attr) bool {
							switch a.Key {
							case "username":
								hasUsername = a.Value.String() == "testuser"
							case "rating":
								hasRating = true
							case "message_index":
								hasIndex = true
							case "question":
								hasQuestion = true
							}
							return true
						})
						if !hasUsername {
							t.Error("slog record missing or incorrect 'username' attribute")
						}
						if !hasRating {
							t.Error("slog record missing 'rating' attribute")
						}
						if !hasIndex {
							t.Error("slog record missing 'message_index' attribute")
						}
						if !hasQuestion {
							t.Error("slog record missing 'question' attribute")
						}
					}
				}
				if !found {
					t.Error("expected slog.Info with message 'chat feedback' to be called")
				}
			}
		})
	}
}

func TestChatFeedback_TruncatesLongQuestion(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	logCapture := &captureLogHandler{}
	origLogger := slog.Default()
	slog.SetDefault(slog.New(logCapture))
	defer slog.SetDefault(origLogger)

	longQuestion := strings.Repeat("x", 200)
	body := `{"message_index":0,"rating":"up","truncated_question":"` + longQuestion + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = middleware.WithUserClaims(req, feedbackTestClaims)
	w := httptest.NewRecorder()

	h.ChatFeedback(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// Verify the logged question is truncated to 100 chars
	for _, rec := range logCapture.records {
		if rec.Message == "chat feedback" {
			rec.Attrs(func(a slog.Attr) bool {
				if a.Key == "question" {
					if len(a.Value.String()) > 100 {
						t.Errorf("expected question to be truncated to 100 chars, got %d", len(a.Value.String()))
					}
				}
				return true
			})
		}
	}
}
