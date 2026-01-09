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

// EstimateTPS estimates TPS for a given cell count using the provided curve.
// If curve is nil/empty, returns 0 and "disabled" (TPS modeling is disabled).
// The frontend must explicitly enable TPS by providing a curve.
func EstimateTPS(cellCount int, curve []models.TPSPt) (tps int, status string) {
	if cellCount <= 0 {
		return 0, "unknown"
	}

	// Skip TPS calculation if no curve provided (TPS disabled)
	if len(curve) == 0 {
		return 0, "disabled"
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

// CalculateConstraints computes both HA Admission Control and N-X constraints
// and determines which is more restrictive.
func CalculateConstraints(totalMemoryGB, hostCount, memoryPerHostGB, haAdmissionPct, usedMemoryGB int) *models.ConstraintAnalysis {
	if hostCount == 0 || memoryPerHostGB == 0 {
		return nil
	}

	// HA Admission Control constraint
	haReservedPct := float64(haAdmissionPct)
	haReservedGB := int(float64(totalMemoryGB) * haReservedPct / 100)
	haUsableGB := totalMemoryGB - haReservedGB
	// N-equivalent: how many hosts worth of capacity does HA% reserve?
	haNEquivalent := 0
	if memoryPerHostGB > 0 {
		haNEquivalent = int(math.Ceil(float64(haReservedGB) / float64(memoryPerHostGB)))
	}

	// N-1 constraint (simple: reserve one host's worth)
	n1ReservedGB := memoryPerHostGB
	n1UsableGB := totalMemoryGB - n1ReservedGB
	n1ReservedPct := 0.0
	if totalMemoryGB > 0 {
		n1ReservedPct = float64(n1ReservedGB) / float64(totalMemoryGB) * 100
	}

	// Calculate utilizations
	haUtil := 0.0
	if haUsableGB > 0 {
		haUtil = float64(usedMemoryGB) / float64(haUsableGB) * 100
	}
	n1Util := 0.0
	if n1UsableGB > 0 {
		n1Util = float64(usedMemoryGB) / float64(n1UsableGB) * 100
	}

	// Determine which is more restrictive (less usable = more restrictive)
	haIsLimiting := haUsableGB <= n1UsableGB

	// Check if HA% provides insufficient protection for N-1
	insufficientHA := haReservedGB < n1ReservedGB

	// Build limiting label
	var limitingLabel, limitingType string
	if haIsLimiting {
		limitingType = "ha_admission"
		limitingLabel = fmt.Sprintf("HA %d%% (≈N-%d)", haAdmissionPct, haNEquivalent)
	} else {
		limitingType = "n_minus_x"
		limitingLabel = "N-1"
	}

	return &models.ConstraintAnalysis{
		HAAdmission: models.CapacityConstraint{
			Type:           "ha_admission",
			ReservedGB:     haReservedGB,
			ReservedPct:    haReservedPct,
			UsableGB:       haUsableGB,
			NEquivalent:    haNEquivalent,
			IsLimiting:     haIsLimiting,
			UtilizationPct: haUtil,
		},
		NMinusX: models.CapacityConstraint{
			Type:           "n_minus_x",
			ReservedGB:     n1ReservedGB,
			ReservedPct:    n1ReservedPct,
			UsableGB:       n1UsableGB,
			NEquivalent:    1,
			IsLimiting:     !haIsLimiting,
			UtilizationPct: n1Util,
		},
		LimitingConstraint:    limitingType,
		LimitingLabel:         limitingLabel,
		InsufficientHAWarning: insufficientHA,
	}
}

// CalculateCurrent computes metrics for the current configuration.
// tpsCurve is optional - if nil, TPS modeling is disabled.
func (c *ScenarioCalculator) CalculateCurrent(state models.InfrastructureState, tpsCurve []models.TPSPt) models.ScenarioResult {
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
		tpsCurve,
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

	// Blast radius: % of capacity lost per single cell failure
	var blastRadiusPct float64
	if cellCount > 0 {
		blastRadiusPct = 100.0 / float64(cellCount)
	}

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
		BlastRadiusPct:     blastRadiusPct,
	}
}

// WarningsContext provides context for generating actionable warnings
type WarningsContext struct {
	State   models.InfrastructureState
	Input   models.ScenarioInput
	Changes []models.ConfigChange
}

// findRelevantChange finds the most relevant ConfigChange for a warning type
func findRelevantChange(changes []models.ConfigChange, preferredFields ...string) *models.ConfigChange {
	// First try to find a change matching preferred fields
	for _, field := range preferredFields {
		for i := range changes {
			if changes[i].Field == field {
				return &changes[i]
			}
		}
	}
	// Fall back to first change if any
	if len(changes) > 0 {
		return &changes[0]
	}
	return nil
}

// GenerateWarnings produces warnings based on proposed scenario.
// The constraints parameter is optional - if provided, the warning messages
// will reflect whether HA Admission Control or N-1 is the limiting factor.
// The ctx parameter is optional - if provided, warnings will include change
// context and fix suggestions.
func (c *ScenarioCalculator) GenerateWarnings(current, proposed models.ScenarioResult, constraints *models.ConstraintAnalysis, ctx *WarningsContext) []models.ScenarioWarning {
	var warnings []models.ScenarioWarning

	// Determine which constraint is limiting for the warning message
	isHALimiting := constraints != nil && constraints.LimitingConstraint == "ha_admission"

	// Capacity utilization warnings - message depends on limiting constraint
	if proposed.N1UtilizationPct > 85 {
		var message string
		if isHALimiting {
			message = fmt.Sprintf("Exceeds HA Admission Control capacity limit (%s)", constraints.LimitingLabel)
		} else {
			message = "Exceeds N-1 capacity safety margin"
		}
		warning := models.ScenarioWarning{
			Severity: "critical",
			Message:  message,
		}
		// Add context if available
		if ctx != nil {
			warning.Change = findRelevantChange(ctx.Changes, "cell_count", "cell_memory_gb")
			warning.Fixes = CalculateCapacityFix(ctx.State, ctx.Input, constraints)
		}
		warnings = append(warnings, warning)
	} else if proposed.N1UtilizationPct > 75 {
		var message string
		if isHALimiting {
			message = fmt.Sprintf("Approaching HA Admission Control capacity limit (%s)", constraints.LimitingLabel)
		} else {
			message = "Approaching N-1 capacity limits"
		}
		warning := models.ScenarioWarning{
			Severity: "warning",
			Message:  message,
		}
		// Add context if available
		if ctx != nil {
			warning.Change = findRelevantChange(ctx.Changes, "cell_count", "cell_memory_gb")
			warning.Fixes = CalculateCapacityFix(ctx.State, ctx.Input, constraints)
		}
		warnings = append(warnings, warning)
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

	// Blast radius warning: warn when single cell failure impact is high
	// Thresholds: >20% is critical (5 or fewer cells), >10% is warning (10 or fewer cells)
	if proposed.BlastRadiusPct > 20 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  fmt.Sprintf("High cell failure impact: single cell loss affects %.0f%% of capacity", proposed.BlastRadiusPct),
		})
	} else if proposed.BlastRadiusPct > 10 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  fmt.Sprintf("Elevated cell failure impact: single cell loss affects %.0f%% of capacity", proposed.BlastRadiusPct),
		})
	}

	return warnings
}

