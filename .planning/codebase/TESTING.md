# Testing Patterns

**Analysis Date:** 2026-02-24

## Test Frameworks

**Backend (Go):**

- Runner: Go standard library `testing` package (Go 1.24+)
- No third-party assertion libraries -- uses plain `t.Errorf`, `t.Fatalf`, `t.Fatal`
- Linting: `staticcheck`
- Config: No separate test config file -- uses `go test ./...`

**Frontend (JavaScript/JSX):**

- Runner: Vitest 4.x
- Config: embedded in `frontend/vite.config.js` under `test:` key
- Assertions: Vitest's built-in `expect` plus `@testing-library/jest-dom`
- Component testing: `@testing-library/react` + `@testing-library/user-event`
- Setup file: `frontend/src/test/setup.js` (imports `@testing-library/jest-dom`)

## Run Commands

```bash
make test                       # Run all tests (backend + frontend + CLI)
make backend-test               # Go tests: cd backend && go test ./...
make backend-test-verbose       # Go tests: go test -v ./...
make frontend-test              # Frontend: cd frontend && npm test (vitest run)
make frontend-test-watch        # Frontend: vitest in watch mode
make frontend-test-coverage     # Frontend: vitest run --coverage
```

## Test File Organization

**Backend -- co-located in same package:**

```
backend/
├── middleware/
│   ├── auth.go
│   ├── auth_test.go          # Same package: package middleware
│   ├── csrf.go
│   ├── csrf_test.go
│   ├── rbac.go
│   ├── rbac_test.go
│   ├── ratelimit.go
│   ├── ratelimit_test.go
│   └── helpers_test.go       # Shared test helpers within the package
├── config/
│   ├── config.go
│   ├── config_test.go
│   └── helpers_test.go
├── models/
│   ├── models.go
│   ├── models_test.go
│   ├── auth_test.go
│   ├── bottleneck_test.go
│   └── scenario_test.go
├── cache/
│   └── cache_test.go
├── handlers/
│   ├── handlers.go
│   ├── handlers_test.go
│   └── auth_test.go
└── e2e/                      # Separate e2e package: package e2e
    ├── helpers_test.go
    ├── auth_test.go
    ├── cors_test.go
    ├── csrf_test.go
    ├── rbac_test.go
    ├── ratelimit_test.go
    ├── scenario_test.go
    └── tls_test.go
```

**Frontend -- co-located with source:**

```
frontend/src/
├── components/
│   ├── ScenarioAnalyzer.jsx
│   ├── ScenarioAnalyzer.test.jsx
│   ├── BottleneckCard.jsx
│   ├── BottleneckCard.test.jsx
│   └── wizard/
│       ├── ScenarioWizard.jsx
│       ├── ScenarioWizard.test.jsx
│       └── steps/
│           ├── CellConfigStep.jsx
│           └── CellConfigStep.test.jsx
├── services/
│   ├── cfAuth.js
│   ├── cfAuth.test.js
│   ├── apiClient.js
│   └── apiClient.test.js
├── utils/
│   ├── metricsCalculations.js
│   └── metricsCalculations.test.js
└── test/
    └── setup.js              # Global test setup only
```

**Naming:**

- Go: `TestFunctionName_Context_ExpectedBehavior` (e.g., `TestAuth_RequiredMode_NoHeader_Returns401`)
- JavaScript: `describe("ComponentName", () => { it("behavior description", ...) })`

## Go Test Structure

**Standard unit test pattern:**

```go
func TestAuth_RequiredMode_NoHeader_Returns401(t *testing.T) {
    // Arrange
    cfg := AuthConfig{Mode: AuthModeRequired}
    handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
        t.Error("Handler should not be called without auth header in required mode")
    })

    // Act
    req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
    rec := httptest.NewRecorder()
    handler(rec, req)

    // Assert
    if rec.Code != http.StatusUnauthorized {
        t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
    }
}
```

