# Backend API Reference

Base URL: `http://localhost:8080` (default)

All endpoints return JSON responses and support CORS.

---

## Health & Status

### GET /api/health

Health check endpoint.

**Response:**

```json
{
  "cf_api": "ok",
  "bosh_api": "ok",
  "cache_status": {
    "cells_cached": false,
    "apps_cached": false
  }
}
```

| Field | Description |
|-------|-------------|
| `cf_api` | CF API connectivity status |
| `bosh_api` | BOSH API status (`ok` or `not_configured`) |
| `cache_status` | Current cache state |

---

## Dashboard

### GET /api/dashboard

Returns live dashboard data from CF and BOSH APIs.

**Response:**

```json
{
  "cells": [
    {
      "name": "diego_cell/0",
      "isolation_segment": "shared",
      "memory_mb": 32768,
      "allocated_mb": 24576,
      "used_mb": 18432,
      "cpu_percent": 45.2
    }
  ],
  "apps": [
    {
      "guid": "abc-123",
      "name": "my-app",
      "instances": 3,
      "requested_mb": 1024,
      "actual_mb": 780,
      "isolation_segment": "shared"
    }
  ],
  "segments": [
    {
      "guid": "seg-123",
      "name": "shared"
    }
  ],
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "cached": false,
    "bosh_available": true
  }
}
```

**Data Sources:**

- `cells`: BOSH API (Diego cell VMs and vitals)
- `apps`: CF API (applications and process stats)
- `segments`: CF API (isolation segments)

---

## Infrastructure

### GET /api/infrastructure

Returns live infrastructure data from vSphere.

**Prerequisites:** Requires vSphere environment variables:

- `VSPHERE_HOST`
- `VSPHERE_USERNAME`
- `VSPHERE_PASSWORD`
- `VSPHERE_DATACENTER`

**Response:**

```json
{
  "name": "vcenter.example.com",
  "source": "vsphere",
  "timestamp": "2024-01-15T10:30:00Z",
  "cached": false,
  "clusters": [
    {
      "name": "TAS-Cluster",
      "host_count": 4,
      "memory_gb": 512,
      "cpu_cores": 128,
      "memory_gb_per_host": 128,
      "cpu_cores_per_host": 32,
      "ha_admission_control_percentage": 25,
      "ha_usable_memory_gb": 384,
      "ha_usable_cpu_cores": 96,
      "ha_host_failures_survived": 1,
      "cells": [
        {
          "name": "diego_cell/0",
          "memory_gb": 64,
          "cpu": 8,
          "disk_gb": 200
        }
      ]
    }
  ],
  "total_host_count": 4,
  "total_cell_count": 10,
  "total_cell_memory_gb": 640,
  "total_cell_cpu": 80,
  "total_cell_disk_gb": 2000,
  "total_app_memory_gb": 450,
  "total_app_disk_gb": 900,
  "total_app_instances": 150,
  "platform_vms_gb": 64
}
```

**Error (503):** vSphere not configured

```json
{
  "error": "vSphere not configured. Set VSPHERE_HOST, VSPHERE_USERNAME, VSPHERE_PASSWORD, and VSPHERE_DATACENTER environment variables.",
  "code": 503
}
```

---

### POST /api/infrastructure/manual

Set infrastructure state from manual input (JSON upload or form data).

**Request Body:**

```json
{
  "name": "My Infrastructure",
  "clusters": [
    {
      "name": "TAS-Cluster",
      "host_count": 4,
      "memory_gb_per_host": 128,
      "cpu_cores_per_host": 32,
      "ha_admission_control_percentage": 25,
      "cells": [
        {
          "name": "diego_cell/0",
          "memory_gb": 64,
          "cpu": 8,
          "disk_gb": 200
        }
      ]
    }
  ],
  "platform_vms_gb": 64,
  "total_app_memory_gb": 450,
  "total_app_disk_gb": 900,
  "total_app_instances": 150
}
```

**Response:** Returns computed `InfrastructureState` (same format as GET /api/infrastructure)

---

### POST /api/infrastructure/state

Set infrastructure state directly (accepts full InfrastructureState object).

**Request Body:** Full `InfrastructureState` object (same format as GET /api/infrastructure response)

**Response:** Returns the stored state

---

### GET /api/infrastructure/status

Returns current infrastructure data source status and capacity metrics.

**Response:**

