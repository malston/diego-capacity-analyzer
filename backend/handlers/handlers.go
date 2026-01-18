// ABOUTME: HTTP handlers for capacity analyzer API endpoints
// ABOUTME: Provides health check, dashboard, and resource-specific endpoints

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

type Handler struct {
	cfg                 *config.Config
	cache               *cache.Cache
	cfClient            *services.CFClient
	boshClient          *services.BOSHClient
	vsphereClient       *services.VSphereClient
	infrastructureState *models.InfrastructureState
	scenarioCalc        *services.ScenarioCalculator
	planningCalc        *services.PlanningCalculator
	infraMutex          sync.RWMutex
}

func NewHandler(cfg *config.Config, cache *cache.Cache) *Handler {
	h := &Handler{
		cfg:          cfg,
		cache:        cache,
		scenarioCalc: services.NewScenarioCalculator(),
		planningCalc: services.NewPlanningCalculator(),
	}

	// CF client is optional (for testing)
	if cfg != nil {
		h.cfClient = services.NewCFClient(cfg.CFAPIUrl, cfg.CFUsername, cfg.CFPassword)

		// BOSH client is optional
		if cfg.BOSHEnvironment != "" {
			h.boshClient = services.NewBOSHClient(
				cfg.BOSHEnvironment,
				cfg.BOSHClient,
				cfg.BOSHSecret,
				cfg.BOSHCACert,
				cfg.BOSHDeployment,
			)
		}

		// vSphere client is optional
		if cfg.VSphereConfigured() {
			h.vsphereClient = services.VSphereClientFromEnv(
				cfg.VSphereHost,
				cfg.VSphereUsername,
				cfg.VSpherePassword,
				cfg.VSphereDatacenter,
			)
		}
	}

	return h
}

// writeJSON writes a JSON response with the given status code.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// writeErrorMethod writes a standardized error response.
// Named writeErrorMethod to avoid conflict with existing writeError function.
// Will be renamed to writeError after handlers are migrated.
func (h *Handler) writeErrorMethod(w http.ResponseWriter, message string, code int) {
	h.writeJSON(w, code, models.ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// writeErrorWithDetails writes an error response with additional details.
func (h *Handler) writeErrorWithDetails(w http.ResponseWriter, message, details string, code int) {
	h.writeJSON(w, code, models.ErrorResponse{
		Error:   message,
		Details: details,
		Code:    code,
	})
}

func (h *Handler) EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (h *Handler) HandleScenarioCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		writeError(w, "No infrastructure data. Set via /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	var input models.ScenarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	comparison := h.scenarioCalc.Compare(*state, input)

	// Add recommendations based on current state
	comparison.Recommendations = models.GenerateRecommendations(*state)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comparison)
}

// HandleBottleneckAnalysis returns multi-resource bottleneck analysis
func (h *Handler) HandleBottleneckAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		writeError(w, "No infrastructure data. Load via /api/infrastructure or /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	analysis := models.AnalyzeBottleneck(*state)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

// HandleRecommendations returns upgrade path recommendations
func (h *Handler) HandleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		writeError(w, "No infrastructure data. Load via /api/infrastructure or /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	analysis := models.AnalyzeBottleneck(*state)
	recommendations := models.GenerateRecommendations(*state)

	response := models.RecommendationsResponse{
		Recommendations:      recommendations,
		ConstrainingResource: analysis.ConstrainingResource,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: message,
		Code:  code,
	})
}
