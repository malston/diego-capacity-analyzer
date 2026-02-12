# Diagnostic Heuristics

Diego Capacity Analyzer provides threshold-based diagnostic heuristics across
several resource dimensions. This document describes each heuristic, its
thresholds, and the remediation guidance the system generates.

## Multi-Resource Bottleneck Detection

**Endpoint:** `GET /api/v1/bottleneck`

Ranks three resource dimensions by utilization and identifies which is the
platform constraint:

| Resource | Metric                                                  | Unit  |
| -------- | ------------------------------------------------------- | ----- |
| Memory   | App memory allocated vs. total cell memory capacity     | GB    |
| CPU      | vCPU:pCPU overcommit ratio (host-level CPU utilization) | cores |
| Disk     | App disk allocated vs. total cell disk capacity         | GB    |

The highest-utilization resource is flagged as `constraining`, with a summary
such as: _"Memory is your constraint at 82.3% utilization. Address Memory
capacity before other resources."_

## Scenario Comparison Warnings

**Endpoint:** `POST /api/v1/scenario/compare`

The warning engine fires threshold-based heuristics across several categories.
Each warning carries a severity of `critical` or `warning`.

### Capacity / N-1 Safety

| Threshold             | Severity | Message                        |
| --------------------- | -------- | ------------------------------ |
| N-1 utilization > 85% | critical | Exceeds capacity safety margin |
| N-1 utilization > 75% | warning  | Approaching capacity limits    |

When HA Admission Control data is available, messages reflect whichever
constraint (HA% or N-1) is more restrictive.

### Staging Capacity (Free Chunks)

Free chunks represent contiguous memory blocks available for app staging.
Chunk size defaults to 4 GB and auto-detects from max instance memory.

| Threshold                 | Severity | Message                        |
| ------------------------- | -------- | ------------------------------ |
| < 10 free chunks (~40 GB) | critical | Critical: Low staging capacity |
| < 20 free chunks (~80 GB) | warning  | Low staging capacity           |

### Cell Memory Utilization

| Threshold | Severity | Message                          |
| --------- | -------- | -------------------------------- |
| > 90%     | critical | Cell utilization critically high |
| > 80%     | warning  | Cell utilization elevated        |

### Disk Utilization

| Threshold | Severity | Message                          |
| --------- | -------- | -------------------------------- |
| > 90%     | critical | Disk utilization critically high |
| > 80%     | warning  | Disk utilization elevated        |

### TPS / Scheduling Performance

Uses an interpolated TPS curve. The default curve peaks at 1,964 TPS for 3
cells and degrades at higher cell counts.

| Threshold          | Severity | Status   |
| ------------------ | -------- | -------- |
| TPS >= 80% of peak | --       | optimal  |
| TPS >= 50% of peak | warning  | degraded |
| TPS < 50% of peak  | critical | critical |

Beyond the curve's last data point, TPS degrades proportionally
(`lastTPS * lastCells / currentCells`).

### Blast Radius (Resilience)

Measures the percentage of total capacity lost when a single cell fails.

| Threshold                 | Severity | Message                      |
| ------------------------- | -------- | ---------------------------- |
| > 20% (5 or fewer cells)  | critical | High cell failure impact     |
| > 10% (10 or fewer cells) | warning  | Elevated cell failure impact |

The scenario delta classifies overall resilience:

| Blast Radius | Classification         |
| ------------ | ---------------------- |
| <= 5%        | low (20+ cells)        |
| 5--15%       | moderate (7--20 cells) |
| > 15%        | high (< 7 cells)       |

### vCPU:pCPU Ratio

| Threshold  | Severity | Risk Level   |
| ---------- | -------- | ------------ |
| <= 4:1     | --       | conservative |
| 4:1 to 8:1 | warning  | moderate     |
| > 8:1      | critical | aggressive   |

When the ratio exceeds the target (default 4:1), a warning advises monitoring
CPU Ready time. At > 8:1 the system flags aggressive overcommit and recommends
watching for CPU Ready > 5%.

## Constraint Analysis

When host configuration is provided, two capacity models are compared:

- **HA Admission Control** -- percentage-based memory reservation from
  vSphere DRS
- **N-X Host Failure** -- reserves one host's worth of memory

The system reports which is more restrictive and warns if the configured HA
percentage is insufficient to survive a single host failure.

## Actionable Fix Suggestions

Warnings include computed remediation suggestions:

- **Capacity fix:** reduce to N cells for 84% utilization, or add M hosts to
  support the proposed cell count
- **CPU ratio fix:** reduce to N cells to hit the target ratio, or reduce
  vCPUs per cell

## Recommendations

**Endpoint:** `GET /api/v1/recommendations`

Three prioritized upgrade paths, targeted at the constraining resource:

| Priority | Type         | Description                                                   |
| -------- | ------------ | ------------------------------------------------------------- |
| 1        | Add cells    | Calculates cells needed to reach 70% utilization              |
| 2        | Resize cells | Suggests doubling memory or adding vCPUs per cell             |
| 3        | Add hosts    | Calculates hosts needed for 70% utilization or 4:1 vCPU ratio |

## Planning Calculator

**Endpoint:** `POST /api/v1/infrastructure/planning`

Capacity ceiling analysis with sizing alternatives:

- Max deployable cells by memory vs. by CPU
- Identifies whether memory or CPU is the bottleneck at each cell size
- Reports utilization percentages and headroom (unused capacity in cells)
- Generates recommendations across 6 standard cell presets:

| Preset | vCPU | Memory |
| ------ | ---- | ------ |
| 1      | 4    | 32 GB  |
| 2      | 4    | 64 GB  |
| 3      | 8    | 64 GB  |
| 4      | 8    | 128 GB |
| 5      | 16   | 128 GB |
| 6      | 16   | 256 GB |

## Summary

The key diagnostic levers are:

1. **Which resource is the constraint** -- memory, CPU, or disk
2. **How close to unsafe capacity** -- N-1 and HA admission thresholds
3. **Platform resilience** -- blast radius per cell failure
4. **CPU overcommit risk** -- vCPU:pCPU ratio tiers
5. **Staging headroom** -- free chunk count for app deployments

The system provides threshold-based severity ratings and computed remediation
steps for each diagnostic.
