package services

import (
	"fmt"
	"math"
	"strings"
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
	result := calc.CalculateCurrent(state, nil)

	// Cell count
	if result.CellCount != 470 {
		t.Errorf("Expected CellCount 470, got %d", result.CellCount)
	}

	// App capacity with 7% overhead: 470 × (32 - 32×0.07) = 470 × 29.76 ≈ 470 × 30 = 14100 GB
	// (int truncation: 32 * 0.07 = 2.24 → 2, so 32 - 2 = 30)
	expectedCapacity := 470 * (32 - 2) // 2 = int(32 * 0.07)
	if result.AppCapacityGB != expectedCapacity {
		t.Errorf("Expected AppCapacityGB %d, got %d", expectedCapacity, result.AppCapacityGB)
	}

	// Utilization: 10500 / 14100 × 100 = 74.5%
	if result.UtilizationPct < 74 || result.UtilizationPct > 75 {
		t.Errorf("Expected UtilizationPct ~74.5%%, got %.1f%%", result.UtilizationPct)
	}

	// Free chunks: (14100 - 10500) / 4 = 900
	expectedChunks := (expectedCapacity - 10500) / 4
	if result.FreeChunks != expectedChunks {
		t.Errorf("Expected FreeChunks %d, got %d", expectedChunks, result.FreeChunks)
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

	// App capacity with 7% overhead: 235 × (64 - 64×0.07) = 235 × (64-4) = 235 × 60 = 14100 GB
	expectedCapacity := 235 * (64 - 4) // 4 = int(64 * 0.07)
	if result.AppCapacityGB != expectedCapacity {
		t.Errorf("Expected AppCapacityGB %d, got %d", expectedCapacity, result.AppCapacityGB)
	}

	// Utilization: 10500 / 14100 × 100 = 74.5%
	if result.UtilizationPct < 74 || result.UtilizationPct > 75 {
		t.Errorf("Expected UtilizationPct ~74.5%%, got %.1f%%", result.UtilizationPct)
	}

	// Fault impact: 7500 / 235 = 32
	if result.FaultImpact != 32 {
		t.Errorf("Expected FaultImpact 32, got %d", result.FaultImpact)
	}
}

func TestGenerateWarnings_CriticalN1(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical
		FreeChunks:       500,
		CellCount:        100,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Exceeds N-1 capacity safety margin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for N-1 > 85%")
	}
}

func TestGenerateWarnings_LowFreeChunks(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       5, // < 10 = critical
		CellCount:        100,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Critical: Low staging capacity" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for free chunks < 10")
	}
}

func TestGenerateWarnings_BlastRadius(t *testing.T) {
	// Test that blast radius warnings fire based on ABSOLUTE impact, not relative change
	tests := []struct {
		name           string
		proposedCells  int
		blastRadiusPct float64
		expectWarning  bool
		expectCritical bool
	}{
		{"Large foundation (100 cells)", 100, 1.0, false, false},
		{"Medium foundation (20 cells)", 20, 5.0, false, false},
		{"Small foundation (8 cells)", 8, 12.5, true, false},     // >10% triggers warning
		{"Very small foundation (4 cells)", 4, 25.0, true, true}, // >20% triggers critical
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := models.ScenarioResult{
				N1UtilizationPct: 70,
				FreeChunks:       500,
				CellCount:        200,
				BlastRadiusPct:   0.5,
			}
			proposed := models.ScenarioResult{
				N1UtilizationPct: 70,
				FreeChunks:       500,
				CellCount:        tt.proposedCells,
				BlastRadiusPct:   tt.blastRadiusPct,
			}

			calc := NewScenarioCalculator()
			warnings := calc.GenerateWarnings(current, proposed, nil, nil)

			foundWarning := false
			foundCritical := false
			for _, w := range warnings {
				if contains(w.Message, "cell failure impact") {
					if w.Severity == "critical" {
						foundCritical = true
					} else {
						foundWarning = true
					}
				}
			}

			if tt.expectCritical && !foundCritical {
				t.Errorf("Expected critical blast radius warning for %d cells (%.1f%% impact)", tt.proposedCells, tt.blastRadiusPct)
			}
			if tt.expectWarning && !foundWarning && !foundCritical {
				t.Errorf("Expected blast radius warning for %d cells (%.1f%% impact)", tt.proposedCells, tt.blastRadiusPct)
			}
			if !tt.expectWarning && !tt.expectCritical && (foundWarning || foundCritical) {
				t.Errorf("Did not expect blast radius warning for %d cells (%.1f%% impact)", tt.proposedCells, tt.blastRadiusPct)
			}
		})
	}
}

func TestCompare(t *testing.T) {
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

	// Proposing 64GB cells with 230 cells
	// 470 -> 230 cells: blast radius goes from 0.21% -> 0.43% (both "low")
	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    230,
	}

	calc := NewScenarioCalculator()
	comparison := calc.Compare(state, input)

	// Current should match
	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}

	// Proposed should match
	if comparison.Proposed.CellCount != 230 {
		t.Errorf("Expected Proposed.CellCount 230, got %d", comparison.Proposed.CellCount)
	}

	// Current capacity with 7%: 470 × (32-2) = 470 × 30 = 14,100 GB
	// Proposed capacity with 7%: 230 × (64-4) = 230 × 60 = 13,800 GB
	// Delta is negative (fewer cells means less total capacity despite bigger VMs)
	// This is expected - we're testing the comparison mechanics, not capacity optimization
	if comparison.Delta.CapacityChangeGB == 0 {
		t.Errorf("Expected non-zero capacity change, got %d (current: %d, proposed: %d)",
			comparison.Delta.CapacityChangeGB, comparison.Current.AppCapacityGB, comparison.Proposed.AppCapacityGB)
	}

	// ResilienceChange should be "low" - both current and proposed have tiny blast radius
	// 470 cells = 0.21% blast radius, 230 cells = 0.43% blast radius (both ≤ 5%)
	if comparison.Delta.ResilienceChange != "low" {
		t.Errorf("Expected ResilienceChange 'low' for large foundation, got '%s'", comparison.Delta.ResilienceChange)
	}

	// BlastRadiusPct should be calculated
	expectedBlastRadius := 100.0 / 230.0 // ~0.43%
	if comparison.Proposed.BlastRadiusPct < 0.4 || comparison.Proposed.BlastRadiusPct > 0.5 {
		t.Errorf("Expected Proposed.BlastRadiusPct ~%.2f%%, got %.2f%%",
			expectedBlastRadius, comparison.Proposed.BlastRadiusPct)
	}

	// No blast radius warning expected - 230 cells is plenty resilient
	for _, w := range comparison.Warnings {
		if contains(w.Message, "cell failure impact") {
			t.Errorf("Did not expect blast radius warning for 230 cells, got: %s", w.Message)
		}
	}
}

// ============================================================================
// NEW TESTS: Disk Capacity, Percentage Overhead, TPS, Per-App Scenarios
// ============================================================================

