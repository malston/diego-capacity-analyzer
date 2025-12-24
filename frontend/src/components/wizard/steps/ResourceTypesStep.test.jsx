// ABOUTME: Tests for resource types step in scenario wizard
// ABOUTME: Covers resource toggle buttons and disk input

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ResourceTypesStep from './ResourceTypesStep';

describe('ResourceTypesStep', () => {
  const defaultProps = {
    selectedResources: ['memory'],
    toggleResource: vi.fn(),
    customDisk: 128,
    setCustomDisk: vi.fn(),
    onContinue: vi.fn(),
    onSkip: vi.fn(),
  };

  it('renders resource type buttons', () => {
    render(<ResourceTypesStep {...defaultProps} />);
    expect(screen.getByRole('button', { name: /memory/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /cpu/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /disk/i })).toBeInTheDocument();
  });

  it('shows selected state for active resources', () => {
    render(<ResourceTypesStep {...defaultProps} selectedResources={['memory', 'cpu']} />);
    const memoryBtn = screen.getByRole('button', { name: /memory/i });
    expect(memoryBtn).toHaveAttribute('aria-pressed', 'true');
  });

  it('calls toggleResource when clicking resource button', async () => {
    const toggleResource = vi.fn();
    render(<ResourceTypesStep {...defaultProps} toggleResource={toggleResource} />);
    await userEvent.click(screen.getByRole('button', { name: /cpu/i }));
    expect(toggleResource).toHaveBeenCalledWith('cpu');
  });

  it('shows disk input only when disk is selected', () => {
    const { rerender } = render(<ResourceTypesStep {...defaultProps} selectedResources={['memory']} />);
    expect(screen.queryByLabelText(/disk per cell/i)).not.toBeInTheDocument();

    rerender(<ResourceTypesStep {...defaultProps} selectedResources={['memory', 'disk']} />);
    expect(screen.getByLabelText(/disk per cell/i)).toBeInTheDocument();
  });

  it('calls onSkip when Skip button clicked', async () => {
    const onSkip = vi.fn();
    render(<ResourceTypesStep {...defaultProps} onSkip={onSkip} />);
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(onSkip).toHaveBeenCalled();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<ResourceTypesStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });
});
