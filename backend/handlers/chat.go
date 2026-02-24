// ABOUTME: SSE streaming chat handler for AI capacity advisor
// ABOUTME: Validates requests, snapshots infrastructure context, and streams token events

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services/ai"
)

const maxChatMessages = 50

// ChatRequest is the POST body for /api/v1/chat.
type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a single conversation turn.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TokenPayload is the SSE data for "token" events.
type TokenPayload struct {
	Text string `json:"text"`
	Seq  int    `json:"seq"`
}

// DonePayload is the SSE data for "done" events.
type DonePayload struct {
	StopReason string     `json:"stop_reason"`
	Usage      UsageStats `json:"usage"`
}

// UsageStats reports token consumption.
type UsageStats struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// ErrorPayload is the SSE data for "error" events.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeSSEEvent writes a single SSE event and flushes immediately.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
	flusher.Flush()
	return nil
}

// buildChatSystemPrompt snapshots the current infrastructure state and builds
// a system prompt combining domain expertise with live context data.
func (h *Handler) buildChatSystemPrompt(username string) string {
	input := ai.ContextInput{
		BOSHConfigured:    h.boshClient != nil,
		VSphereConfigured: h.cfg.VSphereConfigured(),
	}

	if cached, found := h.cache.Get("dashboard:all"); found {
		if dashboard, ok := cached.(models.DashboardResponse); ok {
			input.Dashboard = &dashboard
			// Derive Log Cache availability: true if any app has actual memory data
			for _, app := range dashboard.Apps {
				if app.ActualMB > 0 {
					input.LogCacheAvailable = true
					break
				}
			}
		}
	}

	h.infraMutex.RLock()
	input.Infra = h.infrastructureState
	h.infraMutex.RUnlock()

	h.userScenariosMutex.RLock()
	input.Scenario = h.userScenarios[username]
	h.userScenariosMutex.RUnlock()

	ctx := ai.BuildContext(input)
	return ai.BuildSystemPrompt(ctx)
}

// Chat handles POST /api/v1/chat with SSE streaming responses.
// Phase 1 validates the request and returns JSON errors. Phase 2 snapshots
// infrastructure context. Phase 3 streams SSE token events from the AI provider.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	// Phase 1: Pre-stream validation (JSON errors via h.writeError)

	claims := middleware.GetUserClaims(r)
	if claims == nil {
		h.writeError(w, "authentication required for AI advisor", http.StatusUnauthorized)
		return
	}

	if h.chatProvider == nil {
		h.writeError(w, "AI advisor not configured", http.StatusServiceUnavailable)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		h.writeError(w, "Messages required", http.StatusBadRequest)
		return
	}

	if len(req.Messages) > maxChatMessages {
		h.writeError(w, "Maximum 50 messages per request", http.StatusBadRequest)
		return
	}

	for _, msg := range req.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			h.writeError(w, "Invalid message role: must be 'user' or 'assistant'", http.StatusBadRequest)
			return
		}
		if msg.Content == "" {
			h.writeError(w, "Message content must not be empty", http.StatusBadRequest)
			return
		}
	}

	// Phase 2: Context snapshot
	systemPrompt := h.buildChatSystemPrompt(claims.Username)

	// Phase 3: SSE streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Disable write deadline for long-lived streaming connection
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	// Convert ChatMessages to ai.Messages
	messages := make([]ai.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create a cancelable context so max duration and client disconnect
	// both propagate to the provider via context cancellation.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	tokenCh := h.chatProvider.Chat(ctx, messages, ai.WithSystem(systemPrompt))

	// Max duration timer: caps total stream wall-clock time.
	// When it fires, it signals via maxDurationExceeded before canceling the context.
	maxDurationExceeded := make(chan struct{})
	maxDuration := time.AfterFunc(time.Duration(h.cfg.AIMaxDurationSecs)*time.Second, func() {
		close(maxDurationExceeded)
		cancel()
	})
	defer maxDuration.Stop()

	// Idle timer: fires if no token arrives within the idle window.
	idleTimeout := time.Duration(h.cfg.AIIdleTimeoutSecs) * time.Second
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()

	var seq int
	for {
		select {
		case event, chanOpen := <-tokenCh:
			if !chanOpen {
				// Channel closed without a done event
				return
			}

			if event.Err != nil {
				slog.Warn("AI provider error during streaming",
					"error", event.Err,
					"messages", len(req.Messages),
				)
				writeSSEEvent(w, flusher, "error", ErrorPayload{
					Code:    "provider_error",
					Message: event.Err.Error(),
				})
				return
			}

			if event.Done {
				var usage UsageStats
				if event.Usage != nil {
					usage = UsageStats{
						InputTokens:  event.Usage.InputTokens,
						OutputTokens: event.Usage.OutputTokens,
					}
				}
				writeSSEEvent(w, flusher, "done", DonePayload{
					StopReason: event.StopReason,
					Usage:      usage,
				})
				return
			}

			if event.Text != "" {
				seq++
				writeSSEEvent(w, flusher, "token", TokenPayload{
					Text: event.Text,
					Seq:  seq,
				})
			}

			// Reset idle timer after each event (safe drain pattern)
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(idleTimeout)

		case <-idleTimer.C:
			writeSSEEvent(w, flusher, "error", ErrorPayload{
				Code:    "timeout",
				Message: "No response from AI provider within timeout window",
			})
			return

		case <-ctx.Done():
			// Determine if max duration fired or client disconnected
			select {
			case <-maxDurationExceeded:
				writeSSEEvent(w, flusher, "error", ErrorPayload{
					Code:    "timeout",
					Message: "Response exceeded maximum duration",
				})
			default:
				// Client disconnected -- no one to send to
			}
			return
		}
	}
}
