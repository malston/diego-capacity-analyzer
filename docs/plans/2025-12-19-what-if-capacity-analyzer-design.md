# What-If Capacity Analyzer Design

**Date:** 2025-12-19
**Status:** Approved

---

## Overview

Add what-if analysis capability to the Diego Capacity Analyzer, allowing operators to model VM size changes and cell count adjustments before making infrastructure decisions. The tool shows current state vs proposed state with tradeoff warnings.

## Goals

1. Show current cell density and capacity metrics
2. Model what-if scenarios (VM size changes, cell count changes)
3. Display tradeoffs (capacity vs redundancy vs N-1 utilization)
4. Support multi-cluster environments with aggregate and per-cluster views

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend                                 │
│  ┌─────────────────┐    ┌──────────────────────────────────┐    │
│  │ Current State   │    │ What-If Scenario                 │    │
│  │ (existing)      │    │ ┌────────────┐ ┌──────────────┐  │    │
│  │                 │    │ │ VM Size ▼  │ │ Cell Count   │  │    │
│  │                 │    │ │ 4×32       │ │ [470]        │  │    │
│  │                 │    │ │ 4×64       │ └──────────────┘  │    │
│  │                 │    │ │ 8×64       │ ┌──────────────┐  │    │
│  │                 │    │ └────────────┘ │ Cluster ▼    │  │    │
│  │                 │    │                │ All / specific│  │    │
│  │                 │    │                └──────────────┘  │    │
│  └─────────────────┘    └──────────────────────────────────┘    │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Comparison Table                              │  │
│  │  Metric              │ Current (4×32) │ Proposed (4×64)   │  │
│  │  ─────────────────────────────────────────────────────────│  │
│  │  Cell count          │ 470            │ 235               │  │
│  │  App capacity        │ 12.7 TB        │ 12.7 TB           │  │
│  │  Utilization %       │ 83%            │ 76%               │  │
│  │  Free chunks         │ 547            │ 820               │  │
│  │  N-1 utilization %   │ 71%            │ 71%               │  │
│  │  Fault impact        │ 16 apps/cell   │ 32 apps/cell      │  │
│  │  Instances/cell      │ 16             │ 32                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Per-Cluster Breakdown (expandable)                        │  │
│  │  cluster-01: 8 hosts, 250 cells, 68% N-1                   │  │
│  │  cluster-02: 7 hosts, 220 cells, 74% N-1                   │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                         Backend                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ BOSH Client  │  │ CF Client    │  │ vSphere Client (NEW) │   │
│  │ (existing)   │  │ (existing)   │  │ via govmomi          │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│                           │                                      │
│                    ┌──────┴───────┐                              │
│                    │ Scenario     │                              │
│                    │ Calculator   │                              │
│                    │ (NEW)        │                              │
│                    └──────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

## vSphere Integration

### Credential Flow

Extract vCenter credentials from Ops Manager Director tile:

```bash
om staged-director-config --no-redact | yq '.iaas-configurations[0]'
# Returns: vcenter_host, vcenter_username, vcenter_password, datacenter, clusters
```

### Data Retrieved via govmomi

| Data | API Call | Used For |
|------|----------|----------|
| Host count | `ClusterComputeResource.Host` | N-1 calculation |
| Host memory | `HostSystem.Summary.Hardware.MemorySize` | Total cluster capacity |
| Host CPU | `HostSystem.Summary.Hardware.NumCpuCores` | CPU constraints |
| VM inventory | `VirtualMachine` list | Map Diego cells to ESXi hosts |
| Resource pools | `ResourcePool` | Validate allocation limits |

### Caching

vSphere data cached for 5 minutes (vs 30s for BOSH/CF data).

## Data Models

### Infrastructure State

