package services

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestCalculateCurrentScenario(t *testing.T) {
	// Based on capacity doc: 470 cells (4×32), 7500 instances, 10.5 TB apps
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624, // 14 hosts * 2048 - simulated N-1
		TotalCellCount:    470,
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateCurrent(state)

	// Cell count
	if result.CellCount != 470 {
		t.Errorf("Expected CellCount 470, got %d", result.CellCount)
	}

	// App capacity: 470 × (32 - 5) = 470 × 27 = 12690 GB
	if result.AppCapacityGB != 12690 {
		t.Errorf("Expected AppCapacityGB 12690, got %d", result.AppCapacityGB)
	}

	// Utilization: 10500 / 12690 × 100 = 82.7%
	if result.UtilizationPct < 82 || result.UtilizationPct > 83 {
		t.Errorf("Expected UtilizationPct ~82.7%%, got %.1f%%", result.UtilizationPct)
	}

	// Free chunks: (12690 - 10500) / 4 = 547
	if result.FreeChunks != 547 {
		t.Errorf("Expected FreeChunks 547, got %d", result.FreeChunks)
	}

	// Instances per cell: 7500 / 470 = 15.96
	if result.InstancesPerCell < 15.9 || result.InstancesPerCell > 16.1 {
		t.Errorf("Expected InstancesPerCell ~16, got %.1f", result.InstancesPerCell)
	}

	// Fault impact (apps per cell): 7500 / 470 = 16
	if result.FaultImpact != 16 {
		t.Errorf("Expected FaultImpact 16, got %d", result.FaultImpact)
	}

	// N-1 utilization: (470 × 32 + 4800) / 26624 × 100 = 74.5%
	// Cell memory: 470 × 32 = 15040
	// Total: 15040 + 4800 = 19840
	// 19840 / 26624 = 74.5%
	if result.N1UtilizationPct < 74 || result.N1UtilizationPct > 75 {
		t.Errorf("Expected N1UtilizationPct ~74.5%%, got %.1f%%", result.N1UtilizationPct)
	}
}

func TestCalculateProposedScenario(t *testing.T) {
	// Same infrastructure, but proposing 4×64 cells with 235 cells
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    470,
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    235,
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateProposed(state, input)

	// Cell count
	if result.CellCount != 235 {
		t.Errorf("Expected CellCount 235, got %d", result.CellCount)
	}

	// App capacity: 235 × (64 - 5) = 235 × 59 = 13865 GB
	if result.AppCapacityGB != 13865 {
		t.Errorf("Expected AppCapacityGB 13865, got %d", result.AppCapacityGB)
	}

	// Utilization: 10500 / 13865 × 100 = 75.7%
	if result.UtilizationPct < 75 || result.UtilizationPct > 76 {
		t.Errorf("Expected UtilizationPct ~75.7%%, got %.1f%%", result.UtilizationPct)
	}

	// Fault impact: 7500 / 235 = 32
	if result.FaultImpact != 32 {
		t.Errorf("Expected FaultImpact 32, got %d", result.FaultImpact)
	}
}
