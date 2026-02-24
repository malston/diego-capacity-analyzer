# Codebase Structure

**Analysis Date:** 2026-02-24

## Directory Layout

```
diego-capacity-analyzer/
├── backend/                    # Go HTTP backend service
│   ├── main.go                 # Entry point, server setup, route registration
│   ├── go.mod / go.sum         # Go module (github.com/markalston/diego-capacity-analyzer/backend)
│   ├── config/                 # Environment-based configuration
│   ├── cache/                  # In-memory TTL cache
│   ├── logger/                 # slog initialization
│   ├── middleware/             # Composable HTTP middleware
│   ├── models/                 # Shared data structures
│   ├── services/               # External API clients and calculators
│   ├── handlers/               # HTTP handlers and route table
│   ├── e2e/                    # End-to-end integration tests (black-box)
│   ├── scripts/                # Utility scripts for backend
│   └── tmp/                    # Air (auto-reload) temp artifacts (not committed)
│
├── frontend/                   # React SPA
│   ├── src/
│   │   ├── App.jsx             # Root component (auth gate)
│   │   ├── main.jsx            # React DOM mount point
│   │   ├── TASCapacityAnalyzer.jsx  # Main dashboard container
│   │   ├── components/         # UI components
│   │   │   └── wizard/         # Multi-step scenario wizard
│   │   │       └── steps/      # Individual wizard step components
│   │   ├── contexts/           # React Context providers (Auth, Toast)
│   │   ├── services/           # API client modules
│   │   ├── utils/              # Pure utility functions (CSRF, metrics, export)
│   │   ├── config/             # Static configuration (resource types, VM presets)
│   │   ├── data/               # Static reference data
│   │   └── test/               # Shared test setup
│   ├── public/
│   │   └── samples/            # Sample infrastructure JSON files
│   ├── package.json
│   └── vite.config.js
│
├── cli/                        # Standalone CLI tool (diego-capacity binary)
│   ├── main.go                 # Entry point
│   ├── go.mod / go.sum         # Separate Go module
│   ├── cmd/                    # Cobra command definitions
│   └── internal/
│       ├── client/             # HTTP client for backend REST API
│       └── tui/                # Bubble Tea TUI components
│
├── config/                     # Deployment configuration examples
├── docs/                       # Documentation, images, plans
├── scripts/                    # Top-level deploy and setup scripts
├── Makefile                    # All development tasks
├── .env.example                # Environment variable template
├── generate-env.sh             # Auto-derive env from Ops Manager
└── .planning/
    └── codebase/               # GSD codebase analysis documents
```

## Directory Purposes

**`backend/config/`:**

- Purpose: Typed configuration loaded from environment variables
- Contains: `config.go` with `Config` struct and `Load()` function
- Key files: `backend/config/config.go`

**`backend/middleware/`:**

- Purpose: HTTP middleware implementing security and observability
- Contains: One file per concern, each exporting a constructor returning `func(http.HandlerFunc) http.HandlerFunc`
- Key files: `backend/middleware/chain.go`, `backend/middleware/auth.go`, `backend/middleware/csrf.go`, `backend/middleware/rbac.go`, `backend/middleware/ratelimit.go`, `backend/middleware/cors.go`, `backend/middleware/logging.go`, `backend/middleware/errors.go`

**`backend/models/`:**

- Purpose: API-serializable data structures shared across backend packages
- Contains: Structs with JSON tags, no methods
- Key files: `backend/models/models.go` (core domain), `backend/models/auth.go` (session/login types)

**`backend/services/`:**

- Purpose: External API clients (CF, BOSH, Log Cache, vSphere) and pure calculators (scenario, planning)
- Contains: One service or calculator per file; test files co-located (`*_test.go`)
- Key files: `backend/services/cfapi.go`, `backend/services/boshapi.go`, `backend/services/vsphere.go`, `backend/services/logcache.go`, `backend/services/scenario.go`, `backend/services/planning.go`, `backend/services/session.go`, `backend/services/jwks.go`

**`backend/handlers/`:**

