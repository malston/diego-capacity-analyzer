// ABOUTME: Tests for role-based access control middleware
// ABOUTME: Verifies role enforcement for operator-only endpoints

package middleware

import (
	"context"
	"encoding/json"
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

	// Verify JSON error response format
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var errResp struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Response body is not valid JSON: %v; body: %s", err, rec.Body.String())
	}
	if errResp.Error == "" {
		t.Error("Expected non-empty error field in JSON response")
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

func TestRequireRole_UnknownRole_FailsClosed(t *testing.T) {
	handler := RequireRole(RoleViewer)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for unknown role")
	})

	claims := &UserClaims{Username: "mystery-user", Role: "superadmin"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d; unknown roles should resolve to level 0 (fail-closed)", rec.Code, http.StatusForbidden)
	}
}

func TestRequireRole_UnknownRequiredRole_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("RequireRole should panic for unknown required role")
		}
	}()

	RequireRole("typo-admin")
	t.Fatal("Should not reach here")
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
