# Design Document: API Refactoring

**Feature:** `refactor-api`  
**Author:** Auto-generated  
**Date:** 2026-01-16  
**Status:** Draft

---

## 1. Overview

### What We're Building

A comprehensive refactoring of the Diego Capacity Analyzer backend API layer to improve code organization, reduce boilerplate, and establish patterns for future extensibility. This refactoring focuses on **internal code quality** while maintaining 100% backward compatibility with existing API consumers.

### Why We're Building It

The current API implementation has accumulated technical debt that impedes maintainability:

| Problem | Impact |
|---------|--------|
| **Monolithic handler file** | 586 lines in `handlers.go` with 12 handler methods spanning 4 unrelated domains |
| **Repeated boilerplate** | Every handler manually checks HTTP methods (11 instances of `if r.Method != ...`) |
| **Verbose route registration** | Each route requires `h.EnableCORS(middleware.LogRequest(...))` wrapping |
| **CORS as handler method** | `EnableCORS()` is bound to Handler struct rather than being composable middleware |
| **No API versioning** | All endpoints at `/api/*` with no mechanism for backward-compatible evolution |
| **Inconsistent response patterns** | Mix of inline `json.NewEncoder().Encode()` and `writeError()` helper |

### Goals

1. **Improve maintainability** — Split handlers into domain-focused files under 200 lines each
2. **Reduce boilerplate** — Eliminate repeated method checking and middleware wrapping
3. **Enable extensibility** — Add optional `/api/v1/` prefix for future API evolution
4. **Standardize patterns** — Consistent response helpers across all handlers
5. **Maintain compatibility** — Zero breaking changes to existing consumers (CLI, frontend)

### Non-Goals

- Changing request/response formats
- Adding new endpoints
- Introducing external routing frameworks (Gin, Echo, Chi)
- Modifying business logic in services layer

---

## 2. Requirements

### 2.1 Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-1 | All existing endpoints must continue to work with identical request/response formats | P0 |
| FR-2 | Handler code must be split into domain-focused files (health, infrastructure, scenario, analysis) | P1 |
| FR-3 | HTTP method validation must be handled at the router level, not in each handler | P1 |
| FR-4 | Route registration must be declarative (route table pattern) | P1 |
| FR-5 | Response helpers (`writeJSON`, `writeError`) must be standardized across all handlers | P1 |
| FR-6 | Optional `/api/v1/` prefix must be supported alongside existing `/api/` routes | P2 |
| FR-7 | CORS must be implemented as composable middleware, not a Handler method | P2 |

### 2.2 Non-Functional Requirements

| ID | Requirement | Metric |
|----|-------------|--------|
| NFR-1 | No handler file exceeds 200 lines | Measured by `wc -l` |
| NFR-2 | All existing tests pass without modification | 100% pass rate |
| NFR-3 | No new external dependencies added | `go.mod` unchanged |
| NFR-4 | Request latency unchanged | <1ms overhead |
| NFR-5 | Code coverage maintained | ≥80% on handlers package |

### 2.3 Constraints

- **Go standard library only** — No external routing frameworks
- **Backward compatibility** — CLI (`cli/`) and frontend (`frontend/src/services/`) depend on current API
- **Existing test suite** — 45K+ lines of handler tests must continue passing
- **Go 1.22+ features available** — Can use enhanced `http.ServeMux` with method routing

---

## 3. Design

### 3.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              main.go                                     │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    Router Configuration                          │    │
│  │  routes := []Route{                                              │    │
│  │    {Method: "GET",  Path: "/api/v1/health", Handler: ...},      │    │
│  │    {Method: "POST", Path: "/api/v1/infrastructure/manual", ...}, │    │
│  │  }                                                               │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           middleware/                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │   cors.go    │  │  logging.go  │  │  chain.go    │                  │
│  │              │  │              │  │              │                  │
│  │ EnableCORS() │  │ LogRequest() │  │ Chain(...)   │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            handlers/                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │  handler.go  │  │   health.go  │  │infrastructure│  │ scenario.go│  │
│  │              │  │              │  │     .go      │  │            │  │
│  │ Handler{}    │  │ Health()     │  │ GetInfra()   │  │ Compare()  │  │
│  │ writeJSON()  │  │              │  │ SetManual()  │  │            │  │
│  │ writeError() │  │              │  │ GetStatus()  │  │            │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │
│                                                                         │
│  ┌──────────────┐  ┌──────────────┐                                    │
│  │ analysis.go  │  │  routes.go   │                                    │
│  │              │  │              │                                    │
│  │ Bottleneck() │  │ Route{}      │                                    │
│  │ Recommend()  │  │ Routes()     │                                    │
│  └──────────────┘  └──────────────┘                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Component Design