```go
type InfrastructureState struct {
    Clusters        []ClusterState `json:"clusters"`
    TotalMemoryGB   int            `json:"total_memory_gb"`
    TotalN1MemoryGB int            `json:"total_n1_memory_gb"`
    TotalHostCount  int            `json:"total_host_count"`
    PlatformVMsGB   int            `json:"platform_vms_gb"`
    Timestamp       time.Time      `json:"timestamp"`
    Cached          bool           `json:"cached"`
}

type ClusterState struct {
    Name           string `json:"name"`
    HostCount      int    `json:"host_count"`
    MemoryGB       int    `json:"memory_gb"`
    CPUCores       int    `json:"cpu_cores"`
    N1MemoryGB     int    `json:"n1_memory_gb"`
    UsableMemoryGB int    `json:"usable_memory_gb"` // N1 * 0.9
    DiegoCellCount int    `json:"diego_cell_count"`
}
```

### Scenario Input/Output

```go
type ScenarioInput struct {
    ProposedCellMemoryGB int    `json:"proposed_cell_memory_gb"`
    ProposedCellCPU      int    `json:"proposed_cell_cpu"`
    ProposedCellCount    int    `json:"proposed_cell_count"`
    TargetCluster        string `json:"target_cluster"` // Empty = all clusters
}

type ScenarioResult struct {
    CellCount        int     `json:"cell_count"`
    CellSize         string  `json:"cell_size"`         // e.g., "4×32"
    AppCapacityGB    int     `json:"app_capacity_gb"`
    UtilizationPct   float64 `json:"utilization_pct"`
    FreeChunks       int     `json:"free_chunks"`
    N1UtilizationPct float64 `json:"n1_utilization_pct"`
    FaultImpact      int     `json:"fault_impact"`
    InstancesPerCell float64 `json:"instances_per_cell"`
}

type ScenarioComparison struct {
    Current  ScenarioResult    `json:"current"`
    Proposed ScenarioResult    `json:"proposed"`
    Warnings []ScenarioWarning `json:"warnings"`
    Delta    ScenarioDelta     `json:"delta"`
}

type ScenarioWarning struct {
    Severity string `json:"severity"` // "info", "warning", "critical"
    Message  string `json:"message"`
}

type ScenarioDelta struct {
    CapacityChangeGB     int     `json:"capacity_change_gb"`
    UtilizationChangePct float64 `json:"utilization_change_pct"`
    RedundancyChange     string  `json:"redundancy_change"` // "improved", "reduced", "unchanged"
}
```

## Calculation Formulas

| Metric | Formula |
|--------|---------|
| App capacity | `CellCount × (CellMemoryGB - 5)` where 5GB = Garden overhead |
| Cell utilization % | `TotalAppMemoryGB / AppCapacity × 100` |
| Free chunks (4GB) | `(AppCapacity - TotalAppMemoryGB) / 4` |
| N-1 utilization % | `(CellCount × CellMemoryGB + PlatformVMsGB) / N1MemoryGB × 100` |
| Fault impact | `TotalAppInstances / CellCount` (avg apps per cell) |
| Instances/cell | `TotalAppInstances / CellCount` |

### Tradeoff Warning Thresholds

| Condition | Severity | Message |
|-----------|----------|---------|
| N1UtilizationPct > 85% | critical | "Exceeds N-1 capacity safety margin" |
| N1UtilizationPct > 75% | warning | "Approaching N-1 capacity limits" |
| FreeChunks < 200 | critical | "Critical: Low staging capacity" |
| FreeChunks < 400 | warning | "Low staging capacity" |
| Cell count reduced > 50% | warning | "Significant redundancy reduction" |
| UtilizationPct > 90% | critical | "Cell utilization critically high" |
| UtilizationPct > 80% | warning | "Cell utilization elevated" |

## API Endpoints

### GET /api/infrastructure

Returns vSphere cluster information.

**Response:**
```json
{
    "clusters": [
        {
            "name": "cluster-01",
            "host_count": 8,
            "memory_gb": 16384,
            "cpu_cores": 256,
            "n1_memory_gb": 14336,
            "usable_memory_gb": 12902,
            "diego_cell_count": 250
        },
        {
            "name": "cluster-02",
            "host_count": 7,
            "memory_gb": 14336,
            "cpu_cores": 224,
            "n1_memory_gb": 12288,
            "usable_memory_gb": 11059,
            "diego_cell_count": 220
        }
    ],
    "total_memory_gb": 30720,
    "total_n1_memory_gb": 26624,
    "total_host_count": 15,
    "platform_vms_gb": 4800,
    "cached": true,
    "timestamp": "2025-12-19T10:30:00Z"
}
```

