// ABOUTME: Tests for wizard step indicator component
// ABOUTME: Covers step rendering, click navigation, and visual states

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import StepIndicator from "./StepIndicator";

const STEPS = [
  { id: "cell-config", label: "Cell Config", required: true },
  { id: "resources", label: "Resources", required: false },
  { id: "advanced", label: "Advanced", required: false },
];

describe("StepIndicator", () => {
  const defaultProps = {
    steps: STEPS,
    currentStep: 0,
    completedSteps: [],
    onStepClick: vi.fn(),
  };

  it("renders all step labels", () => {
    render(<StepIndicator {...defaultProps} />);
    expect(screen.getByText("Cell Config")).toBeInTheDocument();
    expect(screen.getByText("Resources")).toBeInTheDocument();
    expect(screen.getByText("Advanced")).toBeInTheDocument();
  });

  it("marks current step as active", () => {
    render(<StepIndicator {...defaultProps} currentStep={1} />);
    const resourcesStep = screen.getByText("Resources").closest("button");
    expect(resourcesStep).toHaveAttribute("aria-current", "step");
  });

  it("marks completed steps with checkmark", () => {
    render(
      <StepIndicator {...defaultProps} completedSteps={["cell-config"]} />,
    );
    const cellConfigStep = screen.getByText("Cell Config").closest("button");
    expect(cellConfigStep).toHaveAttribute("data-completed", "true");
  });

  it("calls onStepClick when clicking completed step", async () => {
    const onStepClick = vi.fn();
    render(
      <StepIndicator
        {...defaultProps}
        currentStep={1}
        completedSteps={["cell-config"]}
        onStepClick={onStepClick}
      />,
    );
    await userEvent.click(screen.getByText("Cell Config"));
    expect(onStepClick).toHaveBeenCalledWith(0);
  });

  it("calls onStepClick when clicking any step (free navigation)", async () => {
    const onStepClick = vi.fn();
    render(
      <StepIndicator
        {...defaultProps}
        currentStep={0}
        completedSteps={[]}
        onStepClick={onStepClick}
      />,
    );
    await userEvent.click(screen.getByText("Advanced"));
    expect(onStepClick).toHaveBeenCalledWith(2);
  });
});