#### 3.2.1 Route Definition (`handlers/routes.go`)

A declarative route table replaces verbose `http.HandleFunc()` calls:

```go
// Route defines an API endpoint with its HTTP method and handler
type Route struct {
    Method  string           // HTTP method (GET, POST, etc.)
    Path    string           // URL path (e.g., "/api/v1/health")
    Handler http.HandlerFunc // Handler function
    Name    string           // Route name for logging/debugging
}

// Routes returns all API routes for registration
func (h *Handler) Routes() []Route {
    return []Route{
        // Health & Status
        {Method: http.MethodGet, Path: "/api/v1/health", Handler: h.Health, Name: "health"},
        {Method: http.MethodGet, Path: "/api/v1/dashboard", Handler: h.Dashboard, Name: "dashboard"},
        
        // Infrastructure
        {Method: http.MethodGet, Path: "/api/v1/infrastructure", Handler: h.GetInfrastructure, Name: "infrastructure.get"},
        {Method: http.MethodPost, Path: "/api/v1/infrastructure/manual", Handler: h.SetManualInfrastructure, Name: "infrastructure.manual"},
        {Method: http.MethodPost, Path: "/api/v1/infrastructure/state", Handler: h.SetInfrastructureState, Name: "infrastructure.state"},
        {Method: http.MethodGet, Path: "/api/v1/infrastructure/status", Handler: h.GetInfrastructureStatus, Name: "infrastructure.status"},
        {Method: http.MethodPost, Path: "/api/v1/infrastructure/planning", Handler: h.PlanInfrastructure, Name: "infrastructure.planning"},
        {Method: http.MethodGet, Path: "/api/v1/infrastructure/apps", Handler: h.GetInfrastructureApps, Name: "infrastructure.apps"},
        
        // Scenario
        {Method: http.MethodPost, Path: "/api/v1/scenario/compare", Handler: h.CompareScenario, Name: "scenario.compare"},
        
        // Analysis
        {Method: http.MethodGet, Path: "/api/v1/bottleneck", Handler: h.AnalyzeBottleneck, Name: "bottleneck"},
        {Method: http.MethodGet, Path: "/api/v1/recommendations", Handler: h.GetRecommendations, Name: "recommendations"},
    }
}
```

#### 3.2.2 Handler Base (`handlers/handler.go`)

Core Handler struct and response helpers:

```go
// Handler provides HTTP handlers for the capacity analyzer API
type Handler struct {
    cfg                 *config.Config
    cache               *cache.Cache
    cfClient            *services.CFClient
    boshClient          *services.BOSHClient
    vsphereClient       *services.VSphereClient
    infrastructureState *models.InfrastructureState
    scenarioCalc        *services.ScenarioCalculator
    planningCalc        *services.PlanningCalculator
    infraMutex          sync.RWMutex
}

// writeJSON writes a JSON response with the given status code
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        slog.Error("Failed to encode JSON response", "error", err)
    }
}

// writeError writes a standardized error response
func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
    h.writeJSON(w, code, models.ErrorResponse{
        Error: message,
        Code:  code,
    })
}

// writeErrorWithDetails writes an error response with additional details
func (h *Handler) writeErrorWithDetails(w http.ResponseWriter, message, details string, code int) {
    h.writeJSON(w, code, models.ErrorResponse{
        Error:   message,
        Details: details,
        Code:    code,
    })
}

// getInfrastructureState safely retrieves current infrastructure state
func (h *Handler) getInfrastructureState() *models.InfrastructureState {
    h.infraMutex.RLock()
    defer h.infraMutex.RUnlock()
    return h.infrastructureState
}

// setInfrastructureState safely updates current infrastructure state
func (h *Handler) setInfrastructureState(state *models.InfrastructureState) {
    h.infraMutex.Lock()
    defer h.infraMutex.Unlock()
    h.infrastructureState = state
}
```

