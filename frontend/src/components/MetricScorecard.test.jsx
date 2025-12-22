// ABOUTME: Tests for MetricScorecard component
// ABOUTME: Covers rendering, status display, and change indicators

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MetricScorecard from './MetricScorecard';

describe('MetricScorecard', () => {
  const defaultProps = {
    label: 'Test Metric',
    currentValue: 50,
    proposedValue: 75,
  };

  it('renders label', () => {
    render(<MetricScorecard {...defaultProps} />);
    expect(screen.getByText('Test Metric')).toBeInTheDocument();
  });

  it('renders current and proposed values', () => {
    render(<MetricScorecard {...defaultProps} />);
    expect(screen.getByText('50')).toBeInTheDocument();
    expect(screen.getByText('75')).toBeInTheDocument();
  });

  it('applies format function to values', () => {
    render(
      <MetricScorecard
        {...defaultProps}
        format={(v) => `${v}GB`}
      />
    );
    expect(screen.getByText('50GB')).toBeInTheDocument();
    expect(screen.getByText('75GB')).toBeInTheDocument();
  });

  it('displays unit suffix', () => {
    render(
      <MetricScorecard
        {...defaultProps}
        unit="%"
      />
    );
    expect(screen.getByText('50%')).toBeInTheDocument();
  });

  it('shows good status when below warning threshold', () => {
    render(
      <MetricScorecard
        {...defaultProps}
        proposedValue={50}
        thresholds={{ warning: 75, critical: 85 }}
      />
    );
    expect(screen.getByText('good')).toBeInTheDocument();
  });

  it('shows warning status when above warning threshold', () => {
    render(
      <MetricScorecard
        {...defaultProps}
        proposedValue={80}
        thresholds={{ warning: 75, critical: 85 }}
      />
    );
    expect(screen.getByText('warning')).toBeInTheDocument();
  });

  it('shows critical status when above critical threshold', () => {
    render(
      <MetricScorecard
        {...defaultProps}
        proposedValue={90}
        thresholds={{ warning: 75, critical: 85 }}
      />
    );
    expect(screen.getByText('critical')).toBeInTheDocument();
  });

  it('displays change amount', () => {
    render(<MetricScorecard {...defaultProps} />);
    // Change is 75 - 50 = +25
    expect(screen.getByText('+25')).toBeInTheDocument();
  });

  it('handles zero current value without NaN', () => {
    render(
      <MetricScorecard
        label="Test"
        currentValue={0}
        proposedValue={100}
      />
    );
    // Should not throw and should render values
    expect(screen.getByText('0')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
  });
});
