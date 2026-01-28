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
	sessionService      *services.SessionService
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
		h.cfClient = services.NewCFClient(cfg.CFAPIUrl, cfg.CFUsername, cfg.CFPassword, cfg.CFSkipSSLValidation)

		// BOSH client is optional
		if cfg.BOSHEnvironment != "" {
			boshClient, err := services.NewBOSHClient(
				cfg.BOSHEnvironment,
				cfg.BOSHClient,
				cfg.BOSHSecret,
				cfg.BOSHCACert,
				cfg.BOSHDeployment,
				cfg.BOSHSkipSSLValidation,
			)
			if err != nil {
				slog.Error("Failed to create BOSH client, running in degraded mode", "error", err)
			} else {
				h.boshClient = boshClient
			}
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

// writeError writes a standardized error response.
func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
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

// SetSessionService sets the session service for auth handlers
func (h *Handler) SetSessionService(svc *services.SessionService) {
	h.sessionService = svc
}
