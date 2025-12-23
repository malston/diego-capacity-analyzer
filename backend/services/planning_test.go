// ABOUTME: Tests for infrastructure planning service
// ABOUTME: Validates cell count calculations and sizing recommendations

package services

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestCalculateMaxCells_MemoryConstrained(t *testing.T) {
	// Scenario: 1024 GB memory, 256 vCPUs available
	// Cell size: 4 vCPU × 32 GB
	// Max by memory: 1024 / 32 = 32 cells
	// Max by CPU: 256 / 4 = 64 cells
	// Result: 32 cells (memory-constrained)

	state := models.InfrastructureState{
		TotalN1MemoryGB: 1024,
		Clusters: []models.ClusterState{
			{
				CPUCores: 256,
			},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	if result.MaxCellsByMemory != 32 {
		t.Errorf("Expected MaxCellsByMemory 32, got %d", result.MaxCellsByMemory)
	}
	if result.MaxCellsByCPU != 64 {
		t.Errorf("Expected MaxCellsByCPU 64, got %d", result.MaxCellsByCPU)
	}
	if result.DeployableCells != 32 {
		t.Errorf("Expected DeployableCells 32, got %d", result.DeployableCells)
	}
	if result.Bottleneck != "memory" {
		t.Errorf("Expected Bottleneck 'memory', got '%s'", result.Bottleneck)
	}
}

func TestCalculateMaxCells_CPUConstrained(t *testing.T) {
	// Scenario: 2048 GB memory, 128 vCPUs available
	// Cell size: 8 vCPU × 64 GB
	// Max by memory: 2048 / 64 = 32 cells
	// Max by CPU: 128 / 8 = 16 cells
	// Result: 16 cells (CPU-constrained)

	state := models.InfrastructureState{
		TotalN1MemoryGB: 2048,
		Clusters: []models.ClusterState{
			{
				CPUCores: 128,
			},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 64,
		CellCPU:      8,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	if result.MaxCellsByMemory != 32 {
		t.Errorf("Expected MaxCellsByMemory 32, got %d", result.MaxCellsByMemory)
	}
	if result.MaxCellsByCPU != 16 {
		t.Errorf("Expected MaxCellsByCPU 16, got %d", result.MaxCellsByCPU)
	}
	if result.DeployableCells != 16 {
		t.Errorf("Expected DeployableCells 16, got %d", result.DeployableCells)
	}
	if result.Bottleneck != "cpu" {
		t.Errorf("Expected Bottleneck 'cpu', got '%s'", result.Bottleneck)
	}
}

func TestCalculateMaxCells_Balanced(t *testing.T) {
	// Scenario: 512 GB memory, 64 vCPUs available
	// Cell size: 4 vCPU × 32 GB
	// Max by memory: 512 / 32 = 16 cells
	// Max by CPU: 64 / 4 = 16 cells
	// Result: 16 cells (balanced)

	state := models.InfrastructureState{
		TotalN1MemoryGB: 512,
		Clusters: []models.ClusterState{
			{
				CPUCores: 64,
			},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	if result.DeployableCells != 16 {
		t.Errorf("Expected DeployableCells 16, got %d", result.DeployableCells)
	}
	if result.Bottleneck != "balanced" {
		t.Errorf("Expected Bottleneck 'balanced', got '%s'", result.Bottleneck)
	}
}

func TestCalculateMaxCells_MultiCluster(t *testing.T) {
	// Scenario: 2 clusters, each with 512 GB memory and 64 vCPUs
	// Total: 1024 GB memory, 128 vCPUs
	// Cell size: 4 vCPU × 32 GB
	// Max by memory: 1024 / 32 = 32 cells
	// Max by CPU: 128 / 4 = 32 cells

	state := models.InfrastructureState{
		TotalN1MemoryGB: 1024, // Aggregate
		Clusters: []models.ClusterState{
			{CPUCores: 64},
			{CPUCores: 64},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	if result.DeployableCells != 32 {
		t.Errorf("Expected DeployableCells 32, got %d", result.DeployableCells)
	}
}

func TestCalculateMaxCells_UtilizationMetrics(t *testing.T) {
	// Verify utilization calculations
	state := models.InfrastructureState{
		TotalN1MemoryGB: 1000,
		Clusters: []models.ClusterState{
			{CPUCores: 100},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	// Max cells by memory: 1000 / 32 = 31 cells
	// Max cells by CPU: 100 / 4 = 25 cells
	// Deployable: 25 cells (CPU constrained)
	// Memory used: 25 × 32 = 800 GB
	// Memory util: 800 / 1000 = 80%
	// CPU used: 25 × 4 = 100
	// CPU util: 100 / 100 = 100%

	if result.MemoryUsedGB != 800 {
		t.Errorf("Expected MemoryUsedGB 800, got %d", result.MemoryUsedGB)
	}
	if result.MemoryAvailGB != 1000 {
		t.Errorf("Expected MemoryAvailGB 1000, got %d", result.MemoryAvailGB)
	}
	if result.CPUUsed != 100 {
		t.Errorf("Expected CPUUsed 100, got %d", result.CPUUsed)
	}
	if result.CPUAvail != 100 {
		t.Errorf("Expected CPUAvail 100, got %d", result.CPUAvail)
	}

	// Allow small tolerance for floating point
	if result.MemoryUtilPct < 79 || result.MemoryUtilPct > 81 {
		t.Errorf("Expected MemoryUtilPct ~80%%, got %.1f%%", result.MemoryUtilPct)
	}
	if result.CPUUtilPct < 99 || result.CPUUtilPct > 101 {
		t.Errorf("Expected CPUUtilPct ~100%%, got %.1f%%", result.CPUUtilPct)
	}
}

func TestCalculateMaxCells_Headroom(t *testing.T) {
	// Headroom = cells possible by non-bottleneck - deployable cells
	state := models.InfrastructureState{
		TotalN1MemoryGB: 1024,
		Clusters: []models.ClusterState{
			{CPUCores: 256},
		},
	}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	// Memory: 1024/32 = 32 cells
	// CPU: 256/4 = 64 cells
	// Deployable: 32 (memory-constrained)
	// Headroom: 64 - 32 = 32 cells worth of CPU unused

	if result.HeadroomCells != 32 {
		t.Errorf("Expected HeadroomCells 32, got %d", result.HeadroomCells)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	state := models.InfrastructureState{
		TotalN1MemoryGB: 2048,
		Clusters: []models.ClusterState{
			{CPUCores: 128},
		},
	}

	calc := NewPlanningCalculator()
	recs := calc.GenerateRecommendations(state)

	// Should have multiple recommendations
	if len(recs) < 3 {
		t.Errorf("Expected at least 3 recommendations, got %d", len(recs))
	}

	// Each recommendation should have valid data
	for _, rec := range recs {
		if rec.CellCPU <= 0 {
			t.Errorf("Recommendation has invalid CellCPU: %d", rec.CellCPU)
		}
		if rec.CellMemoryGB <= 0 {
			t.Errorf("Recommendation has invalid CellMemoryGB: %d", rec.CellMemoryGB)
		}
		if rec.DeployableCells <= 0 {
			t.Errorf("Recommendation has invalid DeployableCells: %d", rec.DeployableCells)
		}
		if rec.Bottleneck == "" {
			t.Error("Recommendation has empty Bottleneck")
		}
		if rec.Label == "" {
			t.Error("Recommendation has empty Label")
		}
	}
}

func TestGenerateRecommendations_ContainsCommonSizes(t *testing.T) {
	state := models.InfrastructureState{
		TotalN1MemoryGB: 4096,
		Clusters: []models.ClusterState{
			{CPUCores: 512},
		},
	}

	calc := NewPlanningCalculator()
	recs := calc.GenerateRecommendations(state)

	// Should contain common cell sizes
	commonSizes := []struct {
		cpu int
		mem int
	}{
		{4, 32},
		{4, 64},
		{8, 64},
		{8, 128},
	}

	for _, size := range commonSizes {
		found := false
		for _, rec := range recs {
			if rec.CellCPU == size.cpu && rec.CellMemoryGB == size.mem {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected recommendation for %d×%d GB", size.cpu, size.mem)
		}
	}
}

func TestCalculateMaxCells_ZeroInputs(t *testing.T) {
	// Edge case: zero or invalid inputs
	state := models.InfrastructureState{
		TotalN1MemoryGB: 1024,
		Clusters: []models.ClusterState{
			{CPUCores: 128},
		},
	}

	calc := NewPlanningCalculator()

	// Zero cell memory
	result := calc.Calculate(state, models.PlanningInput{CellMemoryGB: 0, CellCPU: 4})
	if result.DeployableCells != 0 {
		t.Errorf("Zero cell memory should yield 0 deployable cells, got %d", result.DeployableCells)
	}

	// Zero cell CPU
	result = calc.Calculate(state, models.PlanningInput{CellMemoryGB: 32, CellCPU: 0})
	if result.DeployableCells != 0 {
		t.Errorf("Zero cell CPU should yield 0 deployable cells, got %d", result.DeployableCells)
	}
}

func TestCalculateMaxCells_EmptyState(t *testing.T) {
	// Edge case: empty infrastructure state
	state := models.InfrastructureState{}

	input := models.PlanningInput{
		CellMemoryGB: 32,
		CellCPU:      4,
	}

	calc := NewPlanningCalculator()
	result := calc.Calculate(state, input)

	if result.DeployableCells != 0 {
		t.Errorf("Empty state should yield 0 deployable cells, got %d", result.DeployableCells)
	}
}
