// ABOUTME: Chat feedback endpoint for response quality signaling
// ABOUTME: Logs feedback via slog for analytics without persisting state

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
)

// FeedbackRequest represents a chat feedback submission.
type FeedbackRequest struct {
	MessageIndex      int    `json:"message_index"`
	Rating            string `json:"rating"`
	TruncatedQuestion string `json:"truncated_question"`
}

// validRatings defines the allowed values for the rating field.
var validRatings = map[string]bool{
	"up":   true,
	"down": true,
	"none": true,
}

// maxQuestionLength is the maximum character length for the truncated question.
const maxQuestionLength = 100

// ChatFeedback handles POST /api/v1/chat/feedback requests.
// It validates the request, logs feedback via slog, and returns 204 No Content.
func (h *Handler) ChatFeedback(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		h.writeError(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !validRatings[req.Rating] {
		h.writeError(w, "Invalid rating: must be up, down, or none", http.StatusBadRequest)
		return
	}

	if req.MessageIndex < 0 {
		h.writeError(w, "Invalid message_index: must be non-negative", http.StatusBadRequest)
		return
	}

	question := req.TruncatedQuestion
	if len(question) > maxQuestionLength {
		question = question[:maxQuestionLength]
	}

	slog.Info("chat feedback",
		"username", claims.Username,
		"rating", req.Rating,
		"message_index", req.MessageIndex,
		"question", question,
	)

	w.WriteHeader(http.StatusNoContent)
}
