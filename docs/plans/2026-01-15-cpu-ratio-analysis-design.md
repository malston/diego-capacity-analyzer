# CPU Ratio Analysis for Scenario Comparison

**Date:** 2026-01-15
**Status:** Approved
**Related:** GitHub Issue #10, PR #25

## Problem

The wizard collects CPU configuration (`physicalCoresPerHost`, `targetVCPURatio`) but clicking "Run Analysis" produces no CPU-related output. The frontend sends the data; the backend ignores it.

**Root cause:** Backend `ScenarioInput` lacks fields for `physical_cores_per_host` and `target_vcpu_ratio`. Backend `ScenarioResult` lacks CPU output metrics. No calculation or display code exists.

## Solution

Add end-to-end CPU ratio analysis:
1. Accept CPU config in backend input struct
2. Calculate vCPU:pCPU ratio in scenario results
3. Generate warnings when ratio exceeds target
4. Display CPU gauge in results UI

## Design

### Data Model Changes

**ScenarioInput** (add fields):
```go
PhysicalCoresPerHost int `json:"physical_cores_per_host"`
TargetVCPURatio      int `json:"target_vcpu_ratio"`
```

**ScenarioResult** (add fields):
```go
TotalVCPUs   int     `json:"total_vcpus"`
TotalPCPUs   int     `json:"total_pcpus"`
VCPURatio    float64 `json:"vcpu_ratio"`
CPURiskLevel string  `json:"cpu_risk_level"`
```

**ScenarioDelta** (add field):
```go
VCPURatioChange float64 `json:"vcpu_ratio_change"`
```

### Calculation Logic

In `calculateFull()`, add:
```go
if hostCount > 0 && physicalCoresPerHost > 0 {
    totalVCPUs = cellCount * cellCPU
    totalPCPUs = hostCount * physicalCoresPerHost
    vcpuRatio = float64(totalVCPUs) / float64(totalPCPUs)
    cpuRiskLevel = CPURiskLevel(vcpuRatio)
}
```

Risk level thresholds:
| Ratio | Risk Level | Guidance |
|-------|------------|----------|
| ≤4:1 | conservative | Safe for production |
| 4-8:1 | moderate | Monitor CPU Ready time |
| >8:1 | aggressive | Expect contention |

### Warning Generation

Two warning types:

1. **Ratio exceeds target** (severity: warning)
   - Triggered when `vcpuRatio > targetVCPURatio`
   - Message: "vCPU:pCPU ratio X.X:1 exceeds target Y:1 - expect CPU contention under load"
   - Fix suggestions: reduce cell count OR reduce vCPU per cell

2. **Aggressive ratio** (severity: critical)
   - Triggered when `vcpuRatio > 8`
   - Message: "vCPU:pCPU ratio X.X:1 is aggressive - monitor CPU Ready time (>5% indicates problems)"

### Frontend Display

Add CPU ratio gauge to `ScenarioResults.jsx`:
- Display format: "X.X:1" (e.g., "4.5:1")
- Color by risk level (emerald/amber/red)
- Show breakdown: "X vCPU / Y pCPU"
- Show target comparison below gauge

Tooltip content explains:
- Risk level meanings
- Workload-specific recommendations (general 4-6:1, CPU-intensive 2-3:1, IO-intensive 6-8:1)
- CPU Ready >5% indicates problems

## Files to Modify

| File | Changes |
|------|---------|
| `backend/models/scenario.go` | Add input/output fields |
| `backend/services/scenario.go` | Add calculations, `CPURiskLevel()`, `CalculateCPURatioFix()`, warnings |
| `backend/services/scenario_test.go` | Unit tests |
| `backend/e2e/scenario_cpu_test.go` | E2E test (new file) |
| `frontend/src/components/ScenarioResults.jsx` | CPU gauge, tooltip |
| `frontend/src/components/ScenarioResults.test.jsx` | Component tests |

## Test Plan

### Backend Unit Tests
- `TestCalculateFull_WithCPUConfig` - verify ratio calculation
- `TestCPURiskLevel_Thresholds` - verify threshold boundaries
- `TestGenerateWarnings_CPURatioExceedsTarget` - verify warning generation
- `TestGenerateWarnings_AggressiveRatio` - verify critical warning
- `TestCalculateCPURatioFix` - verify fix suggestions

### Backend E2E Test
- `TestScenarioCompare_WithCPUConfig` - verify API accepts and returns CPU fields

### Frontend Tests
- Verify CPU gauge renders when cpu selected and data available
- Verify CPU gauge hidden when cpu not selected

## Implementation Order

1. Write failing backend test for CPU fields in ScenarioResult
2. Add fields to ScenarioInput and ScenarioResult structs
3. Add `CPURiskLevel()` helper function
4. Update `calculateFull()` with CPU calculations
5. Update `CalculateCurrent()` and `CalculateProposed()` callers
6. Add `CalculateCPURatioFix()` helper
7. Add CPU warnings in `GenerateWarnings()`
8. Verify backend tests pass
9. Write failing frontend test
10. Add CPU gauge to ScenarioResults.jsx
11. Verify all tests pass
12. Manual verification through UI

## Verification

```bash
make backend-test
make frontend-test
make check
```

Manual: Load sample infrastructure → Enable CPU → Fill CPU config → Run Analysis → Verify gauge shows ratio and warnings appear when appropriate.
