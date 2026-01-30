# Diego Capacity Analyzer -- Slide Deck Outline

Use this markdown as a starting point for your slides. Import into your preferred tool (Google Slides, Keynote, PowerPoint, reveal.js, etc.).

---

## Slide 1: Title

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│           DIEGO CAPACITY ANALYZER                       │
│                                                         │
│     Capacity planning for TAS, without the spreadsheets │
│                                                         │
│                                                         │
│                    [Your Name]                          │
│                    [Team Name]                          │
│                    January 2025                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** Brief intro--who you are, what you'll cover in 25 minutes.

---

## Slide 2: The Problem

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│    HOW DO YOU ANSWER "WILL MY WORKLOADS FIT?"           │
│                                                         │
│    • Manual spreadsheet calculations                    │
│      - Pull data from BOSH, CF, vSphere separately      │
│      - Merge and cross-reference by hand                │
│                                                         │
│    • N-1 HA math done manually                          │
│      - "If one host fails, do we have capacity?"        │
│                                                         │
│    • No unified view of constraints                     │
│      - Memory? CPU? Disk? Host count?                   │
│                                                         │
│    • "What-if" = start over from scratch                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** Pause here. Let them feel the pain. They've done this work manually.

---

## Slide 3: The Solution

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│    ONE DASHBOARD. REAL-TIME DATA. INSTANT MODELING.     │
│                                                         │
│    ┌─────────────────────────────────────────────────┐  │
│    │                                                 │  │
│    │     [Screenshot: Dashboard with metrics]        │  │
│    │                                                 │  │
│    │     Or use: docs/images/dashboard.gif           │  │
│    │                                                 │  │
│    └─────────────────────────────────────────────────┘  │
│                                                         │
│                  Let me show you...                     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** Transition to live demo. Switch to browser with Dashboard loaded.

---

## Slide 4: Data Flow Architecture

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              HOW IT WORKS: DATA FLOW                    │
│                                                         │
│                                                         │
│   ┌──────────────┐                                      │
│   │ BOSH Director│───┐                                  │
│   └──────────────┘   │                                  │
│   ┌──────────────┐   │    ┌──────────┐    ┌─────────┐  │
│   │   CF API     │───┼───▶│    Go    │───▶│  React  │  │
│   └──────────────┘   │    │ Backend  │    │Frontend │  │
│   ┌──────────────┐   │    └──────────┘    └─────────┘  │
│   │  Log Cache   │───┤          │                      │
│   └──────────────┘   │          ▼                      │
│   ┌──────────────┐   │   ┌────────────┐                │
│   │   vSphere    │───┘   │  Unified   │                │
│   └──────────────┘       │  Capacity  │                │
│                          │   Model    │                │
│                          └────────────┘                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** After live demo. Explain how the four data sources feed into a unified model.

---

## Slide 5: Integration Details

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              FOUR APIs, ONE UNIFIED VIEW                │
│                                                         │
│   ┌────────────┬────────────────────┬────────────────┐  │
│   │  Source    │  What We Get       │  How           │  │
│   ├────────────┼────────────────────┼────────────────┤  │
│   │  BOSH      │  Diego cell VMs    │  UAA OAuth     │  │
│   │            │  Memory/CPU vitals │  Director API  │  │
│   ├────────────┼────────────────────┼────────────────┤  │
│   │  CF API    │  Apps, processes   │  OAuth2        │  │
│   │            │  Isolation segments│  /v3 endpoints │  │
│   ├────────────┼────────────────────┼────────────────┤  │
│   │  Log Cache │  Actual memory     │  PromQL-style  │  │
│   │            │  (not allocated)   │  queries       │  │
│   ├────────────┼────────────────────┼────────────────┤  │
│   │  vSphere   │  Hosts, clusters   │  govmomi       │  │
│   │            │  VM inventory      │  SDK           │  │
│   └────────────┴────────────────────┴────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** Emphasize Log Cache gives _actual_ memory, not just _allocated_--that's key for accurate planning.

---

## Slide 6: Capacity Engine

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              INTELLIGENT CAPACITY ENGINE                │
│                                                         │
│   N-1 HA CALCULATION                                    │
│   ─────────────────                                     │
│   "If one host fails, can remaining hosts              │
│    absorb all workloads?"                               │
│                                                         │
│   MULTI-RESOURCE BOTTLENECK DETECTION                   │
│   ────────────────────────────────────                  │
│   • Memory utilization                                  │
│   • CPU (with oversubscription modeling)                │
│   • Disk capacity                                       │
│   • Host count constraints                              │
│                                                         │
│   SCENARIO COMPARISON                                   │
│   ───────────────────                                   │
│   Current state vs. proposed changes                    │
│   with delta calculations                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** This is the "secret sauce"--not just showing data, but doing the math automatically.

---

## Slide 7: Get Started

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                   TRY IT YOURSELF                       │
│                                                         │
│                                                         │
│   QUICK START (no credentials needed)                   │
│   ────────────────────────────────────                  │
│                                                         │
│   $ git clone [repo-url]                                │
│   $ make frontend-dev                                   │
│   → Load any sample file from the UI                    │
│                                                         │
│                                                         │
│   CONNECT TO REAL INFRASTRUCTURE                        │
│   ──────────────────────────────────                    │
│                                                         │
│   $ ./generate-env.sh    # pulls creds from Ops Manager │
│   $ make backend-run                                    │
│                                                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** Emphasize low barrier to try--sample data mode means zero setup friction.

---

## Slide 8: Feedback Welcome

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                  FEEDBACK WELCOME                       │
│                                                         │
│                                                         │
│   QUESTIONS TO CONSIDER                                 │
│   ─────────────────────                                 │
│                                                         │
│   • What capacity questions do you wrestle with?        │
│                                                         │
│   • What's missing that would make this useful?         │
│                                                         │
│   • How would this fit into your workflow?              │
│                                                         │
│                                                         │
│                                                         │
│                    [Your Contact Info]                  │
│                    [Repo / Documentation Link]          │
│                                                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Speaker notes:** End with questions--invite dialogue. This builds adoption momentum.

---

## Asset Checklist

Include these visuals from your repo:

| Slide | Asset                       | Path                                         |
| ----- | --------------------------- | -------------------------------------------- |
| 3     | Dashboard screenshot or GIF | `docs/images/dashboard.gif`                  |
| 3     | (alt) Full demo walkthrough | `docs/images/tas-capacity-analyzer-demo.gif` |
| -     | What-If mode interaction    | `docs/images/tas-what-if-mode.gif`           |
| -     | Scenario results            | `docs/images/tas-scenario-results.gif`       |

---

## Presentation Tips

1. **Test screen share** before the demo--make sure resolution works
2. **Use dark mode** in the browser--looks sharper, colorblind-friendly
3. **Pre-load sample data** so you don't fumble during the demo
4. **Have Swagger tab ready** but minimized
5. **Keep terminal open** with CLI built
6. **Practice transitions** between slides and live demo
