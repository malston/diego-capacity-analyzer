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

// rbacTestFixture holds shared RSA key and auth config for RBAC tests
type rbacTestFixture struct {
	privateKey *rsa.PrivateKey
	keyID      string
	authCfg    middleware.AuthConfig
}

// newRBACTestFixture generates an RSA key pair and configures a JWKSClient
// with AuthModeRequired for RBAC integration testing.
func newRBACTestFixture(t *testing.T) rbacTestFixture {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	keyID := "rbac-test-key"
	jwksClient := &services.JWKSClient{}
	jwksClient.SetKeysForTesting(map[string]*rsa.PublicKey{
		keyID: &privateKey.PublicKey,
	})

	return rbacTestFixture{
		privateKey: privateKey,
		keyID:      keyID,
		authCfg: middleware.AuthConfig{
			Mode:       middleware.AuthModeRequired,
			JWKSClient: jwksClient,
		},
	}
}

// bearerRequest creates a signed JWT and returns an httptest request with it set as a Bearer token.
func (f rbacTestFixture) bearerRequest(t *testing.T, method, path string, claims jwtClaims) *http.Request {
	t.Helper()
	token := createTestJWT(t, f.privateKey, f.keyID, claims)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestRBAC_OperatorEndpoint_WithOperatorToken_Returns200(t *testing.T) {
	f := newRBACTestFixture(t)

	var handlerCalled bool
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(f.authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	req := f.bearerRequest(t, http.MethodPost, "/api/v1/infrastructure/manual", jwtClaims{
		UserName: "operator-user",
		UserID:   "op-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.operator"},
	})
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
	f := newRBACTestFixture(t)

	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for viewer accessing operator endpoint")
		},
		middleware.Auth(f.authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	req := f.bearerRequest(t, http.MethodPost, "/api/v1/infrastructure/manual", jwtClaims{
		UserName: "viewer-user",
		UserID:   "view-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.viewer"},
	})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d. Body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRBAC_ViewerEndpoint_WithViewerToken_Returns200(t *testing.T) {
	f := newRBACTestFixture(t)

	var handlerCalled bool
	// No RequireRole middleware -- viewer endpoints have no Role set
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(f.authCfg),
	)

	req := f.bearerRequest(t, http.MethodGet, "/api/v1/dashboard", jwtClaims{
		UserName: "viewer-user",
		UserID:   "view-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.viewer"},
	})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for viewer accessing viewer endpoint")
	}
}

func TestRBAC_NonGatedPOST_WithViewerToken_Returns200(t *testing.T) {
	f := newRBACTestFixture(t)

	var handlerCalled bool
	// No RequireRole -- /scenario/compare has no Role set in the route table
	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
		middleware.Auth(f.authCfg),
	)

	req := f.bearerRequest(t, http.MethodPost, "/api/v1/scenario/compare", jwtClaims{
		UserName: "viewer-user",
		UserID:   "view-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "diego-analyzer.viewer"},
	})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for viewer accessing non-gated POST endpoint")
	}
}

func TestRBAC_OperatorEndpoint_AuthDisabled_NoRBACCheck(t *testing.T) {
	authCfg := middleware.AuthConfig{
		Mode: middleware.AuthModeDisabled,
	}

	var handlerCalled bool
	// Auth disabled means RequireRole is not in the chain (per main.go wiring)
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
	f := newRBACTestFixture(t)

	handler := middleware.Chain(
		func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for user without scopes on operator endpoint")
		},
		middleware.Auth(f.authCfg),
		middleware.RequireRole(middleware.RoleOperator),
	)

	// Token with no diego-analyzer scopes -- defaults to viewer
	req := f.bearerRequest(t, http.MethodPost, "/api/v1/infrastructure/manual", jwtClaims{
		UserName: "no-scope-user",
		UserID:   "ns-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Scope:    []string{"openid", "cloud_controller.read"},
	})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