// Compare computes full comparison between current and proposed scenarios
func (c *ScenarioCalculator) Compare(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioComparison {
	// Use same TPS curve for both current and proposed (if provided)
	current := c.CalculateCurrent(state, input.TPSCurve)
	proposed := c.CalculateProposed(state, input)

	// Calculate constraint analysis FIRST if host config is provided
	// This is needed before generating warnings so we know which constraint is limiting
	var constraints *models.ConstraintAnalysis
	if input.HostCount > 0 && input.MemoryPerHostGB > 0 {
		totalMemoryGB := input.HostCount * input.MemoryPerHostGB
		// Used memory: proposed cell memory + platform VMs
		usedMemoryGB := proposed.CellCount*proposed.CellMemoryGB + state.PlatformVMsGB

		constraints = CalculateConstraints(
			totalMemoryGB,
			input.HostCount,
			input.MemoryPerHostGB,
			input.HAAdmissionPct,
			usedMemoryGB,
		)
	}

	// Detect what changed between current and proposed
	changes := DetectChanges(state, input)

	// Build context for actionable warnings
	ctx := &WarningsContext{
		State:   state,
		Input:   input,
		Changes: changes,
	}

	// Generate warnings - pass constraints and context for actionable messages
	warnings := c.GenerateWarnings(current, proposed, constraints, ctx)

	// Add warning if HA% is insufficient for N-1 protection
	if constraints != nil && constraints.InsufficientHAWarning {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message: fmt.Sprintf(
				"HA Admission Control (%d%%) may be insufficient for N-1 host failure protection. Consider increasing to at least %.0f%%.",
				input.HAAdmissionPct,
				constraints.NMinusX.ReservedPct,
			),
		})
	}

	// Calculate delta
	capacityChange := proposed.AppCapacityGB - current.AppCapacityGB
	diskCapacityChange := proposed.DiskCapacityGB - current.DiskCapacityGB
	utilizationChange := proposed.UtilizationPct - current.UtilizationPct
	diskUtilizationChange := proposed.DiskUtilizationPct - current.DiskUtilizationPct

	// ResilienceChange based on blast radius: what % of capacity is at risk per cell failure
	// "low" = ≤5% blast radius (20+ cells), very resilient
	// "moderate" = 5-15% blast radius (7-20 cells), acceptable for most workloads
	// "high" = >15% blast radius (< 7 cells), concerning for production
	var resilienceChange string
	switch {
	case proposed.BlastRadiusPct <= 5:
		resilienceChange = "low"
	case proposed.BlastRadiusPct <= 15:
		resilienceChange = "moderate"
	default:
		resilienceChange = "high"
	}

	return models.ScenarioComparison{
		Current:     current,
		Proposed:    proposed,
		Warnings:    warnings,
		Constraints: constraints,
		Delta: models.ScenarioDelta{
			CapacityChangeGB:         capacityChange,
			DiskCapacityChangeGB:     diskCapacityChange,
			UtilizationChangePct:     utilizationChange,
			DiskUtilizationChangePct: diskUtilizationChange,
			ResilienceChange:         resilienceChange,
		},
	}
}

