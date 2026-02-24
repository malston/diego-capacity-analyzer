# Coding Conventions

**Analysis Date:** 2026-02-24

## File-Level Comments

Every Go file starts with two `// ABOUTME:` comment lines summarizing the file's purpose. These are the first lines in the file, before the package declaration.

```go
// ABOUTME: Authentication middleware for CF UAA tokens and session cookies
// ABOUTME: Verifies JWT signatures via JWKS client, extracts user claims

package middleware
```

Every JavaScript/JSX test file follows the same pattern:

```js
// ABOUTME: Unit tests for shared API client wrapper
// ABOUTME: Verifies network error classification, HTTP error handling, and JSON parsing
```

This convention applies to all source and test files. Do not use any other top-of-file documentation style.

## Naming Patterns

**Files (Go):**

- `snake_case.go` for all Go files: `auth.go`, `ratelimit.go`, `cfapi.go`, `boshapi.go`
- Test files: `{name}_test.go` co-located in the same package directory
- Helper test files: `helpers_test.go` inside the package under test

**Files (Frontend):**

- `PascalCase.jsx` for React components: `ScenarioAnalyzer.jsx`, `BottleneckCard.jsx`
- `camelCase.js` for services and utilities: `apiClient.js`, `cfAuth.js`, `metricsCalculations.js`
- Test files: `{Name}.test.jsx` or `{name}.test.js` co-located with the source file

**Go Types and Structs:**

- PascalCase exported types: `DiegoCell`, `AuthConfig`, `UserClaims`, `Route`
- Unexported types lowercase: `entry`, `contextKey`

**Go Functions and Methods:**

- PascalCase exported: `NewHandler`, `ValidateAuthMode`, `ResolveRole`, `GetUserClaims`
- camelCase unexported: `writeJSON`, `writeError`, `getEnv`, `getEnvInt`, `ensureScheme`
- Constructor functions: `New{Type}(...)` -- `NewHandler`, `NewRateLimiter`, `NewJWKSClient`

**Go Constants:**

- PascalCase exported: `AuthModeDisabled`, `AuthModeOptional`, `RoleViewer`, `RoleOperator`
- Constants grouped in `const (...)` blocks with inline comments

**Go Variables:**

- camelCase: `authCfg`, `corsMiddleware`, `rateLimiters`, `jwksClient`

**JSON Tags:**

- snake_case for all JSON field names: `memory_mb`, `allocated_mb`, `isolation_segment`
- Use `omitempty` only where fields are genuinely optional: `GUID string \`json:"guid,omitempty"\``

## Code Style

**Formatting:**

- Standard `gofmt` for Go -- no custom formatter configured, staticcheck for linting
- ESLint for frontend (`eslint src --max-warnings 0` enforces zero-warning policy)
- No Prettier config detected; formatting follows Vite/ESLint defaults

**Linting:**

- Backend: `staticcheck ./...` (run via `make backend-lint`)
- Frontend: `eslint` with `eslint-plugin-react` and `eslint-plugin-react-hooks`
- Lint is part of `make check` which runs before any CI-gate merge

**nolint directives:**

- Use inline `//nolint:gosec` with a comment explaining why: `//nolint:gosec // Operator-controlled setting`

## Import Organization

**Go imports follow standard three-group convention:**

```go
import (
    // Standard library
    "context"
    "encoding/json"
    "net/http"

    // Third-party
    "github.com/joho/godotenv"
    "github.com/vmware/govmomi"

    // Internal packages
    "github.com/markalston/diego-capacity-analyzer/backend/config"
    "github.com/markalston/diego-capacity-analyzer/backend/middleware"
)
```

**Frontend imports (JS/JSX):**

```js
// External libraries first
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

// Internal imports
import ScenarioAnalyzer from "./ScenarioAnalyzer";
import { ToastProvider } from "../contexts/ToastContext";
import { scenarioApi } from "../services/scenarioApi";
```

## Error Handling

**Go backend:**

- Functions return `(value, error)` -- callers check `if err != nil` immediately
- Use `fmt.Errorf("context: %w", err)` for wrapping errors with context
- Validation errors use `fmt.Errorf("FIELD_NAME must be ..., got %v", value)`
- Fatal startup errors: `slog.Error(...); os.Exit(1)` -- never `panic` in production paths
- HTTP handlers call `h.writeError(w, "message", http.StatusXxx)` for all error responses
- Middleware uses `writeJSONError(w, "message", code)` from `middleware/errors.go`
- Error response format is always `{"error": "message", "code": N}` -- see `models.ErrorResponse`