```json
{
  "vsphere_configured": true,
  "has_data": true,
  "source": "vsphere",
  "name": "vcenter.example.com",
  "cluster_count": 2,
  "host_count": 8,
  "cell_count": 20,
  "timestamp": "2024-01-15T10:30:00Z",
  "constraining_resource": "memory",
  "bottleneck_summary": "Memory is the primary constraint at 78% utilization",
  "memory_utilization": 78.5,
  "n1_capacity_percent": 72.0,
  "n1_status": "ok",
  "ha_min_host_failures_survived": 1,
  "ha_status": "ok"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `vsphere_configured` | boolean | Whether vSphere credentials are configured |
| `has_data` | boolean | Whether infrastructure data has been loaded |
| `source` | string | Data source: "vsphere", "manual", or "json" |
| `name` | string | Infrastructure name (vCenter hostname or custom) |
| `cluster_count` | integer | Number of clusters |
| `host_count` | integer | Total ESXi hosts |
| `cell_count` | integer | Total Diego cells |
| `timestamp` | string | When data was loaded (ISO 8601) |
| `constraining_resource` | string | Primary bottleneck: "memory", "CPU", or "disk" |
| `bottleneck_summary` | string | Human-readable bottleneck description |
| `memory_utilization` | float | Host memory utilization percentage |
| `n1_capacity_percent` | float | Percentage of N-1 memory capacity used by cells |
| `n1_status` | string | N-1 capacity status: "ok", "warning", "critical", or "unavailable" |
| `ha_min_host_failures_survived` | integer | Number of host failures the cluster can survive |
| `ha_status` | string | HA status: "ok" or "at-risk" |

**Note:** `n1_status` is set to "unavailable" and `n1_capacity_percent` to 0 for single-host clusters where N-1 capacity cannot be calculated.

---

### GET /api/infrastructure/apps

Returns detailed per-app breakdown of memory and instance allocation from Cloud Foundry.

**Prerequisites:** CF API credentials must be configured via `CF_API_URL`, `CF_USERNAME`, and `CF_PASSWORD` environment variables.

**Response:**

```json
{
  "total_app_memory_gb": 5,
  "total_app_instances": 17,
  "apps": [
    {
      "name": "my-app",
      "guid": "abc-123-def",
      "instances": 3,
      "requested_mb": 1536,
      "actual_mb": 512,
      "isolation_segment": "default"
    },
    {
      "name": "worker-app",
      "guid": "xyz-456-ghi",
      "instances": 2,
      "requested_mb": 2048,
      "actual_mb": 1024,
      "isolation_segment": "shared"
    }
  ]
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `total_app_memory_gb` | integer | Total requested memory across all apps (GB) |
| `total_app_instances` | integer | Total running instances across all apps |
| `apps` | array | Per-app details |
| `apps[].name` | string | Application name |
| `apps[].guid` | string | CF application GUID |
| `apps[].instances` | integer | Number of running instances |
| `apps[].requested_mb` | integer | Total requested memory (instances × memory per instance) |
| `apps[].actual_mb` | integer | Actual memory usage from Log Cache (if available) |
| `apps[].isolation_segment` | string | Isolation segment name ("default" if none assigned) |

**Error Responses:**

| Code | Description |
|------|-------------|
| 503 | CF API not configured |
| 503 | CF authentication failed |
| 500 | Failed to fetch apps |

---

## Capacity Planning

### POST /api/infrastructure/planning

Calculate maximum deployable Diego cells given IaaS capacity constraints.

**Prerequisites:** Infrastructure data must be loaded first via `/api/infrastructure` or `/api/infrastructure/manual`

**Request Body:**

```json
{
  "cell_memory_gb": 64,
  "cell_cpu": 8,
  "overhead_pct": 7
}
```

| Field | Type | Description |
|-------|------|-------------|
| `cell_memory_gb` | int | Desired memory per Diego cell (GB) |
| `cell_cpu` | int | Desired vCPUs per Diego cell |
| `overhead_pct` | float | Memory overhead percentage (default: 7) |

**Response:**

```json
{
  "result": {
    "max_cells_by_memory": 12,
    "max_cells_by_cpu": 15,
    "deployable_cells": 12,
    "bottleneck": "memory",
    "memory_used_gb": 768,
    "memory_avail_gb": 896,
    "cpu_used": 96,
    "cpu_avail": 120,
    "memory_util_pct": 85.7,
    "cpu_util_pct": 80.0,
    "headroom_cells": 2
  },
  "recommendations": [
    {
      "action": "add_hosts",
      "description": "Add 2 hosts to increase capacity",
      "impact": "Adds 256 GB memory capacity"
    }
  ]
}
```

---

## Scenario Analysis

### POST /api/scenario/compare

Compare current infrastructure state against a proposed configuration.

**Prerequisites:** Infrastructure data must be loaded first

**Request Body:**

```json
{
  "proposed_cell_memory_gb": 64,
  "proposed_cell_cpu": 8,
  "proposed_cell_disk_gb": 200,
  "proposed_cell_count": 15,
  "target_cluster": "",
  "selected_resources": ["memory", "cpu", "disk"],
  "overhead_pct": 7,
  "additional_app": {
    "name": "new-service",
    "instances": 10,
    "memory_gb": 2,
    "disk_gb": 4
  },
  "tps_curve": [
    {"cells": 1, "tps": 284},
    {"cells": 3, "tps": 1964},
    {"cells": 100, "tps": 1389}
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `proposed_cell_memory_gb` | int | Proposed memory per cell (GB) |
| `proposed_cell_cpu` | int | Proposed vCPUs per cell |
| `proposed_cell_disk_gb` | int | Proposed disk per cell (GB) |
| `proposed_cell_count` | int | Proposed number of cells |
| `target_cluster` | string | Target cluster (empty = all) |
| `selected_resources` | array | Resources to analyze: `memory`, `cpu`, `disk` |
| `overhead_pct` | float | Memory overhead % (default: 7) |
| `additional_app` | object | Optional hypothetical app to model |
| `tps_curve` | array | Optional custom TPS performance curve |

**Response:**

```json
{
  "current": {
    "cell_count": 10,
    "cell_memory_gb": 64,
    "cell_cpu": 8,
    "total_capacity_gb": 640,
    "app_capacity_gb": 595,
    "utilization_pct": 75.6,
    "free_chunks": 450,
    "tps": 1800,
    "tps_status": "optimal",
    "fault_impact": 15,
    "n1_utilization_pct": 72.0
  },
  "proposed": {
    "cell_count": 15,
    "cell_memory_gb": 64,
    "cell_cpu": 8,
    "total_capacity_gb": 960,
    "app_capacity_gb": 893,
    "utilization_pct": 50.4,
    "free_chunks": 680,
    "tps": 1650,
    "tps_status": "optimal",
    "fault_impact": 10,
    "n1_utilization_pct": 68.0
  },
  "delta": {
    "cell_count": 5,
    "capacity_gb": 320,
    "utilization_pct": -25.2,
    "tps": -150,
    "fault_impact": -5
  },
  "warnings": [
    {
      "severity": "warning",
      "message": "Cell count (15) may cause scheduling latency (~1650 TPS)",
      "metric": "tps"
    }
  ],
  "recommendations": [
    {
      "action": "resize_cells",
      "priority": 2,
      "description": "Consider larger cells to improve TPS",
      "impact": "Reduces scheduler coordination overhead"
    }
  ]
}
```

---

## Analysis

### GET /api/bottleneck

Returns multi-resource bottleneck analysis.

**Prerequisites:** Infrastructure data must be loaded first

**Response:**

```json
{
  "constraining_resource": "memory",
  "resources": [
    {
      "name": "memory",
      "utilization_pct": 78.5,
      "status": "warning",
      "headroom_gb": 142
    },
    {
      "name": "cpu",
      "utilization_pct": 45.2,
      "status": "good",
      "headroom_cores": 66
    },
    {
      "name": "disk",
      "utilization_pct": 32.1,
      "status": "good",
      "headroom_gb": 1360
    }
  ],
  "summary": "Memory is the primary constraint at 78% utilization"
}
```

---

### GET /api/recommendations

Returns upgrade path recommendations based on current bottlenecks.

**Prerequisites:** Infrastructure data must be loaded first

**Response:**

```json
{
  "constraining_resource": "memory",
  "recommendations": [
    {
      "action": "add_cells",
      "priority": 1,
      "description": "Add 4 Diego cells",
      "impact": "Adds 256 GB memory capacity"
    },
    {
      "action": "resize_cells",
      "priority": 2,
      "description": "Resize cells from 64 GB to 128 GB",
      "impact": "Doubles per-cell capacity, reduces scheduler overhead"
    },
    {
      "action": "add_hosts",
      "priority": 3,
      "description": "Add 2 ESXi hosts",
      "impact": "Adds infrastructure capacity and improves N-1 tolerance"
    }
  ]
}
```

---

## Error Responses

All endpoints return errors in a consistent format:

```json
{
  "error": "Error message describing what went wrong",
  "details": "Additional details (optional)",
  "code": 400
}
```

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid input |
| 405 | Method Not Allowed |
| 500 | Internal Server Error |
| 503 | Service Unavailable - External service not configured |

---

## Caching

The backend implements in-memory caching with configurable TTLs:

| Cache Key | Default TTL | Environment Variable |
|-----------|-------------|---------------------|
| Dashboard data | 30s | `DASHBOARD_CACHE_TTL` |
| vSphere infrastructure | 300s | `VSPHERE_CACHE_TTL` |
| General cache | 300s | `CACHE_TTL` |

Cached responses include `"cached": true` in the metadata.

---

## Manual Data Collection

When using the manual infrastructure input endpoint (`POST /api/infrastructure/manual`), you may need to collect app-related metrics yourself. This section documents how to obtain these values.

### Data Sources

The following fields require app workload data:

| Field | Description |
|-------|-------------|
| `total_app_memory_gb` | Total memory allocated to all application instances |
| `total_app_disk_gb` | Total disk allocated to all application instances |
| `total_app_instances` | Total number of running application instances |

### Option 1: From Healthwatch / Aria Operations Dashboards

If you have Healthwatch or Aria Operations for Applications deployed, these metrics are already collected.

**Healthwatch (Grafana/Prometheus)**:

```promql
# Total allocated app memory across all Diego cells (result in GB)
sum(rep_CapacityAllocatedMemory) / 1024
```

**Aria Operations for Applications (Wavefront)**:

```bash
# Total allocated app memory across all Diego cells (result in GB)
sum(ts("tas.rep.CapacityAllocatedMemory")) / 1024
```

**Diego Rep Metrics Reference**:

| Metric | Origin | Units | Description |
|--------|--------|-------|-------------|
| `rep.CapacityTotalMemory` | rep | MiB | Max memory available for app allocation |
| `rep.CapacityRemainingMemory` | rep | MiB | Remaining allocatable memory |
| `rep.CapacityAllocatedMemory` | rep | MiB | Memory allocated to containers |

Formula: `TotalMemory = AllocatedMemory + RemainingMemory`

### Option 2: CF CLI Commands

Use these commands to collect app metrics directly from the CF API:

```bash
# Authenticate to CF
cf login -a https://api.sys.example.com

# Get total allocated app memory (sum of memory_in_mb × instances for all processes)
cf curl "/v3/processes" | jq '[.resources[] | .memory_in_mb * .instances] | add'
# Output: total MB (e.g., 10752)

# For large foundations, handle pagination:
total_mb=0
next_url="/v3/processes?per_page=5000"
while [ "$next_url" != "null" ]; do
  response=$(cf curl "$next_url")
  page_total=$(echo "$response" | jq '[.resources[] | .memory_in_mb * .instances] | add // 0')
  total_mb=$((total_mb + page_total))
  next_url=$(echo "$response" | jq -r '.pagination.next.href // "null"')
done
echo "Total app memory: $((total_mb / 1024)) GB"

# Get total app instances
cf curl "/v3/processes?per_page=5000" | jq '[.resources[].instances] | add'

# Get total disk
cf curl "/v3/processes?per_page=5000" | jq '[.resources[] | .disk_in_mb * .instances] | add'
# Convert to GB: divide by 1024
```

### Understanding the Numbers

| Field | CF API Source | Calculation |
|-------|--------------|-------------|
| `total_app_memory_gb` | `/v3/processes` | Sum of (memory_in_mb × instances), convert to GB |
| `total_app_instances` | `/v3/processes` | Sum of instances |
| `total_app_disk_gb` | `/v3/processes` | Sum of (disk_in_mb × instances), convert to GB |

### Example Manual JSON Input

After collecting the values above:

```json
{
  "name": "Production TAS",
  "clusters": [
    {
      "name": "TAS-Cluster",
      "host_count": 8,
      "memory_gb_per_host": 2048,
      "cpu_cores_per_host": 64,
      "diego_cell_count": 250,
      "diego_cell_memory_gb": 32,
      "diego_cell_cpu": 4
    }
  ],
  "platform_vms_gb": 4800,
  "total_app_memory_gb": 10,
  "total_app_disk_gb": 50,
  "total_app_instances": 42
}
```

### Automatic Enrichment

When using live vSphere data (`GET /api/infrastructure`), the backend automatically enriches infrastructure data with app metrics from the CF API if CF credentials are configured. This eliminates the need for manual data collection in most cases.
