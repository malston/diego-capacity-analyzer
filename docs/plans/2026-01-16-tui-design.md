# TUI Design: Charm Bracelet Terminal Interface

**Date:** 2026-01-16
**Status:** Draft
**Author:** Mark & Claude

## Overview

Replace the current `diego-capacity` CLI with a rich terminal user interface (TUI) built on Charm Bracelet libraries. The TUI provides interactive scenario planning with a wizard-style flow and live capacity metrics in a split-pane layout.

## Motivation

- **SSH-only access**: Operators work on jump boxes and bastion hosts where browsers aren't available
- **Unified Go stack**: Faster iteration than React; single language for backend, CLI, and TUI
- **Terminal workflow integration**: Capacity data alongside `cf` and `bosh` commands

## Key Decisions

| Decision | Choice |
|----------|--------|
| Priority feature | Interactive scenario planning with wizard |
| Interaction model | Step-by-step wizard + split-pane live dashboard |
| Data sources | Both vSphere and manual/JSON, user selects at startup |
| CLI relationship | TUI replaces current CLI as single `diego-capacity` binary |
| Scripting support | TUI-first; `--json` flag for non-interactive output |
| Charm stack | Hybrid: `huh` for wizard forms, `bubbletea` for dashboard |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      diego-capacity                          │
│                    (single binary)                           │
├─────────────────────────────────────────────────────────────┤
│  Entry Point (cli/main.go)                                  │
│    ├── TTY detected? → Launch TUI                           │
│    └── --json flag?  → Non-interactive output               │
├─────────────────────────────────────────────────────────────┤
│  TUI Layer (cli/internal/tui/)                              │
│    ├── app.go         → Root bubbletea model                │
│    ├── wizard/        → huh-based scenario wizard           │
│    ├── dashboard/     → Split-pane live view                │
│    └── styles/        → Shared lipgloss styles              │
├─────────────────────────────────────────────────────────────┤
│  API Client (cli/internal/client/)  ← already exists        │
│    └── Calls backend endpoints, returns typed structs       │
├─────────────────────────────────────────────────────────────┤
│  Backend (backend/)  ← unchanged                            │
│    └── 12 REST endpoints for capacity data                  │
└─────────────────────────────────────────────────────────────┘
```

The TUI is a presentation layer only. All business logic stays in the backend.

## User Flow

### Startup Menu

```
┌──────────────────────────────────────────────────────┐
│  Diego Capacity Analyzer                             │
│                                                      │
│  Select data source:                                 │
│                                                      │
│  > ● Live vSphere      (vcenter.example.com)        │
│    ○ Load JSON file                                  │
│    ○ Manual input                                    │
│                                                      │
│  [Enter] Select   [q] Quit                          │
└──────────────────────────────────────────────────────┘
```

### Main View (Split-Pane)

```
┌─────────────────────────────────┬────────────────────────────┐
│  Current Infrastructure         │  Scenario Wizard           │
│  ─────────────────────────      │  ────────────────          │
│  Clusters: 2                    │  Step 2 of 5: Cell Sizing  │
│  Hosts: 8                       │                            │
│  Diego Cells: 24                │  Memory per cell (GB):     │
│                                 │  [64]                      │
│  Memory Utilization             │                            │
│  ████████████░░░░ 78%           │  CPU cores per cell:       │
│                                 │  [8]                       │
│  N-1 Status: ✓ OK               │                            │
│  HA Status:  ✓ OK               │  Disk per cell (GB):       │
│                                 │  [200]                     │
│  Bottleneck: Memory             │                            │
│                                 │  [←Back] [Next→] [q]Quit   │
└─────────────────────────────────┴────────────────────────────┘
```

The left pane shows live metrics (updates on each wizard step). The right pane walks through the scenario wizard. When the wizard completes, the left pane shows a comparison view with current vs. proposed deltas.

## Component Breakdown

| Component | Library | Responsibility |
|-----------|---------|----------------|
| **App** | `bubbletea` | Root model, keyboard routing, manages child components |
| **DataSourceMenu** | `huh` | Startup menu with radio select for vSphere/JSON/manual |
| **Wizard** | `huh` | Multi-step form for scenario configuration |
| **Dashboard** | `bubbletea` | Left pane with live metrics, progress bars, status indicators |
| **ComparisonView** | `bubbletea` | Post-wizard view showing current vs. proposed with deltas |
| **Styles** | `lipgloss` | Shared color palette, borders, spacing |

### Wizard Steps

1. **Data Source** — selected at startup menu
2. **Cluster Selection** — if multiple clusters, pick target (or "all")
3. **Cell Sizing** — memory GB, CPU cores, disk GB per cell
4. **Proposed Cell Count** — how many cells in the scenario
5. **Overhead & HA** — overhead %, HA admission %, resource toggles
6. **Review & Compare** — show delta, warnings, recommendations

### API Calls Per Step

- After step 1: `GET /api/infrastructure` or `POST /api/infrastructure/manual`
- After step 5: `POST /api/scenario/compare`
- Left pane refresh: `GET /api/infrastructure/status` (polled or on-demand)

## File Structure

```
cli/
├── main.go                      # Entry point (add TTY detection)
├── cmd/
│   ├── root.go                  # Cobra root command (exists)
│   ├── health.go                # Existing - add --json flag
│   ├── status.go                # Existing - add --json flag
│   ├── check.go                 # Existing - add --json flag
│   └── scenario.go              # NEW: non-interactive scenario compare
│
├── internal/
│   ├── client/
│   │   └── client.go            # API client (extend for all endpoints)
│   │
│   └── tui/                     # NEW: all TUI code
│       ├── app.go               # Root bubbletea model, keyboard routing
│       ├── styles/
│       │   └── styles.go        # Lipgloss styles, colors, borders
│       ├── components/
│       │   ├── menu.go          # Data source selection menu
│       │   ├── dashboard.go     # Left pane: live metrics display
│       │   ├── comparison.go    # Post-wizard comparison view
│       │   └── progress.go      # Reusable progress bar component
│       └── wizard/
│           ├── wizard.go        # Huh form orchestration
│           ├── cluster.go       # Step: cluster selection
│           ├── cellsize.go      # Step: cell sizing inputs
│           ├── overhead.go      # Step: overhead & HA settings
│           └── review.go        # Step: review before API call
│
└── go.mod                       # Add bubbletea, huh, lipgloss deps
```

## API Client Expansion

Current coverage (3 endpoints):
- `GET /api/health`
- `GET /api/infrastructure/status`
- Capacity check logic

Required additions:
- `GET /api/infrastructure`
- `POST /api/infrastructure/manual`
- `POST /api/infrastructure/state`
- `GET /api/dashboard`
- `GET /api/infrastructure/apps`
- `POST /api/infrastructure/planning`
- `POST /api/scenario/compare`
- `GET /api/bottleneck`
- `GET /api/recommendations`

## Non-Interactive Mode

Existing commands work non-interactively with `--json` flag:

```bash
# Interactive (launches TUI)
diego-capacity

