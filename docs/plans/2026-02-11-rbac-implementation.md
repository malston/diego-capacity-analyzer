# RBAC Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add role-based authorization (viewer/operator) derived from CF UAA scopes to API endpoints.

**Architecture:** JWT `scope` claim flows through existing auth pipeline into `UserClaims`. A `RequireRole` middleware gates state-mutating endpoints. Only 2 of 17 routes require operator role; everything else is viewer-accessible.

**Tech Stack:** Go 1.23+, standard library HTTP middleware, CF UAA JWT scopes

**Design doc:** `docs/plans/2026-02-11-rbac-design.md`

---

### Task 1: Add Scope to JWT Claims

**Files:**

- Modify: `backend/services/jwks.go:96-99` (JWTClaims struct)
- Modify: `backend/services/jwks.go:109-117` (jwtClaimsForVerification struct)
- Modify: `backend/services/jwks.go:228-231` (return statement in verifyJWT)

**Step 1: Add Scope field to jwtClaimsForVerification**

In `backend/services/jwks.go`, add `Scope` to the internal claims struct:

```go
type jwtClaimsForVerification struct {
	Sub      string   `json:"sub"`
	UserName string   `json:"user_name"`
	UserID   string   `json:"user_id"`
	ClientID string   `json:"client_id"`
	Exp      int64    `json:"exp"`
	Nbf      int64    `json:"nbf"`
	Iat      int64    `json:"iat"`
	Scope    []string `json:"scope"`
}
```

**Step 2: Add Scopes field to JWTClaims**

```go
type JWTClaims struct {
	Username string
	UserID   string
	Scopes   []string
}
```

**Step 3: Populate Scopes in verifyJWT return**

Change the return at line ~228:

```go
return &JWTClaims{
	Username: username,
	UserID:   userID,
	Scopes:   claims.Scope,
}, nil
```

**Step 4: Run existing tests to verify no regressions**

Run: `cd backend && go test ./services/ -run TestJWKS -v`
Expected: All existing JWKS tests PASS (Scopes is nil for tokens without scope claim, which is fine)

**Step 5: Commit**

```
feat(auth): add scope extraction to JWT claims parsing
```

---

### Task 2: Add Role Resolution and Scopes to UserClaims

**Files:**

- Modify: `backend/middleware/auth.go:55-58` (UserClaims struct)
- Modify: `backend/middleware/auth.go:112-116` (Bearer token claims conversion)
- Test: `backend/middleware/auth_test.go` (add role resolution tests)

**Step 1: Write failing tests for ResolveRole**

Add to `backend/middleware/auth_test.go`:

```go
func TestResolveRole(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   string
	}{
		{"operator scope", []string{"openid", "diego-analyzer.operator"}, "operator"},
		{"viewer scope", []string{"openid", "diego-analyzer.viewer"}, "viewer"},
		{"both scopes operator wins", []string{"diego-analyzer.viewer", "diego-analyzer.operator"}, "operator"},
		{"no matching scopes defaults to viewer", []string{"openid", "cloud_controller.read"}, "viewer"},
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

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./middleware/ -run TestResolveRole -v`
Expected: FAIL -- `ResolveRole` not defined

**Step 3: Implement ResolveRole and update UserClaims**

In `backend/middleware/auth.go`, update UserClaims:

```go
type UserClaims struct {
	Username string
	UserID   string
	Scopes   []string
	Role     string
}
```

Add constants and ResolveRole function:

```go
const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"

	ScopeViewer   = "diego-analyzer.viewer"
	ScopeOperator = "diego-analyzer.operator"
)

// ResolveRole determines the application role from JWT scopes.
// Operator scope takes precedence. Defaults to viewer if no matching scope found.
func ResolveRole(scopes []string) string {
	for _, s := range scopes {
		if s == ScopeOperator {
			return RoleOperator
		}
	}
	for _, s := range scopes {
		if s == ScopeViewer {
			return RoleViewer
		}
	}
	return RoleViewer
}
```

**Step 4: Update Bearer token claims conversion**

In the Auth middleware, update the Bearer token path (~line 113):

```go
claims := &UserClaims{
	Username: jwtClaims.Username,
	UserID:   jwtClaims.UserID,
	Scopes:   jwtClaims.Scopes,
	Role:     ResolveRole(jwtClaims.Scopes),
}
```

**Step 5: Run tests**

Run: `cd backend && go test ./middleware/ -run "TestResolveRole|TestAuth" -v`
Expected: All PASS

**Step 6: Commit**

