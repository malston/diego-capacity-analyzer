// frontend/src/components/ComparisonTable.jsx
// ABOUTME: Side-by-side comparison table for current vs proposed scenarios
// ABOUTME: Shows metrics with change indicators

import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

const formatNumber = (num) => {
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toFixed(1);
};

const formatGB = (gb) => {
  if (gb >= 1000) return `${(gb / 1000).toFixed(1)} TB`;
  return `${gb} GB`;
};

const ChangeIndicator = ({ current, proposed, inverse = false }) => {
  const diff = proposed - current;
  if (Math.abs(diff) < 0.1) {
    return <span className="text-slate-500"><Minus size={16} /></span>;
  }
  const isPositive = inverse ? diff < 0 : diff > 0;
  return isPositive ? (
    <span className="text-emerald-400 flex items-center gap-1">
      <TrendingUp size={16} />
      {diff > 0 ? '+' : ''}{formatNumber(diff)}
    </span>
  ) : (
    <span className="text-red-400 flex items-center gap-1">
      <TrendingDown size={16} />
      {diff > 0 ? '+' : ''}{formatNumber(diff)}
    </span>
  );
};

const ComparisonTable = ({ comparison }) => {
  if (!comparison) return null;

  const { current, proposed } = comparison;

  const metrics = [
    {
      label: 'Cell Count',
      current: current.cell_count,
      proposed: proposed.cell_count,
      format: (v) => v,
      inverse: false,
    },
    {
      label: 'Cell Size',
      current: `${current.cell_cpu}×${current.cell_memory_gb}`,
      proposed: `${proposed.cell_cpu}×${proposed.cell_memory_gb}`,
      noChange: true,
    },
    {
      label: 'App Capacity',
      current: current.app_capacity_gb,
      proposed: proposed.app_capacity_gb,
      format: formatGB,
      inverse: false,
    },
    {
      label: 'Utilization',
      current: current.utilization_pct,
      proposed: proposed.utilization_pct,
      format: (v) => `${v.toFixed(1)}%`,
      inverse: true, // Lower is better
    },
    {
      label: 'Free Chunks',
      current: current.free_chunks,
      proposed: proposed.free_chunks,
      format: (v) => v,
      inverse: false,
    },
    {
      label: 'N-1 Utilization',
      current: current.n1_utilization_pct,
      proposed: proposed.n1_utilization_pct,
      format: (v) => `${v.toFixed(1)}%`,
      inverse: true, // Lower is better
    },
    {
      label: 'Fault Impact',
      current: current.fault_impact,
      proposed: proposed.fault_impact,
      format: (v) => `${v} apps/cell`,
      inverse: true, // Lower is better
    },
  ];

  return (
    <div className="bg-slate-900/50 rounded-lg border border-slate-700/50 overflow-hidden">
      <table className="w-full">
        <thead className="bg-slate-800/50">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold text-slate-300">
              Metric
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-slate-300">
              Current ({current.cell_cpu}×{current.cell_memory_gb})
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-slate-300">
              Proposed ({proposed.cell_cpu}×{proposed.cell_memory_gb})
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-slate-300">
              Change
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          {metrics.map((m) => (
            <tr key={m.label} className="hover:bg-slate-800/30 transition-colors">
              <td className="px-4 py-3 text-sm text-slate-200">{m.label}</td>
              <td className="px-4 py-3 text-sm text-right text-slate-200">
                {m.format ? m.format(m.current) : m.current}
              </td>
              <td className="px-4 py-3 text-sm text-right text-slate-200">
                {m.format ? m.format(m.proposed) : m.proposed}
              </td>
              <td className="px-4 py-3 text-sm text-right">
                {m.noChange ? (
                  <span className="text-slate-500">—</span>
                ) : (
                  <ChangeIndicator
                    current={m.current}
                    proposed={m.proposed}
                    inverse={m.inverse}
                  />
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default ComparisonTable;
