# Metric Grouping by Scope Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Group scenario results metrics into "Infrastructure Headroom" and "Current Utilization" sections with visual containers and section headers (Issue #55).

**Architecture:** Wrap existing metric elements in `ScenarioResults.jsx` into two container divs with section headers. No changes to metric components, data flow, or calculations -- purely structural reorganization of the JSX.

**Tech Stack:** React, Tailwind CSS, Lucide React icons, Vitest + Testing Library

---

### Task 1: Write failing tests for section group rendering

**Files:**

- Modify: `frontend/src/components/ScenarioResults.test.jsx`

**Step 1: Add tests for section headers and grouping**

Add a new describe block at the end of the test file:

```jsx
describe("ScenarioResults Metric Grouping", () => {
  const groupingComparison = {
    current: {
      cell_count: 10,
      cell_memory_gb: 32,
      cell_cpu: 4,
      app_capacity_gb: 298,
      utilization_pct: 50,
      n1_utilization_pct: 60,
      free_chunks: 100,
      blast_radius_pct: 5,
      instances_per_cell: 5,
      fault_impact: 10,
      estimated_tps: 0,
      tps_status: "disabled",
    },
    proposed: {
      cell_count: 20,
      cell_memory_gb: 32,
      cell_cpu: 4,
      cell_disk_gb: 100,
      app_capacity_gb: 596,
      utilization_pct: 25,
      disk_utilization_pct: 40,
      disk_capacity_gb: 2000,
      n1_utilization_pct: 70,
      free_chunks: 200,
      blast_radius_pct: 2.5,
      instances_per_cell: 2.5,
      fault_impact: 5,
      estimated_tps: 0,
      tps_status: "disabled",
      total_vcpus: 80,
      total_pcpus: 96,
      vcpu_ratio: 0.83,
      cpu_risk_level: "conservative",
      max_cells_by_cpu: 90,
      cpu_headroom_cells: 70,
    },
    delta: {
      capacity_change_gb: 298,
      utilization_change_pct: -25,
      resilience_change: "improved",
    },
    constraints: {
      ha_admission: {
        type: "ha_admission",
        usable_gb: 1280,
        utilization_pct: 46.5,
        is_limiting: true,
        reserved_gb: 640,
      },
      n_minus_x: {
        type: "n_minus_x",
        usable_gb: 1536,
        utilization_pct: 38.8,
        is_limiting: false,
      },
      limiting_constraint: "ha_admission",
      limiting_label: "HA 25%",
    },
  };

  it("renders Infrastructure Headroom section header", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    expect(screen.getByText("Infrastructure Headroom")).toBeInTheDocument();
  });

  it("renders Current Utilization section header", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    expect(screen.getByText("Current Utilization")).toBeInTheDocument();
  });

  it("places N-1 Capacity gauge inside Infrastructure Headroom section", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const infraSection = screen.getByTestId("section-infrastructure-headroom");
    const n1Label = screen.getByText((content, element) => {
      return (
        element.textContent.includes("Capacity") &&
        element.textContent.includes("HA 25%")
      );
    });
    expect(infraSection).toContainElement(n1Label);
  });

  it("places Memory Utilization gauge inside Current Utilization section", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const utilizationSection = screen.getByTestId(
      "section-current-utilization",
    );
    const memLabel = screen.getByText("Memory Utilization");
    expect(utilizationSection).toContainElement(memLabel);
  });

  it("places Maximum Deployable Cells inside Infrastructure Headroom section", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const infraSection = screen.getByTestId("section-infrastructure-headroom");
    const maxCellsLabel = screen.getByText("Maximum Deployable Cells");
    expect(infraSection).toContainElement(maxCellsLabel);
  });

  it("places Staging Capacity inside Current Utilization section", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const utilizationSection = screen.getByTestId(
      "section-current-utilization",
    );
    const stagingLabel = screen.getByText("Staging Capacity");
    expect(utilizationSection).toContainElement(stagingLabel);
  });

  it("places Cell Configuration Change inside Current Utilization section", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const utilizationSection = screen.getByTestId(
      "section-current-utilization",
    );
    const cellConfigLabel = screen.getByText("Cell Configuration Change");
    expect(utilizationSection).toContainElement(cellConfigLabel);
  });

  it("renders Infrastructure Headroom before Current Utilization in DOM order", () => {
    render(
      <ScenarioResults
        comparison={groupingComparison}
        warnings={[]}
        selectedResources={["memory", "cpu"]}
      />,
    );

    const infraSection = screen.getByTestId("section-infrastructure-headroom");
    const utilizationSection = screen.getByTestId(
      "section-current-utilization",
    );

    // compareDocumentPosition bit 4 = DOCUMENT_POSITION_FOLLOWING
    const position = infraSection.compareDocumentPosition(utilizationSection);
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `make frontend-test`
Expected: New "Metric Grouping" tests fail (section headers / test IDs don't exist yet). Existing tests still pass.

**Step 3: Commit failing tests**

```bash
git add frontend/src/components/ScenarioResults.test.jsx
git commit -m "test: add failing tests for metric grouping sections (#55)"
```

---

### Task 2: Implement metric grouping in ScenarioResults

**Files:**

- Modify: `frontend/src/components/ScenarioResults.jsx`

**Step 1: Add section header helper component**

Add a `SectionHeader` component inside the file, after the `TOOLTIPS` constant and before `ScenarioResults`:

```jsx
const SectionHeader = ({ icon: Icon, label }) => (
  <div className="flex items-center gap-2 border-b border-slate-700/50 pb-3 mb-5">
    <Icon size={16} className="text-gray-400" />
    <span className="text-xs uppercase tracking-wider font-medium text-gray-400">
      {label}
    </span>
  </div>
);
```

**Step 2: Wrap Infrastructure Headroom metrics**

In the JSX return, after the Constraint Callout section (line ~200) and before the current Key Gauges Row, add an Infrastructure Headroom container wrapping:

- The N-1 / Constraint Utilization gauge (standalone, extracted from the gauges grid)
- The vCPU:pCPU Ratio gauge (standalone, extracted from the gauges grid)
- The Maximum Deployable Cells section (moved up from its current position at line ~634)

The infrastructure gauges use the same card styling but sit inside a 2-column grid within the container.

Container:

```jsx
{/* Infrastructure Headroom */}
<div
  data-testid="section-infrastructure-headroom"
  className="rounded-2xl border border-slate-600/40 bg-slate-800/20 p-5"
