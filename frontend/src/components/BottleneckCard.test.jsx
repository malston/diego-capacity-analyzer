// ABOUTME: Tests for multi-resource bottleneck display component
// ABOUTME: Covers resource ranking, exhaustion ordering, and bottleneck highlighting

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import BottleneckCard from './BottleneckCard';

describe('BottleneckCard', () => {
  const defaultProps = {
    resources: [
      { name: 'Memory', utilization: 78, type: 'memory' },
      { name: 'Disk', utilization: 45, type: 'disk' },
      { name: 'CPU', utilization: 32, type: 'cpu' },
    ],
  };

  describe('Resource Exhaustion Ordering', () => {
    it('renders section title', () => {
      render(<BottleneckCard {...defaultProps} />);
      expect(screen.getByText(/resource exhaustion/i)).toBeInTheDocument();
    });

    it('displays resources in order of utilization (highest first)', () => {
      render(<BottleneckCard {...defaultProps} />);

      const items = screen.getAllByRole('listitem');
      expect(items).toHaveLength(3);

      // Check ordering - Memory (78%) should be first
      expect(items[0]).toHaveTextContent(/memory/i);
      expect(items[0]).toHaveTextContent(/78%/);

      // Disk (45%) second
      expect(items[1]).toHaveTextContent(/disk/i);

      // CPU (32%) third
      expect(items[2]).toHaveTextContent(/cpu/i);
    });

    it('displays ranking numbers', () => {
      render(<BottleneckCard {...defaultProps} />);

      expect(screen.getByText('1')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
      expect(screen.getByText('3')).toBeInTheDocument();
    });
  });

  describe('Bottleneck Highlighting', () => {
    it('highlights the constraining resource (highest utilization)', () => {
      const { container } = render(<BottleneckCard {...defaultProps} />);

      // First item should have highlight styling
      const firstItem = container.querySelector('[data-constraint="true"]');
      expect(firstItem).toBeInTheDocument();
      expect(firstItem).toHaveTextContent(/memory/i);
    });

    it('shows constraint indicator for first resource', () => {
      render(<BottleneckCard {...defaultProps} />);

      expect(screen.getByText(/closest to limit/i)).toBeInTheDocument();
    });

    it('uses different styling for non-constraint resources', () => {
      const { container } = render(<BottleneckCard {...defaultProps} />);

      const nonConstraints = container.querySelectorAll('[data-constraint="false"]');
      expect(nonConstraints.length).toBe(2);
    });
  });

  describe('Utilization Status Colors', () => {
    it('shows critical color for utilization > 85%', () => {
      render(
        <BottleneckCard
          resources={[
            { name: 'Memory', utilization: 90, type: 'memory' },
          ]}
        />
      );

      const { container } = render(
        <BottleneckCard
          resources={[{ name: 'Memory', utilization: 90, type: 'memory' }]}
        />
      );
      expect(container.querySelector('.text-red-400')).toBeInTheDocument();
    });

    it('shows warning color for utilization 70-85%', () => {
      const { container } = render(
        <BottleneckCard
          resources={[{ name: 'Memory', utilization: 80, type: 'memory' }]}
        />
      );
      expect(container.querySelector('.text-amber-400')).toBeInTheDocument();
    });

    it('shows normal color for utilization < 70%', () => {
      const { container } = render(
        <BottleneckCard
          resources={[{ name: 'Memory', utilization: 50, type: 'memory' }]}
        />
      );
      expect(container.querySelector('.text-cyan-400')).toBeInTheDocument();
    });
  });

  describe('Progress Bars', () => {
    it('renders progress bar for each resource', () => {
      const { container } = render(<BottleneckCard {...defaultProps} />);

      const progressBars = container.querySelectorAll('[role="progressbar"]');
      expect(progressBars.length).toBe(3);
    });

    it('sets progress bar width based on utilization', () => {
      const { container } = render(
        <BottleneckCard
          resources={[{ name: 'Memory', utilization: 50, type: 'memory' }]}
        />
      );

      const progressFill = container.querySelector('[role="progressbar"] > div');
      expect(progressFill).toHaveStyle({ width: '50%' });
    });
  });

  describe('Recommendation Message', () => {
    it('shows recommendation for the constraining resource', () => {
      render(<BottleneckCard {...defaultProps} />);

      // Recommendation contains "Memory" and "constraint" in the text
      expect(screen.getByText(/memory/i, { selector: '.text-amber-400' })).toBeInTheDocument();
      expect(screen.getByText(/is your constraint/i)).toBeInTheDocument();
    });

    it('updates recommendation based on highest utilization resource', () => {
      render(
        <BottleneckCard
          resources={[
            { name: 'CPU', utilization: 85, type: 'cpu' },
            { name: 'Memory', utilization: 60, type: 'memory' },
          ]}
        />
      );

      // CPU should be highlighted as the constraint
      expect(screen.getByText('CPU', { selector: '.text-amber-400' })).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('handles empty resources array', () => {
      render(<BottleneckCard resources={[]} />);
      expect(screen.getByText(/no resources/i)).toBeInTheDocument();
    });

    it('handles single resource', () => {
      render(
        <BottleneckCard
          resources={[{ name: 'Memory', utilization: 70, type: 'memory' }]}
        />
      );
      // Single resource should show in the list
      const items = screen.getAllByRole('listitem');
      expect(items.length).toBe(1);
      expect(items[0]).toHaveTextContent(/memory/i);
    });

    it('handles equal utilization resources', () => {
      render(
        <BottleneckCard
          resources={[
            { name: 'Memory', utilization: 50, type: 'memory' },
            { name: 'CPU', utilization: 50, type: 'cpu' },
          ]}
        />
      );

      const items = screen.getAllByRole('listitem');
      expect(items.length).toBe(2);
    });
  });

  describe('Resource Icons', () => {
    it('displays appropriate icon for each resource type', () => {
      const { container } = render(<BottleneckCard {...defaultProps} />);

      // Should have lucide icons for memory, disk, cpu
      expect(container.querySelector('.lucide-hard-drive')).toBeInTheDocument();
      expect(container.querySelector('.lucide-database')).toBeInTheDocument();
      expect(container.querySelector('.lucide-cpu')).toBeInTheDocument();
    });
  });
});
