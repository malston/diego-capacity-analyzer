# CPU Ratio Analysis Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add vCPU:pCPU ratio analysis to scenario comparison so CPU configuration from the wizard affects what-if analysis results.

**Architecture:** Frontend sends `physical_cores_per_host` and `target_vcpu_ratio` → backend calculates ratio metrics → backend generates warnings when ratio exceeds target → frontend displays CPU gauge with risk level.

**Tech Stack:** Go 1.23 (backend), React 18 + Vitest (frontend), TDD throughout.

**Design Doc:** `docs/plans/2026-01-15-cpu-ratio-analysis-design.md`

**Worktree:** `.worktrees/cpu-ratio-analysis` on branch `feature/cpu-ratio-analysis`

---

## Task 1: Add CPU Input Fields to ScenarioInput

**Files:**
- Modify: `backend/models/scenario.go:9-23`
- Test: `backend/models/scenario_test.go` (new test)

**Step 1: Write the failing test**

Create test in `backend/models/scenario_test.go`:

```go
func TestScenarioInput_CPUFields(t *testing.T) {
	input := ScenarioInput{
		PhysicalCoresPerHost: 32,
		TargetVCPURatio:      4,
	}

	if input.PhysicalCoresPerHost != 32 {
		t.Errorf("PhysicalCoresPerHost = %d, want 32", input.PhysicalCoresPerHost)
	}
	if input.TargetVCPURatio != 4 {
		t.Errorf("TargetVCPURatio = %d, want 4", input.TargetVCPURatio)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./models -run TestScenarioInput_CPUFields -v`
Expected: FAIL with "unknown field 'PhysicalCoresPerHost'"

**Step 3: Add fields to ScenarioInput struct**

In `backend/models/scenario.go`, add after `HAAdmissionPct` field:

```go
	// CPU configuration for vCPU:pCPU ratio analysis
	PhysicalCoresPerHost int `json:"physical_cores_per_host"` // pCPU per ESXi host
	TargetVCPURatio      int `json:"target_vcpu_ratio"`       // User's target ratio (e.g., 4 for 4:1)
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./models -run TestScenarioInput_CPUFields -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/scenario.go backend/models/scenario_test.go
git commit -m "feat(models): add CPU input fields to ScenarioInput"
```

---

## Task 2: Add CPU Output Fields to ScenarioResult

**Files:**
- Modify: `backend/models/scenario.go:46-62`
- Test: `backend/models/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/models/scenario_test.go`:

```go
func TestScenarioResult_CPUFields(t *testing.T) {
	result := ScenarioResult{
		TotalVCPUs:   160,
		TotalPCPUs:   96,
		VCPURatio:    1.67,
		CPURiskLevel: "conservative",
	}

	if result.TotalVCPUs != 160 {
		t.Errorf("TotalVCPUs = %d, want 160", result.TotalVCPUs)
	}
	if result.TotalPCPUs != 96 {
		t.Errorf("TotalPCPUs = %d, want 96", result.TotalPCPUs)
	}
	if result.VCPURatio != 1.67 {
		t.Errorf("VCPURatio = %f, want 1.67", result.VCPURatio)
	}
	if result.CPURiskLevel != "conservative" {
		t.Errorf("CPURiskLevel = %s, want conservative", result.CPURiskLevel)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./models -run TestScenarioResult_CPUFields -v`
Expected: FAIL with "unknown field 'TotalVCPUs'"

**Step 3: Add fields to ScenarioResult struct**

In `backend/models/scenario.go`, add after `BlastRadiusPct` field in ScenarioResult:

```go
	// CPU ratio metrics (only populated when CPU analysis enabled)
	TotalVCPUs   int     `json:"total_vcpus"`    // cellCount * cellCPU
	TotalPCPUs   int     `json:"total_pcpus"`    // hostCount * physicalCoresPerHost
	VCPURatio    float64 `json:"vcpu_ratio"`     // TotalVCPUs / TotalPCPUs (e.g., 4.5 means 4.5:1)
	CPURiskLevel string  `json:"cpu_risk_level"` // "conservative", "moderate", "aggressive"
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./models -run TestScenarioResult_CPUFields -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/scenario.go backend/models/scenario_test.go
git commit -m "feat(models): add CPU output fields to ScenarioResult"
```

---

## Task 3: Add VCPURatioChange to ScenarioDelta

