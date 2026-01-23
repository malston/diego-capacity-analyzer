# Diego Capacity Analyzer — R&D Demo Plan

**Date:** January 2025
**Duration:** ~25 minutes + Q&A
**Format:** Slides + Live Demo
**Audience:** R&D team (deep TAS/Diego expertise)
**Goals:** Adoption pitch + Technical showcase

---

## Demo Structure Overview

| Section               | Time  | Content                                             |
| --------------------- | ----- | --------------------------------------------------- |
| Slides: The Problem   | 3 min | Title, problem statement, solution teaser           |
| Live: Dashboard       | 5 min | Sample data → metrics → What-If mode                |
| Live: Scenario Wizard | 8 min | Full wizard → "Will It Fit?" → bottleneck detection |
| Slides: Architecture  | 5 min | Data flow, integrations, capacity engine            |
| Live: Technical Depth | 4 min | Swagger UI + CLI/TUI                                |
| Close                 | 2 min | Try it yourself + Q&A invitation                    |

---

## Pre-Demo Setup Checklist

- [ ] Frontend running: `make frontend-dev`
- [ ] Browser in dark mode (colorblind-friendly theme)
- [ ] Sample data loaded: `large-foundation.json` for Dashboard
- [ ] Second sample ready: `multi-cluster-enterprise.json` for Scenario Wizard
- [ ] Swagger UI tab open: `http://localhost:8080/docs`
- [ ] Terminal with CLI built: `make build`
- [ ] Slides loaded and tested with projector/screen share

---

## Section 1: Opening Slides (3 min)

### Slide 1: Title

- "Diego Capacity Analyzer"
- Presenter name, team, date
- Tagline: "Capacity planning for TAS, without the spreadsheets"

### Slide 2: The Problem

**Headline:** "How do you answer 'will my workloads fit?'"

Bullets:

- Manual spreadsheet calculations across BOSH, CF, vSphere data
- N-1 HA math done by hand
- No single view of memory, CPU, disk constraints
- "What-if" scenarios require re-pulling all the data

_Goal: Set up the pain they recognize_

### Slide 3: The Solution

**Headline:** "One dashboard. Real-time data. Instant modeling."

- Screenshot or GIF of the dashboard
- Transition: "Let me show you..."

---

## Section 2: Live Demo — Dashboard (5 min)

### Setup

- Pre-load `large-foundation.json` (250 cells, 2 clusters)
- Browser in dark mode

### Demo Flow

**Step 1: Orient (30 sec)**

> "This is a production-scale foundation—250 Diego cells across two clusters. The data came from a JSON export, but in real deployments this pulls live from BOSH, CF, and vSphere."

**Step 2: Metric Cards (30 sec)**

- Point out: Total Cells, Memory Utilization %, Average CPU
  > "At a glance, I can see we're at 73% memory utilization across the foundation."

**Step 3: Cell Capacity Chart (1 min)**

- Highlight stacked bar chart (Used / Allocated / Available)
  > "The green is what's actually consumed. Yellow is allocated but not used—that's your overcommit opportunity. Gray is truly free."

**Step 4: Right-Sizing Recommendations (1 min)**

- Scroll to recommendations section
  > "The system automatically identifies apps that are over-provisioned. This one app alone could free up 8GB if right-sized."

**Step 5: What-If Mode (2 min)** — _The "aha" moment_

- Toggle What-If Mode ON
- Drag Memory Overcommit slider from 1.0x to 1.3x
  > "Watch what happens to available capacity as I model 30% memory overcommit..."
- Show chart updating in real-time
  > "Without changing any infrastructure, I've just unlocked 20% more headroom. That's the kind of insight that used to take a day of spreadsheet work."

### Transition

> "But what if I want to go deeper—model new cell sizes, different host counts, or see exactly what's constraining me? That's where the Scenario Analyzer comes in."

---

## Section 3: Live Demo — Scenario Wizard (8 min)

### Setup

- Switch to `multi-cluster-enterprise.json` (1000 cells, 3 AZs)
- Or continue with large-foundation for continuity

### Demo Flow

**Step 1: Launch Wizard (30 sec)**

- Click "Scenarios" tab → "Run Analysis"
  > "Let's model a real capacity question: what if we doubled the memory on each Diego cell?"

**Step 2: Resource Types (30 sec)**

- Select Memory + CPU + Disk
  > "I want to see constraints across all resource types, not just memory."

**Step 3: Cell Configuration (1.5 min)**

- Show current values (e.g., 64GB cells)
- Change to 128GB cells
  > "We're modeling a scale-up: moving from 64GB to 128GB Diego cells. The system will calculate how many of these larger cells fit in our infrastructure."