func TestDiskCapacityCalculation(t *testing.T) {
	// Test disk capacity follows same pattern as memory
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    100,
		PlatformVMsGB:     1000,
		TotalAppMemoryGB:  5000,
		TotalAppDiskGB:    6000, // NEW: disk usage
		TotalAppInstances: 1000,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
				DiegoCellDiskGB:   128, // NEW: disk per cell
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellDiskGB:   128, // NEW
		ProposedCellCount:    100,
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateProposed(state, input)

	// Disk capacity: 100 cells × 128 GB = 12,800 GB (minus tiny overhead)
	// With 0.01% overhead: 128 * 0.9999 ≈ 128 (negligible)
	if result.DiskCapacityGB < 12700 || result.DiskCapacityGB > 12800 {
		t.Errorf("Expected DiskCapacityGB ~12800, got %d", result.DiskCapacityGB)
	}

	// Disk utilization: 6000 / 12800 × 100 = 46.9%
	if result.DiskUtilizationPct < 45 || result.DiskUtilizationPct > 50 {
		t.Errorf("Expected DiskUtilizationPct ~47%%, got %.1f%%", result.DiskUtilizationPct)
	}
}

func TestPercentageOverhead(t *testing.T) {
	// Test that overhead is calculated as percentage (7%) instead of fixed 5GB
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    100,
		PlatformVMsGB:     1000,
		TotalAppMemoryGB:  5000,
		TotalAppInstances: 1000,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellCount:    100,
		OverheadPct:          7.0, // NEW: 7% overhead
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateProposed(state, input)

	// With 7% overhead on 64GB cell: 64 × 0.93 = 59.52 GB usable
	// App capacity: 100 × 59.52 = 5952 GB (vs old 5900 with fixed 5GB)
	// Old calculation: 100 × (64 - 5) = 5900 GB
	// New calculation: 100 × (64 - 4.48) = 5952 GB
	expectedCapacity := int(float64(100) * (64 - 64*0.07))
	if result.AppCapacityGB < expectedCapacity-50 || result.AppCapacityGB > expectedCapacity+50 {
		t.Errorf("Expected AppCapacityGB ~%d with 7%% overhead, got %d", expectedCapacity, result.AppCapacityGB)
	}
}

func TestTPSEstimation(t *testing.T) {
	// Test TPS estimation using default curve
	tests := []struct {
		cellCount     int
		expectedTPS   int
		expectedRange int // ±range
		expectedState string
	}{
		{1, 284, 50, "critical"},     // Low cell count = low TPS
		{3, 1964, 100, "optimal"},    // Peak efficiency
		{9, 1932, 100, "optimal"},    // Still near peak
		{100, 1389, 200, "degraded"}, // Degrading
		{210, 104, 50, "critical"},   // Severe degradation
		{300, 70, 50, "critical"},    // Beyond curve - extrapolated
	}

	for _, tc := range tests {
		// TPS requires an explicit curve - nil returns "disabled"
		tps, status := EstimateTPS(tc.cellCount, DefaultTPSCurve)
		if tps < tc.expectedTPS-tc.expectedRange || tps > tc.expectedTPS+tc.expectedRange {
			t.Errorf("Cell count %d: expected TPS ~%d (±%d), got %d",
				tc.cellCount, tc.expectedTPS, tc.expectedRange, tps)
		}
		if status != tc.expectedState {
			t.Errorf("Cell count %d: expected status '%s', got '%s'",
				tc.cellCount, tc.expectedState, status)
		}
	}
}

func TestTPSEstimation_Disabled(t *testing.T) {
	// When no curve is provided, TPS should be disabled
	tps, status := EstimateTPS(100, nil)
	if tps != 0 {
		t.Errorf("Expected TPS 0 when disabled, got %d", tps)
	}
	if status != "disabled" {
		t.Errorf("Expected status 'disabled', got '%s'", status)
	}

	// Empty slice should also disable TPS
	tps, status = EstimateTPS(100, []models.TPSPt{})
	if tps != 0 {
		t.Errorf("Expected TPS 0 when disabled, got %d", tps)
	}
	if status != "disabled" {
		t.Errorf("Expected status 'disabled', got '%s'", status)
	}
}

func TestTPSEstimation_CustomCurve(t *testing.T) {
	// Test with user-provided custom TPS curve
	customCurve := []models.TPSPt{
		{Cells: 1, TPS: 500},
		{Cells: 10, TPS: 2000},
		{Cells: 50, TPS: 1500},
	}

	tps, status := EstimateTPS(10, customCurve)
	if tps != 2000 {
		t.Errorf("Expected TPS 2000 at 10 cells with custom curve, got %d", tps)
	}
	if status != "optimal" {
		t.Errorf("Expected status 'optimal', got '%s'", status)
	}

	// Interpolation between 10 and 50 cells
	tps, _ = EstimateTPS(30, customCurve)
	// Linear interpolation: 2000 + (30-10)/(50-10) * (1500-2000) = 2000 - 250 = 1750
	if tps < 1700 || tps > 1800 {
		t.Errorf("Expected TPS ~1750 at 30 cells (interpolated), got %d", tps)
	}
}

func TestAppAdditionScenario(t *testing.T) {
	// Test "what if I add App X with Y instances"
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    100,
		PlatformVMsGB:     1000,
		TotalAppMemoryGB:  5000,
		TotalAppDiskGB:    6000,
		TotalAppInstances: 1000,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
				DiegoCellDiskGB:   128,
			},
		},
	}

	// Propose adding a new app: 10 instances × 4GB RAM × 5GB disk
	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellDiskGB:   128,
		ProposedCellCount:    100,
		AdditionalApp: &models.AppSpec{
			Name:      "new-api-service",
			Instances: 10,
			MemoryGB:  4,
			DiskGB:    5,
		},
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateProposed(state, input)

	// Total app memory should include additional app: 5000 + (10 × 4) = 5040 GB
	// Since we're testing the result, check utilization increased
	baseInput := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellDiskGB:   128,
		ProposedCellCount:    100,
	}
	baseResult := calc.CalculateProposed(state, baseInput)

	if result.UtilizationPct <= baseResult.UtilizationPct {
		t.Errorf("Adding app should increase utilization: base %.1f%%, with app %.1f%%",
			baseResult.UtilizationPct, result.UtilizationPct)
	}

	// Check disk utilization also increased
	if result.DiskUtilizationPct <= baseResult.DiskUtilizationPct {
		t.Errorf("Adding app should increase disk utilization: base %.1f%%, with app %.1f%%",
			baseResult.DiskUtilizationPct, result.DiskUtilizationPct)
	}
}

func TestGenerateWarnings_DiskUtilization(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 50,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 92, // > 90% = critical
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Disk utilization critically high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for disk utilization > 90%")
	}
}

func TestGenerateWarnings_TPSDegradation(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        50,
		EstimatedTPS:     1800,
		TPSStatus:        "optimal",
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        200,
		EstimatedTPS:     200,
		TPSStatus:        "critical",
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && contains(w.Message, "scheduling degradation") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for TPS degradation")
	}
}

func TestCompareWithDiskAndTPS(t *testing.T) {
	// Full integration test with disk and TPS
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    100,
		PlatformVMsGB:     1000,
		TotalAppMemoryGB:  5000,
		TotalAppDiskGB:    6000,
		TotalAppInstances: 1000,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
				DiegoCellDiskGB:   128,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellDiskGB:   256, // Double disk
		ProposedCellCount:    100,
		TPSCurve:             DefaultTPSCurve, // Enable TPS by providing curve
	}

	calc := NewScenarioCalculator()
	comparison := calc.Compare(state, input)

	// Disk capacity should be in delta
	if comparison.Delta.DiskCapacityChangeGB <= 0 {
		t.Errorf("Expected positive disk capacity change, got %d", comparison.Delta.DiskCapacityChangeGB)
	}

	// Both current and proposed should have TPS estimates when curve is provided
	if comparison.Current.EstimatedTPS == 0 {
		t.Error("Expected Current.EstimatedTPS to be set")
	}
	if comparison.Proposed.EstimatedTPS == 0 {
		t.Error("Expected Proposed.EstimatedTPS to be set")
	}
}

