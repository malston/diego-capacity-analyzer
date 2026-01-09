package handlers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// setupMockBOSHServer creates a mock BOSH API server that returns cells with no UsedMB
func setupMockBOSHServer(cellsWithNoUsedMB bool) *httptest.Server {
	taskDone := false

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/info":
			uaaURL := "https://" + r.Host
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "test-bosh",
				"user_authentication": map[string]interface{}{
					"type": "uaa",
					"options": map[string]interface{}{
						"url": uaaURL,
					},
				},
			})
		case "/oauth/token":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"token_type":   "bearer",
				"expires_in":   3600,
			})
		case "/deployments":
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"name": "cf-test"},
			})
		case "/deployments/cf-test/vms":
			if r.URL.Query().Get("format") == "full" {
				// Return a task object
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          123,
					"state":       "queued",
					"description": "retrieve vm-stats",
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case "/tasks/123":
			if !taskDone {
				taskDone = true
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":    123,
					"state": "processing",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":    123,
					"state": "done",
				})
			}
		case "/tasks/123/output":
			if r.URL.Query().Get("type") == "result" {
				// Return two diego cells as NDJSON - with mem.percent = "0" to trigger app calculation
				// This simulates when BOSH vitals don't have rep metrics populated yet
				// usedMB = (memoryMB * 0) / 100 = 0, which triggers needsAppCalculation
				w.Write([]byte(`{"job_name":"diego_cell","index":0,"id":"cell-01","vitals":{"mem":{"kb":"32000000","percent":"0"},"cpu":{"sys":"10","user":"5","wait":"1"},"disk":{"system":{"percent":"30"}}}}
{"job_name":"diego_cell","index":1,"id":"cell-02","vitals":{"mem":{"kb":"32000000","percent":"0"},"cpu":{"sys":"10","user":"5","wait":"1"},"disk":{"system":{"percent":"30"}}}}
`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server
}

// setupMockCFServerWithApps creates a mock CF API server that returns apps with ActualMB
func setupMockCFServerWithApps() (*httptest.Server, *httptest.Server) {
	uaaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test-token","token_type":"bearer"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	var cfServerURL string
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v3/info":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"links":{"self":{"href":"` + cfServerURL + `"},"login":{"href":"` + uaaServer.URL + `"}}}`))

		case r.URL.Path == "/v3/apps":
			w.WriteHeader(http.StatusOK)
			// Return 2 apps in the "shared" segment with 512MB each (2 instances = 1024MB actual)
			w.Write([]byte(`{
				"resources": [
					{
						"guid": "app-1",
						"name": "test-app-1",
						"state": "STARTED",
						"relationships": {
							"space": {"data": {"guid": "space-1"}}
						}
					},
					{
						"guid": "app-2",
						"name": "test-app-2",
						"state": "STARTED",
						"relationships": {
							"space": {"data": {"guid": "space-1"}}
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
					{"guid": "iso-seg-1", "name": "shared"}
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
		CFAPIUrl:     cfServer.URL,
		CFUsername:   "admin",
		CFPassword:   "secret",
		DashboardTTL: 30, // 30 seconds for test
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

func TestHandleManualInfrastructure_CPUMetrics(t *testing.T) {
	// Test that CPU metrics are computed and returned in API response
	body := `{
		"name": "CPU Metrics Test",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 4,
			"memory_gb_per_host": 1024,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 100,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4
		}],
		"platform_vms_gb": 0,
		"total_app_memory_gb": 0,
		"total_app_instances": 0
	}`

	req := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)
	handler.HandleManualInfrastructure(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.InfrastructureState
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify CPU metrics are computed
	// Total pCPU: 4 hosts × 64 cores = 256
	expectedCPUCores := 256
	if response.TotalCPUCores != expectedCPUCores {
		t.Errorf("Expected TotalCPUCores %d, got %d", expectedCPUCores, response.TotalCPUCores)
	}

	// Total vCPU: 100 cells × 4 vCPU = 400
	expectedVCPUs := 400
	if response.TotalVCPUs != expectedVCPUs {
		t.Errorf("Expected TotalVCPUs %d, got %d", expectedVCPUs, response.TotalVCPUs)
	}

	// vCPU:pCPU ratio: 400 / 256 = 1.5625
	expectedRatio := 1.5625
	if response.VCPURatio != expectedRatio {
		t.Errorf("Expected VCPURatio %.4f, got %.4f", expectedRatio, response.VCPURatio)
	}

	// Risk level: 1.5625 ≤ 4.0 = low
	if response.CPURiskLevel != "low" {
		t.Errorf("Expected CPURiskLevel 'low', got '%s'", response.CPURiskLevel)
	}

	// Verify cluster-level CPU metrics
	if len(response.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(response.Clusters))
	}
	cluster := response.Clusters[0]
	if cluster.TotalVCPUs != expectedVCPUs {
		t.Errorf("Expected cluster TotalVCPUs %d, got %d", expectedVCPUs, cluster.TotalVCPUs)
	}
	if cluster.VCPURatio != expectedRatio {
		t.Errorf("Expected cluster VCPURatio %.4f, got %.4f", expectedRatio, cluster.VCPURatio)
	}
}

func TestHandleManualInfrastructure_CPURiskLevels(t *testing.T) {
	tests := []struct {
		name            string
		cellCount       int
		cellCPU         int
		hostCount       int
		cpuCoresPerHost int
		expectedRisk    string
	}{
		{
			name:            "low risk - ratio under 1:1",
			cellCount:       50,
			cellCPU:         4,
			hostCount:       4,
			cpuCoresPerHost: 100,
			expectedRisk:    "low", // 200 vCPU / 400 pCPU = 0.5
		},
		{
			name:            "medium risk - ratio 6:1",
			cellCount:       150,
			cellCPU:         4,
			hostCount:       4,
			cpuCoresPerHost: 25,
			expectedRisk:    "medium", // 600 vCPU / 100 pCPU = 6.0
		},
		{
			name:            "high risk - ratio 10:1",
			cellCount:       100,
			cellCPU:         10,
			hostCount:       4,
			cpuCoresPerHost: 25,
			expectedRisk:    "high", // 1000 vCPU / 100 pCPU = 10.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := models.ManualInput{
				Name: "Risk Test",
				Clusters: []models.ClusterInput{
					{
						Name:              "cluster-01",
						HostCount:         tt.hostCount,
						MemoryGBPerHost:   1024,
						CPUCoresPerHost:   tt.cpuCoresPerHost,
						DiegoCellCount:    tt.cellCount,
						DiegoCellMemoryGB: 32,
						DiegoCellCPU:      tt.cellCPU,
					},
				},
			}

			body, _ := json.Marshal(input)
			req := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			cfg := &config.Config{}
			c := cache.New(5 * time.Minute)
			handler := NewHandler(cfg, c)
			handler.HandleManualInfrastructure(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response models.InfrastructureState
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.CPURiskLevel != tt.expectedRisk {
				t.Errorf("Expected CPURiskLevel '%s', got '%s' (ratio: %.2f)",
					tt.expectedRisk, response.CPURiskLevel, response.VCPURatio)
			}
		})
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

func TestHandleManualInfrastructure_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/infrastructure/manual", nil)
	w := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Error != "Method not allowed" {
		t.Errorf("Expected 'Method not allowed' error, got '%s'", resp.Error)
	}
}

func TestHandleManualInfrastructure_InvalidJSON(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Error != "Invalid JSON" {
		t.Errorf("Expected 'Invalid JSON' error, got '%s'", resp.Error)
	}
}

func TestHandleInfrastructure_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("POST", "/api/infrastructure", nil)
	w := httptest.NewRecorder()
	handler.HandleInfrastructure(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleInfrastructure_VSphereNotConfigured(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/infrastructure", nil)
	w := httptest.NewRecorder()
	handler.HandleInfrastructure(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.Contains(resp.Error, "vSphere not configured") {
		t.Errorf("Expected vSphere not configured error, got '%s'", resp.Error)
	}
}

func TestHandleInfrastructureStatus_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("POST", "/api/infrastructure/status", nil)
	w := httptest.NewRecorder()
	handler.HandleInfrastructureStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleInfrastructureStatus_NoData(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/infrastructure/status", nil)
	w := httptest.NewRecorder()
	handler.HandleInfrastructureStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["vsphere_configured"] != false {
		t.Errorf("Expected vsphere_configured false, got %v", resp["vsphere_configured"])
	}
	if resp["has_data"] != false {
		t.Errorf("Expected has_data false, got %v", resp["has_data"])
	}
}

func TestHandleInfrastructureStatus_WithData(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure
	manualBody := `{
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

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	// Now check status
	req2 := httptest.NewRequest("GET", "/api/infrastructure/status", nil)
	w2 := httptest.NewRecorder()
	handler.HandleInfrastructureStatus(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["has_data"] != true {
		t.Errorf("Expected has_data true, got %v", resp["has_data"])
	}
	if resp["source"] != "manual" {
		t.Errorf("Expected source 'manual', got %v", resp["source"])
	}
	if resp["name"] != "Test Env" {
		t.Errorf("Expected name 'Test Env', got %v", resp["name"])
	}
	if resp["host_count"].(float64) != 8 {
		t.Errorf("Expected host_count 8, got %v", resp["host_count"])
	}
	if resp["cell_count"].(float64) != 250 {
		t.Errorf("Expected cell_count 250, got %v", resp["cell_count"])
	}
}

func TestHandleScenarioCompare_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/scenario/compare", nil)
	w := httptest.NewRecorder()
	handler.HandleScenarioCompare(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleScenarioCompare_NoInfrastructureData(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	body := `{"proposed_cell_memory_gb": 64, "proposed_cell_cpu": 4, "proposed_cell_count": 235}`
	req := httptest.NewRequest("POST", "/api/scenario/compare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleScenarioCompare(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.Contains(resp.Error, "No infrastructure data") {
		t.Errorf("Expected 'No infrastructure data' error, got '%s'", resp.Error)
	}
}

func TestHandleScenarioCompare_InvalidJSON(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure
	manualBody := `{
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

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	// Now try with invalid JSON
	req2 := httptest.NewRequest("POST", "/api/scenario/compare", strings.NewReader("not valid json"))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleScenarioCompare(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w2.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Error != "Invalid JSON" {
		t.Errorf("Expected 'Invalid JSON' error, got '%s'", resp.Error)
	}
}

func TestDashboardHandler_AppMemoryCalculation(t *testing.T) {
	// Set up mock CF server with apps
	cfServer, uaaServer := setupMockCFServerWithApps()
	defer cfServer.Close()
	defer uaaServer.Close()

	// Set up mock BOSH server that returns cells with UsedMB = 0
	boshServer := setupMockBOSHServer(true) // true = cells have no UsedMB
	defer boshServer.Close()

	cfg := &config.Config{
		CFAPIUrl:        cfServer.URL,
		CFUsername:      "admin",
		CFPassword:      "secret",
		BOSHEnvironment: boshServer.URL,
		BOSHClient:      "ops_manager",
		BOSHSecret:      "secret",
		BOSHDeployment:  "cf-test",
		DashboardTTL:    30,
	}
	c := cache.New(5 * time.Minute)

	// Create handler and inject a BOSH client with custom TLS config
	h := &Handler{
		cfg:          cfg,
		cache:        c,
		scenarioCalc: services.NewScenarioCalculator(),
	}
	h.cfClient = services.NewCFClient(cfg.CFAPIUrl, cfg.CFUsername, cfg.CFPassword)

	// Create BOSH client with TLS skip verify for test server
	h.boshClient = services.NewBOSHClient(
		boshServer.URL,
		cfg.BOSHClient,
		cfg.BOSHSecret,
		"", // no CA cert
		cfg.BOSHDeployment,
	)
	// Override HTTP client to skip TLS verification for test
	h.boshClient.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})

	req := httptest.NewRequest("GET", "/api/dashboard", nil)
	w := httptest.NewRecorder()

	h.Dashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.DashboardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify cells are present
	if len(resp.Cells) == 0 {
		t.Fatal("Expected cells in response, got none")
	}

	// Verify apps are present
	if len(resp.Apps) == 0 {
		t.Fatal("Expected apps in response, got none")
	}

	// Verify the needsAppCalculation code path was exercised:
	// - BOSH returned cells with UsedMB=0 (mem.percent="0")
	// - CF returned apps with ActualMB (2 apps × 2 instances × 512MB = 2048MB)
	// - Handler calculated UsedMB = 2048MB / 2 cells = 1024MB per cell
	expectedUsedMB := 1024
	for _, cell := range resp.Cells {
		if cell.UsedMB != expectedUsedMB {
			t.Errorf("Expected UsedMB=%d (calculated from app memory), got %d for cell %s",
				expectedUsedMB, cell.UsedMB, cell.Name)
		}
		if cell.IsolationSegment != "default" {
			t.Errorf("Expected IsolationSegment='default', got '%s' for cell %s",
				cell.IsolationSegment, cell.Name)
		}
	}
}

func TestHandleBottleneckAnalysis(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure with high memory utilization
	manualBody := `{
		"name": "Bottleneck Test",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 4,
			"memory_gb_per_host": 1024,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 100,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4,
			"diego_cell_disk_gb": 100
		}],
		"total_app_memory_gb": 2800,
		"total_app_disk_gb": 4000
	}`

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set manual infrastructure: %s", w1.Body.String())
	}

	// Now get bottleneck analysis
	req2 := httptest.NewRequest("GET", "/api/bottleneck", nil)
	w2 := httptest.NewRecorder()
	handler.HandleBottleneckAnalysis(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var analysis models.BottleneckAnalysis
	if err := json.NewDecoder(w2.Body).Decode(&analysis); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(analysis.Resources) == 0 {
		t.Error("Expected resources in bottleneck analysis")
	}

	if analysis.ConstrainingResource == "" {
		t.Error("Expected a constraining resource")
	}

	if analysis.Summary == "" {
		t.Error("Expected a summary in bottleneck analysis")
	}
}

func TestHandleBottleneckAnalysis_NoData(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/bottleneck", nil)
	w := httptest.NewRecorder()
	handler.HandleBottleneckAnalysis(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRecommendations(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure
	manualBody := `{
		"name": "Recommendations Test",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 4,
			"memory_gb_per_host": 1024,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 100,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4,
			"diego_cell_disk_gb": 100
		}],
		"total_app_memory_gb": 2800,
		"total_app_disk_gb": 4000
	}`

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set manual infrastructure: %s", w1.Body.String())
	}

	// Now get recommendations
	req2 := httptest.NewRequest("GET", "/api/recommendations", nil)
	w2 := httptest.NewRecorder()
	handler.HandleRecommendations(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var response models.RecommendationsResponse
	if err := json.NewDecoder(w2.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Recommendations) == 0 {
		t.Error("Expected at least one recommendation")
	}

	if response.ConstrainingResource == "" {
		t.Error("Expected a constraining resource")
	}

	// Verify recommendations are sorted by priority
	for i := 0; i < len(response.Recommendations)-1; i++ {
		if response.Recommendations[i].Priority > response.Recommendations[i+1].Priority {
			t.Error("Recommendations should be sorted by priority")
		}
	}
}

func TestHandleRecommendations_NoData(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/recommendations", nil)
	w := httptest.NewRecorder()
	handler.HandleRecommendations(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleInfrastructureStatus_WithBottleneck(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure with high utilization
	manualBody := `{
		"name": "Status Test",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 4,
			"memory_gb_per_host": 1024,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 100,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4,
			"diego_cell_disk_gb": 100
		}],
		"total_app_memory_gb": 2800,
		"total_app_disk_gb": 4000
	}`

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	// Now check status
	req2 := httptest.NewRequest("GET", "/api/infrastructure/status", nil)
	w2 := httptest.NewRecorder()
	handler.HandleInfrastructureStatus(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify bottleneck info is present
	if _, ok := resp["constraining_resource"]; !ok {
		t.Error("Expected constraining_resource in status response")
	}
}

func TestEnrichWithCFAppData(t *testing.T) {
	// Set up mock CF server with apps
	cfServer, uaaServer := setupMockCFServerWithApps()
	defer cfServer.Close()
	defer uaaServer.Close()

	cfg := &config.Config{
		CFAPIUrl:   cfServer.URL,
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// Create an empty infrastructure state (simulating vSphere data with no app info)
	state := &models.InfrastructureState{
		Source:            "vsphere",
		Name:              "Test Datacenter",
		TotalAppMemoryGB:  0, // Not populated from vSphere
		TotalAppInstances: 0, // Not populated from vSphere
	}

	// Enrich with CF data
	ctx := context.Background()
	err := handler.enrichWithCFAppData(ctx, state)
	if err != nil {
		t.Fatalf("enrichWithCFAppData failed: %v", err)
	}

	// Mock CF server returns 2 apps, each with 2 instances × 512MB = 2048MB total
	// 2048MB / 1024 = 2GB
	expectedMemoryGB := 2
	if state.TotalAppMemoryGB != expectedMemoryGB {
		t.Errorf("Expected TotalAppMemoryGB=%d, got %d", expectedMemoryGB, state.TotalAppMemoryGB)
	}

	// Total instances: 2 apps × 2 instances = 4
	expectedInstances := 4
	if state.TotalAppInstances != expectedInstances {
		t.Errorf("Expected TotalAppInstances=%d, got %d", expectedInstances, state.TotalAppInstances)
	}
}

func TestEnrichWithCFAppData_NoCFClient(t *testing.T) {
	// Handler with no CF client configured
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	state := &models.InfrastructureState{
		Source:            "vsphere",
		TotalAppMemoryGB:  0,
		TotalAppInstances: 0,
	}

	// Should return nil (no error) when CF client is not configured
	ctx := context.Background()
	err := handler.enrichWithCFAppData(ctx, state)
	if err != nil {
		t.Errorf("Expected no error when CF client not configured, got: %v", err)
	}

	// Values should remain unchanged
	if state.TotalAppMemoryGB != 0 {
		t.Errorf("Expected TotalAppMemoryGB to remain 0, got %d", state.TotalAppMemoryGB)
	}
}

func TestHandleScenarioCompare_WithRecommendations(t *testing.T) {
	cfg := &config.Config{}
	c := cache.New(5 * time.Minute)
	handler := NewHandler(cfg, c)

	// First load manual infrastructure
	manualBody := `{
		"name": "Scenario Recommendations Test",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 4,
			"memory_gb_per_host": 1024,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 100,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4,
			"diego_cell_disk_gb": 100
		}],
		"total_app_memory_gb": 2800,
		"total_app_disk_gb": 4000
	}`

	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	// Now compare scenario
	compareBody := `{
		"proposed_cell_memory_gb": 64,
		"proposed_cell_cpu": 4,
		"proposed_cell_count": 50
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

	// Verify recommendations are included
	if len(comparison.Recommendations) == 0 {
		t.Error("Expected recommendations in scenario comparison")
	}
}
