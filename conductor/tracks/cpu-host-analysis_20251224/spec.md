# Specification: Add CPU and Host-Level Capacity Analysis

## Overview

Extend the TAS Capacity Analyzer beyond memory-only analysis to include CPU utilization, vCPU:pCPU ratios, host-level metrics, multi-resource bottleneck detection, and upgrade path recommendations.

## Problem Statement

The tool currently only analyzes memory capacity at the Diego cell level. Real capacity planning requires answering:

- **CPU:** When is CPU the bottleneck? What's my vCPU:pCPU ratio?
- **Hosts:** When do I need more physical hosts vs larger VMs?
- **Multi-resource:** Which resource will I exhaust first?

Currently, toggling "CPU" in Resource Types doesn't change any gauges or add CPU-specific metrics.

## Requirements

### 1. CPU Analysis

#### New Inputs
- vCPU per cell (already exists)
- Physical cores per host
- Number of hosts
- Target vCPU:pCPU ratio (e.g., 4:1, 8:1)

#### New Metrics
- CPU Utilization gauge (similar to memory gauge)
- vCPU:pCPU ratio indicator
- CPU Ready Time estimate (based on oversubscription)

#### Thresholds
| vCPU:pCPU Ratio | Risk Level |
|-----------------|------------|
| ≤ 4:1 | Low - production safe |
| 4:1 - 8:1 | Medium - monitor CPU ready |
| > 8:1 | High - expect contention |

### 2. Host-Level Analysis

#### New Inputs (optional vSphere integration)
- Number of physical hosts
- Cores per host
- Memory per host
- HA admission control policy (% reserved)

#### New Metrics
- Host memory utilization
- Host CPU utilization
- VMs per host
- HA capacity (can survive N host failures?)

#### Key Questions to Answer
- "Can my hosts handle this cell configuration?"
- "Do I need more hosts, or can I resize existing VMs?"
- "What's my HA headroom at the host level?"

### 3. Multi-Resource Bottleneck Analysis

Display which resource will be exhausted first:

```text
Resource Exhaustion Order:
1. Memory (78% utilized) ← Closest to limit
2. Disk (45% utilized)
3. CPU (32% utilized)

Recommendation: Memory is your constraint. Add cells or right-size apps before worrying about CPU.
```

### 4. Upgrade Path Recommendations

Based on analysis, suggest actionable next steps:
- "Add 2 more Diego cells"
- "Increase cell memory from 64GB to 128GB"
- "Add 1 physical host to your cluster"

## Data Sources

- **vSphere API** - Host-level metrics (already integrated via govmomi)
- **Manual input** - For non-vSphere environments
- **Existing infrastructure state** - Cell counts, memory per cell, vCPU per cell

## UI Changes

### Scenario Wizard Updates
- Add CPU-related inputs to the Resources step
- Add host-level inputs (optional section)
- Display CPU and host metrics in results

### Dashboard Updates
- CPU utilization gauge alongside memory gauge
- vCPU:pCPU ratio indicator with color-coded risk level
- Multi-resource bottleneck summary card
- Upgrade path recommendations section

## Backend Changes

### New API Fields
Extend existing models to include:
- CPU cores per host
- Number of hosts
- HA admission control percentage
- vCPU:pCPU ratio calculations

### New Calculations
- CPU utilization percentage
- vCPU:pCPU ratio
- Resource exhaustion ordering
- Upgrade path logic

## Out of Scope

- Disk I/O analysis (future enhancement)
- Network capacity analysis
- Historical trend analysis
- Automated remediation

## Success Criteria

1. CPU utilization gauge displays accurate data based on vCPU configuration
2. vCPU:pCPU ratio indicator shows risk level with correct color coding
3. Host-level analysis shows when hosts are the constraint
4. Multi-resource bottleneck correctly identifies limiting resource
5. Upgrade recommendations are actionable and accurate
6. All new functionality has >80% test coverage
