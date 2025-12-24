// ABOUTME: Tests for advanced options step in scenario wizard
// ABOUTME: Covers overhead slider, hypothetical app, and TPS curve

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AdvancedStep from './AdvancedStep';

describe('AdvancedStep', () => {
  const defaultProps = {
    overheadPct: 7,
    setOverheadPct: vi.fn(),
    useAdditionalApp: false,
    setUseAdditionalApp: vi.fn(),
    additionalApp: { name: 'test-app', instances: 1, memoryGB: 1, diskGB: 1 },
    setAdditionalApp: vi.fn(),
    tpsCurve: [{ cells: 50, tps: 500 }],
    setTPSCurve: vi.fn(),
    onContinue: vi.fn(),
    onSkip: vi.fn(),
  };

  it('renders overhead slider', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('displays current overhead percentage', () => {
    render(<AdvancedStep {...defaultProps} overheadPct={10} />);
    expect(screen.getByText(/10%/)).toBeInTheDocument();
  });

  it('renders hypothetical app section', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByText(/hypothetical app/i)).toBeInTheDocument();
  });

  it('shows app inputs when checkbox is checked', async () => {
    render(<AdvancedStep {...defaultProps} useAdditionalApp={true} />);
    expect(screen.getByLabelText(/app name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/instances/i)).toBeInTheDocument();
  });

  it('renders TPS curve section', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByText(/tps performance curve/i)).toBeInTheDocument();
  });

  it('calls onSkip when Skip button clicked', async () => {
    const onSkip = vi.fn();
    render(<AdvancedStep {...defaultProps} onSkip={onSkip} />);
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(onSkip).toHaveBeenCalled();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<AdvancedStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it('shows completion message instead of buttons when isLastStep is true', () => {
    render(<AdvancedStep {...defaultProps} isLastStep={true} />);
    expect(screen.getByText(/configuration complete/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /continue/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /skip/i })).not.toBeInTheDocument();
  });
});