#### 3.2.3 Domain Handler Files

**`handlers/health.go`** (~50 lines)
```go
// Health returns API health status
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

// Dashboard returns live dashboard data
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
    // ... existing logic, using h.writeJSON() and h.writeError()
}
```

**`handlers/infrastructure.go`** (~180 lines)
- `GetInfrastructure` — Live vSphere data
- `SetManualInfrastructure` — Manual input
- `SetInfrastructureState` — Direct state setting
- `GetInfrastructureStatus` — Current status
- `PlanInfrastructure` — Capacity planning
- `GetInfrastructureApps` — App details

**`handlers/scenario.go`** (~60 lines)
- `CompareScenario` — What-if comparison

**`handlers/analysis.go`** (~80 lines)
- `AnalyzeBottleneck` — Bottleneck analysis
- `GetRecommendations` — Upgrade recommendations

#### 3.2.4 Middleware (`middleware/`)

**`middleware/cors.go`** — Extracted from Handler:
```go
// CORS returns middleware that adds CORS headers
func CORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        next(w, r)
    }
}
```

**`middleware/chain.go`** — Middleware composition:
```go
// Chain applies middleware in order (first middleware is outermost)
func Chain(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}
```

#### 3.2.5 Route Registration (`main.go`)

Simplified route registration using Go 1.22+ ServeMux patterns:

```go
func main() {
    // ... initialization ...

    h := handlers.NewHandler(cfg, c)
    mux := http.NewServeMux()

    // Register all routes with middleware
    for _, route := range h.Routes() {
        // Go 1.22+ pattern: "METHOD /path"
        pattern := route.Method + " " + route.Path
        handler := middleware.Chain(route.Handler, middleware.CORS, middleware.LogRequest)
        mux.HandleFunc(pattern, handler)
        
        // Backward compatibility: also register without /v1/
        legacyPath := strings.Replace(route.Path, "/api/v1/", "/api/", 1)
        if legacyPath != route.Path {
            legacyPattern := route.Method + " " + legacyPath
            mux.HandleFunc(legacyPattern, handler)
        }
    }

    // Handle OPTIONS for all /api/ paths (CORS preflight)
    mux.HandleFunc("OPTIONS /api/", middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))

    slog.Info("Server listening", "addr", addr)
    http.ListenAndServe(addr, mux)
}
```

### 3.3 File Structure

```
backend/
├── main.go                      # Entry point, route registration
├── handlers/
│   ├── handler.go               # Handler struct, NewHandler(), response helpers
│   ├── routes.go                # Route definitions
│   ├── health.go                # Health, Dashboard
│   ├── infrastructure.go        # Infrastructure CRUD operations
│   ├── scenario.go              # Scenario comparison
│   ├── analysis.go              # Bottleneck, recommendations
│   └── handlers_test.go         # Existing tests (unchanged)
├── middleware/
│   ├── cors.go                  # CORS middleware (new)
│   ├── logging.go               # Request logging (existing)
│   └── chain.go                 # Middleware chaining (new)
└── ... (other packages unchanged)
```

### 3.4 API Versioning Strategy

| Route Pattern | Behavior |
|---------------|----------|
| `/api/v1/health` | New versioned route |
| `/api/health` | Legacy route (maps to same handler) |

Both patterns are registered and work identically. This allows:
1. Existing consumers to continue working unchanged
2. New consumers to use versioned endpoints
3. Future `/api/v2/` introduction without breaking v1

---

## 4. Implementation Approach

### 4.1 Phase 1: Foundation (Low Risk)

**Goal:** Add new infrastructure without changing existing behavior

