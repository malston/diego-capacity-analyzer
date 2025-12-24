# Scenario Analyzer Stepper Design

**Created:** 2025-12-23
**Status:** Approved
**Related:** P4 from UX Improvements Milestone

## Overview

Refactor ScenarioAnalyzer from a single dense form into a linear wizard with clickable step navigation. Reduces cognitive load by presenting inputs in logical groups.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pattern | Linear Wizard | Natural sequence; hides complexity until needed |
| Info displays | Always visible | Context helps users make decisions at each step |
| Navigation | Clickable step indicator | Users can jump to any completed step |
| Run Analysis | Available after Step 1 | Optional steps don't block core functionality |

## Component Architecture

```text
ScenarioAnalyzer.jsx (orchestrator, ~200 lines)
├── Info displays (always visible)
│   ├── Current Configuration card
│   └── IaaS Capacity card
├── ScenarioWizard.jsx (wizard container, ~150 lines)
│   ├── StepIndicator.jsx (clickable progress bar, ~50 lines)
│   └── Step content (one visible at a time)
│       ├── CellConfigStep.jsx (~100 lines)
│       ├── ResourceTypesStep.jsx (~80 lines)
│       └── AdvancedStep.jsx (~150 lines)
├── Run Analysis button (visible after Step 1)
└── ScenarioResults (existing, unchanged)
```

### State Management

- Wizard state (currentStep, completedSteps) lives in ScenarioWizard
- Form values stay in ScenarioAnalyzer (lifted state) so Run Analysis can access them
- Each step receives values + setters as props

### Files

| File | Purpose |
|------|---------|
| `ScenarioAnalyzer.jsx` | Refactored: keeps state, renders info + wizard + results |
| `ScenarioWizard.jsx` | New: wizard container with step navigation |
| `StepIndicator.jsx` | New: clickable step progress bar |
| `steps/CellConfigStep.jsx` | New: VM size, cell count |
| `steps/ResourceTypesStep.jsx` | New: resource type toggles, disk input |
| `steps/AdvancedStep.jsx` | New: overhead, hypothetical app, TPS curve |

## Step Indicator

### Visual

```text
● Cell Config  ───────  ○ Resources  ───────  ○ Advanced
   (current)              (optional)          (optional)
```

### States

| State | Visual | Behavior |
|-------|--------|----------|
| Completed | Filled circle with checkmark, cyan | Clickable - jumps to step |
| Current | Filled circle, pulsing border | Active step content shown |
| Available | Empty circle, gray | Clickable if previous step done |
| Locked | Empty circle, dimmed | Not clickable until prerequisites met |

### Step Metadata

```javascript
const STEPS = [
  { id: 'cell-config', label: 'Cell Config', required: true },
  { id: 'resources', label: 'Resources', required: false },
  { id: 'advanced', label: 'Advanced', required: false },
];
```

### Navigation

- Clicking a completed/available step jumps directly to it
- Each step has "Continue" button (primary) → advances to next
- Optional steps also show "Skip" (secondary) → advances without filling
- No "Back" button needed since step indicator is clickable

### Step 1 Completion Requirement

- VM Size selected (always has default)
- Cell Count > 0

## Step Content

### Step 1: Cell Configuration (Required)

```bash
┌─────────────────────────────────────────────────────────┐
│ VM Size                                                 │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 4 vCPU × 32 GB (Standard)                        ▼ │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ [Custom vCPU input]  [Custom Memory input]  ← if Custom │
│                                                         │
│ Cell Count                                              │
│ ┌──────────┐                                           │
│ │ 100      │  ⚡ For equivalent capacity: use 200 cells │
│ └──────────┘     (suggestion link, if applicable)      │
│                                                         │
│                                    [ Continue → ]       │
└─────────────────────────────────────────────────────────┘
```

### Step 2: Resource Types (Optional)

```bash
┌─────────────────────────────────────────────────────────┐
│ Which resources to analyze?                             │
│                                                         │
│ [● Memory]  [● CPU]  [○ Disk]  ← toggle buttons         │
│                                                         │
│ Disk per Cell (GB)         ← only shown if Disk checked │
│ ┌──────────┐                                           │
│ │ 128      │                                           │
│ └──────────┘                                           │
│                                                         │
│                         [ Skip ]    [ Continue → ]      │
└─────────────────────────────────────────────────────────┘
```

### Step 3: Advanced (Optional)

```bash
┌─────────────────────────────────────────────────────────┐
│ Memory Overhead: 7%                                     │
│ ────────●──────────────────────────────  [1%]    [20%] │
│                                                         │
│ ┌─ Hypothetical App ─────────────────────────────────┐ │
│ │ ☐ Include in analysis                              │ │
│ │ App Name: [hypothetical-app]                       │ │
│ │ Instances: [1]  Memory: [1GB]  Disk: [1GB]        │ │
│ └────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─ TPS Performance Curve ────────────────────────────┐ │
│ │ [50 cells → 500 TPS]  [×]                         │ │
│ │ [100 cells → 450 TPS] [×]                         │ │
│ │ [+ Add Point]  [Reset to Default]                  │ │
│ └────────────────────────────────────────────────────┘ │
│                                                         │
│                         [ Skip ]    [ Continue → ]      │
└─────────────────────────────────────────────────────────┘
```

## Overall Layout

```text
┌─────────────────────────────────────────────────────────────────┐
│ ✨ Capacity Planning                                             │
├─────────────────────────────────────────────────────────────────┤
│ [DataSourceSelector - Load from file, vSphere, presets...]     │
├─────────────────────────────────────────────────────────────────┤
│ ┌─ Current Configuration ─────┐  ┌─ IaaS Capacity ───────────┐ │
│ │ 100 Cells  |  8×64  |  6.4T │  │ 10 Hosts | 2.5T | Max:150 │ │
│ └─────────────────────────────┘  └────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ ● Cell Config ─────── ○ Resources ─────── ○ Advanced            │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │                                                             │ │
│ │              [Current Step Content]                         │ │
│ │                                                             │ │
│ │                           [ Skip ]    [ Continue → ]        │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ ✨ Run Analysis                               [Export ↓]    │ │
│ │    Cell Config: 4×32, 200 cells                             │ │
│ │    Resources: Memory, CPU                                   │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ [ScenarioResults - appears after analysis runs]                 │
└─────────────────────────────────────────────────────────────────┘
```

### Run Analysis Section Behavior

- Hidden until infrastructure loaded AND Step 1 completed
- Shows summary of current selections (helps user verify before running)
- Summary updates as user changes settings in any step
- Button disabled while loading
- Export button appears after results exist

### Summary Display Format

```javascript
// Quick glance at current config without leaving the step
"4×32 GB cells, 200 count | Memory, CPU | 7% overhead"
```

## Future Considerations

Per Issue #10 (Multi-Resource Analysis), additional steps may be added:

```text
Step 1: Cell Configuration (Required)
Step 2: Resource Types (Optional)
Step 3: Host Configuration (Future - cores, HA policy)
Step 4: Advanced (Optional)
```

The wizard design accommodates this by:
- Making all steps after Step 1 optional
- Using clickable navigation so users can skip around
- Keeping Run Analysis available early
