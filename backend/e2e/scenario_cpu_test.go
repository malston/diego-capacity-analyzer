// ABOUTME: E2E tests for CPU ratio analysis in scenario comparison
// ABOUTME: Verifies API accepts CPU config and returns ratio metrics

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestScenarioCompare_WithCPUConfig(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Step 1: Set up infrastructure state with CPU config
	manualInput := models.ManualInput{
		Name: "CPU Test Infrastructure",
		Clusters: []models.ClusterInput{
			{
				Name:              "test-cluster",
				HostCount:         3,
				MemoryGBPerHost:   512,
				CPUCoresPerHost:   32,
				DiegoCellCount:    10,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
				DiegoCellDiskGB:   128,
			},
		},
	}

	body1, _ := json.Marshal(manualInput)
	resp1, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body1))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp1.StatusCode)
	}

	// Step 2: Test scenario compare with CPU config
	scenarioInput := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellDiskGB:   128,
		ProposedCellCount:    20,
		HostCount:            3,
		MemoryPerHostGB:      512,
		HAAdmissionPct:       25,
		PhysicalCoresPerHost: 32,
		TargetVCPURatio:      4,
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
	if err := json.NewDecoder(resp2.Body).Decode(&comparison); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify proposed has CPU metrics
	proposed := comparison.Proposed

	// Check that CPU fields are populated
	if proposed.TotalVCPUs == 0 {
		t.Error("Expected TotalVCPUs to be populated")
	}
	if proposed.TotalPCPUs == 0 {
		t.Error("Expected TotalPCPUs to be populated")
	}
	if proposed.VCPURatio == 0 {
		t.Error("Expected VCPURatio to be populated")
	}
	if proposed.CPURiskLevel == "" {
		t.Error("Expected CPURiskLevel to be populated")
	}

	// Verify calculated values:
	// 20 cells * 4 vCPU = 80 total vCPUs
	// 3 hosts * 32 pCPU = 96 total pCPUs
	// Ratio = 80/96 = 0.833...
	expectedVCPUs := 80
	expectedPCPUs := 96
	expectedRatio := float64(expectedVCPUs) / float64(expectedPCPUs) // ~0.833

	if proposed.TotalVCPUs != expectedVCPUs {
		t.Errorf("TotalVCPUs = %d, want %d", proposed.TotalVCPUs, expectedVCPUs)
	}
	if proposed.TotalPCPUs != expectedPCPUs {
		t.Errorf("TotalPCPUs = %d, want %d", proposed.TotalPCPUs, expectedPCPUs)
	}

	// Allow small floating point tolerance
	if proposed.VCPURatio < expectedRatio-0.01 || proposed.VCPURatio > expectedRatio+0.01 {
		t.Errorf("VCPURatio = %f, want ~%f", proposed.VCPURatio, expectedRatio)
	}

	// At 0.83:1 ratio, risk level should be "conservative" (<=4:1)
	if proposed.CPURiskLevel != "conservative" {
		t.Errorf("CPURiskLevel = %q, want %q for ratio %.2f", proposed.CPURiskLevel, "conservative", proposed.VCPURatio)
	}

	t.Logf("CPU Analysis: %d vCPUs / %d pCPUs = %.2f:1 ratio (%s)",
		proposed.TotalVCPUs, proposed.TotalPCPUs, proposed.VCPURatio, proposed.CPURiskLevel)
}

