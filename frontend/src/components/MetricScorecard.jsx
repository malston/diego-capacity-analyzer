// ABOUTME: Scorecard component for displaying individual metrics
// ABOUTME: Shows value with change indicator and status bar

import React from 'react';
import { TrendingUp, TrendingDown, Minus, ArrowRight } from 'lucide-react';

const MetricScorecard = ({
  label,
  currentValue,
  proposedValue,
  format = (v) => v,
  unit = '',
  inverse = false, // If true, lower is better
  thresholds = { warning: 75, critical: 85 },
  showBar = false,
  max = 100,
}) => {
  const change = proposedValue - currentValue;
  const changePercent = currentValue !== 0 ? (change / currentValue) * 100 : 0;

  // Determine if change is positive (improvement)
  const isImprovement = inverse ? change < 0 : change > 0;
  const isNeutral = Math.abs(change) < 0.1;

  // Determine status based on proposed value
  let status = 'good';
  let statusColor = '#06b6d4'; // cyan
  let barColor = 'bg-cyan-500';

  const checkValue = inverse ? proposedValue : proposedValue;
  if (checkValue >= thresholds.critical) {
    status = 'critical';
    statusColor = '#ef4444';
    barColor = 'bg-red-500';
  } else if (checkValue >= thresholds.warning) {
    status = 'warning';
    statusColor = '#f59e0b';
    barColor = 'bg-amber-500';
  }

  // For utilization metrics, we need to invert the threshold check
  if (inverse) {
    if (proposedValue >= thresholds.critical) {
      status = 'critical';
      statusColor = '#ef4444';
      barColor = 'bg-red-500';
    } else if (proposedValue >= thresholds.warning) {
      status = 'warning';
      statusColor = '#f59e0b';
      barColor = 'bg-amber-500';
    }
  }

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-lg p-4 border border-slate-700/50 hover:border-slate-600/50 transition-colors">
      {/* Label */}
      <div className="text-xs uppercase tracking-wider text-gray-400 font-medium mb-3">
        {label}
      </div>

      {/* Values row */}
      <div className="flex items-center justify-between gap-3 mb-3">
        {/* Current */}
        <div className="flex-1">
          <div className="text-xs text-gray-500 mb-1">Current</div>
          <div className="text-lg font-mono text-gray-300">
            {format(currentValue)}{unit}
          </div>
        </div>

        {/* Arrow */}
        <ArrowRight className="text-gray-600 flex-shrink-0" size={16} />

        {/* Proposed */}
        <div className="flex-1 text-right">
          <div className="text-xs text-gray-500 mb-1">Proposed</div>
          <div className="text-lg font-mono font-semibold" style={{ color: statusColor }}>
            {format(proposedValue)}{unit}
          </div>
        </div>
      </div>

      {/* Change indicator */}
      <div className="flex items-center justify-between">
        <div className={`flex items-center gap-1 text-sm font-mono ${
          isNeutral ? 'text-gray-500' :
          isImprovement ? 'text-emerald-400' : 'text-red-400'
        }`}>
          {isNeutral ? (
            <Minus size={14} />
          ) : isImprovement ? (
            <TrendingUp size={14} />
          ) : (
            <TrendingDown size={14} />
          )}
          <span>
            {change > 0 ? '+' : ''}{format(change)}{unit}
          </span>
          {!isNeutral && Math.abs(changePercent) >= 0.1 && (
            <span className="text-gray-500 text-xs">
              ({changePercent > 0 ? '+' : ''}{changePercent.toFixed(0)}%)
            </span>
          )}
        </div>

        {/* Status badge */}
        <div className={`px-2 py-0.5 rounded text-xs font-medium uppercase tracking-wide ${
          status === 'good' ? 'bg-cyan-900/50 text-cyan-400' :
          status === 'warning' ? 'bg-amber-900/50 text-amber-400' :
          'bg-red-900/50 text-red-400'
        }`}>
          {status}
        </div>
      </div>

      {/* Optional progress bar */}
      {showBar && (
        <div className="mt-3">
          <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
            <div
              className={`h-full ${barColor} rounded-full transition-all duration-700 ease-out`}
              style={{ width: `${Math.min((proposedValue / max) * 100, 100)}%` }}
            />
          </div>
          <div className="flex justify-between mt-1 text-xs text-gray-500 font-mono">
            <span>0</span>
            <span>{max}{unit}</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default MetricScorecard;