**Step 4: CPU Configuration (1.5 min)**

- Show vCPU:pCPU ratio slider
- Point out risk indicators (Conservative / Moderate / Aggressive)
  > "This is where you model CPU oversubscription. At 4:1, we're in moderate territory. Push to 8:1 and the system warns you that you're in aggressive overcommit."

**Step 5: Host Configuration (1 min)**

- Show host count, HA settings, Admission Control %
  > "Here's where N-1 tolerance matters. If I lose one host, can my workloads still run? The system calculates this automatically."

**Step 6: Results (3 min)** — _The payoff_

| Component             | Talking Point                                                                                                           |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| "Will It Fit?" banner | "Green checkmark—yes, 128GB cells will fit. If it didn't, this would be red with the specific constraint."              |
| Capacity Gauges       | "Memory at 68%, CPU at 45%, Disk at 32%. Memory is my tightest resource."                                               |
| Bottleneck Card       | "Here's the insight: I'm constrained by N-1 HA, not raw memory. If I add one more host, I unlock significant headroom." |
| Staging Capacity      | "22 free 4GB chunks—that's how many app instances I can stage concurrently during a push."                              |
| Recommendations       | "Prioritized suggestions: add hosts first (high impact), then consider scale-out over scale-up."                        |

### Transition

> "What you just saw queries four different APIs and runs the calculations in real-time. Let me show you how that works under the hood."

---

## Section 4: Architecture Slides (5 min)

### Slide 4: Data Flow Diagram

Visual showing:

```
BOSH Director ──┐
CF API ─────────┼──▶ Go Backend ──▶ REST API ──▶ React Frontend
Log Cache ──────┤         │
vSphere/vCenter ┘         ▼
                    Unified Capacity Model
```

### Slide 5: Integration Details

| Source    | What We Get                         | How                           |
| --------- | ----------------------------------- | ----------------------------- |
| BOSH      | Diego cell VMs + vitals             | UAA OAuth, deployment queries |
| CF API    | Apps, processes, isolation segments | OAuth2, process stats         |
| Log Cache | Actual container memory             | PromQL-style queries          |
| vSphere   | Host/cluster inventory              | govmomi library               |

### Slide 6: Capacity Engine

- N-1 HA calculation: "If one host fails, do remaining hosts have capacity?"
- Multi-resource bottleneck detection: memory, CPU, disk, host count
- Scenario comparison: current state vs. proposed changes

---

## Section 5: Live Demo — Technical Depth (4 min)

### Part A: Swagger UI (2.5 min)

- Open `/docs` route

  > "The entire backend is documented with OpenAPI. This isn't just generated stubs—it's the live spec powering the app."

- Expand `POST /api/v1/scenario/compare`
- Show request/response schemas

  > "Full type definitions, example payloads, error responses."

- Execute a live call (health endpoint)
  > "This isn't mocked—it's hitting the actual backend."

### Part B: CLI/TUI (1.5 min)

```bash
./diego-capacity
```

- Show full-screen TUI

  > "Same capabilities, terminal interface. For platform engineers who live in the CLI."

- Demo keyboard shortcuts: `w` for wizard, `r` for refresh
  > "Keyboard-driven—no mouse needed. Designed for SSH sessions."

### Transition

> "So that's the stack: React frontend for visual workflows, Go backend for the heavy lifting, CLI for automation."

---

## Section 6: Closing (2 min)

### Slide 7: Get Started

**Headline:** "Try It Yourself"

- GitHub repo location
- Quick start: `make frontend-dev` + load sample file
- No credentials needed for demo mode
  > "You can be running this in 5 minutes with sample data."

### Slide 8: What's Next

**Headline:** "Feedback Welcome"

- Areas for feedback
- How to reach you
  > "I'd love feedback—what would make this useful for your workflow?"

### Verbal Close

> "That's Diego Capacity Analyzer: real-time visibility, instant what-if modeling, and actionable recommendations. Questions?"

---

## Key Messages to Reinforce

1. **Solves a real pain:** Replaces manual spreadsheet work with real-time answers
2. **Multi-resource intelligence:** Memory, CPU, disk, AND N-1 HA in one view
3. **Actionable output:** Not just data—prioritized recommendations
4. **Production-ready:** Full API docs, CLI, test coverage, error handling
5. **Low barrier to try:** Sample data mode, no credentials required

---

## Avoid

- Competitive comparisons (e.g., don't mention Tanzu Hub)
- CI/CD pipeline integration details
- Features that aren't production-ready
