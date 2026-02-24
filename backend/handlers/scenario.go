// ABOUTME: HTTP handler for scenario comparison endpoint
// ABOUTME: Provides what-if analysis comparing current vs proposed configurations

package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const maxUserScenarios = 1000

// CompareScenario compares current infrastructure against a proposed scenario.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) CompareScenario(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DOS attacks (Issue #68)
	// MaxBytesReader only triggers on read, so decode body FIRST before state check
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var input models.ScenarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Check if error is due to body size limit (type assertion is more robust than string matching)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
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

	// Store scenario result for authenticated users so the AI advisor can reference it
	claims := middleware.GetUserClaims(r)
	if claims != nil {
		h.userScenariosMutex.Lock()
		if len(h.userScenarios) >= maxUserScenarios {
			slog.Warn("user scenarios map at capacity, skipping storage",
				"username", claims.Username,
				"capacity", maxUserScenarios,
			)
		} else {
			h.userScenarios[claims.Username] = &comparison
		}
		h.userScenariosMutex.Unlock()
	}

	h.writeJSON(w, http.StatusOK, comparison)
}