**Files:**
- Modify: `backend/models/scenario.go:94-100`
- Test: `backend/models/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/models/scenario_test.go`:

```go
func TestScenarioDelta_VCPURatioChange(t *testing.T) {
	delta := ScenarioDelta{
		VCPURatioChange: 1.5,
	}

	if delta.VCPURatioChange != 1.5 {
		t.Errorf("VCPURatioChange = %f, want 1.5", delta.VCPURatioChange)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./models -run TestScenarioDelta_VCPURatioChange -v`
Expected: FAIL with "unknown field 'VCPURatioChange'"

**Step 3: Add field to ScenarioDelta struct**

In `backend/models/scenario.go`, add to ScenarioDelta struct:

```go
	VCPURatioChange float64 `json:"vcpu_ratio_change"` // Proposed ratio - current ratio
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./models -run TestScenarioDelta_VCPURatioChange -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/scenario.go backend/models/scenario_test.go
git commit -m "feat(models): add VCPURatioChange to ScenarioDelta"
```

---

## Task 4: Add CPURiskLevel Helper Function

**Files:**
- Modify: `backend/services/scenario.go`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services -run TestCPURiskLevel -v`
Expected: FAIL with "undefined: CPURiskLevel"

**Step 3: Implement CPURiskLevel function**

Add to `backend/services/scenario.go` after the constants block:

```go
// CPURiskLevel returns risk classification based on vCPU:pCPU ratio.
// Thresholds based on VMware general guidance (workload-dependent):
// - Conservative (≤4:1): Safe for production workloads
// - Moderate (4-8:1): Monitor CPU Ready time
// - Aggressive (>8:1): Expect contention, requires active monitoring
func CPURiskLevel(ratio float64) string {
	switch {
	case ratio <= 4:
		return "conservative"
	case ratio <= 8:
		return "moderate"
	default:
		return "aggressive"
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestCPURiskLevel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat(services): add CPURiskLevel helper function"
```

---

## Task 5: Update calculateFull to Compute CPU Metrics

**Files:**
- Modify: `backend/services/scenario.go:256-339`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
func TestCalculateFull_WithCPUConfig(t *testing.T) {
	calc := NewScenarioCalculator()

	// Use reflection or create a wrapper test since calculateFull is private
	// For now, test via CalculateProposed which calls calculateFull
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
```

Also add the import for "math" at the top if not present.

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services -run TestCalculateFull_WithCPUConfig -v`
Expected: FAIL (TotalVCPUs = 0, TotalPCPUs = 0)

**Step 3: Update calculateFull signature and implementation**

In `backend/services/scenario.go`:

1. Update `calculateFull` signature to add `hostCount` and `physicalCoresPerHost` parameters:

```go
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
	hostCount int,
	physicalCoresPerHost int,
) models.ScenarioResult {
```

2. Add CPU calculation logic before the return statement:

```go
	// CPU ratio calculations (only when host CPU config provided)
	var totalVCPUs, totalPCPUs int
	var vcpuRatio float64
	var cpuRiskLevel string

	if hostCount > 0 && physicalCoresPerHost > 0 {
		totalVCPUs = cellCount * cellCPU
		totalPCPUs = hostCount * physicalCoresPerHost
		vcpuRatio = float64(totalVCPUs) / float64(totalPCPUs)
		cpuRiskLevel = CPURiskLevel(vcpuRatio)
	}
```

3. Add fields to the return struct:

```go
		TotalVCPUs:   totalVCPUs,
		TotalPCPUs:   totalPCPUs,
		VCPURatio:    vcpuRatio,
		CPURiskLevel: cpuRiskLevel,
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestCalculateFull_WithCPUConfig -v`
Expected: FAIL - callers need to be updated first (compilation errors)

---

## Task 6: Update CalculateCurrent and CalculateProposed Callers

**Files:**
- Modify: `backend/services/scenario.go:194-253`

**Step 1: Verify compilation fails**

Run: `cd backend && go build ./...`
Expected: FAIL with "not enough arguments in call to c.calculateFull"

**Step 2: Update CalculateCurrent**

In `CalculateCurrent`, update the call to `calculateFull` to pass host config (use 0 since current doesn't have input):

```go
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
		0, // hostCount - not available in current state
		0, // physicalCoresPerHost - not available in current state
	)
```

**Step 3: Update CalculateProposed**

In `CalculateProposed`, update the call to pass host config from input:

```go
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
		input.HostCount,
		input.PhysicalCoresPerHost,
	)
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestCalculateFull_WithCPUConfig -v`
Expected: PASS

**Step 5: Run all service tests**

Run: `cd backend && go test ./services -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat(services): compute CPU ratio metrics in calculateFull"
```

---

## Task 7: Add CalculateCPURatioFix Helper

**Files:**
- Modify: `backend/services/scenario.go`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services -run TestCalculateCPURatioFix -v`
Expected: FAIL with "undefined: CalculateCPURatioFix"

