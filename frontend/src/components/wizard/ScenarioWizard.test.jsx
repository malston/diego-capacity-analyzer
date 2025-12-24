// ABOUTME: Tests for scenario wizard container component
// ABOUTME: Covers step navigation, state management, and step rendering

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScenarioWizard from './ScenarioWizard';

describe('ScenarioWizard', () => {
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
    selectedResources: ['memory'],
    toggleResource: vi.fn(),
    customDisk: 128,
    setCustomDisk: vi.fn(),
    // Advanced props
    overheadPct: 7,
    setOverheadPct: vi.fn(),
    useAdditionalApp: false,
    setUseAdditionalApp: vi.fn(),
    additionalApp: { name: 'test', instances: 1, memoryGB: 1, diskGB: 1 },
    setAdditionalApp: vi.fn(),
    tpsCurve: [{ cells: 50, tps: 500 }],
    setTPSCurve: vi.fn(),
    onStepComplete: vi.fn(),
  };

  it('renders step indicator', () => {
    render(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByText('Cell Config')).toBeInTheDocument();
    expect(screen.getByText('Resources')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
  });

  it('shows CellConfigStep initially', () => {
    render(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });

  it('advances to ResourceTypesStep after continuing from Step 1', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(screen.getByText(/which resources to analyze/i)).toBeInTheDocument();
  });

  it('advances to AdvancedStep after continuing from Step 2', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    // Step 1 -> Step 2
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    // Step 2 -> Step 3
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('calls onStepComplete after Step 1', async () => {
    const onStepComplete = vi.fn();
    render(<ScenarioWizard {...defaultProps} onStepComplete={onStepComplete} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onStepComplete).toHaveBeenCalledWith(0);
  });

  it('allows skipping optional steps', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('allows clicking on completed steps to navigate back', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    // Go to step 2
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    // Click on step 1 in indicator
    await userEvent.click(screen.getByText('Cell Config'));
    // Should show step 1 content
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });
});
