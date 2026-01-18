# TUI Visual Redesign

**Date:** 2026-01-16
**Status:** Approved
**Focus:** Data visualization and icons/indicators for TUI screens

## Overview

Enhance the CLI TUI with a vibrant, modern visual style featuring compact dashboard blocks, sparklines, and graceful icon degradation (Nerd Fonts with Unicode fallback).

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary focus | TUI screens (dashboard, wizard, comparison) | Where users spend most time |
| Visual emphasis | Data visualization + Icons | Capacity analyzer benefits from rich metric display |
| Icon strategy | Graceful degradation | Nerd Fonts when available, Unicode fallback |
| Visualization style | Compact dashboard blocks with sparklines | High information density |
| Color theme | Vibrant/Modern | Bold saturated colors, high contrast |

## Icon System

Create `cli/internal/tui/icons/icons.go` with Nerd Font detection:

```go
var (
    Memory    = icon("󰍛", "◆")   // Nerd: memory chip, Fallback: diamond
    CPU       = icon("", "●")   // Nerd: processor, Fallback: circle
    Disk      = icon("󰋊", "■")   // Nerd: hard disk, Fallback: square
    Server    = icon("󰒋", "▣")   // Nerd: server, Fallback: box
    Cluster   = icon("󱃾", "⬡")   // Nerd: cluster, Fallback: hexagon
    CheckOK   = icon("", "✓")   // Nerd: checkmark, Fallback: unicode check
    Warning   = icon("", "⚠")   // Nerd: warning, Fallback: unicode warning
    Critical  = icon("", "✗")   // Nerd: x-circle, Fallback: unicode x
    Trend     = icon("󰄬", "↗")   // Nerd: trending up, Fallback: arrow
    Chart     = icon("󰄭", "▁▂▃") // Nerd: chart, Fallback: mini bars
)
```

**Detection:** Check `$TERM` for known Nerd Font terminals, or `DIEGO_NERD_FONTS=1` env var.

## Visualization Components

### Progress Bars with Threshold Zones

```
Memory   [████████████░░░░░░│░░] 67% ✓
         ←── green ──→←amber→←red→
```

### Sparklines

8-character trend visualization using block characters:
```
▁▂▃▅▆▅▄▃
```

### Compact Metric Blocks

```
┌─ 󰍛 Memory ──────────────┐  ┌─  CPU Ratio ───────────┐
│  67%  ▂▃▅▆▅▄▃▂          │  │  3.2:1  ▅▆▇█▇▆▅▄  ⚠   │
│  [████████░░░░] 128/192 │  │  Moderate Risk         │
└─────────────────────────┘  └────────────────────────┘
```

### Status Badges

Colored inline badges: ` OK ` (green), ` WARN ` (amber), ` CRIT ` (red)

## Dashboard Layout

**Header Bar:**
```
╭─ 󰋊 Diego Capacity Analyzer ──────────────────── Lab-vSphere ─╮
```

**Row 1: Key Metrics (4 compact blocks)**
```
┌─ 󰍛 Memory ─────────┐ ┌─  CPU ───────────┐ ┌─ 󱃾 Clusters ────┐ ┌─ 󰒋 Hosts ───────┐
│  67%  ▂▃▅▆▅▄▃▂     │ │  3.2:1  ▅▆▇█▇▆  │ │       2         │ │       8         │
│  [██████░░░] ✓     │ │  Moderate   ⚠   │ │   clusters      │ │   hosts         │
└────────────────────┘ └──────────────────┘ └─────────────────┘ └─────────────────┘
```

**Row 2: Capacity & HA Status (2 larger panels)**
```
┌─  N-1 Capacity ────────────────────────┐ ┌─ 󰒋 HA Status ─────────────────────┐
│   Utilization: 67%                      │ │   ✓ Can survive 1 host failure    │
│   [████████████████░░░░░░░░│░░░░] ✓     │ │   Current: 8 hosts, need min 7    │
│   Headroom: 33% (63 GB available)       │ │   Cells per host: 4 avg           │
└─────────────────────────────────────────┘ └────────────────────────────────────┘
```

**Footer:**
```
╰─ [r]efresh  [w]izard  [b]ack  [q]uit ────────────────── Updated 2s ago ─╯
```

## Comparison Screen

Side-by-side with visual deltas:

```
┌─  Current ──────────────────────────┐  ┌─  Proposed ─────────────────────────┐
│  Cells:     32                       │  │  Cells:     48         (+16)        │
│  Memory:    64 GB each               │  │  Memory:    64 GB each              │
│  Total:     2,048 GB                 │  │  Total:     3,072 GB   (+1,024)     │
│  ┌─ Utilization ──────────────────┐  │  │  ┌─ Utilization ──────────────────┐  │
│  │  67%  [████████░░░░] ✓         │  │  │  │  45%  [█████░░░░░░] ✓         │  │
│  └────────────────────────────────┘  │  │  └────────────────────────────────┘  │
└──────────────────────────────────────┘  └──────────────────────────────────────┘

┌─ 󰄬 Impact Summary ───────────────────────────────────────────────────────────┐
│   Capacity:      +1,024 GB (+50%)                                            │
│   Utilization:   67% → 45%  (-22%)  ↓                                        │
│   Headroom:      +704 GB available                                           │
└───────────────────────────────────────────────────────────────────────────────┘

┌─  Warnings ──────────────────────────────────────────────────────────────────┐
│   ⚠  Adding 16 cells requires 2 additional hosts at current density          │
│   ✓  N-1 capacity maintained with proposed configuration                      │
└───────────────────────────────────────────────────────────────────────────────┘
```

## Wizard Enhancement

Progress indicator with step icons:

```
┌─ Progress ───────────────────────────────────────────────────────────┐
│   ● Cell Sizing    ○ Cell Count    ○ Overhead & HA                   │
│   ━━━━━━━━━━━━━━━━━╸━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │
└──────────────────────────────────────────────────────────────────────┘
```

Keep huh forms but wrap in styled container with consistent header/footer framing.

## Implementation Structure

**New Files:**
```
cli/internal/tui/
├── icons/
│   └── icons.go           # Icon system with Nerd Font detection
├── widgets/
│   ├── sparkline.go       # Sparkline renderer
│   ├── progressbar.go     # Enhanced progress bar with zones
│   ├── metricblock.go     # Compact metric block component
│   └── badge.go           # Status badge component
```

**Modified Files:**
```
cli/internal/tui/
├── styles/styles.go       # Add new colors, box styles
├── dashboard/dashboard.go # Use new widgets, 2-row layout
├── comparison/comparison.go # Use new widgets, delta styling
├── wizard/wizard.go       # Add progress indicator, framing
└── app.go                 # Add header/footer frame
```

**Color Palette Additions:**
```go
DeltaPositive = lipgloss.Color("#10B981")  // Green - improvements
DeltaNegative = lipgloss.Color("#F59E0B")  // Amber - costs/increases
DeltaNeutral  = lipgloss.Color("#6B7280")  // Gray - no change
Accent        = lipgloss.Color("#8B5CF6")  // Lighter purple - highlights
Surface       = lipgloss.Color("#374151")  // Elevated surface bg
```

## Dependencies

No new external dependencies - uses existing Lipgloss for all rendering.
