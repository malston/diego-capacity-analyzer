# Metric Grouping by Scope (Issue #55)

## Problem

The scenario results page displays infrastructure headroom metrics and current utilization metrics in a flat layout without visual distinction. Users see "Max Deployable Cells: 125" alongside "Free Chunks: 2 (Constrained)" and don't understand how they can have headroom for 125 cells while being constrained on staging.

## Solution

Group metrics into two visually distinct containers with section headers and subtle background differences.

## Layout Structure

Page order (top to bottom):

1. **Overall Status Banner** -- unchanged, above both groups
2. **Constraint Callout** -- unchanged, above both groups
3. **Infrastructure Headroom** group container
4. **Current Utilization** group container
5. **Warnings / All Clear** -- unchanged, below both groups

## Metric Assignment

### Infrastructure Headroom

Answers: "What can my hardware support?"

- N-1 / Constraint Utilization gauge
- vCPU:pCPU Ratio gauge (when CPU selected)
- Maximum Deployable Cells section (memory + CPU constraints, bottleneck indicators)

### Current Utilization

Answers: "How is my deployment performing right now?"

- Memory Utilization gauge (when memory selected)
- Disk Utilization gauge (when disk selected)
- Staging Capacity (free chunks)
- TPS Performance indicator (when enabled)
- Detailed Metrics scorecards (Cell Count, App Capacity, Fault Impact, Instances/Cell)
- Cell Configuration Change comparison

## Visual Treatment

### Group Containers

Each group is wrapped in a container div:

- Infrastructure: `rounded-2xl border border-slate-600/40 bg-slate-800/20 p-5`
- Utilization: `rounded-2xl border border-slate-600/40 bg-slate-800/40 p-5`
- Gap between groups: `space-y-6`

### Section Headers

Each container has a header at the top:

- Small caps text: `text-xs uppercase tracking-wider font-medium text-gray-400`
- Lucide icon to the left: `Server` for Infrastructure, `Activity` for Utilization
- Separated from content by thin border: `border-b border-slate-700/50 pb-3 mb-5`

### Internal Layout

Child elements within each group keep their existing spacing and grid layouts. No changes to individual metric cards, gauges, or scorecards.

## Files Modified

- `frontend/src/components/ScenarioResults.jsx` -- wrap existing metric elements in group containers, add section headers, reorder metrics into groups

## Acceptance Criteria

- User can quickly distinguish infrastructure capacity metrics from current utilization metrics
- The distinction is visually apparent without requiring documentation
- No functional changes to metric calculations or data flow
