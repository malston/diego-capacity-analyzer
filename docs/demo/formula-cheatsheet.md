# Diego Capacity Analyzer — Formula Cheatsheet

Quick reference for all the math behind the capacity calculations.

---

## Core Capacity Formulas

### Max Deployable Cells

```
Max Cells by Memory = Usable Memory GB / Cell Memory GB
Max Cells by CPU    = Total Physical Cores / Cell vCPU
Deployable Cells    = min(Max by Memory, Max by CPU)
```

**Bottleneck** = whichever resource limits you first

---

### N-1 Capacity (Host Failure Tolerance)

```
N-1 Reserved GB     = Memory per Host (one host's worth)
N-1 Usable GB       = Total Memory - N-1 Reserved
N-1 Utilization %   = (Cell Memory + Platform VMs) / N-1 Usable × 100
```

**Goal:** Stay under 85% to survive a host failure

---

### HA Admission Control

```
HA Reserved GB      = Total Memory × (HA% / 100)
HA Usable GB        = Total Memory - HA Reserved
HA Utilization %    = (Cell Memory + Platform VMs) / HA Usable × 100

N-Equivalent        = floor(HA Reserved / Memory per Host)
```

**Example:** 25% HA on 16 hosts = N-4 tolerance (reserves 4 hosts' worth)

**Which constraint wins?** Whichever reserves MORE memory is the limiting constraint.

---

## Memory Calculations

### App Capacity (What's Available for Apps)

```
Memory Overhead     = Cell Memory × 7%    (Garden/system processes)
App Capacity per Cell = Cell Memory - Overhead
Total App Capacity  = Cells × App Capacity per Cell
```

**Example:** 64 GB cell → ~60 GB for apps (4 GB overhead)

---

### Memory Utilization

```
Memory Utilization % = Total App Memory / Total App Capacity × 100
```

| Range  | Status   |
| ------ | -------- |
| < 80%  | Good     |
| 80-90% | Warning  |
| > 90%  | Critical |

---

### Free Chunks (Staging Capacity)

```
Free Chunks = (App Capacity - App Memory Used) / 4 GB
```

**4 GB** = typical staging memory for `cf push`

| Chunks | Status      |
| ------ | ----------- |
| ≥ 20   | Healthy     |
| 10-19  | Limited     |
| < 10   | Constrained |

---

## CPU Calculations

### vCPU:pCPU Ratio

```
Total vCPUs         = Cells × Cell vCPU
Total pCPUs         = Hosts × Cores per Host
vCPU:pCPU Ratio     = Total vCPUs / Total pCPUs
```

| Ratio | Risk Level                     |
| ----- | ------------------------------ |
| ≤ 4:1 | Conservative (production safe) |
| 4-8:1 | Moderate (monitor CPU Ready)   |
| > 8:1 | Aggressive (expect contention) |

---

### Max Cells by CPU (at target ratio)

```
Max vCPU            = Target Ratio × Total pCPUs
Available for Cells = Max vCPU - Platform VMs vCPU
Max Cells by CPU    = Available for Cells / Cell vCPU
```

---

## Memory Overcommit & Ballooning

### What is Memory Overcommit?

Promising more virtual RAM to VMs than physical RAM exists on the host.

```
Overcommit Ratio = Total VM Memory / Physical Host Memory
```

| Ratio    | Risk    | Use Case                        |
| -------- | ------- | ------------------------------- |
| 1.0-1.3x | Low     | Production, mission-critical    |
| 1.3-2.0x | Medium  | Dev/test, predictable workloads |
| 2.0-3.0x | High    | Labs, demos only                |
| > 3.0x   | Extreme | Not recommended                 |

### Why These Thresholds?

When VMs collectively demand more than physical RAM, vSphere must **reclaim memory**. The reclamation hierarchy (in order of severity):

| Technique                | Impact     | What Happens                                    |
| ------------------------ | ---------- | ----------------------------------------------- |
| Transparent Page Sharing | Low        | Dedupe identical memory pages across VMs        |
| **Ballooning**           | Medium     | Balloon driver inflates, guest OS pages to swap |
| Memory Compression       | Medium     | Compress infrequently used pages                |
| Host Swapping            | **Severe** | Hypervisor swaps VM pages to disk               |

### Memory Ballooning Explained

```
Overcommit Pressure
    ↓
vSphere detects memory contention
    ↓
Balloon driver (vmmemctl) inflates inside guest VM
    ↓
Guest OS sees "less available RAM"
    ↓
Guest OS pages to its own swap
    ↓
App performance degrades (latency spikes, GC pauses)
```

**The 1.3x threshold** = the point where ballooning is unlikely under normal load variance.

Above 1.3x, you're gambling that workloads won't spike simultaneously.

### Diego-Specific Concerns

Diego cells run Java processes (Garden, executor) and containers. When ballooning triggers:

- Container memory limits become unreliable (guest is paging)
- Apps get OOM-killed unexpectedly
- Latency spikes cascade through the platform
- `cf push` staging may timeout

### Monitoring for Ballooning

In vSphere, watch these metrics:

| Metric              | Healthy | Warning | Critical  |
| ------------------- | ------- | ------- | --------- |
| Balloon (MB)        | 0       | > 0     | > 1 GB    |
| Swapped (MB)        | 0       | > 0     | Any       |
| Memory Contention   | 0%      | > 1%    | > 5%      |
| Mem Usage vs Active | Similar | Gap     | Large gap |

**If you see ballooning > 0 on Diego cells:** Reduce overcommit or add hosts.

---

## Resilience Calculations

### Blast Radius (Single Cell Failure Impact)

```
Blast Radius % = 100 / Cell Count
```

| Radius | Risk     | Cell Count |
| ------ | -------- | ---------- |
| ≤ 5%   | Low      | 20+ cells  |
| 5-15%  | Moderate | 7-20 cells |
| > 15%  | High     | < 7 cells  |

---

### Fault Impact (Apps Affected per Cell Failure)

```
Fault Impact = Total App Instances / Cell Count
```

| Impact | Status   |
| ------ | -------- |
| < 25   | Good     |
| 25-49  | Warning  |
| ≥ 50   | Critical |

---

## TPS (Scheduler Throughput) Estimation

TPS is **modeled**, not measured live. Default curve:

| Cells | TPS   | Notes              |
| ----- | ----- | ------------------ |
| 1     | 284   | Startup            |
| 3     | 1,964 | **Peak**           |
| 9     | 1,932 | Near peak          |
| 100   | 1,389 | -30%               |
| 210   | 104   | Severe degradation |

**Interpolation:** Linear between points

**Beyond curve:** `TPS = Last TPS × Last Cells / Current Cells`

**Status thresholds:**

- Optimal: ≥ 80% of peak
- Degraded: 50-79% of peak
- Critical: < 50% of peak

---

## Quick Reference: All Thresholds

| Metric             | Good       | Warning | Critical |
| ------------------ | ---------- | ------- | -------- |
| Capacity (HA/N-1)  | < 75%      | 75-85%  | > 85%    |
| Memory Utilization | < 80%      | 80-90%  | > 90%    |
| Disk Utilization   | < 80%      | 80-90%  | > 90%    |
| Free Chunks        | ≥ 20       | 10-19   | < 10     |
| TPS Performance    | ≥ 80% peak | 50-79%  | < 50%    |
| Blast Radius       | ≤ 5%       | 5-20%   | > 20%    |
| Fault Impact       | < 25       | 25-49   | ≥ 50     |
| Instances/Cell     | < 30       | 30-49   | ≥ 50     |
| vCPU:pCPU Ratio    | ≤ 4:1      | 4-8:1   | > 8:1    |

---

## Cell Size Presets

| Preset  | vCPU | Memory |
| ------- | ---- | ------ |
| Small   | 4    | 32 GB  |
| Medium  | 4    | 64 GB  |
| Medium+ | 8    | 64 GB  |
| Large   | 8    | 128 GB |
| XL      | 16   | 128 GB |
| XXL     | 16   | 256 GB |

---

## Constants

| Constant        | Value | Description                       |
| --------------- | ----- | --------------------------------- |
| Memory Overhead | 7%    | Garden/system processes           |
| Disk Overhead   | 0.01% | Negligible                        |
| Chunk Size      | 4 GB  | Staging memory unit               |
| Peak TPS        | 1,964 | Default peak scheduler throughput |

---

## Example Calculation

**Given:**

- 15 hosts × 2 TB each = 30 TB total
- 25% HA Admission Control
- 470 Diego cells @ 32 GB / 4 vCPU
- 960 physical cores

**Calculate:**

```
HA Reserved         = 30,000 × 0.25 = 7,500 GB
HA Usable           = 30,000 - 7,500 = 22,500 GB
Cell Memory Total   = 470 × 32 = 15,040 GB
HA Utilization      = 15,040 / 22,500 = 66.8% ✓ (good)

Max Cells by Memory = 22,500 / 32 = 703 cells
vCPU Total          = 470 × 4 = 1,880
vCPU:pCPU Ratio     = 1,880 / 960 = 1.96:1 ✓ (conservative)

Blast Radius        = 100 / 470 = 0.21% ✓ (excellent)
```