1. Create `middleware/cors.go` — Extract CORS logic from Handler
2. Create `middleware/chain.go` — Add middleware composition helper
3. Create `handlers/routes.go` — Define Route struct and Routes() method
4. Update `handlers/handler.go` — Add `writeJSON()` method (keep existing code working)

**Validation:** All existing tests pass, no behavioral changes

### 4.2 Phase 2: Handler Split (Medium Risk)

**Goal:** Split handlers into domain files

1. Create `handlers/health.go` — Move Health, Dashboard handlers
2. Create `handlers/infrastructure.go` — Move infrastructure handlers
3. Create `handlers/scenario.go` — Move scenario handlers
4. Create `handlers/analysis.go` — Move analysis handlers
5. Update handlers to use `h.writeJSON()` / `h.writeError()`
6. Remove duplicate code from original `handlers.go`

**Validation:** All existing tests pass, behavior identical

### 4.3 Phase 3: Route Registration (Medium Risk)

**Goal:** Switch to declarative route registration

1. Update `main.go` to use Routes() pattern
2. Register both `/api/v1/` and `/api/` routes
3. Remove manual method checking from handlers (router handles it)
4. Delete `EnableCORS()` from Handler struct

**Validation:** All tests pass, both route patterns work

### 4.4 Phase 4: Cleanup (Low Risk)

**Goal:** Remove deprecated code, finalize documentation

1. Remove old `handlers.go` (now empty)
2. Update `docs/API.md` to document v1 routes
3. Add migration notes for consumers

**Validation:** Full test suite, manual API testing

### 4.5 Rollback Strategy

Each phase is independently deployable. If issues arise:
- Phase 1: Delete new middleware files
- Phase 2: Revert handler file changes
- Phase 3: Revert main.go to direct http.HandleFunc() calls
- Phase 4: N/A (documentation only)

---

## 5. Testing Strategy

### 5.1 Existing Test Preservation

The existing `handlers_test.go` (45K+ lines) must pass unchanged:

```bash
cd backend && go test ./handlers/... -v
```

Tests exercise handlers through `httptest.NewRecorder()` which is router-agnostic.

### 5.2 New Tests Required

#### 5.2.1 Middleware Tests (`middleware/cors_test.go`)

```go
func TestCORS_AddsHeaders(t *testing.T) {
    handler := middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
    rec := httptest.NewRecorder()
    handler(rec, req)
    
    assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
    assert.Equal(t, "GET, POST, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORS_HandlesPreflight(t *testing.T) {
    handler := middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
        t.Fatal("Handler should not be called for OPTIONS")
    })
    
    req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
    rec := httptest.NewRecorder()
    handler(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
}
```

#### 5.2.2 Route Table Tests (`handlers/routes_test.go`)

```go
func TestRoutes_AllRoutesHaveRequiredFields(t *testing.T) {
    h := handlers.NewHandler(nil, nil)
    routes := h.Routes()
    
    for _, route := range routes {
        assert.NotEmpty(t, route.Method, "Route must have Method")
        assert.NotEmpty(t, route.Path, "Route must have Path")
        assert.NotNil(t, route.Handler, "Route must have Handler")
        assert.True(t, strings.HasPrefix(route.Path, "/api/v1/"), 
            "Route path must start with /api/v1/")
    }
}

func TestRoutes_NoDuplicatePaths(t *testing.T) {
    h := handlers.NewHandler(nil, nil)
    routes := h.Routes()
    
    seen := make(map[string]bool)
    for _, route := range routes {
        key := route.Method + " " + route.Path
        assert.False(t, seen[key], "Duplicate route: %s", key)
        seen[key] = true
    }
}
```

#### 5.2.3 Integration Tests

Verify both legacy and versioned routes work:

```go
func TestLegacyAndVersionedRoutes(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()
    
    endpoints := []struct {
        legacy    string
        versioned string
    }{
        {"/api/health", "/api/v1/health"},
        {"/api/infrastructure/status", "/api/v1/infrastructure/status"},
        {"/api/bottleneck", "/api/v1/bottleneck"},
    }
    
    for _, ep := range endpoints {
        // Test legacy route
        resp, _ := http.Get(server.URL + ep.legacy)
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        // Test versioned route
        resp, _ = http.Get(server.URL + ep.versioned)
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }
}
```

