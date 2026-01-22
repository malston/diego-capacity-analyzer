// ABOUTME: Declarative route table for API endpoints
// ABOUTME: Defines all routes with their HTTP methods and handlers

package handlers

import "net/http"

// Route defines an API endpoint with its HTTP method and handler.
type Route struct {
	Method  string           // HTTP method (GET, POST, etc.)
	Path    string           // URL path (e.g., "/api/v1/health")
	Handler http.HandlerFunc // Handler function
}

// Routes returns all API routes for registration.
// Routes use /api/v1/ prefix; legacy /api/ routes are registered separately.
func (h *Handler) Routes() []Route {
	return []Route{
		// Health & Status
		{Method: http.MethodGet, Path: "/api/v1/health", Handler: h.Health},
		{Method: http.MethodGet, Path: "/api/v1/dashboard", Handler: h.Dashboard},

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

		// Documentation
		{Method: http.MethodGet, Path: "/api/v1/openapi.yaml", Handler: h.OpenAPISpec},
	}
}
