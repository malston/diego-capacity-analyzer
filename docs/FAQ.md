# Diego Capacity Analyzer -- FAQ

Likely questions from team demos and how to answer them.

---

## General Questions

### Q: What problem does this solve?

**A:** It answers "will my workloads fit?" without spreadsheets.

Before this tool, you'd pull data from BOSH, CF API, and vSphere separately, merge it manually, and do N-1 HA math by hand. Every "what-if" scenario meant starting over. This tool does all that in real-time.

---

### Q: Where does the data come from?

**A:** Four sources:

| Source        | What we get                                    |
| ------------- | ---------------------------------------------- |
| **BOSH**      | Diego cell VMs, memory/CPU vitals              |
| **CF API**    | Apps, processes, isolation segments            |
| **Log Cache** | _Actual_ container memory (not just allocated) |
| **vSphere**   | Host/cluster inventory                         |

The key insight is Log Cache--it tells us what apps actually use, not just what they requested.

---

### Q: Can I use it without connecting to a real environment?

**A:** Yes! Load a sample JSON file. We have 9 pre-built scenarios from small dev to 1000-cell enterprise scale. No credentials needed.

---

### Q: What's the difference between "Allocated" and "Used" memory?

**A:**

- **Allocated** = what developers requested via `cf push -m 1G`
- **Used** = what the app actually consumes at runtime

The gap is wasted capacity. That's your right-sizing opportunity.

---

## Dashboard Questions

### Q: Why is utilization low but I'm being told I need more capacity?

**A:** Look at _Allocated_ vs _Used_. High allocation + low utilization = apps are over-provisioned. Right-size your apps before adding cells.

---

### Q: What's a healthy utilization target?

**A:**

- **60-75%** = Good headroom for spikes and deployments
- **< 50%** = Underutilized, consolidation opportunity
- **> 80%** = Too hot, risk of exhaustion during deploys

---

### Q: What does the What-If slider do?

**A:** It models memory overcommit. Moving from 1.0x to 1.5x means "what if we promised 50% more memory than we physically have?"

This works because apps rarely use 100% of their allocation simultaneously. But overcommit too much and you risk OOM kills if apps spike together.

| Ratio    | Risk   | Use case   |
| -------- | ------ | ---------- |
| 1.0-1.3x | Low    | Production |
| 1.3-2.0x | Medium | Dev/test   |
| 2.0-3.0x | High   | Labs only  |

---

### Q: What's the connection between overcommit and memory ballooning?

**A:** Overcommit creates memory pressure; ballooning is how vSphere reclaims memory when that pressure hits.

When VMs demand more RAM than physically exists, vSphere uses this hierarchy:

1. **Transparent Page Sharing** -- dedupe identical pages (low impact)
2. **Ballooning** -- inflate balloon driver, force guest to page (medium impact)
3. **Compression** -- compress cold pages (medium impact)
4. **Host Swapping** -- hypervisor swaps to disk (severe impact)

The **1.3x threshold** is where ballooning becomes unlikely under normal load. Above that, you're betting workloads won't spike together.

**Diego impact:** When ballooning hits Diego cells, container memory limits become unreliable, apps get unexpected OOM kills, and `cf push` may timeout.

**Bottom line:** If you see `Balloon > 0` on Diego cells in vSphere, reduce overcommit or add hosts.

---

## Scenario Analysis Questions

### Q: Why did I get a "NO" answer?

**A:** Check the warnings. The most common reasons:

1. **Exceeds N-1/HA capacity** -- Your cells + platform VMs exceed what's usable after reserving for host failure tolerance
2. **Memory utilization > 90%** -- Apps fill the cell capacity
3. **vCPU:pCPU ratio too high** -- CPU oversubscription beyond your target

The bottleneck card tells you _which_ resource is constraining you.

---

### Q: What's the difference between N-1 and HA Admission Control?

**A:**

| Constraint       | How it works                                    |
| ---------------- | ----------------------------------------------- |
| **N-1**          | Simple: reserve one host's worth of memory      |
| **HA Admission** | vSphere setting: reserve X% of cluster capacity |

The tool compares both and shows whichever is more restrictive. HA Admission is what vSphere actually enforces--you can't deploy VMs beyond its limit.

**Example:** On a 15-host cluster with 2 TB/host:

- HA 25% reserves 7.5 TB (≈ N-3)
- N-1 reserves 2 TB

HA wins--it's more restrictive.

---

### Q: What's a "free chunk"?

**A:** A memory block available for `cf push` staging. Chunk size is auto-detected from your average app instance memory (defaults to 4GB if unavailable). When you push an app, Diego needs a chunk to stage the droplet before starting containers.

