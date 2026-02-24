# Architecture

**Analysis Date:** 2026-02-24

## Pattern Overview

**Overall:** Full-stack BFF (Backend for Frontend) pattern with three discrete applications sharing no code.

**Key Characteristics:**

- Go HTTP backend acts as secure proxy between frontend and CF/BOSH/vSphere APIs
- OAuth tokens are never exposed to the browser; backend manages session state with httpOnly cookies
- Frontend is a React SPA that communicates exclusively through the backend REST API
- A standalone CLI tool (`cli/`) provides TUI and CI/CD access to the same backend API
- All three applications are independently versioned and built

## Layers

**Configuration:**

- Purpose: Load, validate, and expose typed settings from environment variables
- Location: `backend/config/`
- Contains: `config.go` - struct with all settings, `Load()` function with validation
- Depends on: Standard library `os`, `strconv`, `strings`
- Used by: `backend/main.go` to construct all other components

**Infrastructure/Middleware:**

- Purpose: Cross-cutting HTTP concerns applied per-route
- Location: `backend/middleware/`
- Contains: `chain.go`, `auth.go`, `rbac.go`, `csrf.go`, `cors.go`, `ratelimit.go`, `logging.go`, `errors.go`
- Depends on: `backend/services` (for `JWKSClient`, session validation)
- Used by: `backend/main.go` via `middleware.Chain()`

**Services:**

- Purpose: External API clients and domain calculators
- Location: `backend/services/`
- Contains:
  - `cfapi.go` - CF API client (OAuth2, apps, isolation segments)
  - `boshapi.go` - BOSH Director client (Diego cell VMs and vitals)
  - `logcache.go` - Log Cache client (container memory metrics)
  - `vsphere.go` - vCenter client via govmomi
  - `scenario.go` - Scenario comparison calculator
  - `planning.go` - Infrastructure planning calculator
  - `session.go` - Session store backed by in-memory cache
  - `jwks.go` - JWKS client for JWT signature verification
- Depends on: `backend/models`, `backend/cache`
- Used by: `backend/handlers`

**Handlers:**

- Purpose: HTTP request handling, response serialization, route definition
- Location: `backend/handlers/`
- Contains:
  - `handlers.go` - `Handler` struct, `writeJSON`/`writeError` helpers
  - `routes.go` - Declarative `Routes()` method returning all `Route` definitions
  - `health.go` - Health check and dashboard endpoints
  - `auth.go` - Login, logout, token refresh, session management
  - `cfproxy.go` - CF API proxy endpoints (tokens never forwarded to frontend)
  - `infrastructure.go` - vSphere integration, manual infrastructure input
  - `scenario.go` - Scenario comparison endpoint
  - `analysis.go` - Bottleneck analysis and recommendations
  - `openapi.go` - OpenAPI spec endpoint
- Depends on: `backend/services`, `backend/models`, `backend/cache`, `backend/config`, `backend/middleware`
- Used by: `backend/main.go`

**Models:**

- Purpose: Shared data structures serialized over the API
- Location: `backend/models/`
- Contains: `models.go` (core), `auth.go` (session/login models)
- Depends on: Standard library only
- Used by: All other backend packages

**Cache:**

- Purpose: Thread-safe in-memory TTL cache
- Location: `backend/cache/`
- Contains: `cache.go` - `sync.Map`-backed store with per-entry expiry and background cleanup
- Depends on: Standard library only
- Used by: `backend/handlers`, `backend/services`

**Logger:**

- Purpose: Structured logging initialization
- Location: `backend/logger/`
- Contains: `logger.go` - `Init()` configures `log/slog` from `LOG_LEVEL`/`LOG_FORMAT` env vars
- Depends on: Standard library `log/slog`
- Used by: `backend/main.go` at startup

**Frontend:**