**Table-driven test pattern (preferred for multiple cases):**

```go
func TestResolveRole(t *testing.T) {
    tests := []struct {
        name   string
        scopes []string
        want   string
    }{
        {"operator scope", []string{"openid", "diego-analyzer.operator"}, "operator"},
        {"viewer scope", []string{"openid", "diego-analyzer.viewer"}, "viewer"},
        {"empty scopes defaults to viewer", []string{}, "viewer"},
        {"nil scopes defaults to viewer", nil, "viewer"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ResolveRole(tt.scopes)
            if got != tt.want {
                t.Errorf("ResolveRole(%v) = %q, want %q", tt.scopes, got, tt.want)
            }
        })
    }
}
```

**Assertion style:** Always `t.Errorf("Got = %v, want %v", got, want)` format with "Got" first. Never use `if got == want` when `if got != want` is clearer. Use `t.Fatalf` when the test cannot continue (nil dereference risk).

**Handler tests use `httptest`:**

```go
req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
req.Header.Set("Authorization", "Bearer "+token)
rec := httptest.NewRecorder()
handler(rec, req)

if rec.Code != http.StatusOK {
    t.Errorf("Status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
}
```

Include `rec.Body.String()` in error messages for HTTP status failures -- makes CI failures debuggable.

**t.Helper():** Test helper functions always call `t.Helper()` as their first line.

**t.Cleanup():** Used for environment cleanup instead of `defer` to ensure ordering:

```go
func TestLoadConfig_Defaults(t *testing.T) {
    t.Cleanup(withCleanCFEnv(t))
    // test body
}
```

## JavaScript Test Structure

**Standard vitest + Testing Library pattern:**

```js
describe("apiFetch", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("throws ApiConnectionError on TypeError", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    await expect(apiFetch("/api/v1/health")).rejects.toThrow(
      ApiConnectionError,
    );
  });
});
```

**Component rendering pattern:**

```js
// Helper to render with providers
const renderWithProviders = (ui) => {
  return render(<ToastProvider>{ui}</ToastProvider>);
};

it("shows wizard after infrastructure is loaded", async () => {
  renderWithProviders(<ScenarioAnalyzer />);
  await waitFor(() => {
    expect(screen.getByText("Cell Config")).toBeInTheDocument();
  });
});
```

**Grouping:** Nested `describe` blocks group related scenarios (e.g., `describe("network errors", () => {...})`).

## Mocking

**Backend Go -- no mock framework:**

- Test helpers inject real in-memory implementations, not mocks
- `JWKSClient` exposes `SetKeysForTesting(map[string]*rsa.PublicKey)` for tests
- Session validator is a plain function (`SessionValidatorFunc`) -- tests pass an inline func:
  ```go
  sessionValidator := func(sessionID string) *UserClaims {
      if sessionID == "valid-session-123" {
          return &UserClaims{Username: "session-user", UserID: "session-user-id"}
      }
      return nil
  }
  ```
- `httptest.NewServer` creates real HTTP servers for e2e tests -- mock UAA, real HTTP

**Backend -- what NOT to mock:** Never mock `http.HandlerFunc` behavior. Test the real middleware by wrapping an inline `http.HandlerFunc` that captures state or calls `t.Error` if unexpectedly invoked.

**Frontend -- `vi.mock()` for modules:**

```js
vi.mock("../services/scenarioApi", () => ({
  scenarioApi: {
    setManualInfrastructure: vi.fn(),
    compareScenario: vi.fn(),
    getInfrastructureStatus: vi
      .fn()
      .mockResolvedValue({ vsphere_configured: false }),
  },
}));
```

**Frontend -- global object mocking:**

```js
// Mock fetch directly on global
global.fetch = vi.fn().mockResolvedValue({
  ok: true,
  json: () => Promise.resolve(payload),
});

// Restore after each test
afterEach(() => {
  global.fetch = originalFetch;
  vi.restoreAllMocks();
});
```

