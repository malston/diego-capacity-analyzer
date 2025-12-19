package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// setupMockCFServer creates a mock CF API server with UAA authentication
func setupMockCFServer() (*httptest.Server, *httptest.Server) {
	// Mock UAA server
	uaaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test-token","token_type":"bearer"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	// Mock CF API server
	var cfServerURL string
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v3/info":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"links":{"self":{"href":"` + cfServerURL + `"},"login":{"href":"` + uaaServer.URL + `"}}}`))

		case r.URL.Path == "/v3/apps":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"resources": [
					{
						"guid": "app-1",
						"name": "test-app",
						"state": "STARTED",
						"relationships": {
							"space": {
								"data": {"guid": "space-1"}
							}
						}
					}
				],
				"pagination": {"next": null}
			}`))

		case strings.HasPrefix(r.URL.Path, "/v3/apps/") && strings.HasSuffix(r.URL.Path, "/processes"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"resources": [
					{
						"type": "web",
						"instances": 2,
						"memory_in_mb": 512
					}
				]
			}`))

		case strings.HasPrefix(r.URL.Path, "/v3/spaces/") && strings.HasSuffix(r.URL.Path, "/relationships/isolation_segment"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": null}`))

		case r.URL.Path == "/v3/isolation_segments":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"resources": [
					{
						"guid": "iso-seg-1",
						"name": "production"
					}
				],
				"pagination": {"next": null}
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	cfServerURL = cfServer.URL

	return cfServer, uaaServer
}

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
	cfServer, uaaServer := setupMockCFServer()
	defer cfServer.Close()
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:   cfServer.URL,
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

	// Verify apps are present
	apps := resp["apps"].([]interface{})
	if len(apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(apps))
	}

	// Verify isolation segments are present
	segments := resp["segments"].([]interface{})
	if len(segments) != 1 {
		t.Errorf("Expected 1 isolation segment, got %d", len(segments))
	}
}

func TestDashboardHandler_Cache(t *testing.T) {
	cfServer, uaaServer := setupMockCFServer()
	defer cfServer.Close()
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:   cfServer.URL,
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

	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
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

func TestHandleManualInfrastructure(t *testing.T) {
	body := `{
		"name": "Test Env",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 8,
			"memory_gb_per_host": 2048,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 250,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4
		}],
		"platform_vms_gb": 4800,
		"total_app_memory_gb": 10500,
		"total_app_instances": 7500
	}`

	req := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)
	handler.HandleManualInfrastructure(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.InfrastructureState
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Source != "manual" {
		t.Errorf("Expected source 'manual', got '%s'", response.Source)
	}
	if response.TotalHostCount != 8 {
		t.Errorf("Expected TotalHostCount 8, got %d", response.TotalHostCount)
	}
}

func TestHandleScenarioCompare(t *testing.T) {
	// First, set up manual infrastructure
	manualBody := `{
		"name": "Test Env",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 15,
			"memory_gb_per_host": 2048,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 470,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4
		}],
		"platform_vms_gb": 4800,
		"total_app_memory_gb": 10500,
		"total_app_instances": 7500
	}`

	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// Set manual infrastructure
	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set manual infrastructure: %s", w1.Body.String())
	}

	// Now compare scenario
	compareBody := `{
		"proposed_cell_memory_gb": 64,
		"proposed_cell_cpu": 4,
		"proposed_cell_count": 235
	}`

	req2 := httptest.NewRequest("POST", "/api/scenario/compare", strings.NewReader(compareBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleScenarioCompare(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var comparison models.ScenarioComparison
	if err := json.NewDecoder(w2.Body).Decode(&comparison); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}
	if comparison.Proposed.CellCount != 235 {
		t.Errorf("Expected Proposed.CellCount 235, got %d", comparison.Proposed.CellCount)
	}
}
