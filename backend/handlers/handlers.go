// ABOUTME: HTTP handlers for capacity analyzer API endpoints
// ABOUTME: Provides health check, dashboard, and resource-specific endpoints

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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
	infrastructureState *models.InfrastructureState
	scenarioCalc        *services.ScenarioCalculator
	infraMutex          sync.RWMutex
}

func NewHandler(cfg *config.Config, cache *cache.Cache) *Handler {
	h := &Handler{
		cfg:          cfg,
		cache:        cache,
		cfClient:     services.NewCFClient(cfg.CFAPIUrl, cfg.CFUsername, cfg.CFPassword),
		scenarioCalc: services.NewScenarioCalculator(),
	}

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

	return h
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"cf_api":   "ok",
		"bosh_api": "not_configured",
		"cache_status": map[string]bool{
			"cells_cached": false,
			"apps_cached":  false,
		},
	}

	if h.boshClient != nil {
		resp["bosh_api"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Check cache
	if cached, found := h.cache.Get("dashboard:all"); found {
		log.Println("Serving from cache")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Fetch fresh data
	log.Println("Fetching fresh data")

	resp := models.DashboardResponse{
		Cells:    []models.DiegoCell{},
		Apps:     []models.App{},
		Segments: []models.IsolationSegment{},
		Metadata: models.Metadata{
			Timestamp:     time.Now(),
			Cached:        false,
			BOSHAvailable: h.boshClient != nil,
		},
	}

	// Authenticate with CF API
	if err := h.cfClient.Authenticate(); err != nil {
		log.Printf("CF API authentication error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "CF API authentication failed",
			Details: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Fetch apps from CF API
	apps, err := h.cfClient.GetApps()
	if err != nil {
		log.Printf("CF API GetApps error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "Failed to fetch apps from CF API",
			Details: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}
	resp.Apps = apps

	// Fetch isolation segments from CF API
	segments, err := h.cfClient.GetIsolationSegments()
	if err != nil {
		log.Printf("CF API GetIsolationSegments error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "Failed to fetch isolation segments from CF API",
			Details: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}
	resp.Segments = segments

	// Fetch BOSH cells (optional, degraded mode if fails)
	if h.boshClient != nil {
		cells, err := h.boshClient.GetDiegoCells()
		if err != nil {
			log.Printf("BOSH API error (degraded mode): %v", err)
			resp.Metadata.BOSHAvailable = false
		} else {
			resp.Cells = cells
		}
	}

	// If BOSH didn't provide UsedMB (vitals unavailable), calculate from app metrics
	needsAppCalculation := false
	for _, cell := range resp.Cells {
		if cell.UsedMB == 0 {
			needsAppCalculation = true
			break
		}
	}

	if needsAppCalculation && len(resp.Cells) > 0 && len(resp.Apps) > 0 {
		// Sum actual memory per isolation segment
		segmentMemory := make(map[string]int)
		for _, app := range resp.Apps {
			segmentMemory[app.IsolationSegment] += app.ActualMB
		}

		// Count cells per segment for distribution
		segmentCellCount := make(map[string]int)
		for _, cell := range resp.Cells {
			segmentCellCount[cell.IsolationSegment]++
		}

		// Distribute app memory across cells in segment (only for cells without BOSH data)
		for i := range resp.Cells {
			if resp.Cells[i].UsedMB == 0 {
				segment := resp.Cells[i].IsolationSegment
				cellCount := segmentCellCount[segment]
				if cellCount > 0 && segmentMemory[segment] > 0 {
					resp.Cells[i].UsedMB = segmentMemory[segment] / cellCount
				}
			}
		}
	}

	// Cache result
	h.cache.Set("dashboard:all", resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (h *Handler) HandleManualInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input models.ManualInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	state := input.ToInfrastructureState()

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(state)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comparison)
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: message,
		Code:  code,
	})
}
