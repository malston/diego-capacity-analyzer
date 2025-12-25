// ABOUTME: End-to-end tests for CPU and host-level capacity analysis
// ABOUTME: Tests full flow from manual input through bottleneck analysis and recommendations

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// TestCPUAnalysisE2E tests the CPU analysis flow end-to-end
func TestCPUAnalysisE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create infrastructure with CPU configuration (high vCPU:pCPU ratio scenario)
	manualInput := models.ManualInput{
		Name: "CPU Analysis Test Environment",
		Clusters: []models.ClusterInput{
			{
				Name:              "cpu-test-cluster",
				HostCount:         4,
				MemoryGBPerHost:   512,
				CPUCoresPerHost:   32,
				DiegoCellCount:    40,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      8, // 40 cells × 8 vCPU = 320 vCPUs / 128 pCPUs = 2.5:1 ratio
			},
		},
		PlatformVMsGB:     200,
		TotalAppMemoryGB:  800,
		TotalAppInstances: 500,
	}

	body, _ := json.Marshal(manualInput)
	resp, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var infraState models.InfrastructureState
	json.NewDecoder(resp.Body).Decode(&infraState)

	// Verify CPU-related fields are populated
	if infraState.TotalCPUCores == 0 {
		t.Error("Expected TotalCPUCores to be populated")
	}
	if infraState.TotalCPUCores != 128 { // 4 hosts × 32 cores
		t.Errorf("Expected TotalCPUCores 128, got %d", infraState.TotalCPUCores)
	}

	if infraState.TotalVCPUs == 0 {
		t.Error("Expected TotalVCPUs to be populated")
	}
	if infraState.TotalVCPUs != 320 { // 40 cells × 8 vCPU
		t.Errorf("Expected TotalVCPUs 320, got %d", infraState.TotalVCPUs)
	}

	// Verify vCPU:pCPU ratio calculation
	if infraState.VCPURatio == 0 {
		t.Error("Expected VCPURatio to be calculated")
	}
	expectedRatio := 2.5 // 320 / 128
	if infraState.VCPURatio < expectedRatio-0.1 || infraState.VCPURatio > expectedRatio+0.1 {
		t.Errorf("Expected VCPURatio ~%.1f, got %.2f", expectedRatio, infraState.VCPURatio)
	}

	// Verify CPU risk level
	if infraState.CPURiskLevel == "" {
		t.Error("Expected CPURiskLevel to be set")
	}
	if infraState.CPURiskLevel != "low" { // 2.5:1 ratio is low risk
		t.Errorf("Expected CPURiskLevel 'low' for ratio 2.5:1, got '%s'", infraState.CPURiskLevel)
	}

	t.Logf("CPU Analysis: %d pCPU cores, %d vCPUs, ratio %.2f:1, risk level: %s",
		infraState.TotalCPUCores, infraState.TotalVCPUs,
		infraState.VCPURatio, infraState.CPURiskLevel)
}

// TestCPURiskLevelThresholdsE2E tests different vCPU:pCPU ratio risk levels
func TestCPURiskLevelThresholdsE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)

	server := httptest.NewServer(mux)
	defer server.Close()

	testCases := []struct {
		name          string
		hostCount     int
		cpuPerHost    int
		cellCount     int
		cpuPerCell    int
		expectedRatio float64
		expectedRisk  string
	}{
		{
			name:          "Low Risk (ratio <= 4:1)",
			hostCount:     4,
			cpuPerHost:    32,
			cellCount:     30, // 30 × 4 = 120 vCPU / 128 pCPU ≈ 0.94:1
			cpuPerCell:    4,
			expectedRatio: 0.94,
			expectedRisk:  "low",
		},
		{
			name:          "Medium Risk (4:1 < ratio <= 8:1)",
			hostCount:     4,
			cpuPerHost:    32,
			cellCount:     100, // 100 × 6 = 600 vCPU / 128 pCPU ≈ 4.69:1
			cpuPerCell:    6,
			expectedRatio: 4.69,
			expectedRisk:  "medium",
		},
		{
			name:          "High Risk (ratio > 8:1)",
			hostCount:     4,
			cpuPerHost:    32,
			cellCount:     150, // 150 × 8 = 1200 vCPU / 128 pCPU ≈ 9.38:1
			cpuPerCell:    8,
			expectedRatio: 9.38,
			expectedRisk:  "high",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := models.ManualInput{
				Name: tc.name,
				Clusters: []models.ClusterInput{
					{
						Name:              "test-cluster",
						HostCount:         tc.hostCount,
						MemoryGBPerHost:   512,
						CPUCoresPerHost:   tc.cpuPerHost,
						DiegoCellCount:    tc.cellCount,
						DiegoCellMemoryGB: 32,
						DiegoCellCPU:      tc.cpuPerCell,
					},
				},
				PlatformVMsGB:     100,
				TotalAppMemoryGB:  500,
				TotalAppInstances: 300,
			}

			body, _ := json.Marshal(input)
			resp, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Failed to post: %v", err)
			}
			defer resp.Body.Close()

			var state models.InfrastructureState
			json.NewDecoder(resp.Body).Decode(&state)

			if state.CPURiskLevel != tc.expectedRisk {
				t.Errorf("Expected CPURiskLevel '%s' for ratio %.2f:1, got '%s' (actual ratio: %.2f)",
					tc.expectedRisk, tc.expectedRatio, state.CPURiskLevel, state.VCPURatio)
			}

			t.Logf("Ratio: %.2f:1 → Risk: %s", state.VCPURatio, state.CPURiskLevel)
		})
	}
}