- Purpose: HTTP handler methods and route definitions
- Contains: Domain-grouped handler files plus `routes.go` for the route table
- Key files: `backend/handlers/handlers.go`, `backend/handlers/routes.go`, `backend/handlers/auth.go`, `backend/handlers/health.go`, `backend/handlers/infrastructure.go`, `backend/handlers/scenario.go`, `backend/handlers/analysis.go`, `backend/handlers/cfproxy.go`, `backend/handlers/openapi.go`, `backend/handlers/openapi.yaml`

**`backend/e2e/`:**

- Purpose: Integration tests that start a real server and test the full HTTP stack
- Contains: Test files only (no production code); all files end in `_test.go`
- Key files: `backend/e2e/auth_test.go`, `backend/e2e/csrf_test.go`, `backend/e2e/rbac_test.go`

**`frontend/src/components/`:**

- Purpose: React UI components
- Contains: `.jsx` component files with co-located `.test.jsx` files
- Key files: `frontend/src/components/Login.jsx`, `frontend/src/components/ScenarioAnalyzer.jsx`, `frontend/src/components/ScenarioResults.jsx`, `frontend/src/components/DataSourceSelector.jsx`, `frontend/src/components/BottleneckCard.jsx`

**`frontend/src/services/`:**

- Purpose: HTTP client modules for backend API calls
- Contains: Service class singletons exported as instances
- Key files: `frontend/src/services/cfAuth.js`, `frontend/src/services/cfApi.js`, `frontend/src/services/apiClient.js`, `frontend/src/services/scenarioApi.js`

**`frontend/src/utils/`:**

- Purpose: Pure utility functions with no side effects
- Contains: `csrf.js` (CSRF header helper), `metricsCalculations.js` (metric math), `exportMarkdown.js` (report export)
- Key files: `frontend/src/utils/csrf.js`, `frontend/src/utils/metricsCalculations.js`

**`cli/cmd/`:**

- Purpose: Cobra command definitions
- Contains: One file per subcommand: `root.go`, `health.go`, `status.go`, `check.go`, `scenario.go`

**`cli/internal/tui/`:**

- Purpose: Bubble Tea TUI components organized by screen/feature
- Contains: Subdirectory per TUI screen (`dashboard`, `wizard`, `comparison`, `menu`, etc.)

## Key File Locations

**Entry Points:**

- `backend/main.go`: Backend server startup, route registration
- `frontend/src/main.jsx`: React app mount
- `frontend/src/App.jsx`: Auth gate and root component
- `cli/main.go`: CLI binary entry point

**Configuration:**

- `backend/config/config.go`: All backend config with env var defaults
- `.env.example`: Template for required/optional env vars
- `frontend/vite.config.js`: Vite build and dev server config
- `Makefile`: All development commands with port overrides

**Core Logic:**

- `backend/handlers/routes.go`: Complete API route table (single source of truth)
- `backend/handlers/auth.go`: BFF OAuth2 login/refresh/logout flow
- `backend/services/cfapi.go`: CF API authentication and data retrieval
- `backend/services/boshapi.go`: BOSH Diego cell VM and vitals queries
- `backend/services/vsphere.go`: vCenter infrastructure discovery
- `backend/services/scenario.go`: Capacity scenario comparison math
- `backend/services/planning.go`: Infrastructure planning calculations
- `frontend/src/TASCapacityAnalyzer.jsx`: Main dashboard orchestration
- `frontend/src/contexts/AuthContext.jsx`: Frontend auth state and login/logout

**Testing:**

- `backend/services/*_test.go`: Unit tests co-located with service files
- `backend/handlers/*_test.go`: Handler unit tests
- `backend/middleware/*_test.go`: Middleware unit tests
- `backend/e2e/*_test.go`: Black-box integration tests
- `frontend/src/components/*.test.jsx`: React component tests

**API Contract:**

- `backend/handlers/openapi.yaml`: OpenAPI spec (served at `/api/v1/openapi.yaml`)

## Naming Conventions

**Backend Files:**

