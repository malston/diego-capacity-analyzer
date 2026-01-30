// ABOUTME: HTTP handlers for infrastructure management endpoints
// ABOUTME: Handles vSphere integration, manual input, and planning calculations

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// maxRequestBodySize limits JSON request bodies to 1MB to prevent DOS attacks
const maxRequestBodySize = 1 << 20 // 1MB

// AppDetailsResponse contains per-app breakdown of memory, disk, and instances
type AppDetailsResponse struct {
	TotalAppMemoryGB  int          `json:"total_app_memory_gb"`
	TotalAppDiskGB    int          `json:"total_app_disk_gb"`
	TotalAppInstances int          `json:"total_app_instances"`
	Apps              []models.App `json:"apps"`
}

// GetInfrastructure returns live infrastructure data from vSphere.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) GetInfrastructure(w http.ResponseWriter, r *http.Request) {
	// Check if vSphere is configured
	if h.vsphereClient == nil {
		h.writeError(w, "vSphere not configured. Set VSPHERE_HOST, VSPHERE_USERNAME, VSPHERE_PASSWORD, and VSPHERE_DATACENTER environment variables.", http.StatusServiceUnavailable)
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
		h.writeError(w, "Infrastructure service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	defer h.vsphereClient.Disconnect(ctx)

	// Get infrastructure state
	state, err := h.vsphereClient.GetInfrastructureState(ctx)
	if err != nil {
		slog.Error("vSphere inventory fetch failed", "error", err)
		h.writeError(w, "Failed to retrieve infrastructure data", http.StatusInternalServerError)
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

// SetManualInfrastructure accepts manual infrastructure input.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) SetManualInfrastructure(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DOS attacks (Issue #68)
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var input models.ManualInput
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

	state := input.ToInfrastructureState()

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	h.writeJSON(w, http.StatusOK, state)
}

// SetInfrastructureState accepts an InfrastructureState directly (e.g., from vSphere cache).
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) SetInfrastructureState(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DOS attacks (Issue #68)
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var state models.InfrastructureState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		// Check if error is due to body size limit (type assertion is more robust than string matching)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.writeError(w, "Request body too large", http.StatusBadRequest)
			return
		}
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	h.writeJSON(w, http.StatusOK, state)
}

// GetInfrastructureStatus returns the current data source status.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) GetInfrastructureStatus(w http.ResponseWriter, r *http.Request) {
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

// PlanInfrastructure calculates max deployable cells given IaaS capacity.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) PlanInfrastructure(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DOS attacks (Issue #68)
	// MaxBytesReader only triggers on read, so decode body FIRST before state check
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var input models.PlanningInput
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
		h.writeError(w, "No infrastructure data. Load via /api/v1/infrastructure or /api/v1/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	response := h.planningCalc.Plan(*state, input)

	h.writeJSON(w, http.StatusOK, response)
}

// GetInfrastructureApps returns detailed per-app memory, disk, and instance breakdown.
// HTTP method validation handled by Go 1.22+ router pattern matching.
func (h *Handler) GetInfrastructureApps(w http.ResponseWriter, r *http.Request) {
	// Check if CF is configured
	if h.cfClient == nil || h.cfg == nil || h.cfg.CFAPIUrl == "" {
		h.writeError(w, "CF API not configured. Set CF_API_URL, CF_USERNAME, and CF_PASSWORD environment variables.", http.StatusServiceUnavailable)
		return
	}

	// Authenticate with CF
	if err := h.cfClient.Authenticate(); err != nil {
		slog.Error("CF authentication failed", "error", err)
		h.writeError(w, "Authentication service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	// Fetch apps
	apps, err := h.cfClient.GetApps()
	if err != nil {
		slog.Error("Failed to fetch apps from CF", "error", err)
		h.writeError(w, "Failed to retrieve application data", http.StatusInternalServerError)
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

	// Check if context is already cancelled before starting CF API calls
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before CF enrichment: %w", err)
	}

	if err := h.cfClient.Authenticate(); err != nil {
		return err
	}

	// Check context again after authentication
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled after CF authentication: %w", err)
	}

	apps, err := h.cfClient.GetApps()
	if err != nil {
		return err
	}

	var totalMemoryMB, totalDiskMB, totalInstances int
	var maxMemPerInstanceMB int
	for _, app := range apps {
		totalMemoryMB += app.RequestedMB
		totalDiskMB += app.RequestedDiskMB
		totalInstances += app.Instances

		// Track max per-instance memory for chunk size calculation.
		// CF API returns RequestedMB as total memory allocated to all instances
		// of the app (per-instance limit × instance count). We divide to get
		// the per-instance memory limit, which determines staging chunk size.
		if app.Instances > 0 {
			perInstanceMB := app.RequestedMB / app.Instances
			if perInstanceMB > maxMemPerInstanceMB {
				maxMemPerInstanceMB = perInstanceMB
			}
		}
	}

	// Round to nearest GB instead of truncating (add 512MB before dividing)
	state.TotalAppMemoryGB = (totalMemoryMB + 512) / 1024
	state.TotalAppDiskGB = (totalDiskMB + 512) / 1024
	state.TotalAppInstances = totalInstances
	state.MaxInstanceMemoryMB = maxMemPerInstanceMB

	return nil
}