**Step 3: Implement CalculateCPURatioFix**

Add to `backend/services/scenario.go` after `CalculateCapacityFix`:

```go
// CalculateCPURatioFix suggests how to reduce vCPU:pCPU ratio to target.
// Returns at most 2 fix suggestions.
func CalculateCPURatioFix(state models.InfrastructureState, input models.ScenarioInput,
	currentRatio, targetRatio float64) []models.FixSuggestion {
	var fixes []models.FixSuggestion

	totalPCPUs := input.HostCount * input.PhysicalCoresPerHost
	if totalPCPUs == 0 || input.ProposedCellCPU == 0 {
		return fixes
	}

	// Fix 1: Reduce cell count to achieve target ratio
	// targetRatio = (cells * cellCPU) / totalPCPUs
	// cells = (targetRatio * totalPCPUs) / cellCPU
	targetCells := int(targetRatio * float64(totalPCPUs) / float64(input.ProposedCellCPU))
	if targetCells > 0 && targetCells < input.ProposedCellCount {
		fixes = append(fixes, models.FixSuggestion{
			Description: fmt.Sprintf("Reduce to %d cells to achieve %.0f:1 ratio", targetCells, targetRatio),
			Field:       "cell_count",
			Value:       targetCells,
		})
	}

	// Fix 2: Reduce vCPU per cell
	// targetRatio = (cells * cellCPU) / totalPCPUs
	// cellCPU = (targetRatio * totalPCPUs) / cells
	targetCellCPU := int(targetRatio * float64(totalPCPUs) / float64(input.ProposedCellCount))
	if targetCellCPU > 0 && targetCellCPU < input.ProposedCellCPU {
		fixes = append(fixes, models.FixSuggestion{
			Description: fmt.Sprintf("Reduce cell vCPU to %d to achieve %.0f:1 ratio", targetCellCPU, targetRatio),
			Field:       "cell_cpu",
			Value:       targetCellCPU,
		})
	}

	if len(fixes) > 2 {
		fixes = fixes[:2]
	}
	return fixes
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestCalculateCPURatioFix -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat(services): add CalculateCPURatioFix helper"
```

---

## Task 8: Add CPU Ratio Warnings to GenerateWarnings

**Files:**
- Modify: `backend/services/scenario.go:371-482`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing tests**

Add to `backend/services/scenario_test.go`:

```go
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
```

Also add `"strings"` to imports if not present.

**Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./services -run "TestGenerateWarnings_CPU" -v`
Expected: FAIL (warnings not generated)

**Step 3: Add CPU warnings to GenerateWarnings**

In `backend/services/scenario.go`, add after the blast radius warnings in `GenerateWarnings`:

```go
	// vCPU:pCPU ratio warnings (only when CPU analysis enabled)
	if proposed.TotalPCPUs > 0 {
		targetRatio := 4.0 // Default target
		if ctx != nil && ctx.Input.TargetVCPURatio > 0 {
			targetRatio = float64(ctx.Input.TargetVCPURatio)
		}

		// Warning when ratio exceeds target
		if proposed.VCPURatio > targetRatio {
			warning := models.ScenarioWarning{
				Severity: "warning",
				Message: fmt.Sprintf(
					"vCPU:pCPU ratio %.1f:1 exceeds target %.0f:1 - expect CPU contention under load",
					proposed.VCPURatio, targetRatio,
				),
			}
			if ctx != nil {
				warning.Change = findRelevantChange(ctx.Changes, "cell_count", "cell_cpu")
				warning.Fixes = CalculateCPURatioFix(ctx.State, ctx.Input, proposed.VCPURatio, targetRatio)
			}
			warnings = append(warnings, warning)
		}

		// Critical when ratio is aggressive (>8:1)
		if proposed.CPURiskLevel == "aggressive" {
			warnings = append(warnings, models.ScenarioWarning{
				Severity: "critical",
				Message: fmt.Sprintf(
					"vCPU:pCPU ratio %.1f:1 is aggressive - monitor CPU Ready time (>5%% indicates problems)",
					proposed.VCPURatio,
				),
			})
		}
	}
