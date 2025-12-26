// ABOUTME: Tests for collapsible host configuration section
// ABOUTME: Covers host inputs, HA admission control, and expand/collapse behavior

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import HostConfigSection from './HostConfigSection';

describe('HostConfigSection', () => {
  const defaultProps = {
    hostCount: 3,
    setHostCount: vi.fn(),
    coresPerHost: 32,
    setCoresPerHost: vi.fn(),
    memoryPerHost: 512,
    setMemoryPerHost: vi.fn(),
    haAdmissionPct: 25,
    setHaAdmissionPct: vi.fn(),
  };

  describe('Section Header', () => {
    it('renders section title', () => {
      render(<HostConfigSection {...defaultProps} />);
      expect(screen.getByText(/host configuration/i)).toBeInTheDocument();
    });

    it('renders optional label', () => {
      render(<HostConfigSection {...defaultProps} />);
      expect(screen.getByText(/optional/i)).toBeInTheDocument();
    });

    it('is collapsed by default', () => {
      render(<HostConfigSection {...defaultProps} />);
      // When collapsed, inputs should not be visible
      expect(screen.queryByLabelText(/number of hosts/i)).not.toBeInTheDocument();
    });
  });

  describe('Expand/Collapse Behavior', () => {
    it('expands when header is clicked', async () => {
      render(<HostConfigSection {...defaultProps} />);

      const header = screen.getByRole('button', { name: /host configuration/i });
      await userEvent.click(header);

      // After expanding, inputs should be visible
      expect(screen.getByLabelText(/number of hosts/i)).toBeInTheDocument();
    });

    it('collapses when header is clicked again', async () => {
      render(<HostConfigSection {...defaultProps} />);

      const header = screen.getByRole('button', { name: /host configuration/i });
      await userEvent.click(header); // Expand
      await userEvent.click(header); // Collapse

      expect(screen.queryByLabelText(/number of hosts/i)).not.toBeInTheDocument();
    });

    it('shows expand icon when collapsed', () => {
      const { container } = render(<HostConfigSection {...defaultProps} />);
      // ChevronDown icon indicates collapsed state
      expect(container.querySelector('.lucide-chevron-down')).toBeInTheDocument();
    });

    it('shows collapse icon when expanded', async () => {
      const { container } = render(<HostConfigSection {...defaultProps} />);

      const header = screen.getByRole('button', { name: /host configuration/i });
      await userEvent.click(header);

      // ChevronUp icon indicates expanded state
      expect(container.querySelector('.lucide-chevron-up')).toBeInTheDocument();
    });
  });

  describe('Host Count Input', () => {
    it('renders host count input when expanded', async () => {
      render(<HostConfigSection {...defaultProps} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/number of hosts/i)).toBeInTheDocument();
    });

    it('displays current host count value', async () => {
      render(<HostConfigSection {...defaultProps} hostCount={5} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/number of hosts/i)).toHaveValue(5);
    });

    it('calls setHostCount when value changes', async () => {
      const setHostCount = vi.fn();
      render(<HostConfigSection {...defaultProps} setHostCount={setHostCount} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      const input = screen.getByLabelText(/number of hosts/i);
      await userEvent.clear(input);
      await userEvent.type(input, '6');

      expect(setHostCount).toHaveBeenCalled();
    });
  });

  describe('Cores Per Host Input', () => {
    it('renders cores per host input when expanded', async () => {
      render(<HostConfigSection {...defaultProps} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/cores per host/i)).toBeInTheDocument();
    });

    it('displays current cores value', async () => {
      render(<HostConfigSection {...defaultProps} coresPerHost={64} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/cores per host/i)).toHaveValue(64);
    });

    it('calls setCoresPerHost when value changes', async () => {
      const setCoresPerHost = vi.fn();
      render(<HostConfigSection {...defaultProps} setCoresPerHost={setCoresPerHost} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      const input = screen.getByLabelText(/cores per host/i);
      await userEvent.clear(input);
      await userEvent.type(input, '96');

      expect(setCoresPerHost).toHaveBeenCalled();
    });
  });

  describe('Memory Per Host Input', () => {
    it('renders memory per host input when expanded', async () => {
      render(<HostConfigSection {...defaultProps} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/memory per host/i)).toBeInTheDocument();
    });

    it('displays current memory value', async () => {
      render(<HostConfigSection {...defaultProps} memoryPerHost={768} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/memory per host/i)).toHaveValue(768);
    });

    it('calls setMemoryPerHost when value changes', async () => {
      const setMemoryPerHost = vi.fn();
      render(<HostConfigSection {...defaultProps} setMemoryPerHost={setMemoryPerHost} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      const input = screen.getByLabelText(/memory per host/i);
      await userEvent.clear(input);
      await userEvent.type(input, '1024');

      expect(setMemoryPerHost).toHaveBeenCalled();
    });
  });

  describe('HA Admission Control Input', () => {
    it('renders HA admission control input when expanded', async () => {
      render(<HostConfigSection {...defaultProps} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/ha admission/i)).toBeInTheDocument();
    });

    it('displays current HA percentage value', async () => {
      render(<HostConfigSection {...defaultProps} haAdmissionPct={33} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByLabelText(/ha admission/i)).toHaveValue(33);
    });

    it('calls setHaAdmissionPct when value changes', async () => {
      const setHaAdmissionPct = vi.fn();
      render(<HostConfigSection {...defaultProps} setHaAdmissionPct={setHaAdmissionPct} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      const input = screen.getByLabelText(/ha admission/i);
      await userEvent.clear(input);
      await userEvent.type(input, '50');

      expect(setHaAdmissionPct).toHaveBeenCalled();
    });

    it('shows HA help text', async () => {
      render(<HostConfigSection {...defaultProps} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      expect(screen.getByText(/capacity reserved for ha/i)).toBeInTheDocument();
    });
  });

  describe('Summary Display', () => {
    it('shows total capacity summary when expanded', async () => {
      render(<HostConfigSection {...defaultProps} hostCount={4} coresPerHost={32} memoryPerHost={512} />);
      await userEvent.click(screen.getByRole('button', { name: /host configuration/i }));

      // 4 hosts * 32 cores = 128 total cores
      expect(screen.getByText(/128/)).toBeInTheDocument();
      // 4 hosts * 512 GB = 2048 GB = 2.0 TB
      expect(screen.getByText(/2048/)).toBeInTheDocument();
    });
  });

  describe('Start Expanded Option', () => {
    it('can start expanded when defaultExpanded is true', () => {
      render(<HostConfigSection {...defaultProps} defaultExpanded={true} />);
      expect(screen.getByLabelText(/number of hosts/i)).toBeInTheDocument();
    });
  });
});
