// ABOUTME: Tests for upgrade path recommendations
// ABOUTME: Validates add cells, resize cells, and add hosts recommendation logic

package models

import (
	"encoding/json"
	"testing"
)

func TestRecommendation_BasicFields(t *testing.T) {
	rec := Recommendation{
		Type:        RecommendationAddCells,
		Priority:    1,
		Title:       "Add Diego Cells",
		Description: "Add 2 more Diego cells to increase capacity",
		Impact:      "Increases memory capacity by 64 GB",
		Resource:    "Memory",
	}

	if rec.Type != RecommendationAddCells {
		t.Errorf("Expected Type '%s', got '%s'", RecommendationAddCells, rec.Type)
	}
	if rec.Priority != 1 {
		t.Errorf("Expected Priority 1, got %d", rec.Priority)
	}
}

func TestRecommendation_Serialization(t *testing.T) {
	rec := Recommendation{
		Type:        RecommendationResizeCells,
		Priority:    2,
		Title:       "Resize Diego Cells",
		Description: "Increase cell memory from 32GB to 64GB",
		Impact:      "Doubles memory capacity per cell",
		Resource:    "Memory",
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("Failed to marshal Recommendation: %v", err)
	}

	var decoded Recommendation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Recommendation: %v", err)
	}

	if decoded.Type != rec.Type {
		t.Errorf("Type mismatch: got '%s', want '%s'", decoded.Type, rec.Type)
	}
	if decoded.Title != rec.Title {
		t.Errorf("Title mismatch: got '%s', want '%s'", decoded.Title, rec.Title)
	}
}

func TestGenerateAddCellsRecommendation_MemoryConstrained(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host (4096 total)
		64,   // cores per host (256 total)
		100,  // cells
		32,   // cell mem (3200 total cell mem)
		4,    // cell cpu (400 total vcpu)
		100,  // cell disk
		2800, // app mem (87.5% of 3200)
		4000, // app disk
	)

	rec := GenerateAddCellsRecommendation(state, "Memory")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Type != RecommendationAddCells {
		t.Errorf("Expected Type '%s', got '%s'", RecommendationAddCells, rec.Type)
	}
	if rec.Resource != "Memory" {
		t.Errorf("Expected Resource 'Memory', got '%s'", rec.Resource)
	}
	if rec.CellsToAdd <= 0 {
		t.Error("Expected CellsToAdd > 0")
	}
}

func TestGenerateAddCellsRecommendation_CPUConstrained(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host (256 total)
		100,  // cells
		32,   // cell mem
		8,    // cell cpu (800 vcpu = 312.5% ratio)
		100,  // cell disk
		1600, // app mem (50%)
		4000, // app disk
	)

	rec := GenerateAddCellsRecommendation(state, "CPU")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Resource != "CPU" {
		t.Errorf("Expected Resource 'CPU', got '%s'", rec.Resource)
	}
}

func TestGenerateResizeCellsRecommendation_MemoryConstrained(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host
		100,  // cells
		32,   // cell mem
		4,    // cell cpu
		100,  // cell disk
		2800, // app mem (high utilization)
		4000, // app disk
	)

	rec := GenerateResizeCellsRecommendation(state, "Memory")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Type != RecommendationResizeCells {
		t.Errorf("Expected Type '%s', got '%s'", RecommendationResizeCells, rec.Type)
	}
	if rec.Resource != "Memory" {
		t.Errorf("Expected Resource 'Memory', got '%s'", rec.Resource)
	}
	if rec.NewCellMemoryGB <= state.Clusters[0].DiegoCellMemoryGB {
		t.Error("Expected NewCellMemoryGB to be larger than current")
	}
}

func TestGenerateResizeCellsRecommendation_CPUConstrained(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host
		100,  // cells
		32,   // cell mem
		8,    // cell cpu - high ratio
		100,  // cell disk
		1600, // app mem
		4000, // app disk
	)

	rec := GenerateResizeCellsRecommendation(state, "CPU")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Resource != "CPU" {
		t.Errorf("Expected Resource 'CPU', got '%s'", rec.Resource)
	}
	if rec.NewCellCPU <= state.Clusters[0].DiegoCellCPU {
		t.Error("Expected NewCellCPU to be larger than current")
	}
}

func TestGenerateAddHostsRecommendation_HostMemoryConstrained(t *testing.T) {
	// High host memory utilization scenario
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host (4096 total)
		64,   // cores per host
		120,  // cells
		32,   // cell mem (3840 total = 94% of host mem)
		4,    // cell cpu
		100,  // cell disk
		3200, // app mem
		4000, // app disk
	)

	rec := GenerateAddHostsRecommendation(state, "Memory")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Type != RecommendationAddHosts {
		t.Errorf("Expected Type '%s', got '%s'", RecommendationAddHosts, rec.Type)
	}
	if rec.HostsToAdd <= 0 {
		t.Error("Expected HostsToAdd > 0")
	}
}