# Non-interactive (JSON output for scripts)
diego-capacity status --json
diego-capacity check --memory-threshold 85 --json

# Piping triggers non-interactive
diego-capacity status | jq '.n1_status'
```

New scenario command for CI:

```bash
diego-capacity scenario \
  --cell-memory 64 \
  --cell-cpu 8 \
  --cell-count 20 \
  --json
```

## Testing Strategy

| Layer | Approach |
|-------|----------|
| API Client | Unit tests with mock HTTP server (extend existing) |
| TUI Components | `teatest` package — send key sequences, assert rendered output |
| Wizard Forms | Test `huh` form models directly — set values, validate |
| Integration | End-to-end tests against running backend |

## Dependencies

```go
require (
    github.com/charmbracelet/bubbletea v1.x
    github.com/charmbracelet/huh v0.x
    github.com/charmbracelet/lipgloss v1.x
    github.com/charmbracelet/bubbles v0.x  // for progress bars, spinners
)
```

## Open Questions

1. **Polling interval** — How often should the dashboard refresh? (Suggest: on wizard step change, not continuous polling)
2. **Color theme** — Should we support light/dark terminal detection, or pick one theme?
3. **Window resizing** — Full responsive layout or minimum terminal size requirement?

## Next Steps

1. Create implementation plan with task breakdown
2. Set up git worktree for isolated development
3. Extend API client to cover all endpoints
4. Build TUI components incrementally (menu → dashboard → wizard)
5. Add non-interactive `scenario` command
6. Integration testing against live backend