- **≥ 20 chunks** = healthy
- **10-19** = limited, may queue during busy periods
- **< 10** = constrained, deployment bottleneck

The UI displays the actual chunk size used (e.g., "2GB chunks for staging" for Go/Python workloads, "4GB chunks" for Java-heavy platforms).

---

### Q: Why does TPS drop with more cells?

**A:** Diego's BBS (bulletin board system) scheduler coordinates all cells. More cells = more coordination overhead.

Peak TPS (~1,964) happens around 3 cells. At 100+ cells, you're down to ~1,400 TPS. At 200+, you see severe degradation.

**If you need more capacity:** Consider larger cells instead of more cells to avoid scheduler bottlenecks.

---

### Q: What's "blast radius"?

**A:** The percentage of capacity lost if one cell fails.

```
Blast Radius = 100 / Cell Count
```

- 5 cells = 20% blast radius (bad)
- 20 cells = 5% blast radius (good)
- 100 cells = 1% blast radius (excellent)

Low cell counts mean each failure has outsized impact.

---

### Q: Why 7% memory overhead?

**A:** That's what Garden runtime, the Diego executor, and OS processes consume inside a Diego cell. It's an empirical estimate--your actual overhead may vary.

You can adjust this in Advanced Options if you've measured something different.

---

### Q: Are HA Admission and Memory Overhead the same thing?

**A:** No! They're different layers:

| Setting         | Layer             | What it reserves                         |
| --------------- | ----------------- | ---------------------------------------- |
| HA Admission    | vSphere cluster   | Memory to restart VMs after host failure |
| Memory Overhead | Inside Diego cell | Memory for Garden/system processes       |

You need both. HA determines if you can _deploy_ the VMs. Overhead determines how much _app workload_ fits inside them.

---

## Recommendations Questions

### Q: How do I read the recommendations?

**A:** Each recommendation has:

- **Type** badge: Scale-out, Scale-up, Infrastructure, Optimization
- **Description**: What to change
- **Impact**: Specific improvement (e.g., "+256 GB capacity")
- **Priority**: 1 = most impactful

Start with Priority 1. It addresses your bottleneck.

---

### Q: Why does it recommend adding hosts when I could add cells?

**A:** If you're constrained by N-1/HA capacity, adding cells doesn't help--you've already hit the limit. Adding hosts increases the total pool and reduces the percentage impact of any single host failure.

---

### Q: What if I disagree with the recommendation?

**A:** The recommendations are suggestions based on your current bottleneck. You know your environment best--if a recommendation doesn't fit your constraints (budget, data center space, etc.), consider the alternatives it offers.

---

## Technical Questions

### Q: How accurate is the TPS curve?

**A:** It's modeled from Diego benchmark data, not measured live. Actual TPS varies based on:

- Network latency
- Database backend
- Workload characteristics

You can customize the curve in Advanced Options to match your observed performance.

---

### Q: Does this work with isolation segments?

**A:** Yes. The dashboard shows cells grouped by segment, and the pie chart shows segment distribution. Each segment's cells are counted in capacity calculations.

---

### Q: What about Small Footprint TAS/TPCF?

**A:** Detection matches `compute` VMs (Diego colocated on compute instances) in addition to standard `diego_cell` naming.

---

### Q: Can I export the results?

**A:** Currently the results are displayed in the UI. For CI/CD integration, use the CLI with `--json` flag for machine-readable output.

---

## "But wait..." Questions

### Q: Can I run this in production?

**A:** The tool itself is read-only--it queries APIs but doesn't make changes. Safe to point at production BOSH/CF. The only consideration is API rate limits if you refresh constantly.

---

### Q: What if my BOSH is behind a NAT/jumpbox?

**A:** The backend supports SSH tunneling via SOCKS5 proxy through Ops Manager. Set `OM_PRIVATE_KEY` and let `generate-env.sh` configure the tunnel.

---

### Q: Why don't you integrate with [some other tool]?

**A:** We'd love to hear what integration would help! File an issue or let me know. The architecture is designed to add new data sources.

---

## Quick Answers

| Question                       | One-liner                                |
| ------------------------------ | ---------------------------------------- |
| What's the main metric?        | N-1/HA Utilization--stay under 85%        |
| Healthy utilization?           | 60-75%                                   |
| How many chunks is safe?       | ≥ 20                                     |
| Max vCPU ratio for production? | 4:1 or less                              |
| Why "no" answer?               | Check bottleneck card for which resource |
| What's 7% overhead?            | Garden/system processes inside the cell  |
| Overcommit in production?      | Stay at 1.0-1.3x max                     |