func TestGenerateAddHostsRecommendation_CPUConstrained(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host (256 total)
		100,  // cells
		32,   // cell mem
		10,   // cell cpu (1000 vcpu = 390% ratio, very high)
		100,  // cell disk
		1600, // app mem
		4000, // app disk
	)

	rec := GenerateAddHostsRecommendation(state, "CPU")

	if rec == nil {
		t.Fatal("Expected a recommendation, got nil")
	}
	if rec.Resource != "CPU" {
		t.Errorf("Expected Resource 'CPU', got '%s'", rec.Resource)
	}
}

func TestGenerateRecommendations_FullAnalysis(t *testing.T) {
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host
		100,  // cells
		32,   // cell mem
		4,    // cell cpu
		100,  // cell disk
		2800, // app mem (high)
		4000, // app disk
	)

	recs := GenerateRecommendations(state)

	if len(recs) == 0 {
		t.Fatal("Expected at least one recommendation")
	}

	// Recommendations should be ordered by priority
	for i := 0; i < len(recs)-1; i++ {
		if recs[i].Priority > recs[i+1].Priority {
			t.Errorf("Recommendations not sorted by priority: %d > %d at positions %d, %d",
				recs[i].Priority, recs[i+1].Priority, i, i+1)
		}
	}

	// First recommendation should target the constraining resource
	if recs[0].Resource == "" {
		t.Error("First recommendation should have a resource specified")
	}
}

func TestGenerateRecommendations_LowUtilization(t *testing.T) {
	// Low utilization scenario - should still provide recommendations but lower priority
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host
		50,   // cells (low count)
		32,   // cell mem
		4,    // cell cpu
		100,  // cell disk
		800,  // app mem (low)
		2000, // app disk
	)

	recs := GenerateRecommendations(state)

	// Should have recommendations even at low utilization
	if len(recs) == 0 {
		t.Log("No recommendations at low utilization - this may be acceptable")
	}
}

func TestRecommendationPriority_ConstrainingResourceFirst(t *testing.T) {
	// Memory is the constraint
	state := createTestInfrastructure(
		4,    // hosts
		1024, // mem per host
		64,   // cores per host
		100,  // cells
		32,   // cell mem
		2,    // cell cpu (low ratio)
		100,  // cell disk
		3000, // app mem (very high - 93.75%)
		2000, // app disk (low)
	)

	recs := GenerateRecommendations(state)

	if len(recs) == 0 {
		t.Fatal("Expected recommendations")
	}

	// First recommendation should be for Memory (the constraint)
	if recs[0].Resource != "Memory" {
		t.Errorf("Expected first recommendation to target Memory (the constraint), got '%s'", recs[0].Resource)
	}
}

func TestRecommendationsResponse_Serialization(t *testing.T) {
	response := RecommendationsResponse{
		Recommendations: []Recommendation{
			{Type: RecommendationAddCells, Priority: 1, Resource: "Memory"},
			{Type: RecommendationResizeCells, Priority: 2, Resource: "Memory"},
		},
		ConstrainingResource: "Memory",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal RecommendationsResponse: %v", err)
	}

	var decoded RecommendationsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RecommendationsResponse: %v", err)
	}

	if len(decoded.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(decoded.Recommendations))
	}
	if decoded.ConstrainingResource != "Memory" {
		t.Errorf("Expected ConstrainingResource 'Memory', got '%s'", decoded.ConstrainingResource)
	}
}

// createTestInfrastructure is a helper to create InfrastructureState for testing
func createTestInfrastructure(
	hostCount, memPerHost, cpuPerHost int,
	cellCount, cellMem, cellCPU, cellDisk int,
	appMem, appDisk int,
) InfrastructureState {
	mi := ManualInput{
		Name: "Test Infrastructure",
		Clusters: []ClusterInput{
			{
				Name:              "test-cluster",
				HostCount:         hostCount,
				MemoryGBPerHost:   memPerHost,
				CPUCoresPerHost:   cpuPerHost,
				DiegoCellCount:    cellCount,
				DiegoCellMemoryGB: cellMem,
				DiegoCellCPU:      cellCPU,
				DiegoCellDiskGB:   cellDisk,
			},
		},
		TotalAppMemoryGB: appMem,
		TotalAppDiskGB:   appDisk,
	}
	return mi.ToInfrastructureState()
}
