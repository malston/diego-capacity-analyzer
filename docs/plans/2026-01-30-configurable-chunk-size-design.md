# Configurable Chunk Size for Free Staging Capacity

**Date:** 2026-01-30
**Status:** Approved

## Problem

The free chunks calculation is hardcoded to 4GB (`ChunkSizeGB = 4` in `scenario.go`). This assumes Java-heavy workloads where 4GB is typical. Platforms running mostly Go, Python, or Ruby apps have smaller instance footprints, making the 4GB assumption inaccurate.

## Solution

Make chunk size configurable with auto-detection from live CF data and manual override capability.

---

## Data Model Changes

### InfrastructureState (derived from live data)

```go
AvgInstanceMemoryMB int  // TotalAppMemoryGB * 1024 / TotalAppInstances
```

Calculated automatically when CF API data is available.

### ScenarioInput (user override)

```go
ChunkSizeMB int  // Optional override; 0 means use auto-detect or default
```

### ScenarioResult (transparency)

```go
ChunkSizeMB int  // The chunk size used in calculation (for UI display)
```

### Resolution Order

1. `ScenarioInput.ChunkSizeMB` if provided (> 0)
2. `InfrastructureState.AvgInstanceMemoryMB` if available (> 0)
3. Fall back to 4096 MB (current default)

---

## Calculation Changes

### New helper function in scenario.go

```go
// resolveChunkSizeMB returns the effective chunk size in MB.
// Priority: input override -> state average -> default 4096MB
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

### calculateFull() changes

- Add `chunkSizeMB int` parameter
- Update calculation to use MB precision:

```go
// Before
freeChunks := (appCapacityGB - totalAppMemoryGB) / ChunkSizeGB

// After
freeChunks := (appCapacityGB*1024 - totalAppMemoryGB*1024) / chunkSizeMB
```

---

## API Changes

### POST /api/v1/scenario/compare

Add optional field to request body:

```json
{
  "chunk_size_mb": 2048
}
```

No changes to response structure beyond the new `chunk_size_mb` field in results.

---

## Frontend Changes

### WhatIfPanel.jsx

- Add "Chunk Size" input in advanced settings section
- Default: empty (auto-detect)
- When live data available, show placeholder: "Auto (2.1 GB avg)"
- Input accepts MB value for override

### ScenarioResults.jsx

- Update "Free Chunks" display to show chunk size basis
- Example: "Free Chunks: 47 (@ 2GB)"
- Or use tooltip: "Based on 2GB average instance size"

---

## Sample Data Updates

Add `avgInstanceMemoryMB` to sample infrastructure files:

| Sample                  | Avg Instance | Rationale             |
| ----------------------- | ------------ | --------------------- |
| small-foundation.json   | 1024         | Small apps, Go/Python |
| medium-foundation.json  | 2048         | Mixed workloads       |
| large-foundation.json   | 4096         | Java-heavy enterprise |
| memory-constrained.json | 2048         | Typical mix           |

---

## Testing

1. Unit tests for `resolveChunkSizeMB()` helper
2. Update `scenario_test.go` to test chunk size resolution order
3. Frontend tests for input validation and display

---

## Backward Compatibility

- Existing API calls without `chunk_size_mb` continue to work (falls back to 4GB)
- Existing sample files work (missing field defaults to 4GB)
- No breaking changes
