// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/logger"
	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

func main() {
	// Initialize structured logging
	logger.Init()

	// Load .env file if present (optional, won't fail if missing)
	// Try current directory first, then parent (project root)
	if err := godotenv.Load(); err == nil {
		slog.Info("Loaded .env file", "path", ".env")
	} else if err := godotenv.Load("../.env"); err == nil {
		slog.Info("Loaded .env file", "path", "../.env")
	} else {
		slog.Debug("No .env file found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting Diego Capacity Analyzer Backend")
	slog.Info("CF API configured")
	slog.Debug("CF API endpoint", "url", cfg.CFAPIUrl)
	if cfg.CFSkipSSLValidation {
		slog.Warn("CF_SKIP_SSL_VALIDATION=true, TLS certificate verification disabled for CF/Log Cache")
	}
	if cfg.BOSHEnvironment != "" {
		slog.Info("BOSH configured")
		slog.Debug("BOSH endpoint", "environment", cfg.BOSHEnvironment)
		if cfg.BOSHSkipSSLValidation {
			slog.Warn("BOSH_SKIP_SSL_VALIDATION=true, TLS certificate verification disabled for BOSH")
		}
	} else {
		slog.Warn("BOSH not configured, running in degraded mode")
	}
	if cfg.VSphereConfigured() {
		slog.Info("vSphere configured")
		slog.Debug("vSphere endpoint", "host", cfg.VSphereHost, "datacenter", cfg.VSphereDatacenter)
		if cfg.VSphereInsecure {
			slog.Warn("VSPHERE_INSECURE=true, TLS certificate verification disabled for vSphere")
		}
	} else {
		slog.Info("vSphere not configured, manual mode only")
	}

	// Initialize cache
	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	c := cache.New(cacheTTL)
	slog.Info("Cache initialized", "ttl", cacheTTL)

	// Initialize session service for BFF OAuth pattern
	sessionService := services.NewSessionService(c)
	slog.Info("Session service initialized")

	// Configure authentication middleware with session cookie support
	authMode, err := middleware.ValidateAuthMode(cfg.AuthMode)
	if err != nil {
		slog.Error("Invalid AUTH_MODE", "error", err)
		os.Exit(1)
	}
	// Session validator function for auth middleware
	sessionValidator := func(sessionID string) *middleware.UserClaims {
		session, err := sessionService.Get(sessionID)
		if err != nil {
			return nil
		}
		return &middleware.UserClaims{
			Username: session.Username,
			UserID:   session.UserID,
		}
	}
	authCfg := middleware.AuthConfig{
		Mode:             authMode,
		SessionValidator: sessionValidator,
	}
	slog.Info("Auth mode configured", "mode", authMode)

	// Configure CORS middleware with allowed origins
	corsMiddleware := middleware.CORSWithConfig(cfg.CORSAllowedOrigins)
	if len(cfg.CORSAllowedOrigins) > 0 {
		slog.Info("CORS configured with origin whitelist", "origins", cfg.CORSAllowedOrigins)
	} else {
		slog.Warn("CORS_ALLOWED_ORIGINS not set, cross-origin requests will be blocked")
	}

	// Initialize handlers
	h := handlers.NewHandler(cfg, c)
	h.SetSessionService(sessionService)

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
			handler = middleware.Chain(route.Handler, corsMiddleware, middleware.LogRequest)
		} else {
			// Protected routes: apply auth middleware
			handler = middleware.Chain(route.Handler, corsMiddleware, middleware.Auth(authCfg), middleware.LogRequest)
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
	mux.HandleFunc("OPTIONS /api/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Response is handled by CORS middleware for preflight
	}))

	// Start server
	addr := ":" + cfg.Port
	slog.Info("Server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
