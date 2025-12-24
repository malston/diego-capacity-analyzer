// ABOUTME: Tests for cell configuration step in scenario wizard
// ABOUTME: Covers VM size selection, custom inputs, and cell count

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CellConfigStep from './CellConfigStep';

describe('CellConfigStep', () => {
  const defaultProps = {
    selectedPreset: 0,
    setSelectedPreset: vi.fn(),
    customCPU: 4,
    setCustomCPU: vi.fn(),
    customMemory: 32,
    setCustomMemory: vi.fn(),
    cellCount: 100,
    setCellCount: vi.fn(),
    equivalentCellSuggestion: null,
    onContinue: vi.fn(),
  };

  it('renders VM size dropdown', () => {
    render(<CellConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });

  it('renders cell count input', () => {
    render(<CellConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/cell count/i)).toBeInTheDocument();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<CellConfigStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it('disables Continue when cellCount is 0', () => {
    render(<CellConfigStep {...defaultProps} cellCount={0} />);
    expect(screen.getByRole('button', { name: /continue/i })).toBeDisabled();
  });

  it('shows equivalent cells suggestion when provided', () => {
    render(
      <CellConfigStep
        {...defaultProps}
        equivalentCellSuggestion={{
          equivalentCells: 200,
          currentTotalGB: 6400,
        }}
      />
    );
    expect(screen.getByText(/equivalent capacity/i)).toBeInTheDocument();
    expect(screen.getByText(/200 cells/i)).toBeInTheDocument();
  });
});