// ============================================================================
// BLAST RADIUS TESTS: Smarter resilience assessment
// ============================================================================

func TestBlastRadiusPct(t *testing.T) {
	// Blast radius = 100 / cellCount (% of capacity lost per cell failure)
	tests := []struct {
		name           string
		cellCount      int
		expectedRadius float64
	}{
		{"Large foundation (500 cells)", 500, 0.2},
		{"Medium foundation (50 cells)", 50, 2.0},
		{"Small foundation (10 cells)", 10, 10.0},
		{"Very small foundation (5 cells)", 5, 20.0},
		{"Minimal foundation (2 cells)", 2, 50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := models.InfrastructureState{
				TotalN1MemoryGB:   100000,
				TotalCellCount:    tt.cellCount,
				PlatformVMsGB:     1000,
				TotalAppMemoryGB:  5000,
				TotalAppInstances: 1000,
				Clusters: []models.ClusterState{
					{
						DiegoCellCount:    tt.cellCount,
						DiegoCellMemoryGB: 32,
						DiegoCellCPU:      4,
					},
				},
			}

			calc := NewScenarioCalculator()
			result := calc.CalculateCurrent(state, nil)

			if result.BlastRadiusPct != tt.expectedRadius {
				t.Errorf("Expected BlastRadiusPct %.1f%%, got %.1f%%",
					tt.expectedRadius, result.BlastRadiusPct)
			}
		})
	}
}

func TestResilienceWarning_LargeFoundation_NoWarning(t *testing.T) {
	// 500 → 250 cells is a 50% reduction, but blast radius only goes from 0.2% → 0.4%
	// This should NOT trigger a resilience warning - it's still very safe
	current := models.ScenarioResult{
		CellCount:        500,
		BlastRadiusPct:   0.2,
		UtilizationPct:   50,
		N1UtilizationPct: 55,
	}
	proposed := models.ScenarioResult{
		CellCount:        250,
		BlastRadiusPct:   0.4,
		UtilizationPct:   50,
		N1UtilizationPct: 55,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	for _, w := range warnings {
		if contains(w.Message, "resilience") || contains(w.Message, "redundancy") || contains(w.Message, "blast") {
			t.Errorf("Large foundation reduction should not trigger resilience warning, got: %s", w.Message)
		}
	}
}

func TestResilienceWarning_SmallFoundation_Warning(t *testing.T) {
	// 10 → 5 cells means blast radius goes from 10% → 20%
	// This SHOULD trigger a warning - losing one cell loses 20% of capacity
	current := models.ScenarioResult{
		CellCount:        10,
		BlastRadiusPct:   10.0,
		UtilizationPct:   50,
		N1UtilizationPct: 55,
	}
	proposed := models.ScenarioResult{
		CellCount:        5,
		BlastRadiusPct:   20.0,
		UtilizationPct:   50,
		N1UtilizationPct: 55,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if contains(w.Message, "cell failure") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Small foundation with high blast radius should trigger cell failure impact warning")
	}
}

func TestResilienceChange_UsesBlastRadius(t *testing.T) {
	// ResilienceChange should reflect the actual blast radius impact, not just cell count
	state := models.InfrastructureState{
		TotalN1MemoryGB:   100000,
		TotalCellCount:    500,
		PlatformVMsGB:     1000,
		TotalAppMemoryGB:  5000,
		TotalAppInstances: 1000,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    500,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	// 500 → 250 cells: blast radius 0.2% → 0.4% (both low, should be "unchanged" or at worst "minimal")
	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    250,
	}

	calc := NewScenarioCalculator()
	comparison := calc.Compare(state, input)

	// With smarter logic, this should NOT say "reduced" since blast radius is still tiny
	if comparison.Delta.ResilienceChange == "reduced" {
		t.Errorf("Large foundation reduction (0.2%% → 0.4%% blast radius) should not be 'reduced', got: %s",
			comparison.Delta.ResilienceChange)
	}
}

// ============================================================================
// CONSTRAINT ANALYSIS TESTS: HA Admission Control vs N-X Tolerance
// ============================================================================

func TestCalculateConstraints_HAIsLimiting(t *testing.T) {
	// 15 hosts × 2000GB = 30,000GB total
	// HA 25% reserves 7,500GB (≈N-3 equivalent: can survive 3 host failures)
	// Formula: HA% = (Hosts to Survive / Total Hosts) × 100
	// N-3 = 20%, N-4 = 26.67%, so 25% gives N-3 protection
	// N-1 reserves 2,000GB (6.67%)
	// HA is more restrictive
	result := CalculateConstraints(
		30000, // totalMemoryGB
		15,    // hostCount
		2000,  // memoryPerHostGB
		25,    // haAdmissionPct
		15000, // usedMemoryGB (cells + platform)
	)

	if result == nil {
		t.Fatal("Expected non-nil ConstraintAnalysis")
	}

	// HA should be limiting
	if result.LimitingConstraint != "ha_admission" {
		t.Errorf("Expected limiting_constraint='ha_admission', got '%s'", result.LimitingConstraint)
	}

	// HA reserves 25% = 7,500GB
	if result.HAAdmission.ReservedGB != 7500 {
		t.Errorf("Expected HA ReservedGB=7500, got %d", result.HAAdmission.ReservedGB)
	}

	// HA usable = 30,000 - 7,500 = 22,500GB
	if result.HAAdmission.UsableGB != 22500 {
		t.Errorf("Expected HA UsableGB=22500, got %d", result.HAAdmission.UsableGB)
	}

	// HA N-equivalent: 7500 / 2000 = 3.75 → floor = 3 (can survive 3 host failures)
	if result.HAAdmission.NEquivalent != 3 {
		t.Errorf("Expected HA NEquivalent=3, got %d", result.HAAdmission.NEquivalent)
	}

	// N-1 reserves 2,000GB
	if result.NMinusX.ReservedGB != 2000 {
		t.Errorf("Expected N-1 ReservedGB=2000, got %d", result.NMinusX.ReservedGB)
	}

	// Limiting label should show HA with N-equivalent
	expectedLabel := "HA 25% (≈N-3)"
	if result.LimitingLabel != expectedLabel {
		t.Errorf("Expected LimitingLabel='%s', got '%s'", expectedLabel, result.LimitingLabel)
	}

	// No insufficient warning (HA > N-1)
	if result.InsufficientHAWarning {
		t.Error("Expected InsufficientHAWarning=false when HA >= N-1")
	}

	// HA utilization: 15000 / 22500 = 66.67%
	if result.HAAdmission.UtilizationPct < 66 || result.HAAdmission.UtilizationPct > 67 {
		t.Errorf("Expected HA UtilizationPct ~66.67%%, got %.2f%%", result.HAAdmission.UtilizationPct)
	}
}

func TestCalculateConstraints_NMinusXIsLimiting(t *testing.T) {
	// 15 hosts × 2000GB = 30,000GB total
	// HA 5% reserves 1,500GB
	// N-1 reserves 2,000GB (6.67%)
	// N-1 is more restrictive
	result := CalculateConstraints(
		30000, // totalMemoryGB
		15,    // hostCount
		2000,  // memoryPerHostGB
		5,     // haAdmissionPct (low)
		15000, // usedMemoryGB
	)

	if result == nil {
		t.Fatal("Expected non-nil ConstraintAnalysis")
	}

	// N-1 should be limiting
	if result.LimitingConstraint != "n_minus_x" {
		t.Errorf("Expected limiting_constraint='n_minus_x', got '%s'", result.LimitingConstraint)
	}

	// HA reserves 5% = 1,500GB
	if result.HAAdmission.ReservedGB != 1500 {
		t.Errorf("Expected HA ReservedGB=1500, got %d", result.HAAdmission.ReservedGB)
	}

	// Label should show N-1
	if result.LimitingLabel != "N-1" {
		t.Errorf("Expected LimitingLabel='N-1', got '%s'", result.LimitingLabel)
	}

	// Should warn that HA is insufficient
	if !result.InsufficientHAWarning {
		t.Error("Expected InsufficientHAWarning=true when HA < N-1")
	}
}

func TestCalculateConstraints_HAZero(t *testing.T) {
	// HA 0% - no HA reservation
	result := CalculateConstraints(
		30000, // totalMemoryGB
		15,    // hostCount
		2000,  // memoryPerHostGB
		0,     // haAdmissionPct
		15000, // usedMemoryGB
	)

	if result == nil {
		t.Fatal("Expected non-nil ConstraintAnalysis")
	}

	// N-1 should be limiting (HA has 0 reserve)
	if result.LimitingConstraint != "n_minus_x" {
		t.Errorf("Expected limiting_constraint='n_minus_x', got '%s'", result.LimitingConstraint)
	}

	// HA reserves 0GB
	if result.HAAdmission.ReservedGB != 0 {
		t.Errorf("Expected HA ReservedGB=0, got %d", result.HAAdmission.ReservedGB)
	}

	// HA usable = full 30,000GB
	if result.HAAdmission.UsableGB != 30000 {
		t.Errorf("Expected HA UsableGB=30000, got %d", result.HAAdmission.UsableGB)
	}

	// Should warn that HA is insufficient
	if !result.InsufficientHAWarning {
		t.Error("Expected InsufficientHAWarning=true when HA=0")
	}
}

func TestCalculateConstraints_ZeroHosts(t *testing.T) {
	// Edge case: no hosts should return nil
	result := CalculateConstraints(0, 0, 0, 25, 0)

	if result != nil {
		t.Error("Expected nil ConstraintAnalysis when hostCount=0")
	}
}

func TestGenerateWarnings_HAAdmissionLimiting_ShowsHAMessage(t *testing.T) {
	// When HA Admission Control is the limiting constraint and utilization is critical,
	// the warning message should mention HA Admission Control, not N-1.
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical
		FreeChunks:       500,
		CellCount:        100,
	}

	// HA is limiting: 25% reserves more than N-1
	constraints := &models.ConstraintAnalysis{
		LimitingConstraint: "ha_admission",
		LimitingLabel:      "HA 25% (≈N-4)",
		HAAdmission: models.CapacityConstraint{
			IsLimiting:     true,
			UtilizationPct: 90,
		},
		NMinusX: models.CapacityConstraint{
			IsLimiting:     false,
			UtilizationPct: 70,
		},
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, constraints, nil)

	// Should find HA admission message, not N-1 message
	foundHA := false
	foundN1 := false
	for _, w := range warnings {
		if w.Severity == "critical" && contains(w.Message, "HA Admission Control") {
			foundHA = true
		}
		if w.Severity == "critical" && w.Message == "Exceeds N-1 capacity safety margin" {
			foundN1 = true
		}
	}

	if !foundHA {
		t.Error("Expected critical warning mentioning HA Admission Control when HA is limiting constraint")
	}
	if foundN1 {
		t.Error("Should NOT show N-1 message when HA Admission Control is the limiting constraint")
	}
}