// TestHostLevelAnalysisE2E tests host-level metrics calculation
func TestHostLevelAnalysisE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)

	server := httptest.NewServer(mux)
	defer server.Close()

	manualInput := models.ManualInput{
		Name: "Host Analysis Test Environment",
		Clusters: []models.ClusterInput{
			{
				Name:                         "host-test-cluster",
				HostCount:                    8,
				MemoryGBPerHost:              1024,
				CPUCoresPerHost:              64,
				HAAdmissionControlPercentage: 25,
				DiegoCellCount:               60,
				DiegoCellMemoryGB:            64,
				DiegoCellCPU:                 8,
			},
		},
		PlatformVMsGB:     500,
		TotalAppMemoryGB:  2500,
		TotalAppInstances: 2000,
	}

	body, _ := json.Marshal(manualInput)
	resp, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp.Body.Close()

	var infraState models.InfrastructureState
	json.NewDecoder(resp.Body).Decode(&infraState)

	// Verify host count
	if infraState.TotalHostCount != 8 {
		t.Errorf("Expected TotalHostCount 8, got %d", infraState.TotalHostCount)
	}

	// Verify cluster-level host metrics
	if len(infraState.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(infraState.Clusters))
	}
	cluster := infraState.Clusters[0]

	// VMs per host: 60 cells / 8 hosts = 7.5
	expectedVMsPerHost := 7.5
	if cluster.VMsPerHost != expectedVMsPerHost {
		t.Errorf("Expected VMsPerHost %.1f, got %.1f", expectedVMsPerHost, cluster.VMsPerHost)
	}

	// Host memory utilization: 60 cells × 64 GB = 3840 GB / 8192 GB = 46.9%
	expectedMemUtil := 46.875 // 3840 / 8192 × 100
	if cluster.HostMemoryUtilizationPercent < expectedMemUtil-1 || cluster.HostMemoryUtilizationPercent > expectedMemUtil+1 {
		t.Errorf("Expected HostMemoryUtilizationPercent ~%.1f%%, got %.1f%%",
			expectedMemUtil, cluster.HostMemoryUtilizationPercent)
	}

	// Host CPU utilization: 60 cells × 8 vCPU = 480 / 512 pCPU = 93.75%
	expectedCPUUtil := 93.75 // 480 / 512 × 100
	if cluster.HostCPUUtilizationPercent < expectedCPUUtil-1 || cluster.HostCPUUtilizationPercent > expectedCPUUtil+1 {
		t.Errorf("Expected HostCPUUtilizationPercent ~%.1f%%, got %.1f%%",
			expectedCPUUtil, cluster.HostCPUUtilizationPercent)
	}

	// HA usable memory: 8192 GB × 0.75 = 6144 GB
	expectedHAUsable := 6144
	if cluster.HAUsableMemoryGB != expectedHAUsable {
		t.Errorf("Expected HAUsableMemoryGB %d, got %d", expectedHAUsable, cluster.HAUsableMemoryGB)
	}

	// HA host failures survived (should be > 0 since we have capacity)
	if cluster.HAHostFailuresSurvived < 1 {
		t.Errorf("Expected HAHostFailuresSurvived >= 1, got %d", cluster.HAHostFailuresSurvived)
	}

	// HA status
	if cluster.HAStatus != "ok" {
		t.Errorf("Expected HAStatus 'ok', got '%s'", cluster.HAStatus)
	}

	t.Logf("Host Analysis: %d hosts, %.1f VMs/host, Memory: %.1f%%, CPU: %.1f%%, HA: %d failures survived",
		infraState.TotalHostCount, cluster.VMsPerHost,
		cluster.HostMemoryUtilizationPercent, cluster.HostCPUUtilizationPercent,
		cluster.HAHostFailuresSurvived)
}

