# Diego Capacity Analyzer -- Demo Script (Cheat Sheet)

**Print this single page. Follow it during the demo.**

---

## SETUP (Before Demo)

```bash
make frontend-dev                    # Start frontend
open http://localhost:5173           # Browser in dark mode
```

- [ ] Load `large-foundation.json` in Dashboard
- [ ] Have `multi-cluster-enterprise.json` ready
- [ ] Open `http://localhost:8080/docs` in second tab
- [ ] Terminal with `./diego-capacity` ready

---

## FLOW (25 min)

### 1. SLIDES: THE PROBLEM (3 min)

| Slide    | Say                                                                                                |
| -------- | -------------------------------------------------------------------------------------------------- |
| Title    | "Diego Capacity Analyzer--capacity planning without the spreadsheets."                              |
| Problem  | "How do you answer 'will it fit?' today? Spreadsheets, manual math, start over for every what-if." |
| Solution | "One dashboard, real-time data, instant modeling. Let me show you."                                |

**→ Switch to browser**

---

### 2. DASHBOARD DEMO (5 min)

| Action                  | Say                                                                    |
| ----------------------- | ---------------------------------------------------------------------- |
| Point at sample data    | "500 cells, 2 clusters. JSON file, but could be live BOSH/CF/vSphere." |
| Metric cards            | "73% memory utilization at a glance."                                  |
| Capacity chart          | "Green = used. Yellow = allocated not used. Gray = free."              |
| Recommendations         | "Auto-detected: this app could free 8GB if right-sized."               |
| **Toggle What-If ON**   | "Watch this..."                                                        |
| **Drag slider to 1.3x** | "30% overcommit = 20% more headroom. No infrastructure changes."       |

**→ Transition:** "What if I want to model new cell sizes or see what's actually constraining me?"

---

### 3. SCENARIO WIZARD (8 min)

| Action                         | Say                                                                    |
| ------------------------------ | ---------------------------------------------------------------------- |
| Click Scenarios → Run Analysis | "What if we doubled cell memory?"                                      |
| Step 1: Select all resources   | "Memory, CPU, and disk--full picture."                                  |
| Step 2: Change 64GB → 128GB    | "Scale-up scenario."                                                   |
| Step 3: Show CPU ratio slider  | "4:1 is moderate. 8:1 triggers warnings."                              |
| Step 4: Show HA settings       | "N-1 tolerance calculated automatically."                              |
| **Results: Will It Fit?**      | "Green checkmark--yes, it fits."                                        |
| **Bottleneck card**            | "Constrained by N-1 HA, not raw memory. Add one host = more headroom." |
| **Recommendations**            | "Prioritized: add hosts first, then scale-out."                        |

**→ Transition:** "Four APIs, real-time calculations. Here's how it works."

---

### 4. SLIDES: ARCHITECTURE (5 min)

| Slide           | Say                                                                    |
| --------------- | ---------------------------------------------------------------------- |
| Data flow       | "BOSH, CF, Log Cache, vSphere → unified model → REST API → React."     |
| Integrations    | "Log Cache gives actual memory, not just allocated. Key for accuracy." |
| Capacity engine | "N-1 HA, multi-resource bottlenecks, scenario comparison--automatic."   |

**→ Switch to browser**

---

### 5. TECHNICAL DEPTH (4 min)

| Action                 | Say                                          |
| ---------------------- | -------------------------------------------- |
| Open `/docs` (Swagger) | "Full OpenAPI spec. Live, not mocked."       |
| Expand an endpoint     | "Request schema, response schema, examples." |
| Try It Out → Execute   | "Hitting real backend right now."            |
| **Switch to terminal** |                                              |
| Run `./diego-capacity` | "Same features, terminal interface."         |
| Press `w`, then `r`    | "Keyboard-driven. SSH-friendly."             |

**→ Transition:** "React frontend, Go backend, CLI. Full stack."

---

### 6. CLOSE (2 min)

| Slide       | Say                                                    |
| ----------- | ------------------------------------------------------ |
| Get Started | "5 minutes to run with sample data. No creds needed."  |
| Feedback    | "What would make this useful for you? What's missing?" |

**→ "Questions?"**

---

## KEY PHRASES TO HIT

- "Without changing any infrastructure..."
- "That used to take a day of spreadsheet work"
- "Not just data--prioritized recommendations"
- "Constrained by X, not Y" (bottleneck insight)
- "Five minutes to try it yourself"

---

## DO NOT MENTION

- Tanzu Hub or competitive tools
- CI/CD pipeline integration
- Incomplete features

---

## IF SOMETHING BREAKS

| Problem                | Recovery                                                                                |
| ---------------------- | --------------------------------------------------------------------------------------- |
| Sample won't load      | "Let me switch to another sample file..."                                               |
| Backend not responding | "I'll show the pre-recorded GIF instead" → `docs/images/tas-capacity-analyzer-demo.gif` |
| Lost my place          | Glance at this cheat sheet                                                              |

---

## SAMPLE DATA REFERENCE

| File                            | Scale                 | Best for                     |
| ------------------------------- | --------------------- | ---------------------------- |
| `large-foundation.json`         | 500 cells, 2 clusters | Dashboard demo               |
| `multi-cluster-enterprise.json` | 1000 cells, 3 AZs     | Scenario wizard              |
| `cpu-constrained.json`          | 128 cells, 8:1 vCPU   | Showing constraint detection |