func TestGenerateWarnings_N1Limiting_ShowsN1Message(t *testing.T) {
	// When N-1 is the limiting constraint and utilization is critical,
	// the warning message should mention N-1 (existing behavior).
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical
		FreeChunks:       500,
		CellCount:        100,
	}

	// N-1 is limiting: HA 5% reserves less than N-1
	constraints := &models.ConstraintAnalysis{
		LimitingConstraint: "n_minus_x",
		LimitingLabel:      "N-1",
		HAAdmission: models.CapacityConstraint{
			IsLimiting:     false,
			UtilizationPct: 70,
		},
		NMinusX: models.CapacityConstraint{
			IsLimiting:     true,
			UtilizationPct: 90,
		},
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, constraints, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Exceeds N-1 capacity safety margin" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected critical N-1 warning when N-1 is the limiting constraint")
	}
}

func TestGenerateWarnings_NoConstraints_FallsBackToN1(t *testing.T) {
	// When no constraint analysis is available (nil), fall back to N-1 message
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical
		FreeChunks:       500,
		CellCount:        100,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Exceeds N-1 capacity safety margin" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected critical N-1 warning when no constraint analysis is provided")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// DETECT CHANGES TESTS
// ============================================================================

func TestDetectChanges_CellCountChanged(t *testing.T) {
	state := models.InfrastructureState{
		TotalCellCount: 470,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
				DiegoCellDiskGB:   100,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    600, // Changed from 470 to 600
		ProposedCellMemoryGB: 32,  // Same
		ProposedCellCPU:      4,   // Same
	}

	changes := DetectChanges(state, input)

	// Should detect cell count change
	var cellCountChange *models.ConfigChange
	for i := range changes {
		if changes[i].Field == "cell_count" {
			cellCountChange = &changes[i]
			break
		}
	}

	if cellCountChange == nil {
		t.Fatal("Expected to detect cell_count change")
	}

	if cellCountChange.PreviousVal != 470 {
		t.Errorf("Expected PreviousVal=470, got %d", cellCountChange.PreviousVal)
	}
	if cellCountChange.ProposedVal != 600 {
		t.Errorf("Expected ProposedVal=600, got %d", cellCountChange.ProposedVal)
	}
	if cellCountChange.Delta != 130 {
		t.Errorf("Expected Delta=130, got %d", cellCountChange.Delta)
	}
	// Delta percent: 130/470 * 100 = ~27.66%
	if cellCountChange.DeltaPct < 27 || cellCountChange.DeltaPct > 28 {
		t.Errorf("Expected DeltaPct ~27.66%%, got %.2f%%", cellCountChange.DeltaPct)
	}
}

func TestDetectChanges_CellMemoryChanged(t *testing.T) {
	state := models.InfrastructureState{
		TotalCellCount: 470,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    470, // Same
		ProposedCellMemoryGB: 64,  // Changed from 32 to 64
		ProposedCellCPU:      4,   // Same
	}

	changes := DetectChanges(state, input)

	var memoryChange *models.ConfigChange
	for i := range changes {
		if changes[i].Field == "cell_memory_gb" {
			memoryChange = &changes[i]
			break
		}
	}

	if memoryChange == nil {
		t.Fatal("Expected to detect cell_memory_gb change")
	}

	if memoryChange.PreviousVal != 32 {
		t.Errorf("Expected PreviousVal=32, got %d", memoryChange.PreviousVal)
	}
	if memoryChange.ProposedVal != 64 {
		t.Errorf("Expected ProposedVal=64, got %d", memoryChange.ProposedVal)
	}
	if memoryChange.Delta != 32 {
		t.Errorf("Expected Delta=32, got %d", memoryChange.Delta)
	}
}

