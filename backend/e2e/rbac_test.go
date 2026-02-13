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

func TestRBAC_NonGatedPOST_WithViewerToken_Returns200(t *testing.T) {
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
	// No RequireRole -- /scenario/compare has no Role set in the route table
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenario/compare", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
