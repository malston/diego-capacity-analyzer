# Configurable Chunk Size for Free Staging Capacity

**Date:** 2026-01-30
**Status:** Approved (with post-implementation fix)

> **Bug Fix (2026-01-30):** The original design incorrectly used `AvgInstanceMemoryMB` (average per-instance memory, typically 100-500MB) as chunk size. This was fixed to use `MaxInstanceMemoryMB` (largest app memory limit) with a 1GB minimum floor. See "Resolution Order" section for corrected behavior.

## Problem

The free chunks calculation is hardcoded to 4GB (`ChunkSizeGB = 4` in `scenario.go`). This assumes Java-heavy workloads where 4GB is typical. Platforms running mostly Go, Python, or Ruby apps have smaller instance footprints, making the 4GB assumption inaccurate.

## Solution

Make chunk size configurable with auto-detection from live CF data and manual override capability.

---

## Data Model Changes

### InfrastructureState (derived from live data)

```go
MaxInstanceMemoryMB int  // Largest per-instance memory limit across all apps
AvgInstanceMemoryMB int  // TotalAppMemoryGB * 1024 / TotalAppInstances (for reference only)
```

`MaxInstanceMemoryMB` is calculated from live CF API data by finding the largest per-instance memory allocation.

### ScenarioInput (user override)

```go
ChunkSizeMB int  // Optional override; 0 means use auto-detect or default
```

### ScenarioResult (transparency)

```go
ChunkSizeMB int  // The chunk size used in calculation (for UI display)
```

### Resolution Order

1. `ScenarioInput.ChunkSizeMB` if provided (> 0) - user override, used as-is
2. `InfrastructureState.MaxInstanceMemoryMB` if available (> 0), with minimum floor of 1024 MB
3. Fall back to 4096 MB (default)

> **Note:** The original design used `AvgInstanceMemoryMB` which is typically 100-500MB for CF deployments and far too small for staging. The corrected implementation uses `MaxInstanceMemoryMB` (the largest app's memory limit) which better represents staging requirements.

---

## Calculation Changes

### New helper function in scenario.go

```go
const MinChunkSizeMB = 1024  // 1GB minimum for staging

// resolveChunkSizeMB returns the effective chunk size in MB.
// Priority: input override -> state max instance memory (min 1GB) -> default 4096MB
func resolveChunkSizeMB(inputChunkMB, stateMaxMB int) int {
    if inputChunkMB > 0 {
        return inputChunkMB  // User override, respect exactly
    }
    if stateMaxMB > 0 {
        if stateMaxMB < MinChunkSizeMB {
            return MinChunkSizeMB  // Enforce minimum floor
        }
        return stateMaxMB
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

Add `maxInstanceMemoryMB` to sample infrastructure files (represents the largest app's memory limit):

| Sample                  | Max Instance | Rationale                  |
| ----------------------- | ------------ | -------------------------- |
| small-foundation.json   | 1024         | Small apps, Go/Python      |
| medium-foundation.json  | 2048         | Mixed workloads, some Java |
| large-foundation.json   | 4096         | Java-heavy enterprise      |
| memory-constrained.json | 2048         | Typical mix with some Java |

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