### POST /api/scenario/compare

Calculate scenario comparison.

**Request:**
```json
{
    "proposed_cell_memory_gb": 64,
    "proposed_cell_cpu": 4,
    "proposed_cell_count": 235,
    "target_cluster": ""
}
```

**Response:**
```json
{
    "current": {
        "cell_count": 470,
        "cell_size": "4×32",
        "app_capacity_gb": 12690,
        "utilization_pct": 83.0,
        "free_chunks": 547,
        "n1_utilization_pct": 71.0,
        "fault_impact": 16,
        "instances_per_cell": 16.0
    },
    "proposed": {
        "cell_count": 235,
        "cell_size": "4×64",
        "app_capacity_gb": 13865,
        "utilization_pct": 76.0,
        "free_chunks": 820,
        "n1_utilization_pct": 71.0,
        "fault_impact": 32,
        "instances_per_cell": 32.0
    },
    "warnings": [
        {
            "severity": "warning",
            "message": "Cell count reduced by 50% - failure of one cell affects 2× more apps"
        }
    ],
    "delta": {
        "capacity_change_gb": 1175,
        "utilization_change_pct": -7.0,
        "redundancy_change": "reduced"
    }
}
```

## Frontend Components

### Component Structure

```
TASCapacityAnalyzer.jsx (existing)
└── ScenarioAnalyzer.jsx (NEW)
    ├── ScenarioInputForm
    │   ├── VM size dropdown (presets)
    │   ├── Cell count input
    │   └── Cluster selector (All / specific)
    ├── ComparisonTable
    │   └── 7 metrics with current/proposed/change columns
    ├── WarningsList
    │   └── Severity-colored warning badges
    └── ClusterBreakdown (expandable)
        └── Per-cluster metrics
```

### VM Size Presets

```javascript
const VM_SIZE_PRESETS = [
  { label: "4 vCPU × 32 GB", cpu: 4, memoryGB: 32 },
  { label: "4 vCPU × 64 GB", cpu: 4, memoryGB: 64 },
  { label: "8 vCPU × 64 GB", cpu: 8, memoryGB: 64 },
  { label: "8 vCPU × 128 GB", cpu: 8, memoryGB: 128 },
  { label: "Custom...", cpu: null, memoryGB: null },
];
```

### Visual Indicators

| Indicator | Meaning |
|-----------|---------|
| ▲ (green) | Improvement |
| ▼ (red) | Degradation |
| — (gray) | No change |

## Implementation Phases

### Phase 1: vSphere Client

1. Add `services/vsphere.go` with govmomi integration
2. Extract vCenter credentials from `om staged-director-config`
3. Implement cluster/host inventory queries
4. Support multiple clusters
5. Add `GET /api/infrastructure` endpoint
6. Unit tests with mock vSphere responses

### Phase 2: Scenario Calculator

1. Add `services/scenario.go` with calculation logic
2. Add `models/scenario.go` for input/output types
3. Add `POST /api/scenario/compare` endpoint
4. Implement warning thresholds
5. Unit tests validating formulas

### Phase 3: Frontend

1. Add `ScenarioAnalyzer.jsx` component
2. Implement VM size presets and cluster selector
3. Build comparison table with change indicators
4. Add expandable per-cluster breakdown
5. Style warnings with severity colors
6. Integrate into existing dashboard

### Phase 4: Polish

1. Add "Export to Markdown" button
2. Tune caching durations
3. Error handling for vCenter connectivity
4. Loading states during API calls

## Testing Strategy

### Backend Unit Tests

- `vsphere_test.go`: Mock govmomi responses, test credential extraction
- `scenario_test.go`: Validate calculations against known examples from capacity doc

### Integration Tests

- End-to-end API tests with test fixtures
- Verify warning threshold triggering

### Frontend Tests

- Component rendering tests
- API integration tests with mocked responses

## Dependencies

### New Go Dependencies

```go
require github.com/vmware/govmomi v0.34.0
```

### Existing Dependencies (unchanged)

- React 18
- Vite 5
- Tailwind CSS
- Recharts
