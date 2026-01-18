// ABOUTME: End-to-end test for scenario analysis API
// ABOUTME: Tests full flow from manual input to scenario comparison

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

func TestScenarioAnalysisE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.SetManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.CompareScenario)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Step 1: Set manual infrastructure (based on capacity doc)
	manualInput := models.ManualInput{
		Name: "Customer ACME Production",
		Clusters: []models.ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         8,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    250,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         7,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    220,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
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

	var infraState models.InfrastructureState
	json.NewDecoder(resp1.Body).Decode(&infraState)

	if infraState.TotalCellCount != 470 {
		t.Errorf("Expected TotalCellCount 470, got %d", infraState.TotalCellCount)
	}

	// Step 2: Compare scenario (4×32 current → 4×64 proposed)
	scenarioInput := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    235,
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

	// Validate current state
	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}
	if comparison.Current.CellSize() != "4×32" {
		t.Errorf("Expected Current.CellSize '4×32', got '%s'", comparison.Current.CellSize())
	}

	// Validate proposed state
	if comparison.Proposed.CellCount != 235 {
		t.Errorf("Expected Proposed.CellCount 235, got %d", comparison.Proposed.CellCount)
	}
	if comparison.Proposed.CellSize() != "4×64" {
		t.Errorf("Expected Proposed.CellSize '4×64', got '%s'", comparison.Proposed.CellSize())
	}

	// Validate delta - with 235 cells, blast radius is ~0.43%, so ResilienceChange = "low"
	if comparison.Delta.ResilienceChange != "low" {
		t.Errorf("Expected ResilienceChange 'low' for large foundation, got '%s'", comparison.Delta.ResilienceChange)
	}

	// No blast radius warning expected - 235 cells is plenty resilient
	for _, w := range comparison.Warnings {
		if w.Message == "Elevated cell failure impact" || w.Message == "High cell failure impact" {
			t.Errorf("Did not expect blast radius warning for 235 cells, got: %s", w.Message)
		}
	}

	t.Logf("Comparison: Current %s (%d cells) → Proposed %s (%d cells)",
		comparison.Current.CellSize(), comparison.Current.CellCount,
		comparison.Proposed.CellSize(), comparison.Proposed.CellCount)
	t.Logf("Capacity: %d GB → %d GB (change: %+d GB)",
		comparison.Current.AppCapacityGB, comparison.Proposed.AppCapacityGB,
		comparison.Delta.CapacityChangeGB)
	t.Logf("Warnings: %d", len(comparison.Warnings))
}
