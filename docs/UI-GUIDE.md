# UI Guide - Understanding the Dashboard

A quick reference explaining what each metric and visualization means.

> **Tip:** Hover over metric labels for contextual explanations. Tooltips are available throughout the UI:
>
> - **Dashboard**: Key metrics (Total Cells, Utilization, CPU, Unused Memory)
> - **Host Analysis**: VMs per Host, Memory/CPU Utilization, HA Status (N-X notation)
> - **Bottleneck Analysis**: Resource exhaustion order and constraint indicators
> - **What-If Mode**: Memory overcommit ratio (Diego-level, not vSphere) and capacity metrics
> - **Scenario Wizard**: VM Size presets, Memory Overhead, Hypothetical App, TPS Performance Model

---

## Visual Overview

The TAS Capacity Analyzer provides two main views: the **Dashboard** for real-time monitoring and **Capacity Planning** for what-if analysis.

![Full UI Walkthrough](images/tas-capacity-analyzer-demo.gif)

_Complete walkthrough showing login, dashboard metrics, cell details, and capacity planning wizard._

---

## Key Metrics Cards

| Metric            | What It Means                                                                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| **Total Cells**   | Number of Diego cells (VMs that run app containers). More cells = more capacity for workloads.                                                 |
| **Utilization %** | Percentage of total memory actively being used by running apps. Low = underutilized infrastructure. High (>80%) = risk of capacity exhaustion. |
| **Avg CPU %**     | Average processor load across cells. Sustained >70% indicates CPU contention risk - apps compete for cycles and slow down.                     |
| **Unused Memory** | Memory apps reserved but aren't using. This is "paid for but idle" capacity that could be reclaimed through right-sizing.                      |

---

## Diego Cells Detail Table

| Column              | What It Means                                                                          |
| ------------------- | -------------------------------------------------------------------------------------- |
| **Cell**            | The Diego cell VM name (e.g., `diego_cell/0`)                                          |
| **Segment**         | Isolation segment the cell belongs to - used to separate workloads (e.g., prod vs dev) |
| **Capacity**        | Total memory available on this cell (what the VM was sized to)                         |
| **Allocated**       | Memory reserved by apps scheduled to this cell (sum of app memory quotas)              |
| **Used**            | Memory actually consumed by running processes (always ≤ Allocated)                     |
| **CPU %**           | Current CPU utilization on this cell. Color-coded: 🟢 <50%, 🟡 50-70%, 🔴 >70%         |
| **Utilization Bar** | Visual representation of Used/Capacity. Shows how "full" the cell is.                  |

**Key insight:** The gap between Allocated and Used = memory apps requested but aren't consuming. This is per-cell waste.

---

## Cell Capacity Chart (Stacked Bar)

| Segment                      | What It Means                             |
| ---------------------------- | ----------------------------------------- |
| **Used (blue)**              | Memory actively consumed by app processes |
| **Allocated unused (green)** | Reserved by apps but sitting idle         |
| **Available (gray)**         | Free capacity - room for new apps         |

**Reading the chart:**

- Lots of green? Apps are over-provisioned - right-sizing opportunity
- Little gray? Running hot - may need more cells or right-sizing
- Uneven bars? Workload imbalance across cells

---

## Isolation Segments Pie Chart

Shows distribution of cells across segments. Helps answer:

- "Do we have enough production capacity vs dev?"
- "Are segments balanced appropriately?"
- "Should we rebalance cells between segments?"

---

## Right-Sizing Recommendations

Apps appear here when they have >15% memory overhead (requested vs actual).

| Column               | What It Means                                                                  |
| -------------------- | ------------------------------------------------------------------------------ |
| **App Name**         | Application that's over-provisioned                                            |
| **Instances**        | Number of running instances                                                    |
| **Overhead %**       | How much extra memory was requested vs actually used. >30% = significant waste |
| **Requested**        | Memory quota the app asks for (what developers set via `cf push -m`)           |
| **Actual Usage**     | Memory the app actually consumes at runtime                                    |
| **Recommended**      | Suggested quota (actual usage + 20% headroom)                                  |
| **Savings/Instance** | Memory you'd reclaim per instance by right-sizing                              |

**Example:** "This app requests 1024 MB but only uses 780 MB. Reduce quota to 936 MB (780 + 20% buffer) and reclaim 88 MB × instances."

**Why 20% buffer?** Accounts for memory spikes, garbage collection, and safety margin against OOM kills.

---

## What-If Mode

Toggle via the **What-If Mode** button. Explores: "What if I enabled memory overcommit?"

![What-If Mode Demo](images/tas-what-if-mode.gif)

_Adjusting the Memory Overcommit Ratio slider to see capacity impact._

| Metric                  | What It Means                                                                                                                                             |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Overcommit Ratio**    | Memory multiplier. 1.0x = no overcommit. 1.5x = sell 50% more capacity than physically exists.                                                            |
| **Capacity**            | Total cell memory × overcommit ratio. At 1.0x this is actual capacity; at 1.5x it's 50% more virtual capacity.                                            |
| **Current Instances**   | How many app instances are running now                                                                                                                    |
| **Avg Instance Size**   | Average memory per instance, calculated from your actual apps (total requested memory ÷ total instances)                                                  |
| **Additional Capacity** | How many more instances could fit with overcommit, based on your actual average instance size. Shows negative (red) if current workload exceeds capacity. |

### Overcommit Risk Levels

| Ratio        | Risk      | Typical Use                           |
| ------------ | --------- | ------------------------------------- |
| **1.0-1.3x** | Low       | Production, mission-critical          |
| **1.3-2.0x** | Medium    | Dev/test, well-understood workloads   |
| **2.0-3.0x** | High      | Labs, demos, low-traffic environments |
| **3.0x+**    | Very High | Labs with minimal app utilization     |

