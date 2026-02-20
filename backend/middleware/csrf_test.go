// ABOUTME: Tests for CSRF middleware
// ABOUTME: Validates double-submit cookie pattern implementation

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 44-character tokens matching base64url-encoded 32 bytes
const (
	testCSRFToken  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop=="
	testCSRFToken2 = "ZYXWVUTSRQPONMLKJIHGFEDCBAzyxwvutsrqponmlk=="
)

func TestCSRF_SkipsGETRequests(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for GET, got %d", rr.Code)
	}
}

func TestCSRF_SkipsHEADRequests(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("HEAD", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for HEAD, got %d", rr.Code)
	}
}

func TestCSRF_SkipsOPTIONSRequests(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for OPTIONS, got %d", rr.Code)
	}
}

func TestCSRF_SkipsBearerAuth(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for Bearer auth, got %d", rr.Code)
	}
}

func TestCSRF_DoesNotSkipNonBearerAuth(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Non-Bearer Authorization headers should not bypass CSRF
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for Basic auth with session cookie, got %d", rr.Code)
	}
}

func TestCSRF_SkipsNoSessionCookie(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 when no session cookie, got %d", rr.Code)
	}
}

func TestCSRF_RejectsMissingHeader(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "csrf-token"})
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for missing header, got %d", rr.Code)
	}
}

func TestCSRF_RejectsMissingCookie(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.Header.Set("X-CSRF-Token", "csrf-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for missing cookie, got %d", rr.Code)
	}
}

func TestCSRF_RejectsTokenMismatch(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: testCSRFToken})
	req.Header.Set("X-CSRF-Token", testCSRFToken2)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for mismatch, got %d", rr.Code)
	}
}

func TestCSRF_AcceptsValidToken(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: testCSRFToken})
	req.Header.Set("X-CSRF-Token", testCSRFToken)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid token, got %d", rr.Code)
	}
}

func TestCSRF_WorksWithPUT(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("PUT", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: testCSRFToken})
	req.Header.Set("X-CSRF-Token", testCSRFToken)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for PUT with valid token, got %d", rr.Code)
	}
}

func TestCSRF_RejectsInvalidTokenLength(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: "short"})
	req.Header.Set("X-CSRF-Token", "short")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for short token, got %d", rr.Code)
	}
}

func TestCSRF_SkipsLoginPath(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// POST to login with a stale session cookie but no CSRF token
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "stale-session-id"})
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for login path, got %d", rr.Code)
	}
}

func TestCSRF_SkipsLoginLegacyPath(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Legacy path should also be exempt
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "stale-session-id"})
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for legacy login path, got %d", rr.Code)
	}
}

func TestCSRF_DoesNotSkipNonLoginPaths(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Other POST paths with session cookie should still require CSRF
	paths := []string{
		"/api/v1/infrastructure/manual",
		"/api/v1/auth/logout",
		"/api/v1/scenario/compare",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
			rr := httptest.NewRecorder()
			handler(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Errorf("Expected 403 for %s without CSRF token, got %d", path, rr.Code)
			}
		})
	}
}

func TestCSRF_WorksWithDELETE(t *testing.T) {
	handler := CSRF()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "session-id"})
	req.AddCookie(&http.Cookie{Name: "DIEGO_CSRF", Value: testCSRFToken})
	req.Header.Set("X-CSRF-Token", testCSRFToken)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for DELETE with valid token, got %d", rr.Code)
	}
}
