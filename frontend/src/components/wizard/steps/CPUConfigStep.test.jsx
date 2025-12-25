// ABOUTME: Tests for CPU configuration step in scenario wizard
// ABOUTME: Covers physical cores, host count, and vCPU:pCPU ratio inputs

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CPUConfigStep from './CPUConfigStep';

describe('CPUConfigStep', () => {
  const defaultProps = {
    physicalCoresPerHost: 32,
    setPhysicalCoresPerHost: vi.fn(),
    hostCount: 3,
    setHostCount: vi.fn(),
    targetVCPURatio: 4,
    setTargetVCPURatio: vi.fn(),
    onContinue: vi.fn(),
    onSkip: vi.fn(),
  };

  it('renders physical cores per host input', () => {
    render(<CPUConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/physical cores per host/i)).toBeInTheDocument();
  });

  it('renders host count input', () => {
    render(<CPUConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/number of hosts/i)).toBeInTheDocument();
  });

  it('renders vCPU:pCPU ratio input', () => {
    render(<CPUConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/target vcpu.*ratio/i)).toBeInTheDocument();
  });

  it('displays current values in inputs', () => {
    render(<CPUConfigStep {...defaultProps} />);

    expect(screen.getByLabelText(/physical cores per host/i)).toHaveValue(32);
    expect(screen.getByLabelText(/number of hosts/i)).toHaveValue(3);
    expect(screen.getByLabelText(/target vcpu.*ratio/i)).toHaveValue(4);
  });

  it('calls setPhysicalCoresPerHost when cores input changes', async () => {
    const setPhysicalCoresPerHost = vi.fn();
    render(<CPUConfigStep {...defaultProps} setPhysicalCoresPerHost={setPhysicalCoresPerHost} />);

    const input = screen.getByLabelText(/physical cores per host/i);
    await userEvent.clear(input);
    await userEvent.type(input, '64');

    expect(setPhysicalCoresPerHost).toHaveBeenCalled();
  });

  it('calls setHostCount when host count input changes', async () => {
    const setHostCount = vi.fn();
    render(<CPUConfigStep {...defaultProps} setHostCount={setHostCount} />);

    const input = screen.getByLabelText(/number of hosts/i);
    await userEvent.clear(input);
    await userEvent.type(input, '5');

    expect(setHostCount).toHaveBeenCalled();
  });

  it('calls setTargetVCPURatio when ratio input changes', async () => {
    const setTargetVCPURatio = vi.fn();
    render(<CPUConfigStep {...defaultProps} setTargetVCPURatio={setTargetVCPURatio} />);

    const input = screen.getByLabelText(/target vcpu.*ratio/i);
    await userEvent.clear(input);
    await userEvent.type(input, '8');

    expect(setTargetVCPURatio).toHaveBeenCalled();
  });

  it('shows ratio risk level indicator - low risk for ratio <= 4', () => {
    render(<CPUConfigStep {...defaultProps} targetVCPURatio={4} />);
    expect(screen.getByText(/low.*production safe/i)).toBeInTheDocument();
  });

  it('shows ratio risk level indicator - medium risk for ratio 5-8', () => {
    render(<CPUConfigStep {...defaultProps} targetVCPURatio={6} />);
    expect(screen.getByText(/medium.*monitor cpu ready/i)).toBeInTheDocument();
  });

  it('shows ratio risk level indicator - high risk for ratio > 8', () => {
    render(<CPUConfigStep {...defaultProps} targetVCPURatio={10} />);
    expect(screen.getByText(/high.*expect contention/i)).toBeInTheDocument();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<CPUConfigStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it('calls onSkip when Skip button clicked', async () => {
    const onSkip = vi.fn();
    render(<CPUConfigStep {...defaultProps} onSkip={onSkip} />);
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(onSkip).toHaveBeenCalled();
  });

  it('disables Continue when hostCount is 0', () => {
    render(<CPUConfigStep {...defaultProps} hostCount={0} />);
    expect(screen.getByRole('button', { name: /continue/i })).toBeDisabled();
  });

  it('disables Continue when physicalCoresPerHost is 0', () => {
    render(<CPUConfigStep {...defaultProps} physicalCoresPerHost={0} />);
    expect(screen.getByRole('button', { name: /continue/i })).toBeDisabled();
  });

  it('displays total physical CPU cores calculation', () => {
    render(<CPUConfigStep {...defaultProps} physicalCoresPerHost={32} hostCount={4} />);
    // Check that the total cores calculation is displayed (128 = 32 * 4)
    expect(screen.getByText(/32 cores × 4 hosts = 128 total cores/)).toBeInTheDocument();
  });
});