**Warning:** Overcommit lets you pack more apps, but if apps spike memory simultaneously, you risk OOM kills. Use cautiously and monitor closely.

**Real-world example:** A Small Footprint TPCF lab might run 3.75x overcommit (61 GB advertised on a 16 GB cell) because lab apps have minimal utilization. This would cause OOM kills under production traffic.

### Configuring Overcommit in TAS

Memory overcommit is configured in Ops Manager under **TAS tile → Advanced Features**:

| Field                          | Description                                                            |
| ------------------------------ | ---------------------------------------------------------------------- |
| **Diego Cell memory capacity** | Total memory to advertise per cell (in MB). Overrides Resource Config. |
| **Diego Cell disk capacity**   | Total disk to advertise per cell (in MB). Overrides Resource Config.   |

**Example:** If your Diego Cell VMs have 32 GB RAM (32768 MB in Resource Config) but you want 1.5x overcommit, enter **49152** in the "Diego Cell memory capacity" field.

**How to calculate:**

```
Overcommit Value (MB) = Cell VM Memory (MB) × Desired Ratio
Example: 32768 MB × 1.5 = 49152 MB
```

> **Note:** VMware provides no specific recommendation for overcommit ratios due to the deployment-specific nature of workloads. Use the What-If Mode to model scenarios before making changes.

