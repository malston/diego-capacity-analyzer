# UI Guide - Understanding the Dashboard

A quick reference explaining what each metric and visualization means.

---

## Key Metrics Cards

| Metric | What It Means |
|--------|---------------|
| **Total Cells** | Number of Diego cells (VMs that run app containers). More cells = more capacity for workloads. |
| **Utilization %** | Percentage of total memory actively being used by running apps. Low = underutilized infrastructure. High (>80%) = risk of capacity exhaustion. |
| **Avg CPU %** | Average processor load across cells. Sustained >70% indicates CPU contention risk - apps compete for cycles and slow down. |
| **Wasted Memory** | Memory apps reserved but aren't using. This is "paid for but idle" capacity that could be reclaimed through right-sizing. |

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

| Ratio | Risk | When to Use |
|-------|------|-------------|
| **1.0x** | None | Mission-critical workloads, compliance requirements |
| **1.2-1.3x** | Low | Well-understood workloads with predictable memory |
| **1.5x** | Medium | Dev/test environments, stateless apps |
| **2.0x** | High | Only with robust monitoring; expect occasional OOM |

**Warning:** Overcommit lets you pack more apps, but if apps spike memory simultaneously, you risk OOM kills. Use cautiously and monitor closely.

---

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
