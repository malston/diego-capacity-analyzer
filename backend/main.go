// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/logger"
	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
)

func main() {
	// Initialize structured logging
	logger.Init()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting Diego Capacity Analyzer Backend")
	slog.Info("CF API configured", "url", cfg.CFAPIUrl)
	if cfg.BOSHEnvironment != "" {
		slog.Info("BOSH configured", "environment", cfg.BOSHEnvironment)
	} else {
		slog.Warn("BOSH not configured, running in degraded mode")
	}
	if cfg.VSphereConfigured() {
		slog.Info("vSphere configured", "host", cfg.VSphereHost, "datacenter", cfg.VSphereDatacenter)
	} else {
		slog.Info("vSphere not configured, manual mode only")
	}

	// Initialize cache
	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	c := cache.New(cacheTTL)
	slog.Info("Cache initialized", "ttl", cacheTTL)

	// Initialize handlers
	h := handlers.NewHandler(cfg, c)

	// Register routes with logging middleware
	http.HandleFunc("/api/health", h.EnableCORS(middleware.LogRequest(h.Health)))
	http.HandleFunc("/api/dashboard", h.EnableCORS(middleware.LogRequest(h.Dashboard)))
	http.HandleFunc("/api/infrastructure", h.EnableCORS(middleware.LogRequest(h.GetInfrastructure)))
	http.HandleFunc("/api/infrastructure/manual", h.EnableCORS(middleware.LogRequest(h.SetManualInfrastructure)))
	http.HandleFunc("/api/infrastructure/state", h.EnableCORS(middleware.LogRequest(h.SetInfrastructureState)))
	http.HandleFunc("/api/infrastructure/status", h.EnableCORS(middleware.LogRequest(h.GetInfrastructureStatus)))
	http.HandleFunc("/api/infrastructure/planning", h.EnableCORS(middleware.LogRequest(h.PlanInfrastructure)))
	http.HandleFunc("/api/infrastructure/apps", h.EnableCORS(middleware.LogRequest(h.GetInfrastructureApps)))
	http.HandleFunc("/api/scenario/compare", h.EnableCORS(middleware.LogRequest(h.HandleScenarioCompare)))
	http.HandleFunc("/api/bottleneck", h.EnableCORS(middleware.LogRequest(h.HandleBottleneckAnalysis)))
	http.HandleFunc("/api/recommendations", h.EnableCORS(middleware.LogRequest(h.HandleRecommendations)))

	// Start server
	addr := ":" + cfg.Port
	slog.Info("Server listening", "addr", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