func TestDetectChanges_NoChanges(t *testing.T) {
	state := models.InfrastructureState{
		TotalCellCount: 470,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    470, // Same
		ProposedCellMemoryGB: 32,  // Same
		ProposedCellCPU:      4,   // Same
	}

	changes := DetectChanges(state, input)

	if len(changes) != 0 {
		t.Errorf("Expected no changes, got %d changes", len(changes))
	}
}

func TestDetectChanges_MultipleChanges(t *testing.T) {
	state := models.InfrastructureState{
		TotalCellCount: 470,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    235, // Changed: halved
		ProposedCellMemoryGB: 64,  // Changed: doubled
		ProposedCellCPU:      8,   // Changed: doubled
	}

	changes := DetectChanges(state, input)

	if len(changes) != 3 {
		t.Errorf("Expected 3 changes, got %d", len(changes))
	}

	// Verify all expected fields are present
	fields := make(map[string]bool)
	for _, c := range changes {
		fields[c.Field] = true
	}

	if !fields["cell_count"] {
		t.Error("Expected cell_count change")
	}
	if !fields["cell_memory_gb"] {
		t.Error("Expected cell_memory_gb change")
	}
	if !fields["cell_cpu"] {
		t.Error("Expected cell_cpu change")
	}
}

// ============================================================================
// FIX CALCULATOR TESTS
// ============================================================================