// TestBottleneckAnalysisE2E tests multi-resource bottleneck identification
func TestBottleneckAnalysisE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a memory-constrained scenario
	manualInput := models.ManualInput{
		Name: "Bottleneck Analysis Test",
		Clusters: []models.ClusterInput{
			{
				Name:              "bottleneck-cluster",
				HostCount:         4,
				MemoryGBPerHost:   256,
				CPUCoresPerHost:   64,
				DiegoCellCount:    20,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
				DiegoCellDiskGB:   200,
			},
		},
		PlatformVMsGB:     50,
		TotalAppMemoryGB:  550, // 550 / 640 = 85.9% memory util
		TotalAppDiskGB:    2000, // 2000 / 4000 = 50% disk util
		TotalAppInstances: 300,
	}

	body, _ := json.Marshal(manualInput)
	resp, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to post: %v", err)
	}
	defer resp.Body.Close()

	var infraState models.InfrastructureState
	json.NewDecoder(resp.Body).Decode(&infraState)

	// Perform bottleneck analysis
	analysis := models.AnalyzeBottleneck(infraState)

	// Verify resources are ranked
	if len(analysis.Resources) < 2 {
		t.Fatalf("Expected at least 2 resources in analysis, got %d", len(analysis.Resources))
	}

	// Verify constraining resource is identified
	if analysis.ConstrainingResource == "" {
		t.Error("Expected ConstrainingResource to be identified")
	}

	// The first resource should be marked as constraining
	if !analysis.Resources[0].IsConstraining {
		t.Error("Expected first ranked resource to be marked as constraining")
	}

	// Verify resources are sorted by utilization descending
	for i := 1; i < len(analysis.Resources); i++ {
		if analysis.Resources[i].UsedPercent > analysis.Resources[i-1].UsedPercent {
			t.Errorf("Resources not sorted by utilization: %s (%.1f%%) > %s (%.1f%%)",
				analysis.Resources[i].Name, analysis.Resources[i].UsedPercent,
				analysis.Resources[i-1].Name, analysis.Resources[i-1].UsedPercent)
		}
	}

	// Verify summary is generated
	if analysis.Summary == "" {
		t.Error("Expected Summary to be generated")
	}

	t.Logf("Bottleneck Analysis:")
	for i, res := range analysis.Resources {
		constraining := ""
		if res.IsConstraining {
			constraining = " ← Constraining"
		}
		t.Logf("  %d. %s: %.1f%% utilized (%d/%d %s)%s",
			i+1, res.Name, res.UsedPercent, res.UsedCapacity, res.TotalCapacity, res.Unit, constraining)
	}
	t.Logf("Summary: %s", analysis.Summary)
}

