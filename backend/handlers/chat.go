// ABOUTME: SSE streaming chat handler for AI capacity advisor
// ABOUTME: Validates requests, snapshots infrastructure context, and streams token events

package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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
func (h *Handler) buildChatSystemPrompt() string {
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

	ctx := ai.BuildContext(input)
	return ai.BuildSystemPrompt(ctx)
}

// Chat handles POST /api/v1/chat with SSE streaming responses.
// Phase 1 validates the request and returns JSON errors. Phase 2 snapshots
// infrastructure context. Phase 3 streams SSE token events from the AI provider.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	// Phase 1: Pre-stream validation (JSON errors via h.writeError)

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
	systemPrompt := h.buildChatSystemPrompt()

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

	ctx := r.Context()
	tokenCh := h.chatProvider.Chat(ctx, messages, ai.WithSystem(systemPrompt))

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

		case <-ctx.Done():
			// Client disconnected
			return
		}
	}
}