**Go error response pattern (handlers):**

```go
if state == nil {
    h.writeError(w, "No infrastructure data.", http.StatusBadRequest)
    return
}
```

**Go validation pattern (config):**

```go
if cfg.CFAPIUrl == "" {
    return nil, fmt.Errorf("CF_API_URL is required")
}
```

**Frontend:**

- Custom error classes `ApiConnectionError` and `ApiPermissionError` extend `Error`
- API calls wrapped in `apiFetch()` from `src/services/apiClient.js` for uniform error handling
- User-facing errors include a `detail` field with actionable guidance

## Logging

**Framework:** Go standard library `log/slog` (structured logging)

**Import:** `"log/slog"` -- initialized via `logger.Init()` at startup

**Patterns:**

```go
slog.Info("Server listening", "addr", addr)           // key-value pairs
slog.Debug("Cache hit", "key", key)                   // debug-level operational details
slog.Warn("BOSH not configured, running in degraded mode")
slog.Error("Failed to load configuration", "error", err)
```

- `slog.Info` for startup events and significant state changes
- `slog.Debug` for per-request tracing, cache operations, auth decisions
- `slog.Warn` for degraded-mode operation (missing optional config)
- `slog.Error` before `os.Exit(1)` or when encoding responses fails
- Always use key-value pairs for structured data, not `fmt.Sprintf` interpolation in log calls
- Log `"error", err` not `"error", err.Error()` -- slog handles formatting

## Comments

**ABOUTME pattern:** All files start with two `// ABOUTME:` lines -- mandatory.

**Inline comments:** Explain non-obvious decisions, not what the code does:

```go
// Go 1.22+ pattern: "METHOD /path"
pattern := route.Method + " " + route.Path

// Operator scope takes precedence. Defaults to viewer if no matching scope found.
func ResolveRole(scopes []string) string {
```

**Function doc comments:** Exported functions use standard GoDoc format:

```go
// ValidateAuthMode validates an auth mode string and returns the corresponding AuthMode.
// Empty string defaults to AuthModeOptional.
// Returns error for invalid mode values.
func ValidateAuthMode(mode string) (AuthMode, error) {
```

**No temporal comments:** Never reference "refactored", "new", "old", "legacy", "improved" in comments.

## Function Design

**Go handler methods:** All HTTP handlers have the standard signature:

```go
func (h *Handler) MethodName(w http.ResponseWriter, r *http.Request)
```

**Middleware factory pattern:** Middleware returns a function that wraps handlers:

```go
func Auth(cfg AuthConfig) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // ...
        }
    }
}
```

**Constructor pattern:** `New{Type}` functions return a pointer and optionally an error:

```go
func NewHandler(cfg *config.Config, cache *cache.Cache) *Handler
func NewJWKSClient(url string, client *http.Client) (*JWKSClient, error)
```

**Optional/nil pattern:** Clients are set to nil when not configured; callers check before use:

```go
if cfg.BOSHEnvironment != "" {
    boshClient, err := services.NewBOSHClient(...)
    // ...
}
```

## Module Design

**Go packages:**

- One package per directory, named after the directory
- Package-internal helpers are unexported (lowercase)
- Exported symbols have GoDoc comments

**Frontend services:**

- Singleton service objects exported from `src/services/`: `cfAuth`, `scenarioApi`, `apiClient`
- React contexts in `src/contexts/`: `AuthContext`, `ToastContext`
- Utilities as pure functions in `src/utils/`

**Route registration (Go):**

- Routes declared declaratively in `handlers/routes.go` as a slice of `Route` structs
- Route properties (`Public`, `RateLimit`, `Role`) drive middleware chain construction in `main.go`
- Do not register routes directly in `main.go` -- add to `handlers/routes.go`

## Configuration Pattern

All configuration comes from environment variables via `config/config.go`.

**Helper functions for env loading:**

```go
getEnv(key, defaultValue string) string
getEnvInt(key string, defaultValue int) int
getEnvBool(key string, defaultValue bool) bool
getEnvStringList(key string) []string
```

Use these helpers -- do not call `os.Getenv` directly in `Load()` unless the value must be exactly empty (not defaulted).

---

_Convention analysis: 2026-02-24_
