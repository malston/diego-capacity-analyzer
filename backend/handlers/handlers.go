// ABOUTME: HTTP handlers for capacity analyzer API endpoints
// ABOUTME: Provides health check, dashboard, and resource-specific endpoints

package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
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
		slog.Debug("Dashboard cache hit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Fetch fresh data
	slog.Debug("Dashboard cache miss, fetching fresh data")

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
		slog.Error("CF API authentication failed", "error", err)
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
		slog.Error("CF API GetApps failed", "error", err)
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
		slog.Error("CF API GetIsolationSegments failed", "error", err)
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
			slog.Warn("BOSH API error, entering degraded mode", "error", err)
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

	// Cache result with shorter TTL for live BOSH/CF data
	h.cache.SetWithTTL("dashboard:all", resp, time.Duration(h.cfg.DashboardTTL)*time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
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

// HandleSetInfrastructureState accepts an InfrastructureState directly (e.g., from vSphere cache)
func (h *Handler) HandleSetInfrastructureState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var state models.InfrastructureState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(state)
}

// HandleInfrastructure returns live infrastructure data from vSphere
func (h *Handler) HandleInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if vSphere is configured
	if h.vsphereClient == nil {
		writeError(w, "vSphere not configured. Set VSPHERE_HOST, VSPHERE_USERNAME, VSPHERE_PASSWORD, and VSPHERE_DATACENTER environment variables.", http.StatusServiceUnavailable)
		return
	}

	// Check cache first
	cacheKey := "infrastructure:vsphere"
	if cached, found := h.cache.Get(cacheKey); found {
		slog.Debug("Infrastructure cache hit")
		state := cached.(models.InfrastructureState)
		state.Cached = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
		return
	}

	// Connect to vSphere
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.vsphereClient.Connect(ctx); err != nil {
		slog.Error("vSphere connection failed", "error", err)
		writeError(w, "Failed to connect to vSphere: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer h.vsphereClient.Disconnect(ctx)

	// Get infrastructure state
	state, err := h.vsphereClient.GetInfrastructureState(ctx)
	if err != nil {
		slog.Error("vSphere inventory fetch failed", "error", err)
		writeError(w, "Failed to get vSphere inventory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with CF app data (total app memory, disk, instances)
	if err := h.enrichWithCFAppData(ctx, &state); err != nil {
		slog.Warn("Failed to enrich with CF app data, continuing with vSphere-only data",
			"error", err,
			"cf_configured", h.cfClient != nil,
			"cf_api_url", h.cfg.CFAPIUrl)
		// Continue without CF data - vSphere infrastructure data is still useful
	}

	// Cache result
	h.cache.SetWithTTL(cacheKey, state, time.Duration(h.cfg.VSphereCacheTTL)*time.Second)

	// Store as current infrastructure state for scenario calculations
	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// HandleInfrastructureStatus returns the current data source status
func (h *Handler) HandleInfrastructureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	status := map[string]interface{}{
		"vsphere_configured": h.vsphereClient != nil,
		"has_data":           state != nil,
	}

	if state != nil {
		status["source"] = state.Source
		status["name"] = state.Name
		status["cluster_count"] = len(state.Clusters)
		status["host_count"] = state.TotalHostCount
		status["cell_count"] = state.TotalCellCount
		status["timestamp"] = state.Timestamp

		// Add bottleneck summary
		analysis := models.AnalyzeBottleneck(*state)
		status["constraining_resource"] = analysis.ConstrainingResource
		status["bottleneck_summary"] = analysis.Summary

		// Add capacity metrics for CLI
		status["memory_utilization"] = state.HostMemoryUtilizationPercent
		status["ha_min_host_failures_survived"] = state.HAMinHostFailuresSurvived
		status["ha_status"] = state.HAStatus

		// Calculate N-1 capacity utilization (percentage of N-1 memory used by cells)
		if state.TotalN1MemoryGB > 0 {
			n1CapacityPercent := (float64(state.TotalCellMemoryGB) / float64(state.TotalN1MemoryGB)) * 100.0
			status["n1_capacity_percent"] = n1CapacityPercent
		} else {
			// Single-host cluster or no N-1 capacity available
			status["n1_capacity_percent"] = 0.0
			status["n1_status"] = "unavailable"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
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

// HandleInfrastructurePlanning calculates max deployable cells given IaaS capacity
func (h *Handler) HandleInfrastructurePlanning(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	var input models.PlanningInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := h.planningCalc.Plan(*state, input)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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

// enrichWithCFAppData populates app-related fields from CF API
func (h *Handler) enrichWithCFAppData(ctx context.Context, state *models.InfrastructureState) error {
	if h.cfClient == nil || h.cfg == nil || h.cfg.CFAPIUrl == "" {
		return nil // No CF client configured, skip enrichment
	}

	if err := h.cfClient.Authenticate(); err != nil {
		return err
	}

	apps, err := h.cfClient.GetApps()
	if err != nil {
		return err
	}

	var totalMemoryMB, totalDiskMB, totalInstances int
	for _, app := range apps {
		totalMemoryMB += app.RequestedMB
		totalDiskMB += app.RequestedDiskMB
		totalInstances += app.Instances
	}

	// Round to nearest GB instead of truncating (add 512MB before dividing)
	state.TotalAppMemoryGB = (totalMemoryMB + 512) / 1024
	state.TotalAppDiskGB = (totalDiskMB + 512) / 1024
	state.TotalAppInstances = totalInstances

	return nil
}

// AppDetailsResponse contains per-app breakdown of memory, disk, and instances
type AppDetailsResponse struct {
	TotalAppMemoryGB  int          `json:"total_app_memory_gb"`
	TotalAppDiskGB    int          `json:"total_app_disk_gb"`
	TotalAppInstances int          `json:"total_app_instances"`
	Apps              []models.App `json:"apps"`
}

// HandleInfrastructureApps returns detailed per-app memory, disk, and instance breakdown
func (h *Handler) HandleInfrastructureApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if CF is configured
	if h.cfClient == nil || h.cfg == nil || h.cfg.CFAPIUrl == "" {
		writeError(w, "CF API not configured. Set CF_API_URL, CF_USERNAME, and CF_PASSWORD environment variables.", http.StatusServiceUnavailable)
		return
	}

	// Authenticate with CF
	if err := h.cfClient.Authenticate(); err != nil {
		slog.Error("CF authentication failed", "error", err)
		writeError(w, "Failed to authenticate with CF: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Fetch apps
	apps, err := h.cfClient.GetApps()
	if err != nil {
		slog.Error("Failed to fetch apps from CF", "error", err)
		writeError(w, "Failed to fetch apps: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate totals
	var totalMemoryMB, totalDiskMB, totalInstances int
	for _, app := range apps {
		totalMemoryMB += app.RequestedMB
		totalDiskMB += app.RequestedDiskMB
		totalInstances += app.Instances
	}

	response := AppDetailsResponse{
		// Round to nearest GB instead of truncating (add 512MB before dividing)
		TotalAppMemoryGB:  (totalMemoryMB + 512) / 1024,
		TotalAppDiskGB:    (totalDiskMB + 512) / 1024,
		TotalAppInstances: totalInstances,
		Apps:              apps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
