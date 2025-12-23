// ABOUTME: Scenario calculator for what-if capacity analysis
// ABOUTME: Computes metrics and warnings for current vs proposed configurations

package services

import (
	"fmt"
	"math"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const (
	// DefaultMemoryOverheadPct is the default memory overhead percentage (7% for Garden/system)
	DefaultMemoryOverheadPct = 7.0
	// DefaultDiskOverheadPct is the default disk overhead percentage (negligible)
	DefaultDiskOverheadPct = 0.01
	// ChunkSizeGB is the size of a free chunk for staging
	ChunkSizeGB = 4
	// PeakTPS is the peak TPS used for status determination
	PeakTPS = 1964
)

// DefaultTPSCurve is the default TPS curve - baseline estimates, user can override
var DefaultTPSCurve = []models.TPSPt{
	{Cells: 1, TPS: 284},
	{Cells: 3, TPS: 1964},
	{Cells: 9, TPS: 1932},
	{Cells: 100, TPS: 1389},
	{Cells: 210, TPS: 104},
}

// TPSDataPoint is an alias for models.TPSPt for backward compatibility
type TPSDataPoint = models.TPSPt

// ScenarioCalculator computes capacity metrics for scenarios
type ScenarioCalculator struct{}

// NewScenarioCalculator creates a new calculator
func NewScenarioCalculator() *ScenarioCalculator {
	return &ScenarioCalculator{}
}

// EstimateTPS estimates TPS for a given cell count using the provided curve
// If curve is nil, uses DefaultTPSCurve
func EstimateTPS(cellCount int, curve []models.TPSPt) (tps int, status string) {
	if cellCount <= 0 {
		return 0, "unknown"
	}

	// Use default curve if none provided
	if len(curve) == 0 {
		curve = DefaultTPSCurve
	}

	// Exact match
	for _, pt := range curve {
		if pt.Cells == cellCount {
			tps = pt.TPS
			break
		}
	}

	// Interpolation if no exact match
	if tps == 0 {
		for i := 0; i < len(curve)-1; i++ {
			if cellCount >= curve[i].Cells && cellCount <= curve[i+1].Cells {
				ratio := float64(cellCount-curve[i].Cells) / float64(curve[i+1].Cells-curve[i].Cells)
				tps = int(float64(curve[i].TPS) + ratio*float64(curve[i+1].TPS-curve[i].TPS))
				break
			}
		}
	}

	// Beyond last data point - extrapolate degradation
	if tps == 0 && cellCount > curve[len(curve)-1].Cells {
		lastPt := curve[len(curve)-1]
		// Simple extrapolation: TPS degrades proportionally
		tps = lastPt.TPS * lastPt.Cells / cellCount
		if tps < 1 {
			tps = 1
		}
	}

	// Before first data point
	if tps == 0 && cellCount < curve[0].Cells {
		tps = curve[0].TPS
	}

	// Determine status based on peak TPS
	peakTPS := PeakTPS
	// Find peak in curve
	for _, pt := range curve {
		if pt.TPS > peakTPS {
			peakTPS = pt.TPS
		}
	}

	threshold80 := peakTPS * 80 / 100
	threshold50 := peakTPS * 50 / 100

	if tps >= threshold80 {
		status = "optimal"
	} else if tps >= threshold50 {
		status = "degraded"
	} else {
		status = "critical"
	}

	return tps, status
}

// CalculateCurrent computes metrics for current infrastructure state
func (c *ScenarioCalculator) CalculateCurrent(state models.InfrastructureState) models.ScenarioResult {
	// Get cell config from first cluster (assumes uniform cells)
	var cellMemoryGB, cellCPU, cellDiskGB int
	for _, cluster := range state.Clusters {
		if cluster.DiegoCellMemoryGB > 0 {
			cellMemoryGB = cluster.DiegoCellMemoryGB
			cellCPU = cluster.DiegoCellCPU
			cellDiskGB = cluster.DiegoCellDiskGB
			break
		}
	}

	return c.calculateFull(
		state.TotalCellCount,
		cellMemoryGB,
		cellCPU,
		cellDiskGB,
		state.TotalAppMemoryGB,
		state.TotalAppDiskGB,
		state.TotalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
		DefaultMemoryOverheadPct,
		nil, // default TPS curve
	)
}

// CalculateProposed computes metrics for a proposed scenario
func (c *ScenarioCalculator) CalculateProposed(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioResult {
	// Get overhead percentage (default to 7% if not specified)
	overheadPct := input.OverheadPct
	if overheadPct == 0 {
		overheadPct = DefaultMemoryOverheadPct
	}

	// Calculate app memory/disk including additional app if specified
	totalAppMemoryGB := state.TotalAppMemoryGB
	totalAppDiskGB := state.TotalAppDiskGB
	totalAppInstances := state.TotalAppInstances

	if input.AdditionalApp != nil {
		totalAppMemoryGB += input.AdditionalApp.Instances * input.AdditionalApp.MemoryGB
		totalAppDiskGB += input.AdditionalApp.Instances * input.AdditionalApp.DiskGB
		totalAppInstances += input.AdditionalApp.Instances
	}

	return c.calculateFull(
		input.ProposedCellCount,
		input.ProposedCellMemoryGB,
		input.ProposedCellCPU,
		input.ProposedCellDiskGB,
		totalAppMemoryGB,
		totalAppDiskGB,
		totalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
		overheadPct,
		input.TPSCurve,
	)
}

// calculateFull performs the core metric calculations with all features
func (c *ScenarioCalculator) calculateFull(
	cellCount int,
	cellMemoryGB int,
	cellCPU int,
	cellDiskGB int,
	totalAppMemoryGB int,
	totalAppDiskGB int,
	totalAppInstances int,
	platformVMsGB int,
	n1MemoryGB int,
	overheadPct float64,
	tpsCurve []models.TPSPt,
) models.ScenarioResult {
	// Memory overhead as percentage
	memoryOverhead := int(float64(cellMemoryGB) * (overheadPct / 100))
	appCapacityGB := cellCount * (cellMemoryGB - memoryOverhead)

	// Disk overhead (0.01% - negligible but included for completeness)
	diskOverhead := int(float64(cellDiskGB) * (DefaultDiskOverheadPct / 100))
	diskCapacityGB := 0
	if cellDiskGB > 0 {
		diskCapacityGB = cellCount * (cellDiskGB - diskOverhead)
	}

	// Memory utilization
	var utilizationPct float64
	if appCapacityGB > 0 {
		utilizationPct = float64(totalAppMemoryGB) / float64(appCapacityGB) * 100
	}

	// Disk utilization
	var diskUtilizationPct float64
	if diskCapacityGB > 0 {
		diskUtilizationPct = float64(totalAppDiskGB) / float64(diskCapacityGB) * 100
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

	// TPS estimation
	estimatedTPS, tpsStatus := EstimateTPS(cellCount, tpsCurve)

	return models.ScenarioResult{
		CellCount:          cellCount,
		CellMemoryGB:       cellMemoryGB,
		CellCPU:            cellCPU,
		CellDiskGB:         cellDiskGB,
		AppCapacityGB:      appCapacityGB,
		DiskCapacityGB:     diskCapacityGB,
		UtilizationPct:     utilizationPct,
		DiskUtilizationPct: diskUtilizationPct,
		FreeChunks:         freeChunks,
		N1UtilizationPct:   n1UtilizationPct,
		FaultImpact:        faultImpact,
		InstancesPerCell:   instancesPerCell,
		EstimatedTPS:       estimatedTPS,
		TPSStatus:          tpsStatus,
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

	// Disk utilization warnings
	if proposed.DiskUtilizationPct > 90 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Disk utilization critically high",
		})
	} else if proposed.DiskUtilizationPct > 80 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Disk utilization elevated",
		})
	}

	// TPS degradation warnings
	switch proposed.TPSStatus {
	case "critical":
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  fmt.Sprintf("Cell count (%d) causes severe scheduling degradation (~%d TPS)", proposed.CellCount, proposed.EstimatedTPS),
		})
	case "degraded":
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  fmt.Sprintf("Cell count (%d) may cause scheduling latency (~%d TPS)", proposed.CellCount, proposed.EstimatedTPS),
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
	diskCapacityChange := proposed.DiskCapacityGB - current.DiskCapacityGB
	utilizationChange := proposed.UtilizationPct - current.UtilizationPct
	diskUtilizationChange := proposed.DiskUtilizationPct - current.DiskUtilizationPct

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
			CapacityChangeGB:         capacityChange,
			DiskCapacityChangeGB:     diskCapacityChange,
			UtilizationChangePct:     utilizationChange,
			DiskUtilizationChangePct: diskUtilizationChange,
			RedundancyChange:         redundancyChange,
		},
	}
}
