# Configurable Chunk Size Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make chunk size (used for free staging capacity calculation) configurable with auto-detection from live CF data and manual override capability.

**Architecture:** Add `AvgInstanceMemoryMB` to InfrastructureState (auto-calculated from live data), `ChunkSizeMB` to ScenarioInput (manual override), and `ChunkSizeMB` to ScenarioResult (transparency). Resolution order: input override → state average → 4096MB default.

**Tech Stack:** Go backend, React frontend, JSON sample files

---

## Task 1: Add AvgInstanceMemoryMB to InfrastructureState

**Files:**

- Modify: `backend/models/infrastructure.go:58-84`
- Test: `backend/models/infrastructure_test.go`

**Step 1: Write the failing test**

Add to `backend/models/infrastructure_test.go`:

```go
func TestAvgInstanceMemoryMB(t *testing.T) {
	mi := ManualInput{
		Name:              "test",
		Clusters:          []ClusterInput{{Name: "c1", HostCount: 4, MemoryGBPerHost: 512, CPUCoresPerHost: 32, DiegoCellCount: 10, DiegoCellMemoryGB: 32, DiegoCellCPU: 4}},
		PlatformVMsGB:     200,
		TotalAppMemoryGB:  150,
		TotalAppInstances: 50,
	}
	state := mi.ToInfrastructureState()

	// 150 GB * 1024 MB/GB / 50 instances = 3072 MB
	expected := 3072
	if state.AvgInstanceMemoryMB != expected {
		t.Errorf("Expected AvgInstanceMemoryMB %d, got %d", expected, state.AvgInstanceMemoryMB)
	}
}

func TestAvgInstanceMemoryMB_ZeroInstances(t *testing.T) {
	mi := ManualInput{
		Name:              "test",
		Clusters:          []ClusterInput{{Name: "c1", HostCount: 4, MemoryGBPerHost: 512, CPUCoresPerHost: 32, DiegoCellCount: 10, DiegoCellMemoryGB: 32, DiegoCellCPU: 4}},
		TotalAppMemoryGB:  150,
		TotalAppInstances: 0, // Zero instances
	}
	state := mi.ToInfrastructureState()

	// Should be 0 (not divide by zero)
	if state.AvgInstanceMemoryMB != 0 {
		t.Errorf("Expected AvgInstanceMemoryMB 0 for zero instances, got %d", state.AvgInstanceMemoryMB)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./models/... -run TestAvgInstanceMemoryMB -v`
Expected: FAIL with "state.AvgInstanceMemoryMB undefined"

**Step 3: Add field and calculation**

In `backend/models/infrastructure.go`, add field to InfrastructureState (around line 82):

```go
AvgInstanceMemoryMB      int            `json:"avg_instance_memory_mb"`
```

In `ToInfrastructureState()` method, add calculation before the return statement (around line 245):

```go
// Calculate average instance memory
if state.TotalAppInstances > 0 {
	state.AvgInstanceMemoryMB = state.TotalAppMemoryGB * 1024 / state.TotalAppInstances
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./models/... -run TestAvgInstanceMemoryMB -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/infrastructure.go backend/models/infrastructure_test.go
git commit -m "feat: add AvgInstanceMemoryMB to InfrastructureState"
```

---

## Task 2: Add ChunkSizeMB to ScenarioInput

**Files:**

- Modify: `backend/models/scenario.go:9-30`

**Step 1: Add field to ScenarioInput**

In `backend/models/scenario.go`, add field after line 29 (after PlatformVMsCPU):

```go
// ChunkSizeMB is an optional override for staging chunk size.
// If 0, uses AvgInstanceMemoryMB from state; if that's 0, defaults to 4096 MB.
ChunkSizeMB int `json:"chunk_size_mb"`
```