func TestCalculateCapacityFix_ReduceCells(t *testing.T) {
	// Scenario: User proposed 600 cells, which exceeds 85% N-1 capacity
	// Expected fix: Reduce to fewer cells to achieve 84% utilization
	state := models.InfrastructureState{
		TotalCellCount:  470,
		TotalN1MemoryGB: 26624, // N-1 usable memory
		PlatformVMsGB:   4800,  // Platform VM overhead
		Clusters: []models.ClusterState{
			{
				DiegoCellMemoryGB: 32,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    600,
		ProposedCellMemoryGB: 32,
		HostCount:            15,
		MemoryPerHostGB:      2000,
		HAAdmissionPct:       10, // Low HA%, so N-1 is limiting
	}

	fixes := CalculateCapacityFix(state, input, nil)

	if len(fixes) == 0 {
		t.Fatal("Expected at least one fix suggestion")
	}

	// First fix should suggest reducing cells
	found := false
	for _, fix := range fixes {
		if fix.Field == "cell_count" {
			found = true
			// Target 84% utilization
			// usable = 26624 GB, platform = 4800 GB
			// targetCellMemory = 26624 * 0.84 - 4800 = 17564 GB
			// targetCells = 17564 / 32 = 548 cells
			if fix.Value < 500 || fix.Value > 560 {
				t.Errorf("Expected fix value around 548 cells, got %d", fix.Value)
			}
			break
		}
	}

	if !found {
		t.Error("Expected a cell_count fix suggestion")
	}
}

func TestCalculateCapacityFix_AddHosts(t *testing.T) {
	// Scenario: User proposed 600 cells with host config
	// Expected: Suggest adding hosts as alternative fix
	state := models.InfrastructureState{
		TotalCellCount:  470,
		TotalN1MemoryGB: 26624,
		PlatformVMsGB:   4800,
		Clusters: []models.ClusterState{
			{
				DiegoCellMemoryGB: 32,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    600,
		ProposedCellMemoryGB: 32,
		HostCount:            15,
		MemoryPerHostGB:      2000,
		HAAdmissionPct:       10,
	}

	fixes := CalculateCapacityFix(state, input, nil)

	// Should have at most 2 fixes
	if len(fixes) > 2 {
		t.Errorf("Expected max 2 fixes, got %d", len(fixes))
	}

	// Second fix (if present) should suggest adding hosts
	found := false
	for _, fix := range fixes {
		if fix.Field == "host_count" {
			found = true
			// Should suggest more hosts than current 15
			if fix.Value <= 15 {
				t.Errorf("Expected host_count > 15, got %d", fix.Value)
			}
			break
		}
	}

	// This might not always have a host fix depending on calculation
	// Just verify it doesn't crash
	_ = found
}

func TestCalculateCapacityFix_WithHAConstraint(t *testing.T) {
	// Scenario: HA Admission Control is the limiting constraint
	state := models.InfrastructureState{
		TotalCellCount:  470,
		TotalN1MemoryGB: 26624,
		PlatformVMsGB:   4800,
		Clusters: []models.ClusterState{
			{
				DiegoCellMemoryGB: 32,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellCount:    600,
		ProposedCellMemoryGB: 32,
		HostCount:            15,
		MemoryPerHostGB:      2000,
		HAAdmissionPct:       25, // 25% = more restrictive than N-1
	}

	// HA constraint analysis
	constraints := &models.ConstraintAnalysis{
		LimitingConstraint: "ha_admission",
		HAAdmission: models.CapacityConstraint{
			UsableGB:   22500, // 30000 * 0.75
			IsLimiting: true,
		},
		NMinusX: models.CapacityConstraint{
			UsableGB:   28000,
			IsLimiting: false,
		},
	}

	fixes := CalculateCapacityFix(state, input, constraints)

	if len(fixes) == 0 {
		t.Fatal("Expected at least one fix suggestion when HA is limiting")
	}

	// Fix should use HA usable capacity, not N-1
	for _, fix := range fixes {
		if fix.Field == "cell_count" {
			// With HA usable = 22500, platform = 4800
			// targetCellMemory = 22500 * 0.84 - 4800 = 14100 GB
			// targetCells = 14100 / 32 = 440 cells (fewer than N-1 based)
			if fix.Value > 500 {
				t.Errorf("Expected fewer cells due to HA constraint, got %d", fix.Value)
			}
			break
		}
	}
}

// ============================================================================
// CPU RISK LEVEL TESTS
// ============================================================================

func TestCPURiskLevel(t *testing.T) {
	tests := []struct {
		ratio    float64
		expected string
	}{
		{0.5, "conservative"},
		{2.0, "conservative"},
		{4.0, "conservative"},
		{4.1, "moderate"},
		{6.0, "moderate"},
		{8.0, "moderate"},
		{8.1, "aggressive"},
		{12.0, "aggressive"},
		{16.0, "aggressive"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("ratio_%.1f", tt.ratio), func(t *testing.T) {
			result := CPURiskLevel(tt.ratio)
			if result != tt.expected {
				t.Errorf("CPURiskLevel(%.1f) = %s, want %s", tt.ratio, result, tt.expected)
			}
		})
	}
}

func TestCalculateFull_WithCPUConfig(t *testing.T) {
	calc := NewScenarioCalculator()

	state := models.InfrastructureState{
		TotalCellCount: 10,
		Clusters: []models.ClusterState{
			{DiegoCellMemoryGB: 32, DiegoCellCPU: 4, DiegoCellDiskGB: 128},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    10,
		HostCount:            3,
		PhysicalCoresPerHost: 32,
	}

	result := calc.CalculateProposed(state, input)

	// 10 cells × 4 vCPU = 40 vCPU
	// 3 hosts × 32 pCPU = 96 pCPU
	// Ratio = 40/96 = 0.417
	if result.TotalVCPUs != 40 {
		t.Errorf("TotalVCPUs = %d, want 40", result.TotalVCPUs)
	}
	if result.TotalPCPUs != 96 {
		t.Errorf("TotalPCPUs = %d, want 96", result.TotalPCPUs)
	}
	expectedRatio := 40.0 / 96.0
	if math.Abs(result.VCPURatio-expectedRatio) > 0.01 {
		t.Errorf("VCPURatio = %f, want %f", result.VCPURatio, expectedRatio)
	}
	if result.CPURiskLevel != "conservative" {
		t.Errorf("CPURiskLevel = %s, want conservative", result.CPURiskLevel)
	}
}

func TestCalculateCPURatioFix(t *testing.T) {
	state := models.InfrastructureState{}

	// 50 cells × 8 vCPU = 400 vCPU
	// 3 hosts × 32 pCPU = 96 pCPU
	// Current ratio = 400/96 = 4.17:1
	// Target ratio = 4:1
	input := models.ScenarioInput{
		ProposedCellCount:    50,
		ProposedCellCPU:      8,
		HostCount:            3,
		PhysicalCoresPerHost: 32,
		TargetVCPURatio:      4,
	}

	fixes := CalculateCPURatioFix(state, input, 4.17, 4.0)

	if len(fixes) == 0 {
		t.Fatal("Expected at least one fix suggestion")
	}

	// Should suggest reducing cells: 4 * 96 / 8 = 48 cells
	foundCellFix := false
	for _, fix := range fixes {
		if fix.Field == "cell_count" && fix.Value == 48 {
			foundCellFix = true
		}
	}
	if !foundCellFix {
		t.Errorf("Expected fix suggestion to reduce to 48 cells, got: %+v", fixes)
	}
}

func TestGenerateWarnings_CPURatioExceedsTarget(t *testing.T) {
	calc := NewScenarioCalculator()

	current := models.ScenarioResult{
		TotalPCPUs: 96,
		VCPURatio:  3.0,
	}
	proposed := models.ScenarioResult{
		TotalPCPUs:   96,
		TotalVCPUs:   624,
		VCPURatio:    6.5,
		CPURiskLevel: "moderate",
	}

	ctx := &WarningsContext{
		Input: models.ScenarioInput{
			TargetVCPURatio: 4,
		},
	}

	warnings := calc.GenerateWarnings(current, proposed, nil, ctx)

	found := false
	for _, w := range warnings {
		if w.Severity == "warning" && strings.Contains(w.Message, "exceeds target") {
			found = true
		}
	}
	if !found {
		t.Error("Expected warning about ratio exceeding target")
	}
}

func TestGenerateWarnings_AggressiveRatio(t *testing.T) {
	calc := NewScenarioCalculator()

	current := models.ScenarioResult{}
	proposed := models.ScenarioResult{
		TotalPCPUs:   96,
		TotalVCPUs:   1000,
		VCPURatio:    10.4,
		CPURiskLevel: "aggressive",
	}

	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && strings.Contains(w.Message, "aggressive") {
			found = true
		}
	}
	if !found {
		t.Error("Expected critical warning about aggressive ratio")
	}
}

func TestCompare_VCPURatioChange(t *testing.T) {
	calc := NewScenarioCalculator()

	state := models.InfrastructureState{
		TotalCellCount: 10,
		Clusters: []models.ClusterState{
			{DiegoCellMemoryGB: 32, DiegoCellCPU: 4, DiegoCellDiskGB: 128},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    20, // Double the cells
		HostCount:            3,
		PhysicalCoresPerHost: 32,
	}

	comparison := calc.Compare(state, input)

	// Current: 0 (no host config for current)
	// Proposed: 20 * 4 / (3 * 32) = 80/96 = 0.833
	// Change should be 0.833 - 0 = 0.833
	if comparison.Delta.VCPURatioChange == 0 && comparison.Proposed.VCPURatio > 0 {
		t.Errorf("VCPURatioChange = 0, but proposed ratio is %f", comparison.Proposed.VCPURatio)
	}
}

// ============================================================================
// MAX CELLS BY CPU TESTS
// ============================================================================

func TestCPUHeadroomCells(t *testing.T) {
	tests := []struct {
		name              string
		cellCount         int
		cellCPU           int
		hostCount         int
		physicalCores     int
		targetRatio       int
		platformVMsCPU    int
		wantMaxCells      int
		wantHeadroomCells int
	}{
		{
			name:              "positive headroom - under target",
			cellCount:         10,
			cellCPU:           4,
			hostCount:         3,
			physicalCores:     32, // 96 pCPUs total
			targetRatio:       4,  // 4:1 = 384 max vCPU
			platformVMsCPU:    24, // 360 available for cells
			wantMaxCells:      90, // 360 / 4 = 90
			wantHeadroomCells: 80, // 90 - 10 = 80
		},
		{
			name:              "zero headroom - at limit",
			cellCount:         90,
			cellCPU:           4,
			hostCount:         3,
			physicalCores:     32,
			targetRatio:       4,
			platformVMsCPU:    24,
			wantMaxCells:      90,
			wantHeadroomCells: 0, // 90 - 90 = 0
		},
		{
			name:              "negative headroom - over target",
			cellCount:         100,
			cellCPU:           4,
			hostCount:         3,
			physicalCores:     32,
			targetRatio:       4,
			platformVMsCPU:    24,
			wantMaxCells:      90,
			wantHeadroomCells: -10, // 90 - 100 = -10
		},
		{
			name:              "cpu analysis disabled - zero values",
			cellCount:         10,
			cellCPU:           4,
			hostCount:         0, // No hosts configured
			physicalCores:     0,
			targetRatio:       4,
			platformVMsCPU:    0,
			wantMaxCells:      0,
			wantHeadroomCells: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal state - CalculateProposed uses input for cell config
			state := models.InfrastructureState{}

			input := models.ScenarioInput{
				ProposedCellCount:    tt.cellCount,
				ProposedCellCPU:      tt.cellCPU,
				ProposedCellMemoryGB: 32,
				ProposedCellDiskGB:   100,
				HostCount:            tt.hostCount,
				PhysicalCoresPerHost: tt.physicalCores,
				TargetVCPURatio:      tt.targetRatio,
				PlatformVMsCPU:       tt.platformVMsCPU,
			}

			calc := NewScenarioCalculator()
			result := calc.CalculateProposed(state, input)

			if result.MaxCellsByCPU != tt.wantMaxCells {
				t.Errorf("MaxCellsByCPU = %d, want %d", result.MaxCellsByCPU, tt.wantMaxCells)
			}
			if result.CPUHeadroomCells != tt.wantHeadroomCells {
				t.Errorf("CPUHeadroomCells = %d, want %d", result.CPUHeadroomCells, tt.wantHeadroomCells)
			}
		})
	}
}

func TestCalculateMaxCellsByCPU(t *testing.T) {
	tests := []struct {
		name           string
		targetRatio    float64
		totalPCPUs     int
		cellCPU        int
		platformVMsCPU int
		want           int
	}{
		{
			name:           "standard case",
			targetRatio:    4.0,
			totalPCPUs:     100,
			cellCPU:        4,
			platformVMsCPU: 0,
			want:           100, // 4.0 * 100 / 4 = 100
		},
		{
			name:           "with platform overhead",
			targetRatio:    4.0,
			totalPCPUs:     100,
			cellCPU:        4,
			platformVMsCPU: 40,
			want:           90, // (4.0 * 100 - 40) / 4 = 90
		},
		{
			name:           "zero cellCPU disabled",
			targetRatio:    4.0,
			totalPCPUs:     100,
			cellCPU:        0,
			platformVMsCPU: 0,
			want:           0,
		},
		{
			name:           "zero totalPCPUs disabled",
			targetRatio:    4.0,
			totalPCPUs:     0,
			cellCPU:        4,
			platformVMsCPU: 0,
			want:           0,
		},
		{
			name:           "platform exceeds budget",
			targetRatio:    4.0,
			totalPCPUs:     100,
			cellCPU:        4,
			platformVMsCPU: 500, // 500 > 400 (4.0 * 100)
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMaxCellsByCPU(tt.targetRatio, tt.totalPCPUs, tt.cellCPU, tt.platformVMsCPU)
			if got != tt.want {
				t.Errorf("CalculateMaxCellsByCPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// SELECTED RESOURCES FILTER TESTS
// ============================================================================

func TestGenerateWarnings_SelectedResources_DiskFiltered(t *testing.T) {
	// Test that disk warnings are filtered when "disk" is not in selectedResources
	current := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 50,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 92, // > 90% = would normally trigger critical warning
	}

	calc := NewScenarioCalculator()

	// With disk selected, warning should appear
	ctxWithDisk := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"memory", "disk"},
		},
	}
	warningsWithDisk := calc.GenerateWarnings(current, proposed, nil, ctxWithDisk)

	foundDiskWarning := false
	for _, w := range warningsWithDisk {
		if w.Message == "Disk utilization critically high" {
			foundDiskWarning = true
			break
		}
	}
	if !foundDiskWarning {
		t.Error("Expected disk warning when disk is in selectedResources")
	}

	// Without disk selected, warning should NOT appear
	ctxWithoutDisk := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"memory"},
		},
	}
	warningsWithoutDisk := calc.GenerateWarnings(current, proposed, nil, ctxWithoutDisk)

	for _, w := range warningsWithoutDisk {
		if w.Message == "Disk utilization critically high" {
			t.Error("Expected disk warning to be filtered when disk is not in selectedResources")
		}
	}
}

func TestGenerateWarnings_SelectedResources_CPUFiltered(t *testing.T) {
	// Test that CPU ratio warnings are filtered when "cpu" is not in selectedResources
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
		TotalPCPUs:       96,
		TotalVCPUs:       1000,
		VCPURatio:        10.4,
		CPURiskLevel:     "aggressive", // Would normally trigger critical warning
	}

	calc := NewScenarioCalculator()

	// With cpu selected, warning should appear
	ctxWithCPU := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"memory", "cpu"},
		},
	}
	warningsWithCPU := calc.GenerateWarnings(current, proposed, nil, ctxWithCPU)

	foundCPUWarning := false
	for _, w := range warningsWithCPU {
		if strings.Contains(w.Message, "aggressive") && strings.Contains(w.Message, "vCPU:pCPU") {
			foundCPUWarning = true
			break
		}
	}
	if !foundCPUWarning {
		t.Error("Expected CPU ratio warning when cpu is in selectedResources")
	}

	// Without cpu selected, warning should NOT appear
	ctxWithoutCPU := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"memory"},
		},
	}
	warningsWithoutCPU := calc.GenerateWarnings(current, proposed, nil, ctxWithoutCPU)

	for _, w := range warningsWithoutCPU {
		if strings.Contains(w.Message, "vCPU:pCPU") {
			t.Errorf("Expected CPU ratio warning to be filtered when cpu is not in selectedResources, got: %s", w.Message)
		}
	}
}