// DetectChanges identifies which configuration values were modified between
// the current state and the proposed input. Returns a slice of ConfigChange
// describing each modification with its delta and percentage change.
func DetectChanges(state models.InfrastructureState, input models.ScenarioInput) []models.ConfigChange {
	var changes []models.ConfigChange

	// Get current cell config from first cluster (assumes uniform cells)
	var currentCellMemory, currentCellCPU, currentCellDisk int
	for _, cluster := range state.Clusters {
		if cluster.DiegoCellMemoryGB > 0 {
			currentCellMemory = cluster.DiegoCellMemoryGB
			currentCellCPU = cluster.DiegoCellCPU
			currentCellDisk = cluster.DiegoCellDiskGB
			break
		}
	}
	currentCellCount := state.TotalCellCount

	// Detect cell count change
	if input.ProposedCellCount != currentCellCount && currentCellCount > 0 {
		delta := input.ProposedCellCount - currentCellCount
		deltaPct := float64(delta) / float64(currentCellCount) * 100
		changes = append(changes, models.ConfigChange{
			Field:       "cell_count",
			PreviousVal: currentCellCount,
			ProposedVal: input.ProposedCellCount,
			Delta:       delta,
			DeltaPct:    deltaPct,
		})
	}

	// Detect cell memory change
	if input.ProposedCellMemoryGB != currentCellMemory && currentCellMemory > 0 {
		delta := input.ProposedCellMemoryGB - currentCellMemory
		deltaPct := float64(delta) / float64(currentCellMemory) * 100
		changes = append(changes, models.ConfigChange{
			Field:       "cell_memory_gb",
			PreviousVal: currentCellMemory,
			ProposedVal: input.ProposedCellMemoryGB,
			Delta:       delta,
			DeltaPct:    deltaPct,
		})
	}

	// Detect cell CPU change
	if input.ProposedCellCPU != currentCellCPU && currentCellCPU > 0 {
		delta := input.ProposedCellCPU - currentCellCPU
		deltaPct := float64(delta) / float64(currentCellCPU) * 100
		changes = append(changes, models.ConfigChange{
			Field:       "cell_cpu",
			PreviousVal: currentCellCPU,
			ProposedVal: input.ProposedCellCPU,
			Delta:       delta,
			DeltaPct:    deltaPct,
		})
	}

	// Detect cell disk change
	if input.ProposedCellDiskGB != currentCellDisk && currentCellDisk > 0 {
		delta := input.ProposedCellDiskGB - currentCellDisk
		deltaPct := float64(delta) / float64(currentCellDisk) * 100
		changes = append(changes, models.ConfigChange{
			Field:       "cell_disk_gb",
			PreviousVal: currentCellDisk,
			ProposedVal: input.ProposedCellDiskGB,
			Delta:       delta,
			DeltaPct:    deltaPct,
		})
	}

	return changes
}