**Frontend -- localStorage mock:**

```js
const mockLocalStorage = (() => {
  let store = {};
  return {
    getItem: vi.fn((key) => store[key] || null),
    setItem: vi.fn((key, value) => {
      store[key] = value;
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();
Object.defineProperty(window, "localStorage", { value: mockLocalStorage });
```

## Test Helper Patterns

**Go -- environment setup helpers (reused across packages):**

Both `backend/config/helpers_test.go` and `backend/e2e/helpers_test.go` define similar `withClean*Env` helpers. Pattern:

```go
// withCleanCFEnv clears the environment, sets required CF env vars to test
// values, and returns a cleanup function that restores the original env.
func withCleanCFEnv(t *testing.T) func() {
    t.Helper()
    return withCleanCFEnvAndExtra(t, nil)
}

func withCleanCFEnvAndExtra(t *testing.T, extra map[string]string) func() {
    t.Helper()
    originalEnv := os.Environ()
    os.Clearenv()
    os.Setenv("CF_API_URL", "https://api.sys.test.com")
    // set minimal required env...
    return func() {
        os.Clearenv()
        // restore originalEnv...
    }
}
```

**Go -- JWT test token helpers:** Both unit and e2e test packages define `createSignedTestToken` helpers. These generate real RSA-signed JWTs for testing cryptographic verification:

```go
func createSignedTestToken(t *testing.T, privateKey *rsa.PrivateKey, kid, username, userID string, exp time.Time) string {
    t.Helper()
    // Creates real RS256-signed JWT
}
```

## E2E Test Pattern

The `backend/e2e/` package creates a real `httptest.Server` with the actual handler (nil config) and makes real HTTP calls:

```go
func TestScenarioAnalysisE2E(t *testing.T) {
    handler := handlers.NewHandler(nil, nil)  // nil config is valid for unit testing
    mux := http.NewServeMux()
    mux.HandleFunc("/api/infrastructure/manual", handler.SetManualInfrastructure)

    server := httptest.NewServer(mux)
    defer server.Close()

    resp, err := http.Post(server.URL+"/api/...", "application/json", body)
    // assert on resp.StatusCode and decoded response body
}
```

The e2e package also creates real mock servers (e.g., `createMockUAAServer`) using `httptest.NewServer` to simulate UAA JWKS endpoints -- this is real HTTP, not mocked interfaces.

## Coverage

**Requirements:** No enforced coverage threshold detected.

**View coverage:**

```bash
make frontend-test-coverage     # Frontend with coverage
cd backend && go test -coverprofile=coverage.txt ./... && go tool cover -html=coverage.txt
```

Coverage output file at `backend/coverage.txt` (present in repo, not gitignored).

## Security Test Patterns

Security-related middleware (`auth`, `csrf`, `rbac`, `ratelimit`) follows a specific pattern: test all boundary conditions explicitly, including edge cases. This is enforced by project convention.

**Auth middleware coverage includes:**

- Disabled / optional / required mode for each authentication path
- Valid token, expired token, malformed token, wrong key ID, invalid signature
- Empty username with valid user ID (client credentials tokens)
- Bearer token vs session cookie precedence
- Session cookie present but invalid
- No session validator configured (nil)
- `JWKSClient` nil (Bearer unavailable, session still works)

**RBAC coverage includes:**

- operator/viewer claims for operator-required endpoint
- operator/viewer claims for viewer-required endpoint
- No claims (anonymous) for each required role
- Unknown role fails closed (403)
- Unknown required role panics (programming error)

**CSRF coverage includes:**

- Safe methods (GET, HEAD, OPTIONS) bypass CSRF
- Bearer auth bypasses CSRF (stateless)
- Login path exemption
- Missing header, missing cookie, token mismatch, short token
- All mutating methods: POST, PUT, DELETE

---

_Testing analysis: 2026-02-24_
