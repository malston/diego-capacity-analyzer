# UI Guide - Understanding the Dashboard

A quick reference explaining what each metric and visualization means.

> **Tip:** Most metrics and gauges have hover tooltips in the UI. Hover over any metric to see a brief explanation.

---

## Key Metrics Cards

| Metric | What It Means |
|--------|---------------|
| **Total Cells** | Number of Diego cells (VMs that run app containers). More cells = more capacity for workloads. |
| **Utilization %** | Percentage of total memory actively being used by running apps. Low = underutilized infrastructure. High (>80%) = risk of capacity exhaustion. |
| **Avg CPU %** | Average processor load across cells. Sustained >70% indicates CPU contention risk - apps compete for cycles and slow down. |
| **Unused Memory** | Memory apps reserved but aren't using. This is "paid for but idle" capacity that could be reclaimed through right-sizing. |

---

## Diego Cells Detail Table

| Column | What It Means |
|--------|---------------|
| **Cell** | The Diego cell VM name (e.g., `diego_cell/0`) |
| **Segment** | Isolation segment the cell belongs to - used to separate workloads (e.g., prod vs dev) |
| **Capacity** | Total memory available on this cell (what the VM was sized to) |
| **Allocated** | Memory reserved by apps scheduled to this cell (sum of app memory quotas) |
| **Used** | Memory actually consumed by running processes (always ≤ Allocated) |
| **CPU %** | Current CPU utilization on this cell. Color-coded: 🟢 <50%, 🟡 50-70%, 🔴 >70% |
| **Utilization Bar** | Visual representation of Used/Capacity. Shows how "full" the cell is. |

**Key insight:** The gap between Allocated and Used = memory apps requested but aren't consuming. This is per-cell waste.

---

## Cell Capacity Chart (Stacked Bar)

| Segment | What It Means |
|---------|---------------|
| **Used (blue)** | Memory actively consumed by app processes |
| **Allocated unused (green)** | Reserved by apps but sitting idle |
| **Available (gray)** | Free capacity - room for new apps |

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

| Column | What It Means |
|--------|---------------|
| **App Name** | Application that's over-provisioned |
| **Instances** | Number of running instances |
| **Overhead %** | How much extra memory was requested vs actually used. >30% = significant waste |
| **Requested** | Memory quota the app asks for (what developers set via `cf push -m`) |
| **Actual Usage** | Memory the app actually consumes at runtime |
| **Recommended** | Suggested quota (actual usage + 20% headroom) |
| **Savings/Instance** | Memory you'd reclaim per instance by right-sizing |

**Example:** "This app requests 1024 MB but only uses 780 MB. Reduce quota to 936 MB (780 + 20% buffer) and reclaim 88 MB × instances."

**Why 20% buffer?** Accounts for memory spikes, garbage collection, and safety margin against OOM kills.

---

## What-If Mode

Toggle via the **What-If Mode** button. Explores: "What if I enabled memory overcommit?"

| Metric | What It Means |
|--------|---------------|
| **Overcommit Ratio** | Memory multiplier. 1.0x = no overcommit. 1.5x = sell 50% more capacity than physically exists. |
| **New Capacity** | Virtual capacity after applying overcommit ratio |
| **Current Instances** | How many app instances are running now |
| **Additional Capacity** | How many more 512MB instances could fit with overcommit |

### Overcommit Risk Levels

| Ratio | Risk | Typical Use |
|-------|------|-------------|
| **1.0-1.3x** | Low | Production, mission-critical |
| **1.5-2.0x** | Medium | Dev/test, well-understood workloads |
| **2.0-3.0x** | High | Labs, demos, low-traffic environments |
| **3.0x+** | Very High | Labs with minimal app utilization |

**Warning:** Overcommit lets you pack more apps, but if apps spike memory simultaneously, you risk OOM kills. Use cautiously and monitor closely.