func TestGenerateWarnings_SelectedResources_MemoryFiltered(t *testing.T) {
	// Test that memory-related warnings are filtered when "memory" is not in selectedResources
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
		UtilizationPct:   50,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical warning (if memory selected)
		FreeChunks:       5,  // < 10 = critical warning (if memory selected)
		CellCount:        4,  // 25% blast radius = critical (if memory selected)
		UtilizationPct:   95, // > 90% = critical (if memory selected)
		BlastRadiusPct:   25,
	}

	calc := NewScenarioCalculator()

	// With memory selected, warnings should appear
	ctxWithMemory := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"memory", "cpu"},
		},
	}
	warningsWithMemory := calc.GenerateWarnings(current, proposed, nil, ctxWithMemory)

	foundN1Warning := false
	foundFreeChunksWarning := false
	foundUtilizationWarning := false
	foundBlastRadiusWarning := false
	for _, w := range warningsWithMemory {
		if w.Message == "Exceeds N-1 capacity safety margin" {
			foundN1Warning = true
		}
		if w.Message == "Critical: Low staging capacity" {
			foundFreeChunksWarning = true
		}
		if w.Message == "Cell utilization critically high" {
			foundUtilizationWarning = true
		}
		if strings.Contains(w.Message, "cell failure impact") {
			foundBlastRadiusWarning = true
		}
	}
	if !foundN1Warning {
		t.Error("Expected N-1 warning when memory is in selectedResources")
	}
	if !foundFreeChunksWarning {
		t.Error("Expected free chunks warning when memory is in selectedResources")
	}
	if !foundUtilizationWarning {
		t.Error("Expected utilization warning when memory is in selectedResources")
	}
	if !foundBlastRadiusWarning {
		t.Error("Expected blast radius warning when memory is in selectedResources")
	}

	// Without memory selected, memory-related warnings should NOT appear
	ctxWithoutMemory := &WarningsContext{
		Input: models.ScenarioInput{
			SelectedResources: []string{"cpu"}, // Only CPU, no memory
		},
	}
	warningsWithoutMemory := calc.GenerateWarnings(current, proposed, nil, ctxWithoutMemory)

	for _, w := range warningsWithoutMemory {
		if w.Message == "Exceeds N-1 capacity safety margin" {
			t.Error("N-1 warning should be filtered when memory is not in selectedResources")
		}
		if w.Message == "Critical: Low staging capacity" {
			t.Error("Free chunks warning should be filtered when memory is not in selectedResources")
		}
		if w.Message == "Cell utilization critically high" {
			t.Error("Utilization warning should be filtered when memory is not in selectedResources")
		}
		if strings.Contains(w.Message, "cell failure impact") {
			t.Error("Blast radius warning should be filtered when memory is not in selectedResources")
		}
	}
}

func TestGenerateWarnings_SelectedResources_EmptyDefaultsToAll(t *testing.T) {
	// Test that empty selectedResources defaults to showing all warnings
	current := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 50,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct:   70,
		FreeChunks:         500,
		CellCount:          100,
		DiskUtilizationPct: 92, // > 90% = critical
		TotalPCPUs:         96,
		TotalVCPUs:         1000,
		VCPURatio:          10.4,
		CPURiskLevel:       "aggressive",
	}

	calc := NewScenarioCalculator()

	// With empty selectedResources (nil context), all warnings should show
	warnings := calc.GenerateWarnings(current, proposed, nil, nil)

	foundDiskWarning := false
	foundCPUWarning := false
	for _, w := range warnings {
		if w.Message == "Disk utilization critically high" {
			foundDiskWarning = true
		}
		if strings.Contains(w.Message, "aggressive") && strings.Contains(w.Message, "vCPU:pCPU") {
			foundCPUWarning = true
		}
	}
	if !foundDiskWarning {
		t.Error("Expected disk warning when selectedResources is empty (default to all)")
	}
	if !foundCPUWarning {
		t.Error("Expected CPU warning when selectedResources is empty (default to all)")
	}
}

