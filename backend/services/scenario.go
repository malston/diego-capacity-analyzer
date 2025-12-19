// ABOUTME: Scenario calculator for what-if capacity analysis
// ABOUTME: Computes metrics and warnings for current vs proposed configurations

package services

import (
	"math"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const (
	// GardenOverheadGB is memory reserved per cell for Garden/system
	GardenOverheadGB = 5
	// ChunkSizeGB is the size of a free chunk for staging
	ChunkSizeGB = 4
)

// ScenarioCalculator computes capacity metrics for scenarios
type ScenarioCalculator struct{}

// NewScenarioCalculator creates a new calculator
func NewScenarioCalculator() *ScenarioCalculator {
	return &ScenarioCalculator{}
}

// CalculateCurrent computes metrics for current infrastructure state
func (c *ScenarioCalculator) CalculateCurrent(state models.InfrastructureState) models.ScenarioResult {
	// Get cell config from first cluster (assumes uniform cells)
	var cellMemoryGB, cellCPU int
	for _, cluster := range state.Clusters {
		if cluster.DiegoCellMemoryGB > 0 {
			cellMemoryGB = cluster.DiegoCellMemoryGB
			cellCPU = cluster.DiegoCellCPU
			break
		}
	}

	return c.calculate(
		state.TotalCellCount,
		cellMemoryGB,
		cellCPU,
		state.TotalAppMemoryGB,
		state.TotalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
	)
}

// CalculateProposed computes metrics for a proposed scenario
func (c *ScenarioCalculator) CalculateProposed(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioResult {
	return c.calculate(
		input.ProposedCellCount,
		input.ProposedCellMemoryGB,
		input.ProposedCellCPU,
		state.TotalAppMemoryGB,
		state.TotalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
	)
}

// calculate performs the core metric calculations
func (c *ScenarioCalculator) calculate(
	cellCount int,
	cellMemoryGB int,
	cellCPU int,
	totalAppMemoryGB int,
	totalAppInstances int,
	platformVMsGB int,
	n1MemoryGB int,
) models.ScenarioResult {
	// App capacity: cells × (cellMemory - overhead)
	appCapacityGB := cellCount * (cellMemoryGB - GardenOverheadGB)

	// Utilization: appMemory / capacity × 100
	var utilizationPct float64
	if appCapacityGB > 0 {
		utilizationPct = float64(totalAppMemoryGB) / float64(appCapacityGB) * 100
	}

	// Free chunks: (capacity - used) / chunkSize
	freeChunks := (appCapacityGB - totalAppMemoryGB) / ChunkSizeGB
	if freeChunks < 0 {
		freeChunks = 0
	}

	// Instances per cell
	var instancesPerCell float64
	if cellCount > 0 {
		instancesPerCell = float64(totalAppInstances) / float64(cellCount)
	}

	// Fault impact (rounded)
	faultImpact := int(math.Round(instancesPerCell))

	// N-1 utilization: (cellMemory + platformVMs) / n1Memory × 100
	totalCellMemoryGB := cellCount * cellMemoryGB
	var n1UtilizationPct float64
	if n1MemoryGB > 0 {
		n1UtilizationPct = float64(totalCellMemoryGB+platformVMsGB) / float64(n1MemoryGB) * 100
	}

	return models.ScenarioResult{
		CellCount:        cellCount,
		CellMemoryGB:     cellMemoryGB,
		CellCPU:          cellCPU,
		AppCapacityGB:    appCapacityGB,
		UtilizationPct:   utilizationPct,
		FreeChunks:       freeChunks,
		N1UtilizationPct: n1UtilizationPct,
		FaultImpact:      faultImpact,
		InstancesPerCell: instancesPerCell,
	}
}

// GenerateWarnings produces warnings based on proposed scenario
func (c *ScenarioCalculator) GenerateWarnings(current, proposed models.ScenarioResult) []models.ScenarioWarning {
	var warnings []models.ScenarioWarning

	// N-1 utilization warnings
	if proposed.N1UtilizationPct > 85 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Exceeds N-1 capacity safety margin",
		})
	} else if proposed.N1UtilizationPct > 75 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Approaching N-1 capacity limits",
		})
	}

	// Free chunks warnings
	if proposed.FreeChunks < 200 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Critical: Low staging capacity",
		})
	} else if proposed.FreeChunks < 400 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Low staging capacity",
		})
	}

	// Cell utilization warnings
	if proposed.UtilizationPct > 90 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Cell utilization critically high",
		})
	} else if proposed.UtilizationPct > 80 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Cell utilization elevated",
		})
	}

	// Redundancy reduction warning
	if current.CellCount > 0 {
		reduction := float64(current.CellCount-proposed.CellCount) / float64(current.CellCount) * 100
		if reduction >= 50 {
			warnings = append(warnings, models.ScenarioWarning{
				Severity: "warning",
				Message:  "Significant redundancy reduction",
			})
		}
	}

	return warnings
}

// Compare computes full comparison between current and proposed scenarios
func (c *ScenarioCalculator) Compare(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioComparison {
	current := c.CalculateCurrent(state)
	proposed := c.CalculateProposed(state, input)
	warnings := c.GenerateWarnings(current, proposed)

	// Calculate delta
	capacityChange := proposed.AppCapacityGB - current.AppCapacityGB
	utilizationChange := proposed.UtilizationPct - current.UtilizationPct

	var redundancyChange string
	if proposed.CellCount > current.CellCount {
		redundancyChange = "improved"
	} else if proposed.CellCount < current.CellCount {
		redundancyChange = "reduced"
	} else {
		redundancyChange = "unchanged"
	}

	return models.ScenarioComparison{
		Current:  current,
		Proposed: proposed,
		Warnings: warnings,
		Delta: models.ScenarioDelta{
			CapacityChangeGB:     capacityChange,
			UtilizationChangePct: utilizationChange,
			RedundancyChange:     redundancyChange,
		},
	}
}
