// ABOUTME: HTTP handlers for health and dashboard endpoints
// ABOUTME: Provides API status and live dashboard data

package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// Health returns API health status including CF, BOSH, and cache status.
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

	h.writeJSON(w, http.StatusOK, resp)
}

// Dashboard returns live dashboard data including cells, apps, and segments.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Check cache
	if cached, found := h.cache.Get("dashboard:all"); found {
		slog.Debug("Dashboard cache hit")
		h.writeJSON(w, http.StatusOK, cached)
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
		h.writeError(w, "Authentication service temporarily unavailable", http.StatusInternalServerError)
		return
	}

	// Fetch apps from CF API
	apps, err := h.cfClient.GetApps()
	if err != nil {
		slog.Error("CF API GetApps failed", "error", err)
		h.writeError(w, "Failed to retrieve application data", http.StatusInternalServerError)
		return
	}
	resp.Apps = apps

	// Fetch isolation segments from CF API
	segments, err := h.cfClient.GetIsolationSegments()
	if err != nil {
		slog.Error("CF API GetIsolationSegments failed", "error", err)
		h.writeError(w, "Failed to retrieve isolation segment data", http.StatusInternalServerError)
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

	h.writeJSON(w, http.StatusOK, resp)
}
