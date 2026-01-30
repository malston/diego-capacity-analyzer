// ABOUTME: Tests for host analysis display component
// ABOUTME: Covers host metrics, utilization, VMs per host, and HA status

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import HostAnalysisCard from "./HostAnalysisCard";

describe("HostAnalysisCard", () => {
  const defaultProps = {
    hostCount: 4,
    coresPerHost: 32,
    memoryPerHost: 512,
    totalCells: 40,
    haAdmissionPct: 25,
    memoryUtilization: 65,
    cpuUtilization: 45,
  };

  describe("Host Metrics Display", () => {
    it("renders host count", () => {
      render(<HostAnalysisCard {...defaultProps} />);
      expect(screen.getByText("4")).toBeInTheDocument();
      // Use getAllByText since tooltip text may also contain "hosts"
      expect(screen.getAllByText(/hosts/i).length).toBeGreaterThan(0);
    });

    it("renders total cores", () => {
      render(
        <HostAnalysisCard {...defaultProps} hostCount={4} coresPerHost={32} />,
      );
      // 4 * 32 = 128 total cores
      expect(screen.getByText("128")).toBeInTheDocument();
    });

    it("renders total memory", () => {
      render(
        <HostAnalysisCard
          {...defaultProps}
          hostCount={4}
          memoryPerHost={512}
        />,
      );
      // 4 * 512 = 2048 GB = 2T
      expect(screen.getByText(/2\.0T/)).toBeInTheDocument();
    });
  });

  describe("VMs Per Host Calculation", () => {
    it("displays VMs per host", () => {
      render(
        <HostAnalysisCard {...defaultProps} hostCount={4} totalCells={40} />,
      );
      // 40 cells / 4 hosts = 10.0 VMs per host
      expect(screen.getByText("10.0")).toBeInTheDocument();
      expect(screen.getByText(/vms.*host/i)).toBeInTheDocument();
    });

    it("handles uneven distribution", () => {
      render(
        <HostAnalysisCard {...defaultProps} hostCount={3} totalCells={10} />,
      );
      // 10 cells / 3 hosts = 3.33 VMs per host (shows 3.3)
      expect(screen.getByText(/3\.3/)).toBeInTheDocument();
    });
  });

  describe("HA Capacity Status", () => {
    it("displays HA admission control percentage", () => {
      render(<HostAnalysisCard {...defaultProps} haAdmissionPct={25} />);
      expect(screen.getByText(/25%/)).toBeInTheDocument();
      // Use getAllByText since tooltip text may also contain "HA"
      expect(screen.getAllByText(/ha/i).length).toBeGreaterThan(0);
    });

    it("shows N-1 survivable status when HA is sufficient", () => {
      // With 25% HA reservation and 4 hosts, can survive 1 host failure
      render(
        <HostAnalysisCard
          {...defaultProps}
          hostCount={4}
          haAdmissionPct={25}
        />,
      );
      // Use getAllByText since tooltip text may also contain "N-1"
      expect(screen.getAllByText(/n-1/i).length).toBeGreaterThan(0);
    });

    it("shows warning when HA is insufficient", () => {
      // With only 10% reservation on 4 hosts, may not survive host failure
      render(
        <HostAnalysisCard
          {...defaultProps}
          hostCount={4}
          haAdmissionPct={10}
        />,
      );
      // Use getAllByText since tooltip text may also contain "Risk"
      expect(screen.getAllByText(/warning|risk/i).length).toBeGreaterThan(0);
    });

    it("calculates hosts survivable based on HA percentage", () => {
      // 50% HA = can survive 2 host failures on 4 hosts
      render(
        <HostAnalysisCard
          {...defaultProps}
          hostCount={4}
          haAdmissionPct={50}
        />,
      );
      // Use getAllByText since tooltip text may also contain "N-2"
      expect(screen.getAllByText(/n-2/i).length).toBeGreaterThan(0);
    });
  });

  describe("Utilization Display", () => {
    it("displays memory utilization", () => {
      render(<HostAnalysisCard {...defaultProps} memoryUtilization={65} />);
      expect(screen.getByText(/65%/)).toBeInTheDocument();
    });

    it("displays CPU utilization", () => {
      render(<HostAnalysisCard {...defaultProps} cpuUtilization={45} />);
      expect(screen.getByText(/45%/)).toBeInTheDocument();
    });

    it("shows warning status for high utilization", () => {
      const { container } = render(
        <HostAnalysisCard {...defaultProps} memoryUtilization={85} />,
      );
      // High utilization should trigger warning styling
      expect(container.querySelector(".text-amber-400")).toBeInTheDocument();
    });

    it("shows critical status for very high utilization", () => {
      const { container } = render(
        <HostAnalysisCard {...defaultProps} memoryUtilization={95} />,
      );
      // Very high utilization should trigger critical styling
      expect(container.querySelector(".text-red-400")).toBeInTheDocument();
    });
  });

  describe("Section Header", () => {
    it("renders section title with icon", () => {
      render(<HostAnalysisCard {...defaultProps} />);
      expect(screen.getByText(/host analysis/i)).toBeInTheDocument();
    });
  });

  describe("Edge Cases", () => {
    it("handles zero hosts gracefully", () => {
      render(<HostAnalysisCard {...defaultProps} hostCount={0} />);
      // Should show N/A for hosts
      expect(screen.getByText("N/A")).toBeInTheDocument();
    });

    it("handles zero cells gracefully", () => {
      render(<HostAnalysisCard {...defaultProps} totalCells={0} />);
      // VMs per host should show 0
      expect(screen.getByText("0.0")).toBeInTheDocument();
    });
  });
});