- Go files: `snake_case.go` (e.g., `boshapi.go`, `cfproxy.go`)
- Test files: `snake_case_test.go` co-located with source
- One primary type per file named after the file (e.g., `cfapi.go` defines `CFClient`)

**Frontend Files:**

- React components: `PascalCase.jsx` (e.g., `ScenarioAnalyzer.jsx`, `Login.jsx`)
- Component tests: `PascalCase.test.jsx` co-located (e.g., `ScenarioAnalyzer.test.jsx`)
- Services/utilities: `camelCase.js` (e.g., `apiClient.js`, `cfAuth.js`)
- Service tests: `camelCase.test.js` co-located (e.g., `apiClient.test.js`)

**Directories:**

- Backend: `lowercase/` (matches Go package names)
- Frontend: `camelCase/` for feature groupings (e.g., `components/`, `services/`)
- CLI commands: `lowercase.go` matching `cobra.Command.Use`

**Go Packages:**

- Package name matches directory name exactly (e.g., `package handlers`, `package middleware`)

**API Routes:**

- All routes use `/api/v1/` prefix
- Legacy `/api/` paths are auto-registered for backward compatibility in `backend/main.go`

## Where to Add New Code

**New API Endpoint:**

1. Add handler method to appropriate domain file in `backend/handlers/` (or create a new file for a new domain)
2. Add route entry to `backend/handlers/routes.go` with correct `Public`, `RateLimit`, `Role` fields
3. Add request/response types to `backend/models/` if needed
4. Add handler test to `backend/handlers/handlers_test.go` or domain-specific test file
5. Add e2e test to `backend/e2e/` for security-sensitive endpoints (auth, RBAC, CSRF)

**New External Service Integration:**

1. Create `backend/services/servicename.go` with client struct and `New*()` constructor
2. Add config fields to `backend/config/config.go` for credentials
3. Wire client in `backend/handlers/handlers.go` `NewHandler()` constructor
4. Add test data to `backend/services/testdata/` if needed
5. Add unit tests as `backend/services/servicename_test.go`

**New Frontend Component:**

1. Create `frontend/src/components/ComponentName.jsx`
2. Create `frontend/src/components/ComponentName.test.jsx` alongside it
3. Import into parent component

**New Frontend API Call:**

1. Add method to the appropriate service in `frontend/src/services/` using `apiFetch` from `apiClient.js`
2. Use `withCSRFToken()` from `frontend/src/utils/csrf.js` for state-changing requests

**New CLI Subcommand:**

1. Create `cli/cmd/commandname.go` defining a `cobra.Command`
2. Register with `rootCmd.AddCommand()` in `cli/cmd/root.go`
3. Add tests as `cli/cmd/commandname_test.go`

**New Middleware:**

1. Create `backend/middleware/name.go` returning `func(http.HandlerFunc) http.HandlerFunc`
2. Add to middleware chain in `backend/main.go` where appropriate
3. Add unit tests as `backend/middleware/name_test.go`

## Special Directories

**`backend/e2e/`:**

- Purpose: Black-box integration tests that spin up a real HTTP server
- Generated: No
- Committed: Yes
- Note: All files are `_test.go`; no production Go code lives here

**`backend/tmp/`:**

- Purpose: Air hot-reload tool build artifacts
- Generated: Yes (by `air` tool during `make backend-dev`)
- Committed: No (in `.gitignore`)

**`frontend/node_modules/`:**

- Purpose: npm/bun package dependencies
- Generated: Yes (by `make frontend-install`)
- Committed: No

**`frontend/public/samples/`:**

- Purpose: Sample infrastructure JSON files for testing manual input flow in the UI
- Generated: No
- Committed: Yes

**`.planning/codebase/`:**

- Purpose: GSD codebase analysis documents consumed by planning and execution commands
- Generated: Yes (by `/gsd:map-codebase`)
- Committed: Yes

**`.agent/`:**

- Purpose: Agent context markers, SOPs, and task tracking
- Generated: Mixed
- Committed: Yes

---

_Structure analysis: 2026-02-24_