```

**Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./services -run "TestGenerateWarnings_CPU" -v`
Expected: PASS

**Step 5: Run all backend tests**

Run: `cd backend && go test ./...`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat(services): add CPU ratio warnings to GenerateWarnings"
```

---

## Task 9: Add VCPURatioChange to Compare Function

**Files:**
- Modify: `backend/services/scenario.go:485-565`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services -run TestCompare_VCPURatioChange -v`
Expected: FAIL (VCPURatioChange = 0)

**Step 3: Add VCPURatioChange calculation to Compare**

In `backend/services/scenario.go`, in the `Compare` function, add after the delta calculations:

```go
	// CPU ratio change
	vcpuRatioChange := proposed.VCPURatio - current.VCPURatio
```

And update the Delta struct in the return:

```go
	Delta: models.ScenarioDelta{
		CapacityChangeGB:         capacityChange,
		DiskCapacityChangeGB:     diskCapacityChange,
		UtilizationChangePct:     utilizationChange,
		DiskUtilizationChangePct: diskUtilizationChange,
		ResilienceChange:         resilienceChange,
		VCPURatioChange:          vcpuRatioChange,
	},
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestCompare_VCPURatioChange -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat(services): add VCPURatioChange to Compare delta"
```

---

## Task 10: Backend E2E Test

**Files:**
- Create: `backend/e2e/scenario_cpu_test.go`

**Step 1: Write the E2E test**

Create `backend/e2e/scenario_cpu_test.go`:

```go
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
)

func TestScenarioCompare_WithCPUConfig(t *testing.T) {
	h := handlers.NewHandlers(nil, nil, nil, nil)

	// First, set up infrastructure state
	infraPayload := `{
		"name": "CPU Test Infra",
		"clusters": [{
			"name": "test-cluster",
			"diego_cell_count": 10,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4,
			"diego_cell_disk_gb": 128,
			"host_count": 3,
			"memory_gb_per_host": 512
		}]
	}`

	req := httptest.NewRequest("POST", "/api/infrastructure/manual", bytes.NewBufferString(infraPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleManualInfrastructure(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to set infrastructure: %s", rr.Body.String())
	}

	// Now test scenario compare with CPU config
	scenarioPayload := `{
		"proposed_cell_memory_gb": 32,
		"proposed_cell_cpu": 4,
		"proposed_cell_disk_gb": 128,
		"proposed_cell_count": 20,
		"host_count": 3,
		"physical_cores_per_host": 32,
		"target_vcpu_ratio": 4,
		"memory_per_host_gb": 512,
		"ha_admission_pct": 25
	}`

	req = httptest.NewRequest("POST", "/api/scenario/compare", bytes.NewBufferString(scenarioPayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.HandleScenarioCompare(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Scenario compare failed: %s", rr.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify proposed has CPU metrics
	proposed, ok := result["proposed"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'proposed' field")
	}

	// Check CPU fields exist
	if _, ok := proposed["total_vcpus"]; !ok {
		t.Error("Response missing total_vcpus")
	}
	if _, ok := proposed["total_pcpus"]; !ok {
		t.Error("Response missing total_pcpus")
	}
	if _, ok := proposed["vcpu_ratio"]; !ok {
		t.Error("Response missing vcpu_ratio")
	}
	if _, ok := proposed["cpu_risk_level"]; !ok {
		t.Error("Response missing cpu_risk_level")
	}

	// Verify values
	// 20 cells × 4 vCPU = 80 vCPU
	// 3 hosts × 32 pCPU = 96 pCPU
	// Ratio = 80/96 = 0.833
	if vcpus, ok := proposed["total_vcpus"].(float64); ok && vcpus != 80 {
		t.Errorf("total_vcpus = %f, want 80", vcpus)
	}
	if pcpus, ok := proposed["total_pcpus"].(float64); ok && pcpus != 96 {
		t.Errorf("total_pcpus = %f, want 96", pcpus)
	}
}
```