- Purpose: React SPA rendering capacity analysis dashboard
- Location: `frontend/src/`
- Contains:
  - `App.jsx` - Root; wraps with `AuthProvider`, conditionally renders `Login` or `TASCapacityAnalyzer`
  - `TASCapacityAnalyzer.jsx` - Main dashboard component
  - `components/` - All UI components (see Structure)
  - `contexts/AuthContext.jsx` - React auth state, calls `cfAuth` service
  - `contexts/ToastContext.jsx` - Global toast notification state
  - `services/cfAuth.js` - BFF auth calls (login/logout/me)
  - `services/cfApi.js` - CF API proxy calls (isolation segments, apps)
  - `services/scenarioApi.js` - Scenario comparison API calls
  - `services/apiClient.js` - Shared `apiFetch` wrapper with structured error types
  - `utils/csrf.js` - Reads `DIEGO_CSRF` cookie and returns `X-CSRF-Token` header
  - `config/resourceConfig.js` - Resource type definitions
  - `config/vmPresets.js` - VM size presets

**CLI:**

- Purpose: Terminal UI and CI/CD tooling consuming the backend API
- Location: `cli/`
- Contains:
  - `main.go` - Entry point
  - `cmd/` - Cobra commands: `root.go`, `health.go`, `status.go`, `check.go`, `scenario.go`
  - `internal/client/` - HTTP client wrapping backend REST API
  - `internal/tui/` - Bubble Tea TUI components (dashboard, wizard, comparison, etc.)

## Data Flow

**Browser Authentication:**

1. User submits credentials to `POST /api/v1/auth/login`
2. `handlers/auth.go` performs OAuth2 password grant against CF UAA
3. Backend creates server-side session in `cache` via `SessionService`
4. Backend sets `DIEGO_SESSION` (httpOnly) and `DIEGO_CSRF` (JavaScript-readable) cookies
5. Frontend `AuthContext` updates state; no token ever reaches JavaScript

**Dashboard Data Fetch:**

1. Frontend calls `GET /api/v1/dashboard` with session cookie
2. `middleware.Auth` validates session cookie, injects `UserClaims` into context
3. `handlers.Dashboard` checks in-memory cache (`dashboard:all` key)
4. On cache miss: `CFClient.Authenticate()` gets UAA token, then `GetApps()` and `GetIsolationSegments()`
5. `BOSHClient.GetDiegoCells()` fetches Diego cell VMs and memory vitals (degraded gracefully if unavailable)
6. If BOSH vitals missing, memory is estimated by distributing app memory across cells
7. Response cached with `DashboardTTL` (default 30s), returned as `DashboardResponse`

**Infrastructure Planning (vSphere):**

1. Frontend calls `GET /api/v1/infrastructure`
2. `handlers.GetInfrastructure` connects to vSphere via `govmomi`
3. CF app data is fetched and merged for enrichment
4. State stored in `Handler.infrastructureState` (mutex-protected) and in cache
5. Frontend calls `POST /api/v1/infrastructure/planning` with cell specs
6. `PlanningCalculator.Calculate()` computes max deployable cells considering memory and CPU

**CF API Proxy:**

1. Frontend calls `GET /api/v1/cf/isolation-segments` (or other proxy routes)
2. `handlers.CFProxyIsolationSegments` calls `cfClient` using server-side token
3. CF response is forwarded; CF tokens never leave the backend

**State Management (Frontend):**

- Auth state: `AuthContext` (React Context)
- Toast notifications: `ToastContext` (React Context)
- All other state: local component state within `TASCapacityAnalyzer.jsx`
- No Redux or Zustand; state is passed down as props or read from contexts

## Key Abstractions

**Handler:**

- Purpose: Groups all HTTP handlers with shared dependencies (`cfg`, `cache`, service clients)
- Examples: `backend/handlers/handlers.go` (struct definition)
- Pattern: Method receivers on `*Handler`; response helpers `writeJSON`/`writeError` encapsulate serialization

**Route:**

