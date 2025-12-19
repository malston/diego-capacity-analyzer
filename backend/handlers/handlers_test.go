package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
)

func TestHealthHandler(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["cf_api"] != "ok" {
		t.Errorf("Expected cf_api ok, got %v", resp["cf_api"])
	}
}

func TestHealthHandler_WithBOSH(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:        "https://api.test.com",
		CFUsername:      "admin",
		CFPassword:      "secret",
		BOSHEnvironment: "https://10.0.0.6:25555",
		BOSHClient:      "ops_manager",
		BOSHSecret:      "secret",
		BOSHDeployment:  "cf-test",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["bosh_api"] != "ok" {
		t.Errorf("Expected bosh_api ok, got %v", resp["bosh_api"])
	}
}

func TestDashboardHandler_NoBOSH(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/dashboard", nil)
	w := httptest.NewRecorder()

	h.Dashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	metadata := resp["metadata"].(map[string]interface{})
	if metadata["bosh_available"] != false {
		t.Errorf("Expected bosh_available false, got %v", metadata["bosh_available"])
	}

	if metadata["cached"] != false {
		t.Errorf("Expected cached false on first request, got %v", metadata["cached"])
	}
}

func TestDashboardHandler_Cache(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	// First request
	req1 := httptest.NewRequest("GET", "/api/dashboard", nil)
	w1 := httptest.NewRecorder()
	h.Dashboard(w1, req1)

	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	metadata1 := resp1["metadata"].(map[string]interface{})
	timestamp1 := metadata1["timestamp"].(string)

	// Second request (should be cached)
	time.Sleep(10 * time.Millisecond)
	req2 := httptest.NewRequest("GET", "/api/dashboard", nil)
	w2 := httptest.NewRecorder()
	h.Dashboard(w2, req2)

	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	metadata2 := resp2["metadata"].(map[string]interface{})
	timestamp2 := metadata2["timestamp"].(string)

	// Timestamps should be identical (cached response)
	if timestamp1 != timestamp2 {
		t.Errorf("Expected cached response with same timestamp, got different timestamps")
	}
}

func TestEnableCORS(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	corsHandler := h.EnableCORS(testHandler)
	corsHandler(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
		t.Errorf("Expected CORS methods, got %s", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestEnableCORS_OPTIONS(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	w := httptest.NewRecorder()

	corsHandler := h.EnableCORS(testHandler)
	corsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS header for OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