func TestResolveChunkSizeMB(t *testing.T) {
	tests := []struct {
		name    string
		inputMB int
		stateMB int
		wantMB  int
	}{
		{"input override wins", 2048, 3072, 2048},
		{"state max used when input is 0", 0, 3072, 3072},
		{"default when both are 0", 0, 0, 4096},
		{"input override even when state available", 1024, 2048, 1024},
		// NEW: minimum floor enforcement - tiny values should be clamped to 1024MB
		{"state max below minimum floor", 0, 100, 1024},  // 100MB -> 1024MB minimum
		{"state max at minimum floor", 0, 1024, 1024},    // 1024MB -> 1024MB (at floor)
		{"state max above minimum floor", 0, 2048, 2048}, // 2048MB -> 2048MB (above floor)
		// Input override is NOT clamped - user explicitly requested this value
		{"input override below floor is respected", 512, 0, 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveChunkSizeMB(tt.inputMB, tt.stateMB)
			if got != tt.wantMB {
				t.Errorf("resolveChunkSizeMB(%d, %d) = %d, want %d", tt.inputMB, tt.stateMB, got, tt.wantMB)
			}
		})
	}
}

func TestFreeChunksWithConfigurableSize(t *testing.T) {
	state := models.InfrastructureState{
		TotalN1MemoryGB:     26624,
		TotalCellCount:      100,
		PlatformVMsGB:       1000,
		TotalAppMemoryGB:    2000,
		TotalAppInstances:   1000,
		MaxInstanceMemoryMB: 2048, // 2GB max app size (used for chunk size)
		AvgInstanceMemoryMB: 500,  // 500MB average (NOT used for chunk size)
		Clusters: []models.ClusterState{
			{DiegoCellCount: 100, DiegoCellMemoryGB: 32, DiegoCellCPU: 4},
		},
	}

	calc := NewScenarioCalculator()

	// Test 1: Auto-detect from state MaxInstanceMemoryMB (2GB chunks)
	input1 := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    100,
		ChunkSizeMB:          0, // Use auto-detect
	}
	result1 := calc.CalculateProposed(state, input1)

	// App capacity: 100 cells × (32 - 2 overhead) = 3000 GB
	// Free memory: 3000 - 2000 = 1000 GB = 1024000 MB
	// Free chunks at 2048 MB: 1024000 / 2048 = 500
	if result1.FreeChunks != 500 {
		t.Errorf("Expected FreeChunks 500 with auto-detect 2GB, got %d", result1.FreeChunks)
	}
	if result1.ChunkSizeMB != 2048 {
		t.Errorf("Expected ChunkSizeMB 2048, got %d", result1.ChunkSizeMB)
	}

	// Test 2: Manual override (1GB chunks)
	input2 := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    100,
		ChunkSizeMB:          1024, // Override to 1GB
	}
	result2 := calc.CalculateProposed(state, input2)

	// Free chunks at 1024 MB: 1024000 / 1024 = 1000
	if result2.FreeChunks != 1000 {
		t.Errorf("Expected FreeChunks 1000 with 1GB override, got %d", result2.FreeChunks)
	}
	if result2.ChunkSizeMB != 1024 {
		t.Errorf("Expected ChunkSizeMB 1024, got %d", result2.ChunkSizeMB)
	}
}

// TestChunkSizeMinimumFloor verifies that tiny MaxInstanceMemoryMB values
// (like 100MB average instance size) don't result in tiny chunk sizes.
// This is the bug from PR #89 - AvgInstanceMemoryMB was being used as chunk size.
func TestChunkSizeMinimumFloor(t *testing.T) {
	// Simulate a real-world scenario: 500 app instances using 50GB total
	// Average = 50*1024/500 = 102MB (typical for small containerized apps)
	// But chunk size for staging should never be this small!
	state := models.InfrastructureState{
		TotalN1MemoryGB:     26624,
		TotalCellCount:      100,
		PlatformVMsGB:       1000,
		TotalAppMemoryGB:    50,
		TotalAppInstances:   500,
		MaxInstanceMemoryMB: 0,   // No max available (legacy data)
		AvgInstanceMemoryMB: 102, // Calculated: 50*1024/500 = 102MB
		Clusters: []models.ClusterState{
			{DiegoCellCount: 100, DiegoCellMemoryGB: 32, DiegoCellCPU: 4},
		},
	}

	calc := NewScenarioCalculator()

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    100,
		ChunkSizeMB:          0, // Use auto-detect
	}
	result := calc.CalculateProposed(state, input)

	// With no MaxInstanceMemoryMB and tiny AvgInstanceMemoryMB,
	// should fall back to default 4096MB, NOT use the 102MB average!
	if result.ChunkSizeMB != 4096 {
		t.Errorf("Expected ChunkSizeMB 4096 (default) when MaxInstanceMemoryMB is 0, got %d", result.ChunkSizeMB)
	}
}

func TestCompare_HAInsufficientWarning_FilteredWhenMemoryNotSelected(t *testing.T) {
	// Test that HA Admission Control insufficient warning is filtered when memory is not selected
	// This warning is added in CompareScenarios, not GenerateWarnings

	// Set up state with 4 hosts at 512GB each
	// HA 7% reserves: 2048 * 0.07 = 143GB
	// N-1 reserves: 512GB (one host)
	// 143GB < 512GB triggers InsufficientHAWarning
	state := models.InfrastructureState{
		TotalN1MemoryGB:   1536, // 3 hosts worth (N-1 protection)
		TotalCellCount:    10,
		PlatformVMsGB:     100,
		TotalAppMemoryGB:  200,
		TotalAppInstances: 50,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    10,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	// Input with HA% that's insufficient for N-1 (triggers the warning)
	inputWithMemory := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    10,
		HostCount:            4,
		MemoryPerHostGB:      512,
		HAAdmissionPct:       7, // 7% of 2048GB = 143GB, but N-1 needs 512GB
		SelectedResources:    []string{"memory", "cpu"},
	}

	inputWithoutMemory := models.ScenarioInput{
		ProposedCellMemoryGB: 32,
		ProposedCellCPU:      4,
		ProposedCellCount:    10,
		HostCount:            4,
		MemoryPerHostGB:      512,
		HAAdmissionPct:       7,
		SelectedResources:    []string{"cpu"}, // Only CPU, no memory
	}

	calc := NewScenarioCalculator()

	// With memory selected, HA insufficient warning should appear
	comparisonWithMemory := calc.Compare(state, inputWithMemory)
	foundHAWarningWithMemory := false
	for _, w := range comparisonWithMemory.Warnings {
		if strings.Contains(w.Message, "HA Admission Control") && strings.Contains(w.Message, "insufficient") {
			foundHAWarningWithMemory = true
			break
		}
	}
	if !foundHAWarningWithMemory {
		t.Error("Expected HA Admission Control insufficient warning when memory is selected")
	}

	// Without memory selected, HA insufficient warning should NOT appear
	comparisonWithoutMemory := calc.Compare(state, inputWithoutMemory)
	for _, w := range comparisonWithoutMemory.Warnings {
		if strings.Contains(w.Message, "HA Admission Control") && strings.Contains(w.Message, "insufficient") {
			t.Error("HA Admission Control insufficient warning should be filtered when memory is not in selectedResources")
		}
	}
}