For more details, see [Configure Advanced Features](https://techdocs.broadcom.com/us/en/vmware-tanzu/platform/tanzu-platform-for-cloud-foundry/6-0/tpcf/config-advanced-features.html) in the TAS documentation.

---

# Scenario Analysis Tab

Answers: **"Will my workload fit if I change my cell configuration?"**

---

## Loading Infrastructure Data

| Source           | Description                                                           |
| ---------------- | --------------------------------------------------------------------- |
| **vSphere Live** | Real-time data from vCenter via backend (requires vSphere configured) |
| **Upload JSON**  | Import infrastructure JSON file; also shows sample file picker        |
| **Manual Entry** | Form-based input for custom infrastructure configuration              |

**Sample Data:** When using Upload JSON mode, you can select from 8 pre-built configurations:

- Small Foundation (Dev/Test), Medium Foundation (Staging), Large Foundation (Production)
- Enterprise Multi-Cluster
- CPU-Constrained, Memory-Constrained
- Diego Benchmark 50K, Diego Benchmark 250K

After loading data, a **Current Configuration** summary appears showing your existing cell count, size, and total capacity. This helps you understand what you're comparing against before proposing changes.

---

## IaaS Capacity

After loading infrastructure data, the IaaS Capacity section displays your physical infrastructure limits and calculates the maximum number of Diego cells you can deploy.

| Metric           | What It Means                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **Hosts**        | Total ESXi hosts in your cluster(s). Shows cluster count if multi-cluster.                                                    |
| **Total Memory** | Total RAM across all hosts. Below this, you'll see the **HA-usable memory** based on your HA Admission Control percentage.    |
| **Total vCPUs**  | Total CPU cores available across all hosts.                                                                                   |
| **Max Cells**    | Maximum Diego cells deployable based on HA-usable memory. Shows the resulting vCPU:pCPU ratio at current and max cell counts. |

### HA Admission Control

The available memory is constrained by vSphere HA Admission Control, which reserves a percentage of cluster resources for failover capacity. This is what vSphere actually enforces--you cannot deploy VMs beyond this limit.

| Display          | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| **HA X% (≈N-Y)** | X% reserved for HA, equivalent to surviving Y host failures |

**Example:** With 32TB total memory (16 hosts × 2TB) and 25% HA Admission Control:

- Usable memory: 32TB × 75% = 24TB
- Implied tolerance: 25% of 16 hosts = 4 hosts → N-4 tolerance

### Max Cells Calculation

Max Cells is based on HA-usable memory (the real deployable limit):

```text
Max Cells = HA-Usable Memory GB / Cell Memory GB
```

The vCPU:pCPU ratio is calculated as an **output** to show you the resulting CPU oversubscription:

| Metric            | Formula                                              |
| ----------------- | ---------------------------------------------------- |
| **Current Ratio** | `(Current Cells × Cell vCPU) / Total Physical Cores` |
| **Max Ratio**     | `(Max Cells × Cell vCPU) / Total Physical Cores`     |

Risk levels help you understand if the ratio is acceptable:

- **Low (≤4:1)**: Conservative, typical for production
- **Medium (4:1-8:1)**: Monitor CPU Ready time
- **High (>8:1)**: Aggressive, requires active monitoring

**Example:** With 22.5TB HA-usable memory, 960 physical cores, and cells sized at 32 GB / 4 vCPU:

- Max Cells = 22,500 ÷ 32 = **703 cells**
- Max Ratio = (703 × 4) ÷ 960 = **2.93:1** (Low risk ✓)

If your **Proposed Cell Count** exceeds Max Cells, an amber warning appears showing how many cells over capacity you are.

---

## Proposed Configuration

### Cell Configuration

| Input                 | What It Means                                                                     |
| --------------------- | --------------------------------------------------------------------------------- |
| **VM Size Preset**    | Common cell sizes: Small (4 vCPU/32 GB), Medium (8/64), Large (16/128), or Custom |
| **Cell Count**        | Number of Diego cells in your proposed configuration                              |
| **Memory Overhead %** | System memory reserved for Diego/Garden (default 7%)                              |

### CPU Configuration

The vCPU:pCPU ratio is **calculated as an output**, not configured as an input. This reflects reality: you can't set a "target ratio" in vSphere--the ratio is a consequence of how many cells you deploy and what size they are.

| Metric                   | What It Means                                     |
| ------------------------ | ------------------------------------------------- |
| **Total Physical Cores** | Total pCPU cores available across all hosts       |
| **Total vCPUs**          | Sum of vCPUs allocated to all Diego cells         |
| **Current Ratio**        | Actual vCPU:pCPU ratio at your current cell count |
| **Ratio at Max**         | What the ratio would be if you deployed max cells |

Risk level indicators help you understand if your current or proposed configuration is acceptable:

- **Low (≤4:1)**: Conservative, typical for general production workloads
- **Medium (4:1-8:1)**: Monitor CPU Ready time for contention
- **High (>8:1)**: Aggressive, requires active monitoring

**Note:** To change the ratio, change one of the actual parameters: cell count, cell vCPU size, or number of physical hosts.

### Host Configuration (Optional)

| Input                      | What It Means                             |
| -------------------------- | ----------------------------------------- |
| **Number of Hosts**        | Physical ESXi hosts in your cluster       |
| **Memory per Host (GB)**   | RAM per physical host                     |
| **Cores per Host**         | CPU cores per physical host               |
| **HA Admission Control %** | Cluster capacity reserved for HA failover |

**Hypothetical App:** Add a theoretical app to see if it would fit. Enter instance count and memory per instance.

---

## Results Layout

Scenario results are organized into two visual sections that separate "can we deploy this?" from "how full is it?":

- **Infrastructure Headroom** -- Physical capacity limits and deployment constraints
- **Current Utilization** -- Workload usage, performance, and configuration details

![Scenario Results Demo](images/tas-scenario-results.gif)

_Scenario results showing Infrastructure Headroom and Current Utilization sections with gauges, scorecards, and cell configuration comparison._

---

## Results: Infrastructure Headroom

Answers: **"Does the physical infrastructure have room for this configuration?"**

### Capacity (HA) / N-1 Capacity

This gauge shows utilization against whichever capacity constraint is more restrictive. The label changes dynamically:

| Label                       | When Shown                                                             | What It Measures                                |
| --------------------------- | ---------------------------------------------------------------------- | ----------------------------------------------- |
| **Capacity (HA X% (≈N-Y))** | When HA Admission Control is the limiting constraint                   | Utilization of HA-usable capacity               |
| **N-1 Capacity**            | When N-1 host capacity is limiting, or when no host config is provided | Utilization of N-1 capacity (one host reserved) |

The system automatically determines which constraint is more restrictive by comparing:

- **HA Admission Control**: Reserves X% of total cluster memory (e.g., 25% = 7,500 GB on 30 TB cluster)
- **N-1 Host Capacity**: Reserves one host's worth of memory (e.g., 2,000 GB per host)

Whichever reserves more capacity is the limiting constraint and is displayed in the gauge.

| Value      | Status   | Meaning                               |
| ---------- | -------- | ------------------------------------- |
| **< 75%**  | Good     | Safe headroom within capacity limits  |
| **75-85%** | Warning  | Approaching capacity limits           |
| **> 85%**  | Critical | Near or exceeding deployable capacity |

**Key insight:** HA Admission Control is what vSphere actually enforces. If you configure 25% HA, vSphere reserves 25% of cluster resources and won't let you deploy VMs beyond the remaining 75%. This is equivalent to N-4 host failure tolerance on a 16-host cluster.

**Example:** On a 16-host cluster with 2 TB per host (32 TB total):

- HA 25% reserves 8 TB (= N-4) → HA is limiting
- HA 5% reserves 1.6 TB (< N-1's 2 TB) → N-1 is limiting

### CPU Utilization (vCPU:pCPU Ratio)

Shown when CPU is selected as an analysis resource. The vCPU:pCPU ratio shows how many virtual CPUs are allocated per physical CPU core. This is a **calculated output** based on your cell count and cell vCPU size--it's not a configurable setting.

| Ratio         | Risk Level | Meaning                                                   |
| ------------- | ---------- | --------------------------------------------------------- |
| **≤ 4:1**     | Low        | Conservative, typical for general production workloads    |
| **4:1 - 8:1** | Medium     | Monitor CPU Ready time for contention                     |
| **> 8:1**     | High       | Aggressive, requires active monitoring; expect contention |

The display shows the proposed ratio with vCPU and pCPU counts, plus a CPU headroom indicator showing how many additional cells can be added before reaching the target ratio.

**Note:** VMware's current guidance emphasizes monitoring actual CPU Ready Time (target <5%) rather than adhering to fixed ratio thresholds. The ratio indicators help you understand the implications of your cell configuration, but actual performance depends on workload characteristics.

### Maximum Deployable Cells

Shows the upper bound on how many Diego cells your infrastructure can support, broken down by resource constraint. Each constraint is shown as a line item:

| Constraint | What It Measures                                                  |
| ---------- | ----------------------------------------------------------------- |
| **Memory** | Max cells based on HA-usable memory (or N-1) divided by cell size |
| **CPU**    | Max cells before exceeding the target vCPU:pCPU ratio             |

When both Memory and CPU resources are selected, the more restrictive constraint is highlighted with a **BOTTLENECK** indicator. Each line also shows headroom (additional cells beyond the current count).

**Example:** With 22.5TB HA-usable memory, 960 physical cores, and cells sized at 32 GB / 4 vCPU:

- Memory: 22,500 / 32 = **703 cells** (+603 headroom from 100 current)
- CPU at 4:1 target: 960 × 4 / 4 = **960 cells** (+860 headroom)
- Memory is the bottleneck (703 < 960)

---

## Results: Current Utilization

Answers: **"How full is the proposed configuration with the current workload?"**

### Memory Utilization

| Value      | Status   | Meaning                  |
| ---------- | -------- | ------------------------ |
| **< 80%**  | Good     | Healthy headroom         |
| **80-90%** | Warning  | Getting tight            |
| **> 90%**  | Critical | Near capacity exhaustion |

### Disk Utilization

Shown when disk is selected as an analysis resource.

| Value      | Status   | Meaning                  |
| ---------- | -------- | ------------------------ |
| **< 80%**  | Good     | Healthy headroom         |
| **80-90%** | Warning  | Getting tight            |
| **> 90%**  | Critical | Near capacity exhaustion |

### Staging Capacity (Free Chunks)

Available memory chunks for `cf push` staging operations. The chunk size is **auto-detected** from your application portfolio's average instance memory, with a 4GB fallback when no data is available.

| Chunks    | Status      | Meaning                       |
| --------- | ----------- | ----------------------------- |
| **≥ 20**  | Healthy     | Plenty of staging capacity    |
| **10-19** | Limited     | May queue during busy periods |
| **< 10**  | Constrained | Deployment bottleneck likely  |

**Chunk size detection:** The system calculates `Total App Memory / Total App Instances` to determine your typical app footprint. Java-heavy platforms typically show ~4GB chunks, while Go/Python workloads may show 1-2GB. The UI displays the actual chunk size used in calculations (e.g., "2.5GB chunks for staging").

### Scheduling Performance (TPS)

**TPS = Tasks Per Second** -- how fast Diego's scheduler can place app instances. Displayed as a current → proposed comparison with status badges.

| Cell Count | TPS    | Notes              |
| ---------- | ------ | ------------------ |
| 3          | ~1,964 | Peak efficiency    |
| 100        | ~1,389 | ~30% degradation   |
| 210        | ~104   | Severe degradation |

**Why it matters:** More cells = more coordination overhead. If you need more capacity, consider larger cells instead of more cells to avoid scheduler bottlenecks.

> **Note:** These values are modeled estimates, not live measurements. See [TPS Performance (Modeled)](#tps-performance-modeled) for methodology and customization options.

### Metric Scorecards

| Metric             | What It Means                             | Good Direction               |
| ------------------ | ----------------------------------------- | ---------------------------- |
| **Cell Count**     | Number of Diego cell VMs                  | Depends on strategy          |
| **App Capacity**   | Total memory available for apps           | Higher = more headroom       |
| **Fault Impact**   | App instances displaced if one cell fails | Lower = smaller blast radius |
| **Instances/Cell** | Average app instances per cell            | Lower = more distributed     |

---

## Host-Level Analysis

The Host Analysis card shows physical infrastructure metrics for capacity planning.

| Metric                      | What It Means                                          |
| --------------------------- | ------------------------------------------------------ |
| **Total Hosts**             | Physical ESXi hosts in the cluster                     |
| **VMs per Host**            | Average Diego cells per physical host                  |
| **Host Memory Utilization** | Percentage of physical memory allocated to Diego cells |
| **Host CPU Utilization**    | Percentage of physical CPU cores allocated as vCPUs    |
| **HA Hosts Survived**       | Number of host failures the cluster can tolerate       |
| **HA Status**               | "ok" if cluster can survive at least 1 host failure    |

### HA Admission Control

HA admission control reserves cluster capacity to ensure workloads can be restarted after host failures.

| Percentage | Use Case                                           |
| ---------- | -------------------------------------------------- |
| **0%**     | Dev/test environments, no HA protection            |
| **15-20%** | Standard production, single host failure tolerance |
| **25%**    | High availability, can tolerate larger failures    |
| **>25%**   | Mission-critical, multi-host failure tolerance     |

#### Calculating HA Admission Percentage

To determine the HA percentage needed to survive N host failures:

```text
HA % = (Hosts to Survive / Total Hosts) × 100
```

The number of survivable host failures is calculated as:

```text
Hosts Survivable = floor(HA % / 100 × Total Hosts)
```

**Examples:**

| Cluster Size | Target Survivability  | Required HA %    |
| ------------ | --------------------- | ---------------- |
| 4 hosts      | N-1 (1 host failure)  | 25%              |
| 4 hosts      | N-2 (2 host failures) | 50%              |
| 15 hosts     | N-1 (1 host failure)  | 7% (round to 8%) |
| 15 hosts     | N-2 (2 host failures) | 14%              |
| 3 hosts      | N-1 (1 host failure)  | 34%              |

**Note:** Always round up to ensure the `floor()` calculation yields the desired survivability.

#### HA Admission vs. Memory Overhead: Not Double-Counting

These two percentages operate at different layers and are **not** redundant:

| Calculation              | Layer             | What It Measures                                   |
| ------------------------ | ----------------- | -------------------------------------------------- |
| **HA Admission %**       | vSphere cluster   | Memory reserved to restart VMs after host failure  |
| **Memory Overhead (7%)** | Inside Diego cell | Memory consumed by Garden runtime and OS processes |

**How they work together:**

1. **vSphere perspective**: A 32GB Diego cell consumes 32GB of cluster memory. HA admission reserves capacity based on this full VM footprint--vSphere doesn't know or care what runs inside the VM.

2. **Diego perspective**: Of that 32GB cell, ~30GB is available for application containers. The remaining ~2GB (7%) runs Garden, system processes, and the Diego executor.

**Example with 15 hosts × 2TB each (30TB cluster), 10% HA, 470 cells @ 32GB:**

```text
Cluster level (HA Admission):
  Total memory:     30,000 GB
  HA reserved:       3,000 GB (10%)
  Usable for VMs:   27,000 GB
  Diego cells use:  15,040 GB (470 × 32 GB)  ← full VM footprint
  Utilization:      55.7% of HA-usable capacity

Cell level (Memory Overhead):
  Cell size:        32 GB
  OS/Garden:         2 GB (7%)
  App capacity:     30 GB per cell
  Total app capacity: 14,100 GB (470 × 30 GB)
```

Both calculations are needed: HA admission determines if you can _deploy_ the VMs; memory overhead determines how much _workload_ fits inside them.

#### vSphere Memory Reservations for Diego Cells

**Best practice:** Set memory reservations on Diego cell VMs equal to their configured memory.

| Cell Size | Reservation |
| --------- | ----------- |
| 32 GB     | 32 GB       |
| 48 GB     | 48 GB       |
| 64 GB     | 64 GB       |

**Why this matters:**

When a vSphere host is under memory pressure, it uses these reclamation techniques (in order):

1. **Ballooning** - VMware Tools balloon driver inflates inside guest VMs, forcing the guest OS to release memory
2. **Compression** - vSphere compresses memory pages
3. **Host swapping** - vSphere swaps VM memory to disk (severe performance impact)

Diego cells should **never** be subject to these techniques. If they are:

- Apps see sudden, unexplained memory pressure
- OOM kills and container crashes follow
- Performance becomes unpredictable

Setting a memory reservation tells vSphere "guarantee this VM's full memory allocation--never reclaim from it." This ensures Diego cells have dedicated physical memory and aren't sacrificed when other workloads cause host pressure.

**Note:** Memory reservations reduce the pool of memory available for other VMs. If you reserve 32GB for each of 470 Diego cells, that's 15TB that cannot be overcommitted. This is intentional--Diego cells need predictable memory access.

---

## Bottleneck Analysis

The Bottleneck card identifies which resource will be exhausted first.

### Resource Exhaustion Order

Resources are ranked by utilization percentage:

```text
Example:
1. Memory (78% utilized) ← Constraining
2. CPU (45% utilized)
3. Disk (32% utilized)
```

The **constraining resource** is the one closest to capacity. Address this resource first before optimizing others.

### Upgrade Recommendations

Based on bottleneck analysis, the system suggests prioritized actions:

| Recommendation         | When Suggested                            |
| ---------------------- | ----------------------------------------- |
| **Add Diego Cells**    | When you need more capacity quickly       |
| **Resize Diego Cells** | When larger cells would be more efficient |
| **Add Physical Hosts** | When infrastructure is the constraint     |

Each recommendation includes:

- **Impact**: Specific improvement (e.g., "Adds 256 GB memory capacity")
- **Priority**: 1 = most impactful, 3 = least impactful

---

## Overall Status Banner

| Status              | Meaning                                |
| ------------------- | -------------------------------------- |
| **✓ YES** (green)   | Configuration meets all requirements   |
| **⚠ MAYBE** (amber) | Warnings to review before proceeding   |
| **✗ NO** (red)      | Critical issues - adjust configuration |

---

# Reference

## How Metrics Are Calculated

### Dashboard Tab (Live Data)

| Metric            | Formula                                                              | Data Source             |
| ----------------- | -------------------------------------------------------------------- | ----------------------- |
| **Total Cells**   | Count of Diego cell VMs                                              | BOSH API: `bosh vms`    |
| **Utilization %** | `(Total Used Memory / Total Cell Capacity) × 100`                    | BOSH vitals + Log Cache |
| **Avg CPU %**     | `Sum(cell.cpu_percent) / cell_count`                                 | BOSH API: VM vitals     |
| **Unused Memory** | `Sum(app.requested_mb × instances) - Sum(app.actual_mb × instances)` | CF API + Log Cache      |

**Data flow:** BOSH Director → Backend → Frontend

- Cell capacity: BOSH deployment manifest (VM type memory)
- Cell vitals: BOSH `/vms?vitals=true` endpoint
- App quotas: CF API `/v3/apps` and `/v3/processes`
- Actual memory: Log Cache gauge metrics (`memory_bytes`)

### Scenario Analysis Tab (Calculated)

Results are organized into two sections: **Infrastructure Headroom** (deployment limits) and **Current Utilization** (workload usage).

#### Infrastructure Headroom

| Gauge                    | Formula                                                                                      | Thresholds                              |
| ------------------------ | -------------------------------------------------------------------------------------------- | --------------------------------------- |
| **Capacity (HA/N-1)**    | `(Cell Memory + Platform VMs) / Usable Capacity × 100`                                       | Warning: 75%, Critical: 85%             |
| **vCPU:pCPU Ratio**      | `(Cell Count × Cell vCPU) / Total Physical Cores`                                            | Low: ≤4:1, Medium: 4-8:1, High: >8:1    |
| **Max Deployable Cells** | `Usable Memory / Cell Memory` (memory) and `Physical Cores × Target Ratio / Cell vCPU` (CPU) | Bottleneck indicator when both selected |

Where:

- **Usable Capacity** = Total cluster memory - Reserved capacity (HA% or N-1, whichever reserves more)

#### Current Utilization

| Gauge                  | Formula                               | Thresholds                                     |
| ---------------------- | ------------------------------------- | ---------------------------------------------- |
| **Memory Utilization** | `App Memory / App Capacity × 100`     | Warning: 80%, Critical: 90%                    |
| **Disk Utilization**   | `App Disk / Disk Capacity × 100`      | Warning: 80%, Critical: 90%                    |
| **Staging Capacity**   | Raw count of free chunks (auto-sized) | Healthy: ≥20, Limited: 10-19, Constrained: <10 |

Where:

- **App Capacity** = `cells × (cell_memory_gb - 7% overhead)`
- **Free Chunks** = `(App Capacity - App Memory) / Chunk Size`
- **Chunk Size** = Auto-detected from `Max Instance Memory` (largest app memory limit, min 1GB, defaults to 4GB if unavailable). We use MAX because staging requires contiguous memory--if your largest app needs 4GB, you need 4GB chunks available even if most apps are small.

#### TPS Performance

Compares current vs proposed scheduler throughput with status indicators. See [TPS Performance (Modeled)](#tps-performance-modeled) below.

#### Metric Scorecards

Grid of cards showing current → proposed values with change indicators:

| Scorecard          | Formula                               | Notes                            |
| ------------------ | ------------------------------------- | -------------------------------- |
| **Cell Count**     | Direct count                          | Number of Diego cell VMs         |
| **App Capacity**   | `cells × (cell_memory - 7% overhead)` | Total memory available for apps  |
| **Fault Impact**   | `Total App Instances / Cell Count`    | Apps displaced if one cell fails |
| **Instances/Cell** | `Total App Instances / Cell Count`    | Distribution density             |

#### Cell Configuration Change

Visual comparison of cell specs (vCPU × GB) between current and proposed, showing:

- Current cell size and count
- Proposed cell size and count
- Resilience risk indicator (low/moderate/high based on blast radius)
- Capacity change summary in GB
- Utilization change summary in percentage

#### Advanced Options

Expandable panel with configuration overrides:

**Memory Overhead**

- Slider: 1% to 20% (default: 7%)
- Adjusts the percentage of cell memory reserved for Garden runtime and system processes
- Formula: `App Capacity = cells × (cell_memory × (1 - overhead%))`
- The 7% default is an empirical estimate; verify against your actual cell utilization if precision matters
- This is separate from HA Admission Control--see [HA Admission vs. Memory Overhead](#ha-admission-vs-memory-overhead-not-double-counting) for details

**Add Hypothetical App**

- Model the impact of deploying a new application before actually deploying it
- Configure: app name, instance count, memory per instance, disk per instance
- When enabled, adds to total app memory/disk/instances in calculations
- Useful for capacity planning: "Can my foundation handle this new workload?"

**TPS Performance Curve**

- Customize the scheduler throughput benchmark data points
- Each point maps cell count → expected TPS
- Default values are baseline estimates from internal benchmarks
- Add/remove points, adjust values to match your observed scheduler performance
- See [TPS Performance (Modeled)](#tps-performance-modeled) for details on the curve

### TPS Performance (Modeled)

TPS is **not a live metric**. It's estimated from Diego benchmark data using linear interpolation:

```text
Benchmark curve (default):
  1 cell   →    284 TPS
  3 cells  →  1,964 TPS (peak)
  9 cells  →  1,932 TPS
  100 cells → 1,389 TPS
  210 cells →   104 TPS
```

The curve models BBS scheduler coordination overhead as cell count increases. Values between points are interpolated. Beyond the curve, TPS degrades proportionally.

**Important:** The default curve is a baseline estimate derived from internal platform engineering benchmarks. Actual TPS varies significantly based on infrastructure, network latency, database backend, and workload characteristics. **We recommend validating against your own environment** and customizing the curve in Advanced Options to match observed performance.

**References:**

- [Diego Scaling & Performance Tuning](https://github.com/cloudfoundry/diego-release/blob/develop/docs/030-scaling-and-performance-tuning.md) - Official guidance on benchmarking methodology and VM sizing
- [Diego Performance Measurement Proposal](https://github.com/cloudfoundry/diego-notes/blob/main/proposals/measuring_performance.md) - Explains how Diego performance benchmarks are structured

**Status thresholds:**

- Optimal: ≥80% of peak TPS
- Degraded: 50-79% of peak TPS
- Critical: <50% of peak TPS

---

## Data Sources

| Source                 | Description                                 |
| ---------------------- | ------------------------------------------- |
| **CF API**             | App instances, memory quotas, process stats |
| **BOSH**               | Diego cell VMs, capacity, vitals            |
| **Log Cache**          | Real-time container memory metrics          |
| **vSphere** (optional) | VM-level infrastructure metrics             |

---

## Common Questions

**Q: Why is utilization low but we're told we need more capacity?**
A: Look at Allocated vs Used. High allocation with low utilization means apps are over-provisioned. Right-size apps before adding cells.

**Q: What's a healthy utilization target?**
A: 60-75% gives headroom for spikes and deployments. Below 50% suggests consolidation opportunity. Above 80% risks capacity exhaustion during deploys.

**Q: Should we enable overcommit?**
A: Only if you understand your workload patterns and have monitoring in place. Start conservative (1.2x) and increase gradually while watching for OOM events.

**Q: How do we know which apps to right-size first?**
A: Sort by Potential Savings (overhead × instances). Apps with high instance counts and high overhead yield the biggest wins.

---

## Metric Scorecard Status Thresholds

The metric scorecards in the results section use color-coded status badges to indicate health:

| Status       | Color | Meaning                                    |
| ------------ | ----- | ------------------------------------------ |
| **good**     | Cyan  | Within healthy limits                      |
| **warning**  | Amber | Approaching threshold, monitor closely     |
| **critical** | Red   | Exceeds safe threshold, action recommended |

### Scorecard Thresholds

| Scorecard          | Warning Threshold | Critical Threshold | Notes                                                  |
| ------------------ | ----------------- | ------------------ | ------------------------------------------------------ |
| **Cell Count**     | --                | --                 | Informational only; no status thresholds               |
| **App Capacity**   | --                | --                 | Informational only; no status thresholds               |
| **Fault Impact**   | ≥25 apps/cell     | ≥50 apps/cell      | Lower is better; high values mean larger blast radius  |
| **Instances/Cell** | ≥30               | ≥50                | Lower is better; high density increases failure impact |

**Note:** Cell Count and App Capacity display status based on the direction of change (improvement vs regression) rather than absolute thresholds. These metrics don't have inherently "bad" values--500 cells isn't worse than 50 cells; it depends on your workload requirements.

---

## Recommendations Reference

The Recommendations section displays warnings and suggestions based on your proposed configuration. Each message is triggered by a specific metric exceeding a threshold.

### Capacity Constraint Warnings

Measures whether your cluster can handle VM load within capacity constraints. The warning message reflects which constraint is limiting.

#### When HA Admission Control is the Limiting Constraint

| Message                                                            | Severity | Triggered When                       | What It Means                                                                                                       |
| ------------------------------------------------------------------ | -------- | ------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| **Exceeds HA Admission Control capacity limit (HA X% (≈N-Y))**     | Critical | Utilization > 85% and HA is limiting | vSphere HA reserves X% of cluster resources. You're approaching or exceeding what vSphere will allow you to deploy. |
| **Approaching HA Admission Control capacity limit (HA X% (≈N-Y))** | Warning  | Utilization > 75% and HA is limiting | Getting close to the HA-enforced limit. Consider reducing cell count or increasing HA percentage tolerance.         |

#### When N-1 Host Capacity is the Limiting Constraint

| Message                                | Severity | Triggered When                        | What It Means                                                                                                             |
| -------------------------------------- | -------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Exceeds N-1 capacity safety margin** | Critical | Utilization > 85% and N-1 is limiting | If one host fails, remaining hosts cannot accommodate all VMs. Immediate risk of workload loss during host failure.       |
| **Approaching N-1 capacity limits**    | Warning  | Utilization > 75% and N-1 is limiting | Getting close to the threshold. A host failure would leave little headroom. Consider adding hosts or reducing cell count. |

**How the limiting constraint is determined:**

- System compares HA reserved capacity vs N-1 reserved capacity
- Whichever reserves MORE is the limiting constraint (less usable capacity)
- Example: HA 25% on 32 TB = 8 TB reserved; N-1 = 2 TB reserved → HA is limiting

**Formula:** `Utilization = (Total Cell Memory + Platform VMs) / Usable Capacity × 100`

Where Usable Capacity = Total Cluster Memory - Reserved Capacity (HA or N-1, whichever is greater)

### Staging Capacity (Free Chunks)

Available memory chunks for `cf push` staging operations. Chunk size is auto-detected from your workload profile.

| Message                            | Severity | Triggered When   | What It Means                                                                 |
| ---------------------------------- | -------- | ---------------- | ----------------------------------------------------------------------------- |
| **Critical: Low staging capacity** | Critical | Free Chunks < 10 | Fewer than 10 concurrent staging operations possible. Deployments will queue. |
| **Low staging capacity**           | Warning  | Free Chunks < 20 | Limited staging parallelism. May queue during busy deployment periods.        |

**Formula:** `Free Chunks = (App Capacity - Total App Memory) / Chunk Size`

**Chunk Size:** Auto-detected from `Total App Memory / Total App Instances`. Defaults to 4GB when no app data is available. Java-heavy platforms typically show ~4GB chunks; Go/Python workloads may show 1-2GB.

### Cell Memory Utilization

Percentage of Diego cell memory capacity consumed by running apps.

| Message                              | Severity | Triggered When    | What It Means                                                                                       |
| ------------------------------------ | -------- | ----------------- | --------------------------------------------------------------------------------------------------- |
| **Cell utilization critically high** | Critical | Utilization > 90% | Near capacity exhaustion. New apps may fail to stage. Existing apps at risk during cell evacuation. |
| **Cell utilization elevated**        | Warning  | Utilization > 80% | Limited headroom remaining. Monitor closely and plan capacity expansion.                            |

**Formula:** `Utilization = Total App Memory / App Capacity × 100`

### Disk Utilization

Percentage of Diego cell disk capacity consumed by app droplets and containers.

| Message                              | Severity | Triggered When         | What It Means                                                                        |
| ------------------------------------ | -------- | ---------------------- | ------------------------------------------------------------------------------------ |
| **Disk utilization critically high** | Critical | Disk Utilization > 90% | Near disk exhaustion. Staging failures likely. Apps with local file writes may fail. |
| **Disk utilization elevated**        | Warning  | Disk Utilization > 80% | Limited disk headroom. Large droplets or file-heavy apps may encounter issues.       |

**Formula:** `Disk Utilization = Total App Disk / Disk Capacity × 100`

### Scheduling Performance (TPS)

Modeled scheduler throughput based on cell count.

| Message                                                          | Severity | Triggered When                           | What It Means                                                                                                                              |
| ---------------------------------------------------------------- | -------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| **Cell count (N) causes severe scheduling degradation (~X TPS)** | Critical | TPS Status = "critical" (< 50% of peak)  | Diego's BBS scheduler is overwhelmed by coordination overhead. App starts/restarts will be severely delayed. Consider larger, fewer cells. |
| **Cell count (N) may cause scheduling latency (~X TPS)**         | Warning  | TPS Status = "degraded" (50-79% of peak) | Noticeable scheduling delays during scaling events or deployments. Monitor app start times.                                                |

**Reference:** Peak TPS occurs around 3 cells (~1,964 TPS). Performance degrades as cell count increases due to coordination overhead.

### Cell Failure Resilience (Blast Radius)

Measures the percentage of total capacity at risk if a single Diego cell fails.

| Message                                                                   | Severity | Triggered When     | What It Means                                                                                                                |
| ------------------------------------------------------------------------- | -------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| **High cell failure impact: single cell loss affects X% of capacity**     | Critical | Blast Radius > 20% | Very few cells (5 or fewer). A single cell failure has outsized impact on workload capacity. Not recommended for production. |
| **Elevated cell failure impact: single cell loss affects X% of capacity** | Warning  | Blast Radius > 10% | Low cell count (10 or fewer). Consider whether this resilience level is acceptable for your workload criticality.            |

**Formula:** `Blast Radius = 100 / Cell Count`

### Resilience Change Indicator

The cell configuration comparison shows a resilience indicator between current and proposed:

| Indicator           | Blast Radius | Cell Count | Meaning                                                             |
| ------------------- | ------------ | ---------- | ------------------------------------------------------------------- |
| **✓ Low risk**      | ≤ 5%         | 20+ cells  | Highly resilient; single cell failures have minimal impact          |
| **⚠ Moderate risk** | 5-15%        | 7-20 cells | Acceptable for most workloads; monitor during failures              |
| **⚠ High risk**     | > 15%        | < 7 cells  | Significant impact from single failures; consider for dev/test only |

---

## Quick Reference: All Thresholds

| Metric             | Good       | Warning     | Critical   |
| ------------------ | ---------- | ----------- | ---------- |
| Capacity (HA/N-1)  | < 75%      | 75-85%      | > 85%      |
| Memory Utilization | < 80%      | 80-90%      | > 90%      |
| Disk Utilization   | < 80%      | 80-90%      | > 90%      |
| Free Chunks        | ≥ 20       | 10-19       | < 10       |
| TPS Performance    | ≥ 80% peak | 50-79% peak | < 50% peak |
| Blast Radius       | ≤ 5%       | 5-20%       | > 20%      |
| Fault Impact       | < 25       | 25-49       | ≥ 50       |
| Instances/Cell     | < 30       | 30-49       | ≥ 50       |
| vCPU:pCPU Ratio    | ≤ 4:1      | 4:1-8:1     | > 8:1      |

---

## Glossary

Common terms used throughout the TAS Capacity Analyzer:

| Term                            | Definition                                                                                                                                                                                                                                                                 |
| ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Diego Cell**                  | A VM that runs application containers in TAS/Cloud Foundry. Each cell can host multiple app instances. Diego is the container orchestration system that schedules and manages apps.                                                                                        |
| **Isolation Segment**           | A dedicated pool of Diego cells for running specific workloads. Apps in one segment cannot run on another segment's cells. Used to separate production from dev, or to isolate tenants.                                                                                    |
| **Garden**                      | The container runtime that runs on Diego cells. Manages the lifecycle of application containers, similar to Docker's role in Kubernetes.                                                                                                                                   |
| **N-1 / N-X Tolerance**         | Fault tolerance notation. N-1 means the system can survive 1 host failure; N-2 means 2 host failures, etc. Determined by how much spare capacity is reserved for failover.                                                                                                 |
| **HA Admission Control**        | A vSphere cluster setting that reserves a percentage of resources for VM failover. If set to 25%, vSphere won't let you deploy VMs that would consume more than 75% of cluster capacity.                                                                                   |
| **Memory Overcommit (Diego)**   | An advanced TAS feature that makes Diego cells advertise more memory than they physically have. Configured in Ops Manager under TAS → Advanced Features. Different from vSphere memory overcommit.                                                                         |
| **Memory Overcommit (vSphere)** | Hypervisor-level memory oversubscription where total VM memory exceeds physical RAM. Triggers ballooning and swapping under pressure. Not recommended for Diego cells.                                                                                                     |
| **Ballooning**                  | A vSphere memory reclamation technique. When hosts run low on RAM, the balloon driver inflates inside guest VMs, forcing the guest OS to page memory to disk. Causes unpredictable latency.                                                                                |
| **vCPU:pCPU Ratio**             | The ratio of virtual CPU cores allocated to VMs versus physical CPU cores available. A 4:1 ratio means 4 vCPUs per physical core. Higher ratios increase CPU contention risk.                                                                                              |
| **Staging**                     | The `cf push` process where Cloud Foundry builds your app into a runnable droplet. Requires temporary memory (typically 4GB) on Diego cells. Low staging capacity causes deployment queuing.                                                                               |
| **Free Chunks**                 | Available memory blocks for staging operations. Chunk size is auto-detected from average app instance memory (defaults to 4GB). Calculated as (App Capacity - App Memory) / Chunk Size. Below 10 chunks indicates deployment bottleneck risk.                              |
| **Chunk Size**                  | The memory allocation unit used for staging capacity calculations. Auto-detected from your workload profile (Total App Memory / Total Instances). Java-heavy platforms typically show ~4GB; Go/Python workloads show 1-2GB. Defaults to 4GB when no app data is available. |
| **TPS (Tasks Per Second)**      | Diego scheduler throughput, measuring how fast the BBS can place app instances. More cells = more coordination overhead = lower TPS. This is a trade-off: capacity vs. scheduling speed.                                                                                   |
| **Blast Radius**                | The percentage of total capacity affected by a single cell failure. Calculated as 100 / Cell Count. Lower is better for fault tolerance.                                                                                                                                   |
| **App Capacity**                | Memory available for running application containers, after subtracting system overhead. Calculated as Cell Memory × (1 - Memory Overhead %).                                                                                                                               |
| **Memory Overhead**             | The percentage of Diego cell memory consumed by Garden runtime, system processes, and the Diego executor (default 7%). Not available for application containers.                                                                                                           |
| **BOSH**                        | The deployment and lifecycle management tool for Cloud Foundry. Manages Diego cell VMs, handles health monitoring, and provides the vitals data shown in the dashboard.                                                                                                    |
| **Log Cache**                   | A CF component that stores recent container metrics. Used by the analyzer to get actual (not just allocated) memory consumption per app.                                                                                                                                   |
