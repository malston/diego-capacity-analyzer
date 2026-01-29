// ABOUTME: Declarative route table for API endpoints
// ABOUTME: Defines all routes with their HTTP methods and handlers

package handlers

import "net/http"

// Route defines an API endpoint with its HTTP method and handler.
type Route struct {
	Method  string           // HTTP method (GET, POST, etc.)
	Path    string           // URL path (e.g., "/api/v1/health")
	Handler http.HandlerFunc // Handler function
	Public  bool             // If true, no authentication required
}

// Routes returns all API routes for registration.
// Routes use /api/v1/ prefix; legacy /api/ routes are registered separately.
func (h *Handler) Routes() []Route {
	return []Route{
		// Health & Status (public - no auth required)
		{Method: http.MethodGet, Path: "/api/v1/health", Handler: h.Health, Public: true},
		{Method: http.MethodGet, Path: "/api/v1/dashboard", Handler: h.Dashboard},

		// Authentication (public - handles own auth)
		{Method: http.MethodPost, Path: "/api/v1/auth/login", Handler: h.Login, Public: true},
		{Method: http.MethodGet, Path: "/api/v1/auth/me", Handler: h.Me, Public: true},
		{Method: http.MethodPost, Path: "/api/v1/auth/logout", Handler: h.Logout, Public: true},
		{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Handler: h.Refresh, Public: true},

		// Infrastructure
		{Method: http.MethodGet, Path: "/api/v1/infrastructure", Handler: h.GetInfrastructure},
		{Method: http.MethodPost, Path: "/api/v1/infrastructure/manual", Handler: h.SetManualInfrastructure},
		{Method: http.MethodPost, Path: "/api/v1/infrastructure/state", Handler: h.SetInfrastructureState},
		{Method: http.MethodGet, Path: "/api/v1/infrastructure/status", Handler: h.GetInfrastructureStatus},
		{Method: http.MethodPost, Path: "/api/v1/infrastructure/planning", Handler: h.PlanInfrastructure},
		{Method: http.MethodGet, Path: "/api/v1/infrastructure/apps", Handler: h.GetInfrastructureApps},

		// Scenario
		{Method: http.MethodPost, Path: "/api/v1/scenario/compare", Handler: h.CompareScenario},

		// Analysis
		{Method: http.MethodGet, Path: "/api/v1/bottleneck", Handler: h.AnalyzeBottleneck},
		{Method: http.MethodGet, Path: "/api/v1/recommendations", Handler: h.GetRecommendations},

		// CF API Proxy (requires valid session - tokens never exposed to frontend)
		{Method: http.MethodGet, Path: "/api/v1/cf/isolation-segments", Handler: h.CFProxyIsolationSegments},
		{Method: http.MethodGet, Path: "/api/v1/cf/isolation-segments/{guid}", Handler: h.CFProxyIsolationSegmentByGUID},
		{Method: http.MethodGet, Path: "/api/v1/cf/apps", Handler: h.CFProxyApps},
		{Method: http.MethodGet, Path: "/api/v1/cf/apps/{guid}/processes", Handler: h.CFProxyAppProcesses},
		{Method: http.MethodGet, Path: "/api/v1/cf/processes/{guid}/stats", Handler: h.CFProxyProcessStats},
		{Method: http.MethodGet, Path: "/api/v1/cf/spaces/{guid}", Handler: h.CFProxySpaces},

		// Documentation (public - no auth required)
		{Method: http.MethodGet, Path: "/api/v1/openapi.yaml", Handler: h.OpenAPISpec, Public: true},
	}
}