**Real-world example:** A Small Footprint TPCF lab might run 3.75x overcommit (61 GB advertised on a 16 GB cell) because lab apps have minimal utilization. This would cause OOM kills under production traffic.

---

# Scenario Analysis Tab

Answers: **"Will my workload fit if I change my cell configuration?"**

---

## Loading Infrastructure Data

| Source | Description |
|--------|-------------|
| **Upload JSON** | Manual infrastructure data (clusters, cells, apps) |
| **vSphere Live** | Real-time data from vSphere via backend |
| **BOSH Live** | Real-time data from BOSH Director via backend |
| **Sample Data** | Pre-loaded example datasets for demos |

After loading data, a **Current Configuration** summary appears showing your existing cell count, size, and total capacity. This helps you understand what you're comparing against before proposing changes.

---

## Proposed Configuration

| Input | What It Means |
|-------|---------------|
| **VM Size Preset** | Common cell sizes: Small (4 vCPU/32 GB), Medium (8/64), Large (16/128), or Custom |
| **Cell Count** | Number of Diego cells in your proposed configuration |
| **Memory Overhead %** | System memory reserved for Diego/Garden (default 7%) |

**Hypothetical App:** Add a theoretical app to see if it would fit. Enter instance count and memory per instance.

---

## Results: Capacity Gauges

### N-1 Capacity

Can all VMs fit on remaining hosts if one ESXi host fails?

| Value | Status | Meaning |
|-------|--------|---------|
| **< 75%** | Good | Safe headroom for host failure |
| **75-85%** | Warning | Tight - may struggle after host loss |
| **> 85%** | Critical | Cannot survive a host failure |

**Key insight:** N-1 is about **host** failure (losing all cells on one ESXi host), not individual cell failure. If you can survive losing ~30 cells at once (one host), you can easily handle BOSH rolling upgrades which only remove one cell at a time.

### Memory Utilization

| Value | Status | Meaning |
|-------|--------|---------|
| **< 80%** | Good | Healthy headroom |
| **80-90%** | Warning | Getting tight |
| **> 90%** | Critical | Near capacity exhaustion |

### Staging Capacity (Free Chunks)

Available 4GB chunks for `cf push` staging operations.

| Chunks | Status | Meaning |
|--------|--------|---------|
| **> 400** | Good | Plenty of staging capacity |
| **200-400** | Warning | May queue during busy periods |
| **< 200** | Critical | Deployment bottleneck likely |

---

## Results: TPS Performance

**TPS = Tasks Per Second** - how fast Diego's scheduler can place app instances.

| Cell Count | TPS | Notes |
|------------|-----|-------|
| 3 | ~1,964 | Peak efficiency |
| 100 | ~1,389 | ~30% degradation |
| 210 | ~104 | Severe degradation |

**Why it matters:** More cells = more coordination overhead. If you need more capacity, consider larger cells instead of more cells to avoid scheduler bottlenecks.

---

## Results: Metric Scorecards

| Metric | What It Means | Good Direction |
|--------|---------------|----------------|
| **Cell Count** | Number of Diego cell VMs | Depends on strategy |
| **App Capacity** | Total memory available for apps | Higher = more headroom |
| **Fault Impact** | App instances displaced if one cell fails | Lower = smaller blast radius |
| **Instances/Cell** | Average app instances per cell | Lower = more distributed |

---

## Overall Status Banner

| Status | Meaning |
|--------|---------|
| **✓ YES** (green) | Configuration meets all requirements |
| **⚠ MAYBE** (amber) | Warnings to review before proceeding |
| **✗ NO** (red) | Critical issues - adjust configuration |

---

# Reference

## Data Sources

| Source | Description |
|--------|-------------|
| **CF API** | App instances, memory quotas, process stats |
| **BOSH** | Diego cell VMs, capacity, vitals |
| **Log Cache** | Real-time container memory metrics |
| **vSphere** (optional) | VM-level infrastructure metrics |

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