**Step 2: Run existing tests to verify no regression**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./models/... -v`
Expected: PASS (adding a field doesn't break existing tests)

**Step 3: Commit**

```bash
git add backend/models/scenario.go
git commit -m "feat: add ChunkSizeMB override to ScenarioInput"
```

---

## Task 3: Add ChunkSizeMB to ScenarioResult

**Files:**

- Modify: `backend/models/scenario.go:53-76`

**Step 1: Add field to ScenarioResult**

In `backend/models/scenario.go`, add field after line 62 (after FreeChunks):

```go
ChunkSizeMB        int     `json:"chunk_size_mb"` // Chunk size used in calculation (for UI transparency)
```

**Step 2: Run existing tests to verify no regression**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./models/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add backend/models/scenario.go
git commit -m "feat: add ChunkSizeMB to ScenarioResult for UI transparency"
```

---

## Task 4: Add resolveChunkSizeMB helper function

**Files:**

- Modify: `backend/services/scenario.go`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
func TestResolveChunkSizeMB(t *testing.T) {
	tests := []struct {
		name       string
		inputMB    int
		stateMB    int
		wantMB     int
	}{
		{"input override wins", 2048, 3072, 2048},
		{"state average used when input is 0", 0, 3072, 3072},
		{"default when both are 0", 0, 0, 4096},
		{"input override even when state available", 1024, 2048, 1024},
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./services/... -run TestResolveChunkSizeMB -v`
Expected: FAIL with "undefined: resolveChunkSizeMB"

**Step 3: Implement the helper function**

Add to `backend/services/scenario.go` after the constants block (around line 23):

```go
// resolveChunkSizeMB returns the effective chunk size in MB.
// Priority: input override → state average → default 4096MB
func resolveChunkSizeMB(inputChunkMB, stateAvgMB int) int {
	if inputChunkMB > 0 {
		return inputChunkMB
	}
	if stateAvgMB > 0 {
		return stateAvgMB
	}
	return 4096 // Default 4GB
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./services/... -run TestResolveChunkSizeMB -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat: add resolveChunkSizeMB helper function"
```

---

## Task 5: Update calculateFull to use configurable chunk size

**Files:**

- Modify: `backend/services/scenario.go:280-397`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Add to `backend/services/scenario_test.go`:

```go
func TestFreeChunksWithConfigurableSize(t *testing.T) {
	state := models.InfrastructureState{
		TotalN1MemoryGB:      26624,
		TotalCellCount:       100,
		PlatformVMsGB:        1000,
		TotalAppMemoryGB:     2000,
		TotalAppInstances:    1000,
		AvgInstanceMemoryMB:  2048, // 2GB average
		Clusters: []models.ClusterState{
			{DiegoCellCount: 100, DiegoCellMemoryGB: 32, DiegoCellCPU: 4},
		},
	}

	calc := NewScenarioCalculator()

	// Test 1: Auto-detect from state (2GB chunks)
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./services/... -run TestFreeChunksWithConfigurableSize -v`
Expected: FAIL (FreeChunks calculation still uses hardcoded 4GB)

**Step 3: Update calculateFull signature and implementation**

In `backend/services/scenario.go`, update `calculateFull`:

1. Add parameters to function signature (around line 280):

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
	targetVCPURatio float64,
	platformVMsCPU int,
	chunkSizeMB int, // NEW: configurable chunk size
) models.ScenarioResult {
```

2. Update free chunks calculation (around line 320-324):

```go
// Free chunks: (capacity - used) / chunkSize
// Convert GB to MB for precision
freeMemoryMB := (appCapacityGB - totalAppMemoryGB) * 1024
freeChunks := freeMemoryMB / chunkSizeMB
if freeChunks < 0 {
	freeChunks = 0
}
```

3. Add ChunkSizeMB to return struct (around line 383):

```go
ChunkSizeMB:        chunkSizeMB,
```

**Step 4: Update callers of calculateFull**

In `CalculateCurrent` (around line 238), add the chunk size parameter:

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
	0,
	0,
	0,
	0,
	resolveChunkSizeMB(0, state.AvgInstanceMemoryMB), // Auto-detect for current
)
```

In `CalculateProposed` (around line 276), add the chunk size parameter:

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
	float64(input.TargetVCPURatio),
	input.PlatformVMsCPU,
	resolveChunkSizeMB(input.ChunkSizeMB, state.AvgInstanceMemoryMB),
)
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./services/... -run TestFreeChunksWithConfigurableSize -v`
Expected: PASS

**Step 6: Run all scenario tests to verify no regression**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./services/... -v`
Expected: PASS (some existing tests may need chunk size adjustments)

**Step 7: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat: use configurable chunk size in free chunks calculation"
```

---

## Task 6: Update existing tests for new chunk calculation

**Files:**

- Modify: `backend/services/scenario_test.go`

**Step 1: Update TestCalculateCurrentScenario**

The existing test at line 49-53 expects hardcoded 4GB chunks. With 0 instances in state, it should fall back to 4096MB default. If state has AvgInstanceMemoryMB set, update the expected value.

Review the test and update the expected FreeChunks value based on the new logic. If `state.AvgInstanceMemoryMB` is 0 (which it is in the existing test because `TotalAppInstances` is 7500 and `TotalAppMemoryGB` is 10500, so avg = 10500\*1024/7500 = 1433 MB):

```go
// Free chunks: (14100 - 10500) * 1024 / 1433 = 2573 (with auto-detect ~1.4GB)
// OR if we want to keep the old behavior for this test, set AvgInstanceMemoryMB explicitly
```

Actually, let's recalculate: 10500 GB _ 1024 / 7500 = 1433 MB per instance.
Free memory = (14100 - 10500) _ 1024 = 3,686,400 MB
Free chunks = 3,686,400 / 1433 = 2572

Update the test expectation or add AvgInstanceMemoryMB = 0 to state to force default.

**Step 2: Run all tests and fix any failures**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/backend && go test ./... -v`

Fix any test failures by updating expected values.

**Step 3: Commit**

```bash
git add backend/services/scenario_test.go
git commit -m "test: update scenario tests for configurable chunk size"
```

---

## Task 7: Update formula-cheatsheet.md documentation

**Files:**

- Modify: `docs/demo/formula-cheatsheet.md:77-90`

**Step 1: Update Free Chunks section**

Replace the Free Chunks section with:

```markdown
### Free Chunks (Staging Capacity)
```

Chunk Size = AvgInstanceMemoryMB (auto) or ChunkSizeMB (override) or 4096 MB (default)
Free Chunks = (App Capacity - App Memory Used) \* 1024 / Chunk Size MB

```

**Chunk size** = typical app instance size, auto-detected from your workload

| Workload Type | Typical Avg Instance | Chunk Size |
|---------------|---------------------|------------|
| Go/Python     | 512-1024 MB         | ~1 GB      |
| Node.js       | 1-2 GB              | ~1.5 GB    |
| Java          | 2-4 GB              | ~4 GB      |

| Chunks | Status      |
| ------ | ----------- |
| >= 20   | Healthy     |
| 10-19  | Limited     |
| < 10   | Constrained |
```

**Step 2: Update Constants section**

Update the Chunk Size constant entry:

```markdown
| Chunk Size | Auto/4 GB | Avg instance memory (auto-detect or 4GB default) |
```

**Step 3: Commit**

```bash
git add docs/demo/formula-cheatsheet.md
git commit -m "docs: update formula cheatsheet for configurable chunk size"
```

---

## Task 8: Update ScenarioResults.jsx to show chunk size

**Files:**

- Modify: `frontend/src/components/ScenarioResults.jsx:228-258`

**Step 1: Update TOOLTIPS constant**

Update the `stagingCapacity` tooltip (around line 14):

```javascript
stagingCapacity: "Available chunks for staging new apps. Chunk size is auto-detected from your average app instance size, or defaults to 4GB. When you cf push, Diego needs a chunk to build your app. Low chunks = deployment queues.",
```

**Step 2: Update the Free Chunks display**

Update the staging capacity card (around line 255-257) to show chunk size:

```jsx
<div className="mt-4 text-center text-xs text-gray-500">
  {proposed.chunk_size_mb
    ? `${(proposed.chunk_size_mb / 1024).toFixed(1)}GB chunks for staging`
    : "4GB chunks for concurrent staging"}
</div>
```

**Step 3: Run frontend tests**

Run: `cd /Users/markalston/code/diego-capacity-analyzer/frontend && bun test`
Expected: PASS

**Step 4: Commit**

```bash
git add frontend/src/components/ScenarioResults.jsx
git commit -m "feat: display chunk size in ScenarioResults"
```

---

## Task 9: Update sample files with avgInstanceMemoryMB

**Files:**

- Modify: `frontend/public/samples/small-foundation.json`
- Modify: `frontend/public/samples/medium-foundation.json`
- Modify: `frontend/public/samples/large-foundation.json`
- Modify: `frontend/public/samples/memory-constrained.json`
- Modify: `frontend/public/samples/cpu-constrained.json`
- Modify: `frontend/public/samples/multi-cluster-enterprise.json`
- Modify: `frontend/public/samples/diego-benchmark-50k.json`
- Modify: `frontend/public/samples/diego-benchmark-250k.json`
- Modify: `frontend/public/samples/THD.json`

**Step 1: Calculate and add avgInstanceMemoryMB to each sample**

For each sample, calculate: `total_app_memory_gb * 1024 / total_app_instances`

| Sample             | App Memory GB | Instances | Avg MB    | Rationale                 |
| ------------------ | ------------- | --------- | --------- | ------------------------- |
| small-foundation   | 150           | 50        | 3072      | Dev/test with bigger apps |
| medium-foundation  | ?             | ?         | calculate | Mixed workloads           |
| large-foundation   | ?             | ?         | calculate | Enterprise Java           |
| memory-constrained | ?             | ?         | calculate | Typical mix               |
| cpu-constrained    | ?             | ?         | calculate | Compute-heavy             |

Note: The field is calculated automatically by the backend from total_app_memory_gb/total_app_instances. The sample files don't need this field added since the backend computes it. However, for clarity in documentation, we can document expected values.

**Step 2: Verify samples work with backend**

Run: `cd /Users/markalston/code/diego-capacity-analyzer && make backend-test`
Expected: PASS

**Step 3: Commit**

```bash
git commit --allow-empty -m "docs: document avgInstanceMemoryMB calculation in samples"
```

---

## Task 10: Run full test suite and verify

**Step 1: Run all backend tests**

Run: `cd /Users/markalston/code/diego-capacity-analyzer && make backend-test`
Expected: PASS

**Step 2: Run all frontend tests**

Run: `cd /Users/markalston/code/diego-capacity-analyzer && make frontend-test`
Expected: PASS

**Step 3: Run linting**

Run: `cd /Users/markalston/code/diego-capacity-analyzer && make lint`
Expected: PASS

**Step 4: Manual verification**

1. Start backend: `make backend-run`
2. Start frontend: `make frontend-dev`
3. Load a sample file and verify free chunks display shows chunk size
4. Verify API response includes `chunk_size_mb` in scenario results

**Step 5: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address any remaining issues from testing"
```

---

## Summary

| Task | Description                                    | Files Modified                            |
| ---- | ---------------------------------------------- | ----------------------------------------- |
| 1    | Add AvgInstanceMemoryMB to InfrastructureState | infrastructure.go, infrastructure_test.go |
| 2    | Add ChunkSizeMB to ScenarioInput               | scenario.go                               |
| 3    | Add ChunkSizeMB to ScenarioResult              | scenario.go                               |
| 4    | Add resolveChunkSizeMB helper                  | scenario.go, scenario_test.go             |
| 5    | Update calculateFull for configurable chunks   | scenario.go, scenario_test.go             |
| 6    | Update existing tests                          | scenario_test.go                          |
| 7    | Update documentation                           | formula-cheatsheet.md                     |
| 8    | Update frontend display                        | ScenarioResults.jsx                       |
| 9    | Verify sample files                            | samples/\*.json                           |
| 10   | Full test suite verification                   | -                                         |
