# CPU Ratio as Capacity Constraint

**Date:** 2026-01-15
**Status:** Approved
**Related:** Extends CPU ratio analysis from PR #39

## Problem

The target vCPU:pCPU ratio is currently only a warning threshold. When users set a target ratio (e.g., 4:1), the system warns if the calculated ratio exceeds it, but doesn't show how this constraint affects capacity planning.

Users need to see:
- How many cells they can deploy before hitting their CPU target
- Whether CPU, memory, or disk is the limiting factor
- How much headroom remains within their CPU budget

## Solution

Make target vCPU:pCPU ratio a capacity constraint alongside memory and disk:
1. Calculate max cells by CPU based on target ratio
2. Identify bottleneck across all three resources
3. Show headroom for non-limiting resources
4. Account for platform VM overhead in calculations

## Design

### Data Model Changes

**ScenarioInput** (add field):
```go
PlatformVMsCPU int `json:"platform_vms_cpu"` // Total vCPUs for non-Diego platform VMs
```

**ScenarioResult** (add fields):
```go
MaxCellsByCPU    int `json:"max_cells_by_cpu"`    // Max cells before hitting target ratio
CPUHeadroomCells int `json:"cpu_headroom_cells"`  // Additional cells available within target
```

**PlanningResult** (add fields):
```go
MaxCellsByCPU int    `json:"max_cells_by_cpu"` // Max cells by CPU constraint
Bottleneck    string `json:"bottleneck"`       // "memory", "cpu", or "disk"
```

### Calculation Logic

```go
func CalculateMaxCellsByCPU(
    targetRatio float64,
    totalPCPUs int,
    cellCPU int,
    platformVMsCPU int,
) int {
    if cellCPU == 0 || totalPCPUs == 0 {
        return 0 // CPU analysis disabled
    }

    // maxVCPU = targetRatio × totalPCPUs
    // maxVCPU = (cells × cellCPU) + platformVMsCPU
    // cells = (maxVCPU - platformVMsCPU) / cellCPU

    maxVCPU := targetRatio * float64(totalPCPUs)
    availableForCells := maxVCPU - float64(platformVMsCPU)

    if availableForCells <= 0 {
        return 0 // Platform VMs already exceed target
    }

    return int(availableForCells) / cellCPU
}
```

**Bottleneck Detection:**
```go
bottleneck := "memory"
maxCells := maxCellsByMemory

if maxCellsByCPU > 0 && maxCellsByCPU < maxCells {
    bottleneck = "cpu"
    maxCells = maxCellsByCPU
}
if maxCellsByDisk > 0 && maxCellsByDisk < maxCells {
    bottleneck = "disk"
    maxCells = maxCellsByDisk
}
```

### Frontend Display

**Planning Calculator Results:**
```
Maximum Deployable Cells
━━━━━━━━━━━━━━━━━━━━━━━━
Memory:  42 cells  ← BOTTLENECK
CPU:     58 cells  (16 headroom)
Disk:    95 cells  (53 headroom)
```

**Scenario Comparison CPU Gauge:**
```
vCPU:pCPU Ratio
    3.2:1
   [====----] (target: 4:1)

   Headroom: +12 cells
   (before reaching 4:1 target)
```

**New Input Field (CPU Configuration step):**
```
Platform VM vCPUs [________] (optional)
  Total vCPUs allocated to control plane VMs
  (BOSH, Diego Brain, Router, etc.)
```

## Files to Modify

| File | Changes |
|------|---------|
| `backend/models/scenario.go` | Add PlatformVMsCPU input, MaxCellsByCPU and CPUHeadroomCells outputs |
| `backend/models/planning.go` | Add MaxCellsByCPU and Bottleneck fields |
| `backend/services/scenario.go` | Add CalculateMaxCellsByCPU(), integrate into calculateFull() |
| `backend/services/planning.go` | Add CPU constraint, bottleneck detection |
| `backend/services/scenario_test.go` | Unit tests for CPU constraint calculations |
| `backend/services/planning_test.go` | Unit tests for bottleneck detection |
| `backend/e2e/planning_cpu_test.go` | E2E test for planning with CPU constraint |
| `frontend/src/components/wizard/steps/CPUStep.jsx` | Add Platform VM vCPUs input |
| `frontend/src/components/ScenarioResults.jsx` | Add CPU headroom display |
| `frontend/src/components/PlanningResults.jsx` | Show all constraints with bottleneck |

## Test Plan

### Backend Unit Tests

1. **TestCalculateMaxCellsByCPU** - Core calculation:
   - Standard case: 100 pCPUs, 4:1 target, 4 vCPU cells → 100 max cells
   - With platform overhead: same + 40 platform vCPUs → 90 max cells
   - Zero cellCPU returns 0 (disabled)
   - Platform VMs exceed budget returns 0

2. **TestBottleneckDetection** - Multi-resource comparison:
   - Memory-limited (memory < CPU < disk)
   - CPU-limited (CPU < memory < disk)
   - Disk-limited (disk < memory < CPU)

3. **TestCPUHeadroomCells** - Headroom calculation:
   - Positive headroom (current < max)
   - Zero headroom (at limit)
   - Negative headroom (over target)

### Backend E2E Test

- **TestPlanningCalculator_WithCPUConstraint** - Verify API returns max_cells_by_cpu and bottleneck fields

### Frontend Tests

- CPU headroom displays when positive
- CPU headroom shows warning styling when negative
- Bottleneck indicator highlights limiting resource
- Platform VM input field updates calculations

## Verification

```bash
make backend-test
make frontend-test
make check
```

Manual: Configure CPU settings → Set platform VM vCPUs → Run analysis → Verify bottleneck identification and headroom display.