### 5.3 Test Coverage Targets

| Package | Current | Target |
|---------|---------|--------|
| `handlers/` | ~85% | ≥85% |
| `middleware/` | ~70% | ≥90% |

### 5.4 Manual Testing Checklist

- [ ] CLI commands work (`dcli health`, `dcli status`, `dcli check`)
- [ ] Frontend loads dashboard data
- [ ] Frontend scenario comparison works
- [ ] CORS preflight requests succeed from browser
- [ ] Error responses have correct format

---

## 6. Open Questions

| # | Question | Status | Notes |
|---|----------|--------|-------|
| 1 | Should we deprecate `/api/` routes with a warning header? | **Open** | Could add `Deprecation` header to legacy routes to encourage migration |
| 2 | Should method mismatch return 405 with `Allow` header? | **Open** | Go 1.22 ServeMux does this automatically; verify behavior |
| 3 | Should we add request validation middleware? | **Deferred** | Out of scope for this refactor; consider for future |
| 4 | Should handler methods be renamed for consistency? | **Open** | Current: `HandleInfrastructure`, `HandleScenarioCompare`. Proposed: `GetInfrastructure`, `CompareScenario` |
| 5 | Should we use Go 1.22 ServeMux or stay with explicit method checks? | **Decided: Go 1.22** | Project uses Go 1.23, can leverage method-aware routing |

---

## 7. Appendix

### A. Current Handler Method Inventory

| Handler | Method | Path | Domain |
|---------|--------|------|--------|
| `Health` | GET | `/api/health` | health |
| `Dashboard` | GET | `/api/dashboard` | health |
| `HandleInfrastructure` | GET | `/api/infrastructure` | infrastructure |
| `HandleManualInfrastructure` | POST | `/api/infrastructure/manual` | infrastructure |
| `HandleSetInfrastructureState` | POST | `/api/infrastructure/state` | infrastructure |
| `HandleInfrastructureStatus` | GET | `/api/infrastructure/status` | infrastructure |
| `HandleInfrastructurePlanning` | POST | `/api/infrastructure/planning` | infrastructure |
| `HandleInfrastructureApps` | GET | `/api/infrastructure/apps` | infrastructure |
| `HandleScenarioCompare` | POST | `/api/scenario/compare` | scenario |
| `HandleBottleneckAnalysis` | GET | `/api/bottleneck` | analysis |
| `HandleRecommendations` | GET | `/api/recommendations` | analysis |

### B. Proposed Handler Naming

| Current Name | Proposed Name | Rationale |
|--------------|---------------|-----------|
| `HandleInfrastructure` | `GetInfrastructure` | Verb matches HTTP method |
| `HandleManualInfrastructure` | `SetManualInfrastructure` | Clearer intent |
| `HandleSetInfrastructureState` | `SetInfrastructureState` | Remove redundant "Handle" |
| `HandleInfrastructureStatus` | `GetInfrastructureStatus` | Verb matches HTTP method |
| `HandleInfrastructurePlanning` | `PlanInfrastructure` | Action-oriented |
| `HandleInfrastructureApps` | `GetInfrastructureApps` | Verb matches HTTP method |
| `HandleScenarioCompare` | `CompareScenario` | Noun-verb order |
| `HandleBottleneckAnalysis` | `AnalyzeBottleneck` | Action-oriented |
| `HandleRecommendations` | `GetRecommendations` | Verb matches HTTP method |

### C. Dependencies

**No new dependencies required.** Implementation uses only:
- `net/http` (standard library)
- `encoding/json` (standard library)
- `log/slog` (standard library, Go 1.21+)

### D. References

- [Go 1.22 ServeMux Enhancements](https://go.dev/blog/routing-enhancements)
- [HTTP Method Routing in Go 1.22](https://pkg.go.dev/net/http#ServeMux)
- Existing API documentation: `docs/API.md`
