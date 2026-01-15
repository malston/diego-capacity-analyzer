// ABOUTME: Infrastructure planning calculator for cell capacity analysis
// ABOUTME: Computes max deployable cells given IaaS memory and CPU constraints

package services

import (
	"fmt"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// Common cell size presets for recommendations
var cellSizePresets = []struct {
	cpu int
	mem int
}{
	{4, 32},
	{4, 64},
	{8, 64},
	{8, 128},
	{16, 128},
	{16, 256},
}

// PlanningCalculator computes infrastructure planning metrics
type PlanningCalculator struct{}

// NewPlanningCalculator creates a new planning calculator
func NewPlanningCalculator() *PlanningCalculator {
	return &PlanningCalculator{}
}

// Calculate computes max deployable cells given infrastructure and cell specs
func (c *PlanningCalculator) Calculate(state models.InfrastructureState, input models.PlanningInput) models.PlanningResult {
	// Handle invalid inputs
	if input.CellMemoryGB <= 0 || input.CellCPU <= 0 {
		return models.PlanningResult{Bottleneck: "invalid"}
	}

	// Get total available resources
	memoryAvail := state.TotalN1MemoryGB
	cpuAvail := state.TotalCPUCores
	// Fallback: compute from clusters if TotalCPUCores is not set
	if cpuAvail == 0 {
		for _, cluster := range state.Clusters {
			cpuAvail += cluster.CPUCores
		}
	}

	// Handle empty state
	if memoryAvail <= 0 && cpuAvail <= 0 {
		return models.PlanningResult{Bottleneck: "none"}
	}

	// Check which resources are selected for analysis
	cpuSelected := isResourceSelected(input.SelectedResources, "cpu")
	memorySelected := isResourceSelected(input.SelectedResources, "memory")

	// Calculate max cells by each resource
	maxByMemory := 0
	if memoryAvail > 0 {
		maxByMemory = memoryAvail / input.CellMemoryGB
	}

	maxByCPU := 0
	if cpuAvail > 0 {
		maxByCPU = cpuAvail / input.CellCPU
	}

	// Deployable is the minimum of SELECTED resources only
	deployable := 0
	if memorySelected && cpuSelected {
		// Both selected - use minimum
		deployable = maxByMemory
		if maxByCPU < deployable {
			deployable = maxByCPU
		}
	} else if memorySelected {
		// Only memory selected
		deployable = maxByMemory
	} else if cpuSelected {
		// Only CPU selected
		deployable = maxByCPU
	} else {
		// Neither selected - default to memory (legacy behavior)
		deployable = maxByMemory
	}

	// Determine bottleneck - only consider selected resources
	bottleneck := "balanced"
	if memorySelected && cpuSelected {
		// Both selected - report actual bottleneck
		if maxByMemory < maxByCPU {
			bottleneck = "memory"
		} else if maxByCPU < maxByMemory {
			bottleneck = "cpu"
		}
	} else if memorySelected && !cpuSelected {
		// Only memory selected - always report memory as constraint
		bottleneck = "memory"
	} else if cpuSelected && !memorySelected {
		// Only CPU selected - always report cpu as constraint
		bottleneck = "cpu"
	}

	// Calculate actual resource usage
	memoryUsed := deployable * input.CellMemoryGB
	cpuUsed := deployable * input.CellCPU

	// Calculate utilization percentages
	var memoryUtilPct, cpuUtilPct float64
	if memoryAvail > 0 {
		memoryUtilPct = float64(memoryUsed) / float64(memoryAvail) * 100
	}
	if cpuAvail > 0 {
		cpuUtilPct = float64(cpuUsed) / float64(cpuAvail) * 100
	}

	// Calculate headroom (unused capacity in cells)
	headroom := 0
	if maxByMemory > maxByCPU {
		headroom = maxByMemory - deployable
	} else if maxByCPU > maxByMemory {
		headroom = maxByCPU - deployable
	}

	return models.PlanningResult{
		MaxCellsByMemory: maxByMemory,
		MaxCellsByCPU:    maxByCPU,
		DeployableCells:  deployable,
		Bottleneck:       bottleneck,
		MemoryUsedGB:     memoryUsed,
		MemoryAvailGB:    memoryAvail,
		CPUUsed:          cpuUsed,
		CPUAvail:         cpuAvail,
		MemoryUtilPct:    memoryUtilPct,
		CPUUtilPct:       cpuUtilPct,
		HeadroomCells:    headroom,
	}
}

// GenerateRecommendations produces sizing alternatives for the given infrastructure.
// selectedResources filters which resources are considered for bottleneck reporting.
func (c *PlanningCalculator) GenerateRecommendations(state models.InfrastructureState, selectedResources []string) []models.SizingRecommendation {
	var recommendations []models.SizingRecommendation

	for _, preset := range cellSizePresets {
		input := models.PlanningInput{
			CellCPU:           preset.cpu,
			CellMemoryGB:      preset.mem,
			SelectedResources: selectedResources,
		}

		result := c.Calculate(state, input)

		// Skip if no cells can be deployed
		if result.DeployableCells <= 0 {
			continue
		}

		recommendations = append(recommendations, models.SizingRecommendation{
			CellCPU:         preset.cpu,
			CellMemoryGB:    preset.mem,
			DeployableCells: result.DeployableCells,
			Bottleneck:      result.Bottleneck,
			MemoryUtilPct:   result.MemoryUtilPct,
			CPUUtilPct:      result.CPUUtilPct,
			Label:           fmt.Sprintf("%d×%d GB", preset.cpu, preset.mem),
		})
	}

	return recommendations
}

// Plan computes both the result for given input and recommendations
func (c *PlanningCalculator) Plan(state models.InfrastructureState, input models.PlanningInput) models.PlanningResponse {
	return models.PlanningResponse{
		Result:          c.Calculate(state, input),
		Recommendations: c.GenerateRecommendations(state, input.SelectedResources),
	}
}