**Step 2: Run E2E test**

Run: `cd backend && go test ./e2e -run TestScenarioCompare_WithCPUConfig -v`
Expected: PASS

**Step 3: Run all backend tests**

Run: `cd backend && go test ./...`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add backend/e2e/scenario_cpu_test.go
git commit -m "test(e2e): add E2E test for CPU ratio analysis"
```

---

## Task 11: Add CPU Gauge to ScenarioResults Frontend

**Files:**
- Modify: `frontend/src/components/ScenarioResults.jsx`
- Test: `frontend/src/components/ScenarioResults.test.jsx`

**Step 1: Write the failing test**

Add to `frontend/src/components/ScenarioResults.test.jsx`:

```javascript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ScenarioResults from './ScenarioResults';

describe('ScenarioResults CPU Gauge', () => {
  it('displays CPU ratio gauge when cpu selected and data available', () => {
    const comparison = {
      current: {
        cell_count: 10,
        cell_memory_gb: 32,
        cell_cpu: 4,
        app_capacity_gb: 298,
        utilization_pct: 50,
        n1_utilization_pct: 60,
        free_chunks: 100,
        fault_impact: 5,
        instances_per_cell: 5,
        total_vcpus: 40,
        total_pcpus: 96,
        vcpu_ratio: 0.42,
        cpu_risk_level: 'conservative',
      },
      proposed: {
        cell_count: 20,
        cell_memory_gb: 32,
        cell_cpu: 4,
        app_capacity_gb: 596,
        utilization_pct: 25,
        n1_utilization_pct: 70,
        free_chunks: 200,
        fault_impact: 2.5,
        instances_per_cell: 2.5,
        total_vcpus: 80,
        total_pcpus: 96,
        vcpu_ratio: 0.83,
        cpu_risk_level: 'conservative',
      },
      delta: {
        capacity_change_gb: 298,
        utilization_change_pct: -25,
        resilience_change: 'low',
        vcpu_ratio_change: 0.41,
      },
      warnings: [],
    };

    render(
      <ScenarioResults
        comparison={comparison}
        warnings={[]}
        selectedResources={['memory', 'cpu']}
      />
    );

    // Should display the ratio
    expect(screen.getByText('0.8:1')).toBeInTheDocument();
    // Should show risk level
    expect(screen.getByText('conservative')).toBeInTheDocument();
  });

  it('hides CPU gauge when cpu not in selectedResources', () => {
    const comparison = {
      current: {
        cell_count: 10,
        cell_memory_gb: 32,
        cell_cpu: 4,
        app_capacity_gb: 298,
        utilization_pct: 50,
        n1_utilization_pct: 60,
        free_chunks: 100,
        fault_impact: 5,
        instances_per_cell: 5,
        total_vcpus: 80,
        total_pcpus: 96,
        vcpu_ratio: 0.83,
        cpu_risk_level: 'conservative',
      },
      proposed: {
        cell_count: 20,
        cell_memory_gb: 32,
        cell_cpu: 4,
        app_capacity_gb: 596,
        utilization_pct: 25,
        n1_utilization_pct: 70,
        free_chunks: 200,
        fault_impact: 2.5,
        instances_per_cell: 2.5,
        total_vcpus: 80,
        total_pcpus: 96,
        vcpu_ratio: 0.83,
        cpu_risk_level: 'conservative',
      },
      delta: {
        capacity_change_gb: 298,
        utilization_change_pct: -25,
        resilience_change: 'low',
      },
      warnings: [],
    };

    render(
      <ScenarioResults
        comparison={comparison}
        warnings={[]}
        selectedResources={['memory']}
      />
    );

    // Should NOT display vCPU:pCPU label when cpu not selected
    expect(screen.queryByText('vCPU:pCPU Ratio')).not.toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun run test -- --reporter=verbose 2>&1 | grep -A5 "CPU Gauge"`
Expected: FAIL (text not found)

**Step 3: Add CPU gauge to ScenarioResults.jsx**

In `frontend/src/components/ScenarioResults.jsx`:

1. Add to TOOLTIPS object:

```javascript
  cpuRatio: "vCPU:pCPU ratio measures CPU oversubscription. Conservative (≤4:1): safe for production. Moderate (4-8:1): monitor CPU Ready time. Aggressive (>8:1): expect contention. Recommended by workload: General 4-6:1, CPU-intensive 2-3:1, IO-intensive 6-8:1.",
```

2. Update the grid layout (around line 144) to handle CPU gauge:

```jsx
      {/* Key Gauges Row */}
      <div className={`grid gap-6 ${
        selectedResources.includes('cpu') && proposed.total_pcpus > 0
          ? (selectedResources.includes('disk') && proposed.disk_capacity_gb > 0
              ? 'grid-cols-2 lg:grid-cols-5'
              : 'grid-cols-2 lg:grid-cols-4')
          : (selectedResources.includes('disk') && proposed.disk_capacity_gb > 0
              ? 'grid-cols-2 lg:grid-cols-4'
              : 'grid-cols-3')
      }`}>
```

3. Add CPU gauge after the Disk Utilization Gauge and before Free Chunks (around line 215):

```jsx
        {/* CPU Ratio Gauge - only if cpu selected and data available */}
        {selectedResources.includes('cpu') && proposed.total_pcpus > 0 && (
          <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center gap-2 mb-4 text-gray-400">
              <Cpu size={16} />
              <Tooltip text={TOOLTIPS.cpuRatio} position="bottom" showIcon>
                <span className="text-xs uppercase tracking-wider font-medium">vCPU:pCPU Ratio</span>
              </Tooltip>
            </div>
            <div className="flex flex-col items-center justify-center h-[120px]">
              <div className={`text-4xl font-mono font-bold ${
                proposed.cpu_risk_level === 'conservative' ? 'text-emerald-400' :
                proposed.cpu_risk_level === 'moderate' ? 'text-amber-400' :
                'text-red-400'
              }`}>
                {proposed.vcpu_ratio.toFixed(1)}:1
              </div>
              <div className="text-sm text-gray-400 mt-2">
                {proposed.total_vcpus.toLocaleString()} vCPU / {proposed.total_pcpus.toLocaleString()} pCPU
              </div>
              <div className={`text-xs mt-2 px-2 py-0.5 rounded ${
                proposed.cpu_risk_level === 'conservative' ? 'bg-emerald-900/30 text-emerald-400' :
                proposed.cpu_risk_level === 'moderate' ? 'bg-amber-900/30 text-amber-400' :
                'bg-red-900/30 text-red-400'
              }`}>
                {proposed.cpu_risk_level}
              </div>
            </div>
            <div className="mt-4 text-center text-xs text-gray-500">
              Target: 4:1
            </div>
          </div>
        )}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun run test -- --reporter=verbose 2>&1 | grep -A10 "CPU Gauge"`
Expected: PASS

**Step 5: Run all frontend tests**

Run: `cd frontend && bun run test`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add frontend/src/components/ScenarioResults.jsx frontend/src/components/ScenarioResults.test.jsx
git commit -m "feat(frontend): add CPU ratio gauge to ScenarioResults"
```

---

## Task 12: Final Verification and Cleanup

**Step 1: Run all tests**

```bash
cd backend && go test ./...
cd ../frontend && bun run test
```

Expected: All tests PASS

**Step 2: Run linting**

```bash
make lint
```

Expected: No errors

**Step 3: Manual verification**

1. Start backend: `make backend-run`
2. Start frontend: `make frontend-dev`
3. Load sample infrastructure
4. Enable CPU in Resource Types
5. Fill out CPU Config step (physical cores: 32, hosts: 3, target ratio: 4)
6. Click "Run Analysis"
7. Verify CPU gauge appears with ratio and risk level
8. Verify warnings appear when ratio exceeds target

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete CPU ratio analysis implementation"
```

**Step 5: Push feature branch**

```bash
git push -u origin feature/cpu-ratio-analysis
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add CPU input fields | models/scenario.go |
| 2 | Add CPU output fields | models/scenario.go |
| 3 | Add VCPURatioChange | models/scenario.go |
| 4 | CPURiskLevel helper | services/scenario.go |
| 5-6 | calculateFull CPU metrics | services/scenario.go |
| 7 | CalculateCPURatioFix helper | services/scenario.go |
| 8 | CPU ratio warnings | services/scenario.go |
| 9 | VCPURatioChange in Compare | services/scenario.go |
| 10 | Backend E2E test | e2e/scenario_cpu_test.go |
| 11 | Frontend CPU gauge | ScenarioResults.jsx |
| 12 | Final verification | - |