- Purpose: Typed route definition with middleware metadata
- Examples: `backend/handlers/routes.go`
- Pattern: Struct with `Method`, `Path`, `Handler`, `Public`, `RateLimit`, `Role` fields; `Routes()` returns slice

**Middleware:**

- Purpose: Composable `func(http.HandlerFunc) http.HandlerFunc` transformers
- Examples: `backend/middleware/auth.go`, `backend/middleware/csrf.go`, `backend/middleware/rbac.go`
- Pattern: Each middleware is a constructor returning the transformer; chained via `middleware.Chain()`

**Service Client:**

- Purpose: Encapsulates external API authentication and data retrieval
- Examples: `backend/services/cfapi.go`, `backend/services/boshapi.go`, `backend/services/vsphere.go`
- Pattern: Struct with credentials and HTTP client; `New*()` constructor; methods accept `context.Context`

**Calculator:**

- Purpose: Pure computation over models, no I/O
- Examples: `backend/services/scenario.go` (`ScenarioCalculator`), `backend/services/planning.go` (`PlanningCalculator`)
- Pattern: Struct with no state; `New*Calculator()` constructor; methods take and return model types

## Entry Points

**Backend HTTP Server:**

- Location: `backend/main.go`
- Triggers: Process start, reads `PORT` env var (default 8080)
- Responsibilities: Initialize logger, load config, create cache, create services, create handlers, build per-route middleware chains, register routes with Go 1.22 `METHOD /path` pattern, start `http.ListenAndServe`

**Frontend SPA:**

- Location: `frontend/src/main.jsx`
- Triggers: Browser load
- Responsibilities: Mount React app, `AuthProvider` wraps entire tree, renders `Login` or `TASCapacityAnalyzer` based on session

**CLI Entry Point:**

- Location: `cli/main.go`
- Triggers: Terminal invocation of `diego-capacity` binary
- Responsibilities: Execute Cobra root command; on TTY without `--json` launches Bubble Tea TUI; subcommands provide non-interactive output

## Error Handling

**Strategy:** Errors surface to handlers, which translate them to structured JSON responses.

**Patterns:**

- Services return `(value, error)` using Go idioms; errors are wrapped with `fmt.Errorf("context: %w", err)`
- Handlers call `h.writeError(w, message, code)` or `h.writeErrorWithDetails(w, message, details, code)` for all error responses
- All error responses use `models.ErrorResponse{Error, Details, Code}` shape
- Middleware uses `middleware.writeJSONError(w, message, code)` (in `errors.go`) for consistency
- External service failures are logged with `slog.Error`/`slog.Warn`; degraded mode is used (BOSH failures do not abort dashboard)
- Frontend `apiFetch` catches `TypeError` network errors and raises `ApiConnectionError`; 403s raise `ApiPermissionError`

## Cross-Cutting Concerns

**Logging:** `log/slog` throughout backend; `logger.Init()` configures format (text/JSON) and level from `LOG_LEVEL`/`LOG_FORMAT` env vars; frontend uses `console.warn/error` for diagnostic logging

**Validation:** Input validation in handlers using JSON decode + manual field checks; config validation in `config.Load()` with explicit field checks; request body size capped at 1MB (`maxRequestBodySize`)

**Authentication:** BFF OAuth2 pattern; `middleware.Auth` supports three modes (`disabled`/`optional`/`required`); Bearer token validated via JWKS signature verification; session cookie validated via `SessionService`; CSRF protection via double-submit cookie pattern for session-authenticated POST/PUT/DELETE

**RBAC:** Two roles - `viewer` (scope `diego-analyzer.viewer`) and `operator` (scope `diego-analyzer.operator`); `middleware.RequireRole` gates specific routes; operator scope required for write endpoints (`SetManualInfrastructure`, `SetInfrastructureState`)

**Context Propagation:** All service client methods accept `context.Context` as first argument, enabling cancellation and timeout propagation from request context

---

_Architecture analysis: 2026-02-24_
