// ABOUTME: HTTP handlers for bottleneck analysis and recommendations endpoints
// ABOUTME: Provides multi-resource analysis and upgrade path recommendations

package handlers

import (
	"net/http"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// AnalyzeBottleneck returns multi-resource bottleneck analysis.
func (h *Handler) AnalyzeBottleneck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		h.writeError(w, "No infrastructure data. Load via /api/infrastructure or /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	analysis := models.AnalyzeBottleneck(*state)

	h.writeJSON(w, http.StatusOK, analysis)
}

// GetRecommendations returns upgrade path recommendations.
func (h *Handler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		h.writeError(w, "No infrastructure data. Load via /api/infrastructure or /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	analysis := models.AnalyzeBottleneck(*state)
	recommendations := models.GenerateRecommendations(*state)

	response := models.RecommendationsResponse{
		Recommendations:      recommendations,
		ConstrainingResource: analysis.ConstrainingResource,
	}

	h.writeJSON(w, http.StatusOK, response)
}
