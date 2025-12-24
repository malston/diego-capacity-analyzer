# UX Improvements Milestone

**Created:** 2025-12-23
**Status:** Planning
**Branch:** `feature/ux-improvements-v2`

## Overview

Incremental UX improvements to make the Diego Capacity Analyzer more intuitive and polished. Ordered by impact/effort ratio.

---

## Priority 1: Hide Developer Tools from Production UI

**Impact:** High | **Effort:** Low | **Risk:** None

### Problem
The Mock Data / Live API toggle and "Test Connection" button are developer debugging tools exposed in the header settings. This makes the app feel unfinished to end users.

### Solution
- Move data source controls behind a feature flag (`?dev=true` query param or `VITE_DEV_MODE` env var)
- In production, auto-connect to backend API on load
- Show loading/error states gracefully without exposing the toggle

### Files Affected
- `frontend/src/components/SettingsPanel.jsx`
- `frontend/src/components/Header.jsx`
- `frontend/src/TASCapacityAnalyzer.jsx`

### Acceptance Criteria
- [ ] Dev tools hidden by default in production builds
- [ ] `?dev=true` enables data source controls for debugging
- [ ] Clean loading state when connecting to backend

---

## Priority 2: Unify Accent Colors

**Impact:** Medium | **Effort:** Low | **Risk:** None

### Problem
Inconsistent focus/accent colors across components:
- Scenario Analyzer: `focus:border-cyan-500`, cyan gradients
- Infrastructure Planning: `focus:border-emerald-500`, emerald gradients
- Dashboard: blue accents

### Solution
Establish a consistent color system:
- **Primary action (CTA buttons):** cyan-to-blue gradient
- **Focus states:** `cyan-500`
- **Success indicators:** emerald
- **Warnings:** amber
- **Errors:** red

### Files Affected
- `frontend/src/components/InfrastructurePlanning.jsx`
- `frontend/src/components/ScenarioAnalyzer.jsx`
- `frontend/src/TASCapacityAnalyzer.css`

### Acceptance Criteria
- [ ] All focus states use `cyan-500`
- [ ] Primary CTAs use consistent cyan-to-blue gradient
- [ ] Color usage follows documented system

---

## Priority 3: Clarify Tab Navigation

**Impact:** High | **Effort:** Medium | **Risk:** Low

### Problem
- "Infrastructure Planning" and "Scenario Analysis" overlap significantly
- Both have `DataSourceSelector`, both calculate cell capacity
- "What-If Mode" toggle on Dashboard vs "What-If Scenario Analysis" tab is confusing

### Solution Options

**Option A: Rename for clarity (minimal change)**
- Dashboard → "Current State"
- Infrastructure Planning → "IaaS Capacity"
- Scenario Analysis → "What-If Planning"

**Option B: Consolidate (recommended)**
- Merge Infrastructure Planning into Scenario Analysis as "Step 1: Load Infrastructure"
- Remove duplicate DataSourceSelector
- Single flow: Load Data → See Current → Run What-If

**Option C: Guided workflow**
- Add breadcrumb/stepper showing: Infrastructure → Scenarios → Results
- Make tab progression feel intentional

### Files Affected
- `frontend/src/components/Header.jsx` (tab definitions)
- `frontend/src/components/ScenarioAnalyzer.jsx`
- `frontend/src/components/InfrastructurePlanning.jsx`
- `frontend/src/TASCapacityAnalyzer.jsx`

### Acceptance Criteria
- [ ] User understands which tab to use without guessing
- [ ] No redundant data loading across tabs
- [ ] Clear workflow from infrastructure → scenarios

---

## Priority 4: Simplify Scenario Analyzer Form

**Impact:** High | **Effort:** High | **Risk:** Medium

### Problem
ScenarioAnalyzer.jsx is 750+ lines with many inputs competing for attention:
- Resource Type toggles
- VM Size + custom inputs
- Cell Count
- Advanced Options (overhead, hypothetical app, TPS curve)

### Solution
Implement a stepper/wizard pattern:

```bash
Step 1: Cell Configuration
  - VM Size preset or custom
  - Cell count

Step 2: Resource Types (optional)
  - Which resources to analyze
  - Disk settings if selected

Step 3: Advanced (optional, collapsed)
  - Memory overhead
  - Hypothetical app
  - TPS curve

→ Run Analysis
```

### Files Affected
- `frontend/src/components/ScenarioAnalyzer.jsx` (major refactor)
- New: `frontend/src/components/ScenarioStepper.jsx`
- New: `frontend/src/components/steps/CellConfigStep.jsx`
- New: `frontend/src/components/steps/ResourceTypeStep.jsx`
- New: `frontend/src/components/steps/AdvancedStep.jsx`

### Acceptance Criteria
- [ ] Form broken into clear steps
- [ ] User can skip optional steps
- [ ] Current functionality preserved
- [ ] Reduced visual clutter on initial view

---

## Priority 5: Add Success Feedback

**Impact:** Medium | **Effort:** Low | **Risk:** None

### Problem
No feedback after "Run Analysis" completes successfully. User has to notice the results appeared.

### Solution
Add a lightweight toast notification system:
- Success: "Analysis complete"
- Error: "Analysis failed: {reason}"
- Auto-dismiss after 3 seconds

### Files Affected
- New: `frontend/src/components/Toast.jsx`
- New: `frontend/src/contexts/ToastContext.jsx`
- `frontend/src/TASCapacityAnalyzer.jsx`
- `frontend/src/components/ScenarioAnalyzer.jsx`

### Acceptance Criteria
- [ ] Toast appears on analysis completion
- [ ] Toast auto-dismisses
- [ ] Accessible (ARIA live region)

---

## Implementation Order

| Phase | Items | Est. Effort |
|-------|-------|-------------|
| Phase 1 | P1 + P2 | ~2 hours |
| Phase 2 | P3 | ~3-4 hours |
| Phase 3 | P4 | ~6-8 hours |
| Phase 4 | P5 | ~1-2 hours |

Each phase should be a separate PR for easier review.

---

## Open Questions

1. **P3 Tab Strategy:** Option A (rename), B (consolidate), or C (guided)? Need Mark's input.
2. **P4 Stepper:** Should steps be tabs, accordion, or linear wizard?
3. **P5 Toast:** Use existing library (react-hot-toast) or build minimal custom?
