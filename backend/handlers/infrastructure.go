// ABOUTME: HTTP handlers for infrastructure management endpoints
// ABOUTME: Handles vSphere integration, manual input, and planning calculations

package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// AppDetailsResponse contains per-app breakdown of memory, disk, and instances
type AppDetailsResponse struct {
	TotalAppMemoryGB  int          `json:"total_app_memory_gb"`
	TotalAppDiskGB    int          `json:"total_app_disk_gb"`
	TotalAppInstances int          `json:"total_app_instances"`
	Apps              []models.App `json:"apps"`
}

// GetInfrastructure returns live infrastructure data from vSphere
func (h *Handler) GetInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if vSphere is configured
	if h.vsphereClient == nil {
		h.writeErrorMethod(w, "vSphere not configured. Set VSPHERE_HOST, VSPHERE_USERNAME, VSPHERE_PASSWORD, and VSPHERE_DATACENTER environment variables.", http.StatusServiceUnavailable)
		return
	}

	// Check cache first
	cacheKey := "infrastructure:vsphere"
	if cached, found := h.cache.Get(cacheKey); found {
		slog.Debug("Infrastructure cache hit")
		state := cached.(models.InfrastructureState)
		state.Cached = true
		h.writeJSON(w, http.StatusOK, state)
		return
	}

	// Connect to vSphere
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.vsphereClient.Connect(ctx); err != nil {
		slog.Error("vSphere connection failed", "error", err)
		h.writeErrorMethod(w, "Failed to connect to vSphere: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer h.vsphereClient.Disconnect(ctx)

	// Get infrastructure state
	state, err := h.vsphereClient.GetInfrastructureState(ctx)
	if err != nil {
		slog.Error("vSphere inventory fetch failed", "error", err)
		h.writeErrorMethod(w, "Failed to get vSphere inventory: "+err.Error(), http.StatusInternalServerError)
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

	h.writeJSON(w, http.StatusOK, state)
}

// SetManualInfrastructure accepts manual infrastructure input
func (h *Handler) SetManualInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input models.ManualInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeErrorMethod(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	state := input.ToInfrastructureState()

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	h.writeJSON(w, http.StatusOK, state)
}

// SetInfrastructureState accepts an InfrastructureState directly (e.g., from vSphere cache)
func (h *Handler) SetInfrastructureState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var state models.InfrastructureState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		h.writeErrorMethod(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	h.writeJSON(w, http.StatusOK, state)
}

// GetInfrastructureStatus returns the current data source status
func (h *Handler) GetInfrastructureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	h.writeJSON(w, http.StatusOK, status)
}

// PlanInfrastructure calculates max deployable cells given IaaS capacity
func (h *Handler) PlanInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		h.writeErrorMethod(w, "No infrastructure data. Load via /api/infrastructure or /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	var input models.PlanningInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeErrorMethod(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := h.planningCalc.Plan(*state, input)

	h.writeJSON(w, http.StatusOK, response)
}

// GetInfrastructureApps returns detailed per-app memory, disk, and instance breakdown
func (h *Handler) GetInfrastructureApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorMethod(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if CF is configured
	if h.cfClient == nil || h.cfg == nil || h.cfg.CFAPIUrl == "" {
		h.writeErrorMethod(w, "CF API not configured. Set CF_API_URL, CF_USERNAME, and CF_PASSWORD environment variables.", http.StatusServiceUnavailable)
		return
	}

	// Authenticate with CF
	if err := h.cfClient.Authenticate(); err != nil {
		slog.Error("CF authentication failed", "error", err)
		h.writeErrorMethod(w, "Failed to authenticate with CF: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Fetch apps
	apps, err := h.cfClient.GetApps()
	if err != nil {
		slog.Error("Failed to fetch apps from CF", "error", err)
		h.writeErrorMethod(w, "Failed to fetch apps: "+err.Error(), http.StatusInternalServerError)
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

	h.writeJSON(w, http.StatusOK, response)
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
