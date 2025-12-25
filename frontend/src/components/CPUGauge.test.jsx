// ABOUTME: Tests for CPU utilization gauge component
// ABOUTME: Covers gauge rendering, risk levels, and vCPU:pCPU ratio indicator

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CPUGauge from './CPUGauge';

describe('CPUGauge', () => {
  const defaultProps = {
    cpuUtilization: 45,
    vcpuRatio: 4,
  };

  describe('CPU Utilization Gauge', () => {
    it('renders CPU utilization value', () => {
      render(<CPUGauge {...defaultProps} />);
      expect(screen.getByText(/45\.0%/)).toBeInTheDocument();
    });

    it('renders CPU label', () => {
      render(<CPUGauge {...defaultProps} />);
      expect(screen.getByText(/cpu/i)).toBeInTheDocument();
    });

    it('shows warning color for utilization 70-85%', () => {
      const { container } = render(<CPUGauge {...defaultProps} cpuUtilization={75} />);
      // Amber color for warning: #f59e0b
      const progressCircle = container.querySelector('circle[stroke="#f59e0b"]');
      expect(progressCircle).toBeInTheDocument();
    });

    it('shows critical color for utilization > 85%', () => {
      const { container } = render(<CPUGauge {...defaultProps} cpuUtilization={90} />);
      // Red color for critical: #ef4444
      const progressCircle = container.querySelector('circle[stroke="#ef4444"]');
      expect(progressCircle).toBeInTheDocument();
    });

    it('shows normal color for utilization < 70%', () => {
      const { container } = render(<CPUGauge {...defaultProps} cpuUtilization={50} />);
      // Cyan color for normal: #06b6d4
      const progressCircle = container.querySelector('circle[stroke="#06b6d4"]');
      expect(progressCircle).toBeInTheDocument();
    });
  });

  describe('vCPU:pCPU Ratio Indicator', () => {
    it('renders vCPU ratio value', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={4} />);
      expect(screen.getByText(/4:1/)).toBeInTheDocument();
    });

    it('shows low risk for ratio <= 4', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={4} />);
      expect(screen.getByText(/low/i)).toBeInTheDocument();
    });

    it('shows medium risk for ratio 5-8', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={6} />);
      expect(screen.getByText(/medium/i)).toBeInTheDocument();
    });

    it('shows high risk for ratio > 8', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={10} />);
      expect(screen.getByText(/high/i)).toBeInTheDocument();
    });

    it('displays ratio with correct color - green for low', () => {
      const { container } = render(<CPUGauge {...defaultProps} vcpuRatio={4} />);
      const ratioBadge = container.querySelector('.text-emerald-400');
      expect(ratioBadge).toBeInTheDocument();
    });

    it('displays ratio with correct color - amber for medium', () => {
      const { container } = render(<CPUGauge {...defaultProps} vcpuRatio={6} />);
      const ratioBadge = container.querySelector('.text-amber-400');
      expect(ratioBadge).toBeInTheDocument();
    });

    it('displays ratio with correct color - red for high', () => {
      const { container } = render(<CPUGauge {...defaultProps} vcpuRatio={10} />);
      const ratioBadge = container.querySelector('.text-red-400');
      expect(ratioBadge).toBeInTheDocument();
    });
  });

  describe('Size and Layout', () => {
    it('uses default size of 120', () => {
      const { container } = render(<CPUGauge {...defaultProps} />);
      const svg = container.querySelector('svg');
      expect(svg).toHaveAttribute('width', '120');
      expect(svg).toHaveAttribute('height', '120');
    });

    it('accepts custom size', () => {
      const { container } = render(<CPUGauge {...defaultProps} size={150} />);
      const svg = container.querySelector('svg');
      expect(svg).toHaveAttribute('width', '150');
      expect(svg).toHaveAttribute('height', '150');
    });
  });

  describe('Edge Cases', () => {
    it('handles 0% utilization', () => {
      render(<CPUGauge {...defaultProps} cpuUtilization={0} />);
      expect(screen.getByText(/0\.0%/)).toBeInTheDocument();
    });

    it('handles 100% utilization', () => {
      render(<CPUGauge {...defaultProps} cpuUtilization={100} />);
      expect(screen.getByText(/100\.0%/)).toBeInTheDocument();
    });

    it('handles ratio of 1', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={1} />);
      expect(screen.getByText(/1:1/)).toBeInTheDocument();
    });

    it('handles very high ratio', () => {
      render(<CPUGauge {...defaultProps} vcpuRatio={16} />);
      expect(screen.getByText(/16:1/)).toBeInTheDocument();
      expect(screen.getByText(/high/i)).toBeInTheDocument();
    });
  });
});