```
feat(auth): add role resolution from JWT scopes
```

---

### Task 3: Store Scopes in Session

**Files:**

- Modify: `backend/models/auth.go:31-39` (Session struct)
- Modify: `backend/services/session.go:28` (Create function signature)
- Modify: `backend/handlers/auth.go:50-57` (Login handler session creation)
- Modify: `backend/main.go:111-119` (session validator function)

**Step 1: Add Scopes to Session struct**

In `backend/models/auth.go`:

```go
type Session struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	UserID       string    `json:"user_id"`
	Scopes       []string  `json:"scopes"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	CSRFToken    string    `json:"-"`
	TokenExpiry  time.Time `json:"token_expiry"`
	CreatedAt    time.Time `json:"created_at"`
}
```

**Step 2: Add scopes parameter to SessionService.Create**

In `backend/services/session.go`, update the Create signature:

```go
func (s *SessionService) Create(username, userID, accessToken, refreshToken string, scopes []string, tokenExpiry time.Time) (string, error) {
```

And set it in the session struct:

```go
session := &models.Session{
	ID:           sessionID,
	Username:     username,
	UserID:       userID,
	Scopes:       scopes,
	AccessToken:  accessToken,
	RefreshToken: refreshToken,
	CSRFToken:    csrfToken,
	TokenExpiry:  tokenExpiry,
	CreatedAt:    time.Now(),
}
```

**Step 3: Extract scopes from access token in Login handler**

In `backend/handlers/auth.go`, after `authenticateWithCFUAA` returns, extract scopes from the access token before creating the session. Add a helper to parse scopes from the JWT payload without full verification (the token was just received from UAA over TLS):

```go
// extractScopesFromToken parses the scope claim from a JWT payload.
// The token is not verified here because it was just received from UAA.
func extractScopesFromToken(accessToken string) []string {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims struct {
		Scope []string `json:"scope"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims.Scope
}
```

Update the Login handler to extract scopes and pass them to Create:

```go
scopes := extractScopesFromToken(tokenResp.AccessToken)

sessionID, err := h.sessionService.Create(
	req.Username,
	tokenResp.UserID,
	tokenResp.AccessToken,
	tokenResp.RefreshToken,
	scopes,
	expiry,
)
```

**Step 4: Update session validator in main.go to populate Role**

In `backend/main.go`, update the session validator closure:

```go
sessionValidator := func(sessionID string) *middleware.UserClaims {
	session, err := sessionService.Get(sessionID)
	if err != nil {
		return nil
	}
	return &middleware.UserClaims{
		Username: session.Username,
		UserID:   session.UserID,
		Scopes:   session.Scopes,
		Role:     middleware.ResolveRole(session.Scopes),
	}
}
```

**Step 5: Fix any compilation errors and run tests**

Run: `cd backend && go build ./... && go test ./... -count=1`
Expected: All PASS (may need to update test callers of `sessionService.Create` if any)

**Step 6: Commit**

```
feat(auth): store scopes in session for role resolution
```

---

### Task 4: RequireRole Middleware

**Files:**

- Create: `backend/middleware/rbac.go`
- Create: `backend/middleware/rbac_test.go`

**Step 1: Write failing tests**

Create `backend/middleware/rbac_test.go`:

```go
// ABOUTME: Tests for role-based access control middleware
// ABOUTME: Verifies role enforcement for operator-only endpoints

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireRole_OperatorClaims_OperatorRequired_Passes(t *testing.T) {
	handler := RequireRole(RoleOperator)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	claims := &UserClaims{Username: "op-user", Role: RoleOperator}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_ViewerClaims_OperatorRequired_Returns403(t *testing.T) {
	handler := RequireRole(RoleOperator)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for viewer accessing operator endpoint")
	})

	claims := &UserClaims{Username: "view-user", Role: RoleViewer}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireRole_ViewerClaims_ViewerRequired_Passes(t *testing.T) {
	handler := RequireRole(RoleViewer)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	claims := &UserClaims{Username: "view-user", Role: RoleViewer}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_NoClaims_ViewerRequired_Passes(t *testing.T) {
	handler := RequireRole(RoleViewer)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_NoClaims_OperatorRequired_Returns403(t *testing.T) {
	handler := RequireRole(RoleOperator)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for anonymous accessing operator endpoint")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireRole_OperatorClaims_ViewerRequired_Passes(t *testing.T) {
	handler := RequireRole(RoleViewer)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	claims := &UserClaims{Username: "op-user", Role: RoleOperator}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./middleware/ -run TestRequireRole -v`
Expected: FAIL -- `RequireRole` not defined

**Step 3: Implement RequireRole middleware**

Create `backend/middleware/rbac.go`:

```go
// ABOUTME: Role-based access control middleware for API endpoints
// ABOUTME: Gates endpoints by required role derived from JWT scopes

package middleware

import (
	"log/slog"
	"net/http"
)

// roleHierarchy defines the privilege level for each role.
// Higher value means more privilege.
var roleHierarchy = map[string]int{
	RoleViewer:   1,
	RoleOperator: 2,
}

// RequireRole returns middleware that enforces a minimum role.
// Anonymous requests (no UserClaims in context) are treated as viewer.
// Returns 403 Forbidden if the caller's role is insufficient.
func RequireRole(requiredRole string) func(http.HandlerFunc) http.HandlerFunc {
	requiredLevel := roleHierarchy[requiredRole]

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Determine caller's role
			callerRole := RoleViewer // default for anonymous
			claims := GetUserClaims(r)
			if claims != nil && claims.Role != "" {
				callerRole = claims.Role
			}

			callerLevel := roleHierarchy[callerRole]
			if callerLevel < requiredLevel {
				slog.Debug("RBAC rejected: insufficient role",
					"path", r.URL.Path,
					"required", requiredRole,
					"actual", callerRole,
				)
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./middleware/ -run TestRequireRole -v`
Expected: All PASS

**Step 5: Commit**

```
feat(auth): add RequireRole middleware for RBAC enforcement
```

---

### Task 5: Wire RBAC into Route Table and Middleware Chain

**Files:**

- Modify: `backend/handlers/routes.go:9-15` (Route struct)
- Modify: `backend/handlers/routes.go:33-34` (infrastructure/manual and infrastructure/state routes)
- Modify: `backend/main.go:176-189` (middleware chain builder)

**Step 1: Add Role field to Route struct**

In `backend/handlers/routes.go`:

```go
type Route struct {
	Method    string
	Path      string
	Handler   http.HandlerFunc
	Public    bool
	RateLimit string
	Role      string // Required role: "operator", "viewer", or "" (no RBAC check)
}
```

**Step 2: Set Role on operator-only routes**

Update the two state-mutating routes:

```go
{Method: http.MethodPost, Path: "/api/v1/infrastructure/manual", Handler: h.SetManualInfrastructure, RateLimit: "write", Role: "operator"},
{Method: http.MethodPost, Path: "/api/v1/infrastructure/state", Handler: h.SetInfrastructureState, RateLimit: "write", Role: "operator"},
```

**Step 3: Wire RequireRole into middleware chain in main.go**

In `backend/main.go`, after the auth middleware and before rate limiting, add the RBAC check. Update the middleware chain builder (around line 176-189):

```go
mws := []func(http.HandlerFunc) http.HandlerFunc{corsMiddleware, middleware.CSRF()}
if !route.Public {
	mws = append(mws, middleware.Auth(authCfg))
}
if route.Role != "" && authCfg.Mode != middleware.AuthModeDisabled {
	mws = append(mws, middleware.RequireRole(route.Role))
}
if route.RateLimit != "none" {
	rlMiddleware, ok := rateLimiters[route.RateLimit]
	if !ok {
		slog.Error("Unknown rate limit tier", "tier", route.RateLimit, "path", route.Path)
		os.Exit(1)
	}
	mws = append(mws, rlMiddleware)
}
mws = append(mws, middleware.LogRequest)
```

**Step 4: Build and run all tests**

Run: `cd backend && go build ./... && go test ./... -count=1`
Expected: All PASS

**Step 5: Commit**

```
feat(auth): wire RBAC into route table and middleware chain
```

---

### Task 6: E2E Authorization Tests

**Files:**

- Create: `backend/e2e/rbac_test.go`

**Step 1: Write E2E tests**

Create `backend/e2e/rbac_test.go`:

```go
// ABOUTME: Integration tests for role-based access control on API endpoints
// ABOUTME: Verifies operator-only endpoints reject viewer tokens with 403

package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

func TestRBAC_OperatorEndpoint_WithOperatorToken_Returns200(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	keyID := "rbac-test-key"
	jwksClient := &services.JWKSClient{}
	jwksClient.SetKeysForTesting(map[string]*rsa.PublicKey{
		keyID: &privateKey.PublicKey,
	})

	authCfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}

	var handlerCalled bool
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	token := createTestJWT(t, privateKey, keyID, jwtClaims{
		UserName: "operator-user",
		UserID:   "op-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.operator"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !handlerCalled {
		t.Error("Handler should be called for operator accessing operator endpoint")
	}
}

func TestRBAC_OperatorEndpoint_WithViewerToken_Returns403(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	keyID := "rbac-test-key"
	jwksClient := &services.JWKSClient{}
	jwksClient.SetKeysForTesting(map[string]*rsa.PublicKey{
		keyID: &privateKey.PublicKey,
	})

	authCfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}

	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for viewer accessing operator endpoint")
		},
		middleware.Auth(authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	token := createTestJWT(t, privateKey, keyID, jwtClaims{
		UserName: "viewer-user",
		UserID:   "view-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.viewer"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d. Body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRBAC_ViewerEndpoint_WithViewerToken_Returns200(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	keyID := "rbac-test-key"
	jwksClient := &services.JWKSClient{}
	jwksClient.SetKeysForTesting(map[string]*rsa.PublicKey{
		keyID: &privateKey.PublicKey,
	})

	authCfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}

	var handlerCalled bool
	// No RequireRole middleware -- viewer endpoints have no Role set
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(authCfg),
	)

	token := createTestJWT(t, privateKey, keyID, jwtClaims{
		UserName: "viewer-user",
		UserID:   "view-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.viewer"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for viewer accessing viewer endpoint")
	}
}

func TestRBAC_OperatorEndpoint_AuthDisabled_NoRBACCheck(t *testing.T) {
	authCfg := middleware.AuthConfig{
		Mode: middleware.AuthModeDisabled,
	}

	var handlerCalled bool
	// Auth disabled means RequireRole is not in the chain (per main.go wiring)
	// Simulate by only including Auth middleware (which passes through in disabled mode)
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(authCfg),
		// No RequireRole -- main.go skips it when auth is disabled
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called when auth is disabled")
	}
}

func TestRBAC_OperatorEndpoint_NoScopesDefaultsViewer_Returns403(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	keyID := "rbac-test-key"
	jwksClient := &services.JWKSClient{}
	jwksClient.SetKeysForTesting(map[string]*rsa.PublicKey{
		keyID: &privateKey.PublicKey,
	})

	authCfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}

	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for user without scopes on operator endpoint")
		},
		middleware.Auth(authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	// Token with no diego-analyzer scopes -- defaults to viewer
	token := createTestJWT(t, privateKey, keyID, jwtClaims{
		UserName: "no-scope-user",
		UserID:   "ns-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "cloud_controller.read"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/infrastructure/manual", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
```

**Step 2: Add Scope field to E2E jwtClaims test helper**

In `backend/e2e/auth_test.go`, add `Scope` to the `jwtClaims` struct:

```go
type jwtClaims struct {
	Sub      string   `json:"sub,omitempty"`
	UserName string   `json:"user_name,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Exp      int64    `json:"exp,omitempty"`
	Iat      int64    `json:"iat,omitempty"`
	Scope    []string `json:"scope,omitempty"`
}
```

**Step 3: Run E2E tests**

Run: `cd backend && go test ./e2e/ -run TestRBAC -v`
Expected: All PASS

**Step 4: Run full test suite**

Run: `cd backend && go test ./... -count=1`
Expected: All PASS

**Step 5: Commit**

```
test(auth): add E2E tests for RBAC endpoint authorization
```

---

### Task 7: Update Documentation

**Files:**

- Modify: `docs/AUTHENTICATION.md`

**Step 1: Add RBAC section to AUTHENTICATION.md**

Add a section covering:

- Role model (viewer/operator)
- UAA group setup commands (`uaac group add`, `uaac member add`)
- UAA client scope configuration
- Authorization matrix
- Default behavior when groups are not configured

**Step 2: Commit**

```
docs: add RBAC setup to authentication guide
```

---

### Task 8: Final Verification

**Step 1: Run full test suite**

Run: `cd backend && go test ./... -count=1 -v`
Expected: All PASS

**Step 2: Run linter**

Run: `cd backend && make lint` (or `golangci-lint run` if available)
Expected: Clean

**Step 3: Verify build**

Run: `cd backend && go build ./...`
Expected: Success

**Step 4: Manual smoke test (optional)**

Start the server with `AUTH_MODE=disabled` and verify all endpoints work.
Start with `AUTH_MODE=required` and verify operator endpoints return 401 without token.

**Step 5: Commit any fixes, then squash-review**

Review all commits on the branch for completeness and quality.
