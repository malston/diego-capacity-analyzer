// ABOUTME: Tests for scenario wizard container component
// ABOUTME: Covers step navigation, state management, and step rendering

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ScenarioWizard from "./ScenarioWizard";
import { ToastProvider } from "../../contexts/ToastContext";

// Helper to wrap component with ToastProvider
const renderWithToast = (ui) => render(<ToastProvider>{ui}</ToastProvider>);

describe("ScenarioWizard", () => {
  const defaultProps = {
    // Cell config props
    selectedPreset: 0,
    setSelectedPreset: vi.fn(),
    customCPU: 4,
    setCustomCPU: vi.fn(),
    customMemory: 32,
    setCustomMemory: vi.fn(),
    cellCount: 100,
    setCellCount: vi.fn(),
    equivalentCellSuggestion: null,
    // Resource props
    selectedResources: ["memory"],
    toggleResource: vi.fn(),
    customDisk: 128,
    setCustomDisk: vi.fn(),
    // CPU config props
    physicalCoresPerHost: 32,
    setPhysicalCoresPerHost: vi.fn(),
    hostCount: 3,
    setHostCount: vi.fn(),
    targetVCPURatio: 4,
    setTargetVCPURatio: vi.fn(),
    platformVMsCPU: 0,
    setPlatformVMsCPU: vi.fn(),
    totalVCPUs: 400,
    // Host config props (for Advanced step)
    memoryPerHost: 512,
    setMemoryPerHost: vi.fn(),
    haAdmissionPct: 7,
    setHaAdmissionPct: vi.fn(),
    // Advanced props
    overheadPct: 7,
    setOverheadPct: vi.fn(),
    useAdditionalApp: false,
    setUseAdditionalApp: vi.fn(),
    additionalApp: { name: "test", instances: 1, memoryGB: 1, diskGB: 1 },
    setAdditionalApp: vi.fn(),
    tpsCurve: [{ cells: 50, tps: 500 }],
    setTPSCurve: vi.fn(),
    enableTPS: false,
    setEnableTPS: vi.fn(),
  };

  it("renders step indicator", () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByText("Resources")).toBeInTheDocument();
    expect(screen.getByText("Cell Config")).toBeInTheDocument();
    expect(screen.getByText("Advanced")).toBeInTheDocument();
  });

  it("shows ResourceTypesStep initially", () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByText(/which resources to analyze/i)).toBeInTheDocument();
  });

  it("advances to CellConfigStep after continuing from Step 1", async () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });

  it("advances to AdvancedStep after continuing from Step 2", async () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    // Step 1 (Resources) -> Step 2 (Cell Config)
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));
    // Step 2 (Cell Config) -> Step 3 (Advanced)
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it("does not show Skip button on required steps", () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    // Resources step is required, no Skip button
    expect(
      screen.queryByRole("button", { name: /skip/i }),
    ).not.toBeInTheDocument();
  });

  it("marks departing step as completed when clicking another step in indicator", async () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    // We're on step 0 (Resources). Click directly on step 1 (Cell Config) in indicator.
    await userEvent.click(screen.getByText("Cell Config"));
    // Step 0 should now be marked completed (green checkmark)
    const resourcesButton = screen.getByText("Resources").closest("button");
    expect(resourcesButton).toHaveAttribute("data-completed", "true");
  });

  it("does not mark current step as completed when clicking it again", async () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    // Click on step 0 (Resources) while already on step 0
    await userEvent.click(screen.getByText("Resources"));
    // Step 0 should NOT be marked completed -- user hasn't left it
    const resourcesButton = screen.getByText("Resources").closest("button");
    expect(resourcesButton).not.toHaveAttribute("data-completed", "true");
  });

  it("allows clicking on completed steps to navigate back", async () => {
    renderWithToast(<ScenarioWizard {...defaultProps} />);
    // Go to step 2 (Cell Config)
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));
    // Click on step 1 (Resources) in indicator
    await userEvent.click(screen.getByText("Resources"));
    // Should show step 1 content
    expect(screen.getByText(/which resources to analyze/i)).toBeInTheDocument();
  });

  it("shows CPU Config step when cpu resource is selected", async () => {
    renderWithToast(
      <ScenarioWizard
        {...defaultProps}
        selectedResources={["memory", "cpu"]}
      />,
    );
    // Should have CPU Config step in indicator
    expect(screen.getByText("CPU Config")).toBeInTheDocument();
    // Navigate to CPU Config step: Resources -> Cell Config -> CPU Config
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // to Cell Config
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // to CPU Config
    // Should show CPU Config step content
    expect(
      screen.getByLabelText(/physical cores per host/i),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/number of hosts/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/target vcpu.*ratio/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/platform vm vcpus/i)).toBeInTheDocument();
  });

  it("hides CPU Config step when cpu resource is not selected", () => {
    renderWithToast(
      <ScenarioWizard {...defaultProps} selectedResources={["memory"]} />,
    );
    // Should NOT have CPU Config step in indicator
    expect(screen.queryByText("CPU Config")).not.toBeInTheDocument();
  });

  it("preserves correct completion state when CPU step is toggled off", async () => {
    // Start with CPU selected (steps: Resources, Cell Config, CPU Config, Advanced)
    const { rerender } = renderWithToast(
      <ScenarioWizard
        {...defaultProps}
        selectedResources={["memory", "cpu"]}
      />,
    );

    // Advance through Resources -> Cell Config -> CPU Config (completing 0, 1, 2)
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // Resources -> Cell Config
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // Cell Config -> CPU Config
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // CPU Config -> Advanced

    // Verify Resources and Cell Config are completed
    const resourcesBefore = screen.getByText("Resources").closest("button");
    const cellConfigBefore = screen.getByText("Cell Config").closest("button");
    expect(resourcesBefore).toHaveAttribute("data-completed", "true");
    expect(cellConfigBefore).toHaveAttribute("data-completed", "true");

    // Rerender without CPU (steps become: Resources, Cell Config, Advanced)
    rerender(
      <ToastProvider>
        <ScenarioWizard {...defaultProps} selectedResources={["memory"]} />
      </ToastProvider>,
    );

    // Resources and Cell Config should still be completed
    const resourcesAfter = screen.getByText("Resources").closest("button");
    const cellConfigAfter = screen.getByText("Cell Config").closest("button");
    expect(resourcesAfter).toHaveAttribute("data-completed", "true");
    expect(cellConfigAfter).toHaveAttribute("data-completed", "true");

    // Advanced should NOT be completed (it was never completed, only navigated to)
    const advancedAfter = screen.getByText("Advanced").closest("button");
    expect(advancedAfter).not.toHaveAttribute("data-completed", "true");
  });

  it("passes platformVMsCPU prop to CPUConfigStep", async () => {
    const setPlatformVMsCPU = vi.fn();
    renderWithToast(
      <ScenarioWizard
        {...defaultProps}
        selectedResources={["memory", "cpu"]}
        platformVMsCPU={120}
        setPlatformVMsCPU={setPlatformVMsCPU}
      />,
    );
    // Navigate to CPU Config step
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // to Cell Config
    await userEvent.click(screen.getByRole("button", { name: /continue/i })); // to CPU Config
    // Verify platformVMsCPU value is displayed
    const input = screen.getByLabelText(/platform vm vcpus/i);
    expect(input).toHaveValue(120);
    // Change value and verify setter is called
    await userEvent.clear(input);
    await userEvent.type(input, "200");
    expect(setPlatformVMsCPU).toHaveBeenCalled();
  });
});
