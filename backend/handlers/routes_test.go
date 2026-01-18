// ABOUTME: Tests for route table definitions
// ABOUTME: Verifies all routes have required fields and no duplicates

package handlers

import (
	"strings"
	"testing"
)

func TestRoutes_AllRoutesHaveRequiredFields(t *testing.T) {
	h := NewHandler(nil, nil)
	routes := h.Routes()

	if len(routes) == 0 {
		t.Fatal("Routes() returned empty slice")
	}

	for i, route := range routes {
		if route.Method == "" {
			t.Errorf("Route %d: Method is empty", i)
		}
		if route.Path == "" {
			t.Errorf("Route %d: Path is empty", i)
		}
		if route.Handler == nil {
			t.Errorf("Route %d: Handler is nil", i)
		}
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			t.Errorf("Route %d: Path %q must start with /api/v1/", i, route.Path)
		}
	}
}

func TestRoutes_NoDuplicatePaths(t *testing.T) {
	h := NewHandler(nil, nil)
	routes := h.Routes()

	seen := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if seen[key] {
			t.Errorf("Duplicate route: %s", key)
		}
		seen[key] = true
	}
}

func TestRoutes_ExpectedEndpoints(t *testing.T) {
	h := NewHandler(nil, nil)
	routes := h.Routes()

	expected := map[string]bool{
		"GET /api/v1/health":                   false,
		"GET /api/v1/dashboard":                false,
		"GET /api/v1/infrastructure":           false,
		"POST /api/v1/infrastructure/manual":   false,
		"POST /api/v1/infrastructure/state":    false,
		"GET /api/v1/infrastructure/status":    false,
		"POST /api/v1/infrastructure/planning": false,
		"GET /api/v1/infrastructure/apps":      false,
		"POST /api/v1/scenario/compare":        false,
		"GET /api/v1/bottleneck":               false,
		"GET /api/v1/recommendations":          false,
	}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}

	for key, found := range expected {
		if !found {
			t.Errorf("Missing expected route: %s", key)
		}
	}
}
