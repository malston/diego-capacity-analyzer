// ABOUTME: Tests for CF API proxy handlers
// ABOUTME: Verifies session-based authentication and proxying to CF API

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

func TestCFProxyIsolationSegments(t *testing.T) {
	// Mock CF API server
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v3/isolation_segments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": []map[string]interface{}{
					{"guid": "seg-1", "name": "segment-1"},
					{"guid": "seg-2", "name": "segment-2"},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	// Create a session with CF token
	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns isolation segments with valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})

		rr := httptest.NewRecorder()
		h.CFProxyIsolationSegments(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		resources, ok := resp["resources"].([]interface{})
		if !ok || len(resources) != 2 {
			t.Errorf("Expected 2 resources, got %v", resp)
		}
	})

	t.Run("returns 401 without session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments", nil)
		rr := httptest.NewRecorder()
		h.CFProxyIsolationSegments(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("returns 401 with invalid session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "invalid-session-id"})

		rr := httptest.NewRecorder()
		h.CFProxyIsolationSegments(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})
}

func TestCFProxyApps(t *testing.T) {
	// Mock CF API server
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v3/apps") && !strings.Contains(r.URL.Path, "/processes") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": []map[string]interface{}{
					{"guid": "app-1", "name": "myapp"},
				},
				"pagination": map[string]interface{}{
					"total_results": 1,
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns apps with valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/apps", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})

		rr := httptest.NewRecorder()
		h.CFProxyApps(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		resources, ok := resp["resources"].([]interface{})
		if !ok || len(resources) != 1 {
			t.Errorf("Expected 1 resource, got %v", resp)
		}
	})
}

func TestCFProxyAppProcesses(t *testing.T) {
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v3/apps/app-123/processes" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": []map[string]interface{}{
					{"guid": "proc-1", "type": "web", "instances": 2},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns app processes with valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/apps/app-123/processes", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
		// Set path value for Go 1.22+ pattern matching
		req.SetPathValue("guid", "app-123")

		rr := httptest.NewRecorder()
		h.CFProxyAppProcesses(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestCFProxyProcessStats(t *testing.T) {
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v3/processes/proc-123/stats" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": []map[string]interface{}{
					{"index": 0, "state": "RUNNING"},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns process stats with valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/processes/proc-123/stats", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
		req.SetPathValue("guid", "proc-123")

		rr := httptest.NewRecorder()
		h.CFProxyProcessStats(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestCFProxySpaces(t *testing.T) {
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v3/spaces/space-123" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"guid": "space-123",
				"name": "dev-space",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns space with valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/spaces/space-123", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
		req.SetPathValue("guid", "space-123")

		rr := httptest.NewRecorder()
		h.CFProxySpaces(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestCFProxyIsolationSegmentByGUID(t *testing.T) {
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-cf-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v3/isolation_segments/seg-123" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"guid": "seg-123",
				"name": "production",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("returns isolation segment by GUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments/seg-123", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
		req.SetPathValue("guid", "seg-123")

		rr := httptest.NewRecorder()
		h.CFProxyIsolationSegmentByGUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestCFProxyHandlesCFAPIErrors(t *testing.T) {
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 500 error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	sessionID, _ := sessionSvc.Create("testuser", "user-123", "test-cf-token", "test-refresh", time.Now().Add(time.Hour))

	t.Run("propagates CF API errors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments", nil)
		req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})

		rr := httptest.NewRecorder()
		h.CFProxyIsolationSegments(rr, req)

		// Should propagate the error status
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})
}

func TestCFProxySessionTokenUsedCorrectly(t *testing.T) {
	var capturedAuth string

	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"resources": []}`)
	}))
	defer cfServer.Close()

	c := cache.New(5 * time.Minute)
	cfg := &config.Config{CFAPIUrl: cfServer.URL}
	h := NewHandler(cfg, c)

	sessionSvc := services.NewSessionService(c)
	h.SetSessionService(sessionSvc)

	// Create session with specific token
	sessionID, _ := sessionSvc.Create("testuser", "user-123", "unique-cf-token-12345", "refresh", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cf/isolation-segments", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})

	rr := httptest.NewRecorder()
	h.CFProxyIsolationSegments(rr, req)

	// Verify the correct token from session was used
	expectedAuth := "Bearer unique-cf-token-12345"
	if capturedAuth != expectedAuth {
		t.Errorf("Expected Authorization header %q, got %q", expectedAuth, capturedAuth)
	}
}
