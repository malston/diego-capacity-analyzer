// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
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

	// Initialize JWKS client for JWT signature verification (optional, graceful degradation)
	var jwksClient *services.JWKSClient
	uaaURL := discoverUAAURL(cfg)

	// Create HTTP client with same TLS settings as CF API
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.CFSkipSSLValidation, //nolint:gosec // Operator-controlled setting
			},
		},
	}

	jwksClient, err = services.NewJWKSClient(uaaURL, httpClient)
	if err != nil {
		slog.Warn("Failed to initialize JWKS client, Bearer token authentication unavailable",
			"error", err,
			"uaa_url", uaaURL,
		)
	} else {
		slog.Info("JWKS client initialized", "uaa_url", uaaURL)
	}

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
		JWKSClient:       jwksClient,
	}
	slog.Info("Auth mode configured", "mode", authMode)

	// Configure CORS middleware with allowed origins
	corsMiddleware := middleware.CORSWithConfig(cfg.CORSAllowedOrigins)
	if len(cfg.CORSAllowedOrigins) > 0 {
		slog.Info("CORS configured with origin whitelist", "origins", cfg.CORSAllowedOrigins)
	} else {
		slog.Warn("CORS_ALLOWED_ORIGINS not set, cross-origin requests will be blocked")
	}

	// Configure rate limiters (nil if disabled)
	var rateLimiters map[string]func(http.HandlerFunc) http.HandlerFunc
	if cfg.RateLimitEnabled {
		window := time.Minute
		rateLimiters = map[string]func(http.HandlerFunc) http.HandlerFunc{
			"auth":    middleware.RateLimit(middleware.NewRateLimiter(cfg.RateLimitAuth, window), middleware.ClientIP),
			"refresh": middleware.RateLimit(middleware.NewRateLimiter(cfg.RateLimitRefresh, window), middleware.SessionKey),
			"write":   middleware.RateLimit(middleware.NewRateLimiter(cfg.RateLimitWrite, window), middleware.UserOrIP),
			"":        middleware.RateLimit(middleware.NewRateLimiter(cfg.RateLimitDefault, window), middleware.UserOrIP),
		}
		slog.Info("Rate limiting enabled",
			"auth", cfg.RateLimitAuth,
			"refresh", cfg.RateLimitRefresh,
			"write", cfg.RateLimitWrite,
			"default", cfg.RateLimitDefault,
		)
	} else {
		// All tiers map to a nil-limiter no-op
		noOp := middleware.RateLimit(nil, nil)
		rateLimiters = map[string]func(http.HandlerFunc) http.HandlerFunc{
			"auth": noOp, "refresh": noOp, "write": noOp, "": noOp,
		}
		slog.Info("Rate limiting disabled")
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

		// Build middleware chain based on route properties
		// Order: CORS -> CSRF -> Auth (if protected) -> RateLimit (if not exempt) -> LogRequest -> Handler
		var handler http.HandlerFunc
		if route.RateLimit == "none" {
			// Exempt routes: no rate limiting
			if route.Public {
				handler = middleware.Chain(route.Handler, corsMiddleware, middleware.CSRF(), middleware.LogRequest)
			} else {
				handler = middleware.Chain(route.Handler, corsMiddleware, middleware.CSRF(), middleware.Auth(authCfg), middleware.LogRequest)
			}
		} else {
			// Rate-limited routes
			rlMiddleware, ok := rateLimiters[route.RateLimit]
			if !ok {
				slog.Error("Unknown rate limit tier", "tier", route.RateLimit, "path", route.Path)
				os.Exit(1)
			}
			if route.Public {
				handler = middleware.Chain(route.Handler, corsMiddleware, middleware.CSRF(), rlMiddleware, middleware.LogRequest)
			} else {
				handler = middleware.Chain(route.Handler, corsMiddleware, middleware.CSRF(), middleware.Auth(authCfg), rlMiddleware, middleware.LogRequest)
			}
		}
		mux.HandleFunc(pattern, handler)

		// Backward compatibility: also register without /v1/
		legacyPath := strings.Replace(route.Path, "/api/v1/", "/api/", 1)
		if legacyPath != route.Path {
			legacyPattern := route.Method + " " + legacyPath
			mux.HandleFunc(legacyPattern, handler)
			slog.Debug("Registered route", "pattern", pattern, "legacy", legacyPattern, "public", route.Public, "rateLimit", route.RateLimit)
		} else {
			slog.Debug("Registered route", "pattern", pattern, "public", route.Public, "rateLimit", route.RateLimit)
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

// discoverUAAURL discovers the UAA URL from the CF API /v3/info endpoint.
// Falls back to deriveUAAFromCFAPI if discovery fails (network error, non-200, invalid JSON).
// This function always returns a valid URL string (never fails).
func discoverUAAURL(cfg *config.Config) string {
	// Create HTTP client with same TLS settings as CF API
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.CFSkipSSLValidation, //nolint:gosec // Operator-controlled setting
			},
		},
	}

	// Fetch CF API info endpoint
	infoURL := strings.TrimSuffix(cfg.CFAPIUrl, "/") + "/v3/info"
	resp, err := httpClient.Get(infoURL)
	if err != nil {
		slog.Debug("Failed to fetch CF API info, using fallback URL derivation", "error", err)
		return deriveUAAFromCFAPI(cfg.CFAPIUrl)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("CF API info returned non-200, using fallback URL derivation", "status", resp.StatusCode)
		return deriveUAAFromCFAPI(cfg.CFAPIUrl)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Debug("Failed to read CF API info response, using fallback URL derivation", "error", err)
		return deriveUAAFromCFAPI(cfg.CFAPIUrl)
	}

	// Parse the info response to extract UAA URL
	var info struct {
		Links struct {
			Login struct {
				Href string `json:"href"`
			} `json:"login"`
			UAA struct {
				Href string `json:"href"`
			} `json:"uaa"`
		} `json:"links"`
	}

	if err := json.Unmarshal(body, &info); err != nil {
		slog.Debug("Failed to parse CF API info response, using fallback URL derivation", "error", err)
		return deriveUAAFromCFAPI(cfg.CFAPIUrl)
	}

	// Try login.href first (preferred for JWKS), then uaa.href
	if info.Links.Login.Href != "" {
		slog.Debug("Discovered UAA URL from CF API links.login", "url", info.Links.Login.Href)
		return info.Links.Login.Href
	}

	if info.Links.UAA.Href != "" {
		slog.Debug("Discovered UAA URL from CF API links.uaa", "url", info.Links.UAA.Href)
		return info.Links.UAA.Href
	}

	slog.Debug("CF API info missing login/uaa links, using fallback URL derivation")
	return deriveUAAFromCFAPI(cfg.CFAPIUrl)
}

// deriveUAAFromCFAPI derives the UAA URL by replacing "api." with "login." in the CF API URL.
// Example: https://api.sys.example.com -> https://login.sys.example.com
//
// Limitations: This simple replacement only works for standard CF deployments where the
// login server uses the same domain structure. For non-standard deployments (custom domains,
// different UAA hostnames), the CF API /v3/info endpoint should be used for discovery.
func deriveUAAFromCFAPI(cfAPIURL string) string {
	return strings.Replace(cfAPIURL, "api.", "login.", 1)
}
