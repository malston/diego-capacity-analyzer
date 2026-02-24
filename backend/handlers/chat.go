// ABOUTME: SSE streaming chat handler for AI capacity advisor
// ABOUTME: Validates requests, snapshots infrastructure context, and streams token events

package handlers

import "net/http"

// Chat handles POST /api/v1/chat with SSE streaming responses.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, "Not implemented", http.StatusNotImplemented)
}
