// ABOUTME: Tests for CPU ratio gauge in ScenarioResults component
// ABOUTME: Verifies gauge renders when CPU selected and hides when not selected

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ScenarioResults from './ScenarioResults';

describe('ScenarioResults CPU Gauge', () => {
  const baseComparison = {
    current: {
      cell_count: 10,
      cell_memory_gb: 32,
      cell_cpu: 4,
      app_capacity_gb: 298,
      utilization_pct: 50,
      n1_utilization_pct: 60,
      free_chunks: 100,
      blast_radius_pct: 5,
      instances_per_cell: 5,
      fault_impact: 10,
      estimated_tps: 0,
      tps_status: 'disabled',
    },
    proposed: {
      cell_count: 20,
      cell_memory_gb: 32,
      cell_cpu: 4,
      app_capacity_gb: 596,
      utilization_pct: 25,
      n1_utilization_pct: 70,
      free_chunks: 200,
      blast_radius_pct: 2.5,
      instances_per_cell: 2.5,
      fault_impact: 5,
      estimated_tps: 0,
      tps_status: 'disabled',
      total_vcpus: 80,
      total_pcpus: 96,
      vcpu_ratio: 0.83,
      cpu_risk_level: 'conservative',
    },
    delta: {
      capacity_change_gb: 298,
      utilization_change_pct: -25,
      resilience_change: 'improved',
    },
    warnings: [],
  };

  it('displays CPU ratio gauge when cpu selected and data available', () => {
    render(
      <ScenarioResults
        comparison={baseComparison}
        warnings={[]}
        selectedResources={['memory', 'cpu']}
      />
    );

    // Should display the ratio (rounded to 1 decimal)
    expect(screen.getByText('0.8:1')).toBeInTheDocument();
    // Should show risk level badge
    expect(screen.getByText('conservative')).toBeInTheDocument();
    // Should show the vCPU:pCPU breakdown
    expect(screen.getByText(/80.*vCPU/)).toBeInTheDocument();
  });

  it('hides CPU gauge when cpu not in selectedResources', () => {
    render(
      <ScenarioResults
        comparison={baseComparison}
        warnings={[]}
        selectedResources={['memory']}
      />
    );

    // Should NOT display vCPU:pCPU Ratio label
    expect(screen.queryByText('vCPU:pCPU Ratio')).not.toBeInTheDocument();
  });

  it('displays moderate risk level with amber color', () => {
    const moderateComparison = {
      ...baseComparison,
      proposed: {
        ...baseComparison.proposed,
        vcpu_ratio: 5.5,
        cpu_risk_level: 'moderate',
      },
    };

    render(
      <ScenarioResults
        comparison={moderateComparison}
        warnings={[]}
        selectedResources={['memory', 'cpu']}
      />
    );

    // Should display the ratio
    expect(screen.getByText('5.5:1')).toBeInTheDocument();
    // Should show moderate badge
    expect(screen.getByText('moderate')).toBeInTheDocument();
  });

  it('displays aggressive risk level with red color', () => {
    const aggressiveComparison = {
      ...baseComparison,
      proposed: {
        ...baseComparison.proposed,
        vcpu_ratio: 10.2,
        cpu_risk_level: 'aggressive',
      },
    };

    render(
      <ScenarioResults
        comparison={aggressiveComparison}
        warnings={[]}
        selectedResources={['memory', 'cpu']}
      />
    );

    // Should display the ratio
    expect(screen.getByText('10.2:1')).toBeInTheDocument();
    // Should show aggressive badge
    expect(screen.getByText('aggressive')).toBeInTheDocument();
  });

  it('hides CPU gauge when total_pcpus is 0', () => {
    const noPcpuComparison = {
      ...baseComparison,
      proposed: {
        ...baseComparison.proposed,
        total_pcpus: 0,
      },
    };

    render(
      <ScenarioResults
        comparison={noPcpuComparison}
        warnings={[]}
        selectedResources={['memory', 'cpu']}
      />
    );

    // Should NOT display vCPU:pCPU Ratio label when no pCPU data
    expect(screen.queryByText('vCPU:pCPU Ratio')).not.toBeInTheDocument();
  });
});