>
  <SectionHeader icon={Server} label="Infrastructure Headroom" />
  <div className="space-y-6">
    {/* Infrastructure gauges grid */}
    <div className={`grid gap-6 ${/* dynamic cols based on which infra gauges show */}`}>
      {/* N-1 Capacity gauge (existing JSX, unchanged) */}
      {/* vCPU:pCPU Ratio gauge (existing JSX, unchanged) */}
    </div>
    {/* Maximum Deployable Cells section (existing JSX, moved here) */}
  </div>
</div>
```

**Step 3: Wrap Current Utilization metrics**

After the Infrastructure Headroom container, add a Current Utilization container wrapping:

- Memory Utilization gauge
- Disk Utilization gauge
- Staging Capacity (free chunks)
- TPS Performance section
- Detailed Metrics Grid (Cell Count, App Capacity, Fault Impact, Instances/Cell)
- Cell Configuration Change

Container:

```jsx
{/* Current Utilization */}
<div
  data-testid="section-current-utilization"
  className="rounded-2xl border border-slate-600/40 bg-slate-800/40 p-5"
>
  <SectionHeader icon={Activity} label="Current Utilization" />
  <div className="space-y-6">
    {/* Utilization gauges grid */}
    <div className={`grid gap-6 ${/* dynamic cols based on which util gauges show */}`}>
      {/* Memory Utilization gauge (existing JSX, unchanged) */}
      {/* Disk Utilization gauge (existing JSX, unchanged) */}
      {/* Staging Capacity (existing JSX, unchanged) */}
    </div>
    {/* TPS Performance (existing JSX, unchanged) */}
    {/* Detailed Metrics Grid (existing JSX, unchanged) */}
    {/* Cell Configuration Change (existing JSX, unchanged) */}
  </div>
</div>
```

**Key implementation notes:**

- The gauge grid column calculation needs splitting. Infrastructure grid: `grid-cols-1` or `grid-cols-2` (N-1 always shows; CPU is conditional). Utilization grid: dynamic based on which of memory/disk/staging are visible (staging always shows).
- Move existing JSX blocks -- do NOT rewrite them. Cut and paste within the file.
- The `data-testid` attributes are required for the containment tests.

**Step 4: Run tests to verify they pass**

Run: `make frontend-test`
Expected: All 250+ tests pass, including the new Metric Grouping tests.

**Step 5: Commit**

```bash
git add frontend/src/components/ScenarioResults.jsx
git commit -m "feat: group metrics by scope into Infrastructure Headroom and Current Utilization (#55)"
```

---

### Task 3: Visual review and adjustments

**Files:**

- Modify: `frontend/src/components/ScenarioResults.jsx` (if adjustments needed)

**Step 1: Start the dev servers**

Run: `make backend-dev` (in one terminal)
Run: `make frontend-dev` (in another terminal)

Open browser to http://localhost:5173, load scenario data, and visually inspect the grouping.

**Step 2: Check visual rendering**

Verify:

- Both section headers render with correct icons and labels
- Background tint difference is visible but subtle
- Inner metric cards maintain their existing appearance
- Responsive layout works at mobile and desktop widths
- The gauge grids within each section have correct column counts

**Step 3: Make any Tailwind adjustments**

If padding, gaps, or border weights need tweaking, adjust the container classes.

**Step 4: Run tests after any adjustments**

Run: `make frontend-test`
Expected: All tests pass.

**Step 5: Commit any adjustments**

```bash
git add frontend/src/components/ScenarioResults.jsx
git commit -m "style: adjust metric group container spacing (#55)"
```

---

### Task 4: Final verification and cleanup

**Step 1: Run full test suite**

Run: `make check`
Expected: All tests and linters pass.

**Step 2: Verify no unrelated changes**

Run: `git diff main --stat`
Expected: Only `ScenarioResults.jsx`, `ScenarioResults.test.jsx`, and docs files changed.

**Step 3: Push and open PR**

```bash
git push -u origin issue-55-metric-grouping
gh pr create --title "UX: Group metrics by scope (#55)" --body "..."
```
