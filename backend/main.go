// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
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
	slog.Info("Auth mode", "mode", cfg.AuthMode)
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

	// Configure authentication middleware
	authCfg := middleware.AuthConfig{
		Mode: middleware.AuthMode(cfg.AuthMode),
	}

	// Initialize cache
	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	c := cache.New(cacheTTL)
	slog.Info("Cache initialized", "ttl", cacheTTL)

	// Initialize handlers
	h := handlers.NewHandler(cfg, c)

	// Register all routes with middleware
	mux := http.NewServeMux()
	for _, route := range h.Routes() {
		if route.Handler == nil {
			slog.Error("nil handler during route registration", "path", route.Path, "method", route.Method)
			os.Exit(1)
		}
		// Go 1.22+ pattern: "METHOD /path"
		pattern := route.Method + " " + route.Path

		// Apply middleware chain: CORS -> Auth (if not public) -> LogRequest -> Handler
		var handler http.HandlerFunc
		if route.Public {
			// Public routes: no auth
			handler = middleware.Chain(route.Handler, middleware.CORS, middleware.LogRequest)
		} else {
			// Protected routes: apply auth middleware
			handler = middleware.Chain(route.Handler, middleware.CORS, middleware.Auth(authCfg), middleware.LogRequest)
		}
		mux.HandleFunc(pattern, handler)

		// Backward compatibility: also register without /v1/
		legacyPath := strings.Replace(route.Path, "/api/v1/", "/api/", 1)
		if legacyPath != route.Path {
			legacyPattern := route.Method + " " + legacyPath
			mux.HandleFunc(legacyPattern, handler)
			slog.Debug("Registered route", "pattern", pattern, "legacy", legacyPattern, "public", route.Public)
		} else {
			slog.Debug("Registered route", "pattern", pattern, "public", route.Public)
		}
	}

	// Handle OPTIONS for all /api/ paths (CORS preflight)
	mux.HandleFunc("OPTIONS /api/", middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Start server
	addr := ":" + cfg.Port
	slog.Info("Server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