// CalculateCapacityFix calculates fix suggestions for N-1/HA capacity warnings.
// It suggests reducing cell count to achieve 84% utilization, or adding hosts.
// Returns at most 2 fix suggestions.
func CalculateCapacityFix(state models.InfrastructureState, input models.ScenarioInput, constraints *models.ConstraintAnalysis) []models.FixSuggestion {
	var fixes []models.FixSuggestion

	// Determine usable capacity based on which constraint is limiting
	usableGB := state.TotalN1MemoryGB
	if constraints != nil && constraints.LimitingConstraint == "ha_admission" {
		usableGB = constraints.HAAdmission.UsableGB
	}

	if usableGB == 0 || input.ProposedCellMemoryGB == 0 {
		return fixes
	}

	// Fix 1: Reduce cell count to achieve 84% utilization
	// Formula: targetCells = (targetUtil * usableGB - platformVMs) / cellMemory
	targetUtil := 0.84
	targetCellMemoryGB := int(float64(usableGB)*targetUtil) - state.PlatformVMsGB
	if targetCellMemoryGB > 0 {
		targetCells := targetCellMemoryGB / input.ProposedCellMemoryGB
		if targetCells > 0 && targetCells < input.ProposedCellCount {
			fixes = append(fixes, models.FixSuggestion{
				Description: fmt.Sprintf("Reduce to %d cells to achieve 84%% capacity utilization", targetCells),
				Field:       "cell_count",
				Value:       targetCells,
			})
		}
	}

	// Fix 2: Add hosts (only if host config is provided)
	if input.HostCount > 0 && input.MemoryPerHostGB > 0 {
		// Calculate how many hosts needed for proposed cells at 84% utilization
		proposedCellMemoryGB := input.ProposedCellCount * input.ProposedCellMemoryGB
		totalNeededGB := proposedCellMemoryGB + state.PlatformVMsGB

		// At 84% utilization: totalNeeded / usableCapacity = 0.84
		// usable = totalNeeded / 0.84
		usableNeededGB := float64(totalNeededGB) / targetUtil

		// For N-1: usable = (hosts - 1) * memPerHost
		// hosts = usable / memPerHost + 1
		hostsNeeded := int(math.Ceil(usableNeededGB/float64(input.MemoryPerHostGB))) + 1
		hostsToAdd := hostsNeeded - input.HostCount

		if hostsToAdd > 0 {
			fixes = append(fixes, models.FixSuggestion{
				Description: fmt.Sprintf("Add %d host(s) to support %d cells", hostsToAdd, input.ProposedCellCount),
				Field:       "host_count",
				Value:       hostsNeeded,
			})
		}
	}

	// Return at most 2 fixes
	if len(fixes) > 2 {
		fixes = fixes[:2]
	}

	return fixes
}
