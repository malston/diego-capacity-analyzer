// ABOUTME: HTTP handler for scenario comparison endpoint
// ABOUTME: Provides what-if analysis comparing current vs proposed configurations

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// CompareScenario compares current infrastructure against a proposed scenario.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) CompareScenario(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DOS attacks (Issue #68)
	// MaxBytesReader only triggers on read, so decode body FIRST before state check
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var input models.ScenarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Check if error is due to body size limit
		if err.Error() == "http: request body too large" {
			h.writeError(w, "Request body too large", http.StatusBadRequest)
			return
		}
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		h.writeError(w, "No infrastructure data. Set via /api/v1/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	comparison := h.scenarioCalc.Compare(*state, input)

	// Add recommendations based on current state
	comparison.Recommendations = models.GenerateRecommendations(*state)

	h.writeJSON(w, http.StatusOK, comparison)
}