func TestScenarioCompare_CPURiskLevels(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up infrastructure
	manualInput := models.ManualInput{
		Name: "Risk Level Test",
		Clusters: []models.ClusterInput{
			{
				Name:              "test-cluster",
				HostCount:         2,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   32,
				DiegoCellCount:    10,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
			},
		},
	}

	body1, _ := json.Marshal(manualInput)
	resp1, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body1))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp1.Body.Close()

	tests := []struct {
		name             string
		cellCount        int
		cellCPU          int
		physicalCores    int
		expectedRiskLevel string
	}{
		{
			name:             "Conservative ratio (<= 4:1)",
			cellCount:        8,
			cellCPU:          8,
			physicalCores:    32,
			// 8 cells * 8 vCPU = 64 vCPU / (2 hosts * 32 pCPU) = 64/64 = 1:1
			expectedRiskLevel: "conservative",
		},
		{
			name:             "Moderate ratio (4-8:1)",
			cellCount:        40,
			cellCPU:          8,
			physicalCores:    32,
			// 40 cells * 8 vCPU = 320 vCPU / (2 hosts * 32 pCPU) = 320/64 = 5:1
			expectedRiskLevel: "moderate",
		},
		{
			name:             "Aggressive ratio (> 8:1)",
			cellCount:        80,
			cellCPU:          8,
			physicalCores:    32,
			// 80 cells * 8 vCPU = 640 vCPU / (2 hosts * 32 pCPU) = 640/64 = 10:1
			expectedRiskLevel: "aggressive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioInput := models.ScenarioInput{
				ProposedCellMemoryGB: 64,
				ProposedCellCPU:      tt.cellCPU,
				ProposedCellCount:    tt.cellCount,
				HostCount:            2,
				MemoryPerHostGB:      1024,
				PhysicalCoresPerHost: tt.physicalCores,
			}

			body, _ := json.Marshal(scenarioInput)
			resp, err := http.Post(server.URL+"/api/scenario/compare", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Failed to post scenario compare: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected 200, got %d", resp.StatusCode)
			}

			var comparison models.ScenarioComparison
			if err := json.NewDecoder(resp.Body).Decode(&comparison); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if comparison.Proposed.CPURiskLevel != tt.expectedRiskLevel {
				t.Errorf("CPURiskLevel = %q, want %q (ratio: %.2f)",
					comparison.Proposed.CPURiskLevel, tt.expectedRiskLevel, comparison.Proposed.VCPURatio)
			}

			t.Logf("  %d cells * %d vCPU = %d vCPU / %d pCPU = %.2f:1 -> %s",
				tt.cellCount, tt.cellCPU, comparison.Proposed.TotalVCPUs,
				comparison.Proposed.TotalPCPUs, comparison.Proposed.VCPURatio,
				comparison.Proposed.CPURiskLevel)
		})
	}
}

func TestScenarioCompare_NoCPUConfigDisablesAnalysis(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up infrastructure
	manualInput := models.ManualInput{
		Name: "No CPU Config Test",
		Clusters: []models.ClusterInput{
			{
				Name:              "test-cluster",
				HostCount:         3,
				MemoryGBPerHost:   512,
				DiegoCellCount:    10,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	body1, _ := json.Marshal(manualInput)
	resp1, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body1))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp1.Body.Close()

	// Scenario without PhysicalCoresPerHost - CPU analysis should be disabled
	scenarioInput := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    20,
		HostCount:            3,
		MemoryPerHostGB:      512,
		// PhysicalCoresPerHost is 0/omitted - CPU analysis disabled
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
	if err := json.NewDecoder(resp2.Body).Decode(&comparison); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// When CPU analysis is disabled, these fields should be zero/empty
	if comparison.Proposed.TotalVCPUs != 0 {
		t.Errorf("Expected TotalVCPUs = 0 when CPU analysis disabled, got %d", comparison.Proposed.TotalVCPUs)
	}
	if comparison.Proposed.TotalPCPUs != 0 {
		t.Errorf("Expected TotalPCPUs = 0 when CPU analysis disabled, got %d", comparison.Proposed.TotalPCPUs)
	}
	if comparison.Proposed.VCPURatio != 0 {
		t.Errorf("Expected VCPURatio = 0 when CPU analysis disabled, got %f", comparison.Proposed.VCPURatio)
	}
	if comparison.Proposed.CPURiskLevel != "" {
		t.Errorf("Expected CPURiskLevel = \"\" when CPU analysis disabled, got %q", comparison.Proposed.CPURiskLevel)
	}

	t.Log("CPU analysis correctly disabled when PhysicalCoresPerHost not provided")
}