// TestRecommendationsE2E tests upgrade path recommendations
func TestRecommendationsE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a constrained infrastructure
	manualInput := models.ManualInput{
		Name: "Recommendations Test",
		Clusters: []models.ClusterInput{
			{
				Name:              "rec-cluster",
				HostCount:         4,
				MemoryGBPerHost:   512,
				CPUCoresPerHost:   32,
				DiegoCellCount:    50,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
		PlatformVMsGB:     200,
		TotalAppMemoryGB:  1200, // High memory utilization
		TotalAppInstances: 800,
	}

	body, _ := json.Marshal(manualInput)
	resp, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to post: %v", err)
	}
	defer resp.Body.Close()

	var infraState models.InfrastructureState
	json.NewDecoder(resp.Body).Decode(&infraState)

	// Generate recommendations
	recommendations := models.GenerateRecommendations(infraState)

	// Verify recommendations are generated
	if len(recommendations) == 0 {
		t.Fatal("Expected recommendations to be generated")
	}

	// Verify recommendation types
	hasAddCells := false
	hasResizeCells := false
	hasAddHosts := false

	for _, rec := range recommendations {
		switch rec.Type {
		case models.RecommendationAddCells:
			hasAddCells = true
			if rec.CellsToAdd < 1 {
				t.Error("Expected AddCells recommendation to specify cells to add")
			}
		case models.RecommendationResizeCells:
			hasResizeCells = true
			if rec.NewCellMemoryGB == 0 && rec.NewCellCPU == 0 {
				t.Error("Expected ResizeCells recommendation to specify new cell size")
			}
		case models.RecommendationAddHosts:
			hasAddHosts = true
			if rec.HostsToAdd < 1 {
				t.Error("Expected AddHosts recommendation to specify hosts to add")
			}
		}

		// Verify recommendation fields
		if rec.Title == "" {
			t.Error("Expected recommendation Title to be set")
		}
		if rec.Description == "" {
			t.Error("Expected recommendation Description to be set")
		}
		if rec.Impact == "" {
			t.Error("Expected recommendation Impact to be set")
		}
	}

	if !hasAddCells {
		t.Error("Expected at least one 'add_cells' recommendation")
	}
	if !hasResizeCells {
		t.Error("Expected at least one 'resize_cells' recommendation")
	}
	if !hasAddHosts {
		t.Error("Expected at least one 'add_hosts' recommendation")
	}

	// Verify recommendations are sorted by priority
	for i := 1; i < len(recommendations); i++ {
		if recommendations[i].Priority < recommendations[i-1].Priority {
			t.Errorf("Recommendations not sorted by priority: %d < %d",
				recommendations[i].Priority, recommendations[i-1].Priority)
		}
	}

	t.Logf("Recommendations generated:")
	for _, rec := range recommendations {
		t.Logf("  [Priority %d] %s: %s", rec.Priority, rec.Title, rec.Description)
		t.Logf("    Impact: %s", rec.Impact)
	}
}

// TestScenarioCompareWithCPUE2E tests scenario comparison including CPU metrics
func TestScenarioCompareWithCPUE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Step 1: Set up infrastructure
	manualInput := models.ManualInput{
		Name: "Scenario Compare CPU Test",
		Clusters: []models.ClusterInput{
			{
				Name:              "compare-cluster",
				HostCount:         4,
				MemoryGBPerHost:   512,
				CPUCoresPerHost:   32,
				DiegoCellCount:    40,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
		PlatformVMsGB:     200,
		TotalAppMemoryGB:  900,
		TotalAppInstances: 600,
	}

	body1, _ := json.Marshal(manualInput)
	resp1, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body1))
	if err != nil {
		t.Fatalf("Failed to post infrastructure: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp1.StatusCode)
	}

	// Step 2: Compare scenario with increased CPU
	scenarioInput := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8, // Doubled vCPU
		ProposedCellCount:    20,
	}

	body2, _ := json.Marshal(scenarioInput)
	resp2, err := http.Post(server.URL+"/api/scenario/compare", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("Failed to post scenario compare: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp2.StatusCode)
	}

	var comparison models.ScenarioComparison
	json.NewDecoder(resp2.Body).Decode(&comparison)

	// Verify current state reflects CPU config
	if comparison.Current.CellCPU == 0 {
		t.Error("Expected Current.CellCPU to be set")
	}

	// Verify proposed state reflects new CPU config
	if comparison.Proposed.CellCPU != 8 {
		t.Errorf("Expected Proposed.CellCPU 8, got %d", comparison.Proposed.CellCPU)
	}

	// Verify recommendations are included
	if len(comparison.Recommendations) == 0 {
		t.Log("Note: No recommendations in comparison (may be expected if infrastructure is well-balanced)")
	}

	t.Logf("Scenario Comparison:")
	t.Logf("  Current: %s (%d cells)", comparison.Current.CellSize(), comparison.Current.CellCount)
	t.Logf("  Proposed: %s (%d cells)", comparison.Proposed.CellSize(), comparison.Proposed.CellCount)
	t.Logf("  Recommendations: %d", len(comparison.Recommendations))
}
