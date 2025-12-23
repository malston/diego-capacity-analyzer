// ABOUTME: Key metrics display cards for TAS Capacity Analyzer dashboard
// ABOUTME: Shows total cells, utilization, CPU, and unused memory with tooltips

import { Server, Activity, TrendingUp, AlertTriangle } from 'lucide-react';
import Tooltip from './Tooltip';

const TOOLTIPS = {
  totalCells: "Number of Diego cells (VMs that run app containers). More cells = more capacity for workloads.",
  utilization: "Percentage of total memory actively consumed by running apps. Below 50% = consolidation opportunity. Above 80% = running hot.",
  avgCpu: "Average processor load across all cells. Sustained >70% means apps are competing for CPU cycles.",
  unusedMemory: "Memory apps reserved but aren't actually using. May be an optimization opportunity for right-sizing.",
};

const MetricCards = ({ metrics }) => {
  const cards = [
    {
      id: 'totalCells',
      label: 'Total Cells',
      value: metrics.totalCells,
      subtext: `${(metrics.totalMemory / 1024).toFixed(1)} GB capacity`,
      icon: Server,
      iconColor: 'text-blue-400',
    },
    {
      id: 'utilization',
      label: 'Utilization',
      value: `${metrics.utilizationPercent.toFixed(1)}%`,
      subtext: `${(metrics.totalUsed / 1024).toFixed(1)} GB / ${(metrics.totalMemory / 1024).toFixed(1)} GB`,
      icon: Activity,
      iconColor: 'text-emerald-400',
    },
    {
      id: 'avgCpu',
      label: 'Avg CPU',
      value: `${metrics.avgCpu.toFixed(1)}%`,
      subtext: 'Across all cells',
      icon: TrendingUp,
      iconColor: 'text-amber-400',
    },
    {
      id: 'unusedMemory',
      label: 'Unused Memory',
      value: `${(metrics.unusedMemory / 1024).toFixed(1)} GB`,
      subtext: `${metrics.unusedPercent.toFixed(1)}% over-allocated`,
      icon: AlertTriangle,
      iconColor: 'text-amber-400',
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      {cards.map(({ id, label, value, subtext, icon: Icon, iconColor }) => (
        <div key={id} className="metric-card p-6 rounded-xl">
          <div className="flex items-center justify-between mb-2">
            <Tooltip text={TOOLTIPS[id]} position="bottom" showIcon>
              <span className="text-slate-400 text-sm uppercase tracking-wide">{label}</span>
            </Tooltip>
            <Icon className={`w-5 h-5 ${iconColor}`} aria-hidden="true" />
          </div>
          <div className="text-3xl font-bold text-white mb-1">{value}</div>
          <div className="text-xs text-slate-400">{subtext}</div>
        </div>
      ))}
    </div>
  );
};

export default MetricCards;
