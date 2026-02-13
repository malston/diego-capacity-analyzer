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
