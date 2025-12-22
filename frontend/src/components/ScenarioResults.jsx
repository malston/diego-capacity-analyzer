// ABOUTME: Complete scenario results dashboard with scorecards and gauges
// ABOUTME: Replaces basic ComparisonTable with visual capacity analysis

import React from 'react';
import { CheckCircle2, XCircle, AlertTriangle, Zap, HardDrive, Cpu, Server, Shield, Activity, Database, Gauge } from 'lucide-react';
import CapacityGauge from './CapacityGauge';
import MetricScorecard from './MetricScorecard';
import Tooltip from './Tooltip';
import { TPS_STATUS_COLORS, TPS_STATUS_BG_COLORS } from '../config/resourceConfig';

const TOOLTIPS = {
  n1Capacity: "Utilization if you lose one cell (host failure scenario). Below 75% = safe headroom. Above 85% = cannot survive cell loss.",
  memoryUtilization: "App memory divided by total cell capacity. Below 80% = healthy headroom. Above 90% = near capacity exhaustion.",
  diskUtilization: "App disk usage divided by total cell disk capacity. Same thresholds as memory.",
  stagingCapacity: "Available 4GB chunks for staging new apps. When you cf push, Diego needs a 4GB chunk to build your app. Low chunks = deployment queues.",
  tps: "Tasks Per Second - how fast Diego's scheduler can place app instances. Higher = faster deploys and scaling.",
  tpsStatus: "Optimal (≥80% of peak): scheduler performing well. Degraded (50-79%): noticeable slowdown. Critical (<50%): severe delays.",
  cellCount: "Number of Diego cell VMs running your apps.",
  appCapacity: "Total memory available for apps after system overhead.",
  faultImpact: "Average app instances displaced if one cell fails. Lower = smaller blast radius.",
  instancesPerCell: "Average app instances per cell. Lower = more distributed workload.",
};

const ScenarioResults = ({ comparison, warnings = [], selectedResources = ['memory'] }) => {
  if (!comparison) return null;

  const { current, proposed, delta } = comparison;

  // Check for over-capacity (utilization > 100%)
  const isOverCapacity = proposed.utilization_pct > 100;

  // Overall status
  const criticalCount = warnings.filter(w => w.severity === 'critical').length;
  const warningCount = warnings.filter(w => w.severity === 'warning').length;

  let overallStatus = 'good';
  let statusMessage = 'Configuration looks healthy';
  let StatusIcon = CheckCircle2;
  let statusColor = 'text-emerald-400';
  let statusBg = 'bg-emerald-900/20 border-emerald-700/30';
  let statusAnswer = '✓ YES';

  // Over capacity is the most critical issue
  if (isOverCapacity) {
    overallStatus = 'critical';
    statusMessage = `Insufficient capacity - needs ${(proposed.utilization_pct - 100).toFixed(0)}% more space`;
    StatusIcon = XCircle;
    statusColor = 'text-red-400';
    statusBg = 'bg-red-900/30 border-red-600/50';
    statusAnswer = '✗ NO';
  } else if (criticalCount > 0) {
    overallStatus = 'critical';
    statusMessage = `${criticalCount} critical issue${criticalCount > 1 ? 's' : ''} detected`;
    StatusIcon = XCircle;
    statusColor = 'text-red-400';
    statusBg = 'bg-red-900/20 border-red-700/30';
    statusAnswer = '✗ NO';
  } else if (warningCount > 0) {
    overallStatus = 'warning';
    statusMessage = `${warningCount} warning${warningCount > 1 ? 's' : ''} to review`;
    StatusIcon = AlertTriangle;
    statusColor = 'text-amber-400';
    statusBg = 'bg-amber-900/20 border-amber-700/30';
    statusAnswer = '⚠ MAYBE';
  }

  // Format helpers
  const formatGB = (gb) => gb >= 1000 ? `${(gb / 1000).toFixed(1)}T` : `${gb}G`;
  const formatNum = (n) => n >= 1000 ? `${(n / 1000).toFixed(1)}K` : n.toString();

  return (
    <div className="space-y-6">
      {/* Overall Status Banner */}
      <div className={`${statusBg} border rounded-xl p-4 flex items-center justify-between`}>
        <div className="flex items-center gap-3">
          <StatusIcon className={`${statusColor}`} size={24} />
          <div>
            <div className={`font-semibold ${statusColor}`}>
              {overallStatus === 'good' ? 'Will It Fit?' : 'Capacity Check'}
            </div>
            <div className="text-sm text-gray-400">{statusMessage}</div>
          </div>
        </div>
        <div className={`text-3xl font-mono font-bold ${statusColor}`}>
          {statusAnswer}
        </div>
      </div>

      {/* Key Gauges Row */}
      <div className={`grid gap-6 ${
        selectedResources.includes('disk') && proposed.disk_capacity_gb > 0
          ? 'grid-cols-2 lg:grid-cols-4'
          : 'grid-cols-3'
      }`}>
        {/* N-1 Utilization Gauge */}
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Shield size={16} />
            <Tooltip text={TOOLTIPS.n1Capacity} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">N-1 Capacity</span>
            </Tooltip>
          </div>
          <CapacityGauge
            value={proposed.n1_utilization_pct}
            label="Utilization"
            thresholds={{ warning: 75, critical: 85 }}
            inverse={true}
          />
          <div className="mt-4 text-center text-xs text-gray-500">
            Safe under host failure
          </div>
        </div>

        {/* Cell Utilization Gauge */}
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Activity size={16} />
            <Tooltip text={TOOLTIPS.memoryUtilization} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">Memory Utilization</span>
            </Tooltip>
          </div>
          <CapacityGauge
            value={proposed.utilization_pct}
            label="Memory Used"
            thresholds={{ warning: 80, critical: 90 }}
            inverse={true}
          />
          <div className="mt-4 text-center text-xs text-gray-500">
            App memory / capacity
          </div>
        </div>

        {/* Disk Utilization Gauge - only if disk selected and has data */}
        {selectedResources.includes('disk') && proposed.disk_capacity_gb > 0 && (
          <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center gap-2 mb-4 text-gray-400">
              <Database size={16} />
              <Tooltip text={TOOLTIPS.diskUtilization} position="bottom" showIcon>
                <span className="text-xs uppercase tracking-wider font-medium">Disk Utilization</span>
              </Tooltip>
            </div>
            <CapacityGauge
              value={proposed.disk_utilization_pct}
              label="Disk Used"
              thresholds={{ warning: 80, critical: 90 }}
              inverse={true}
            />
            <div className="mt-4 text-center text-xs text-gray-500">
              App disk / capacity
            </div>
          </div>
        )}

        {/* Free Chunks Gauge */}
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Zap size={16} />
            <Tooltip text={TOOLTIPS.stagingCapacity} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">Staging Capacity</span>
            </Tooltip>
          </div>
          <CapacityGauge
            value={Math.min((proposed.free_chunks / 800) * 100, 100)}
            label={`${formatNum(proposed.free_chunks)} chunks`}
            thresholds={{ warning: 50, critical: 25 }}
            inverse={false}
            suffix=""
          />
          <div className="mt-4 text-center text-xs text-gray-500">
            Available 4GB chunks
          </div>
        </div>
      </div>

      {/* TPS Performance Indicator */}
      {proposed.estimated_tps > 0 && (
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Gauge size={16} />
            <Tooltip text={TOOLTIPS.tps} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">Scheduling Performance (TPS)</span>
            </Tooltip>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-6">
              {/* Current TPS */}
              <div className="text-center">
                <div className="text-xs text-gray-500 mb-1">Current</div>
                <div className="text-2xl font-mono font-bold text-gray-300">
                  {current.estimated_tps.toLocaleString()}
                </div>
                <div className={`text-xs mt-1 px-2 py-0.5 rounded ${TPS_STATUS_BG_COLORS[current.tps_status] || 'bg-gray-500/20'} ${TPS_STATUS_COLORS[current.tps_status] || 'text-gray-400'}`}>
                  {current.tps_status}
                </div>
              </div>

              {/* Arrow */}
              <div className="text-3xl text-cyan-500">→</div>

              {/* Proposed TPS */}
              <div className="text-center">
                <div className="text-xs text-gray-500 mb-1">Proposed</div>
                <div className={`text-2xl font-mono font-bold ${TPS_STATUS_COLORS[proposed.tps_status] || 'text-gray-300'}`}>
                  {proposed.estimated_tps.toLocaleString()}
                </div>
                <div className={`text-xs mt-1 px-2 py-0.5 rounded ${TPS_STATUS_BG_COLORS[proposed.tps_status] || 'bg-gray-500/20'} ${TPS_STATUS_COLORS[proposed.tps_status] || 'text-gray-400'}`}>
                  {proposed.tps_status}
                </div>
              </div>
            </div>

            {/* TPS Change Indicator */}
            <div className="text-right">
              <div className="text-xs text-gray-500 mb-1">Change</div>
              <div className={`text-xl font-mono font-bold ${
                proposed.estimated_tps >= current.estimated_tps ? 'text-emerald-400' : 'text-amber-400'
              }`}>
                {proposed.estimated_tps >= current.estimated_tps ? '+' : ''}
                {((proposed.estimated_tps - current.estimated_tps) / Math.max(1, current.estimated_tps) * 100).toFixed(0)}%
              </div>
              <div className="text-xs text-gray-500">
                {proposed.estimated_tps - current.estimated_tps >= 0 ? '+' : ''}
                {(proposed.estimated_tps - current.estimated_tps).toLocaleString()} TPS
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Detailed Metrics Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricScorecard
          label="Cell Count"
          currentValue={current.cell_count}
          proposedValue={proposed.cell_count}
          icon={Server}
          inverse={false}
          thresholds={{ warning: 0, critical: 0 }}
          tooltip={TOOLTIPS.cellCount}
        />
        <MetricScorecard
          label="App Capacity"
          currentValue={current.app_capacity_gb}
          proposedValue={proposed.app_capacity_gb}
          format={formatGB}
          unit="B"
          inverse={false}
          thresholds={{ warning: 0, critical: 0 }}
          tooltip={TOOLTIPS.appCapacity}
        />
        <MetricScorecard
          label="Fault Impact"
          currentValue={current.fault_impact}
          proposedValue={proposed.fault_impact}
          unit=" apps/cell"
          inverse={true}
          thresholds={{ warning: 25, critical: 50 }}
          tooltip={TOOLTIPS.faultImpact}
        />
        <MetricScorecard
          label="Instances/Cell"
          currentValue={current.instances_per_cell}
          proposedValue={proposed.instances_per_cell}
          format={(v) => v.toFixed(1)}
          inverse={true}
          thresholds={{ warning: 30, critical: 50 }}
          tooltip={TOOLTIPS.instancesPerCell}
        />
      </div>

      {/* Cell Size Comparison */}
      <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center gap-2 mb-4 text-gray-400">
          <Cpu size={16} />
          <span className="text-xs uppercase tracking-wider font-medium">Cell Configuration Change</span>
        </div>
        <div className="flex items-center justify-center gap-8">
          {/* Current */}
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-2">CURRENT</div>
            <div className="bg-slate-700/50 rounded-lg px-6 py-4 border border-slate-600/50">
              <div className="text-3xl font-mono font-bold text-gray-300">
                {current.cell_cpu} <span className="text-gray-500">×</span> {current.cell_memory_gb}
              </div>
              <div className="text-xs text-gray-500 mt-1">vCPU × GB</div>
            </div>
            <div className="mt-2 text-sm text-gray-400">
              {formatNum(current.cell_count)} cells
            </div>
          </div>

          {/* Arrow */}
          <div className="flex flex-col items-center">
            <div className="text-4xl text-cyan-500">→</div>
            <div className={`text-xs mt-1 ${
              delta.redundancy_change === 'improved' ? 'text-emerald-400' :
              delta.redundancy_change === 'reduced' ? 'text-amber-400' :
              'text-gray-500'
            }`}>
              {delta.redundancy_change === 'improved' ? '↑ More redundant' :
               delta.redundancy_change === 'reduced' ? '↓ Less redundant' :
               'No change'}
            </div>
          </div>

          {/* Proposed */}
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-2">PROPOSED</div>
            <div className="bg-cyan-900/20 rounded-lg px-6 py-4 border border-cyan-700/30">
              <div className="text-3xl font-mono font-bold text-cyan-400">
                {proposed.cell_cpu} <span className="text-cyan-600">×</span> {proposed.cell_memory_gb}
              </div>
              <div className="text-xs text-cyan-600 mt-1">vCPU × GB</div>
            </div>
            <div className="mt-2 text-sm text-gray-400">
              {formatNum(proposed.cell_count)} cells
            </div>
          </div>
        </div>

        {/* Capacity Change Summary */}
        <div className="mt-6 pt-4 border-t border-slate-700/50 flex justify-center gap-8">
          <div className="text-center">
            <div className="text-xs text-gray-500">Capacity Change</div>
            <div className={`text-lg font-mono font-bold ${
              delta.capacity_change_gb >= 0 ? 'text-emerald-400' : 'text-red-400'
            }`}>
              {delta.capacity_change_gb >= 0 ? '+' : ''}{formatGB(delta.capacity_change_gb)}B
            </div>
          </div>
          <div className="text-center">
            <div className="text-xs text-gray-500">Utilization Change</div>
            <div className={`text-lg font-mono font-bold ${
              delta.utilization_change_pct <= 0 ? 'text-emerald-400' : 'text-amber-400'
            }`}>
              {delta.utilization_change_pct >= 0 ? '+' : ''}{delta.utilization_change_pct.toFixed(1)}%
            </div>
          </div>
        </div>
      </div>

      {/* Warnings Section */}
      {warnings.length > 0 && (
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <AlertTriangle size={16} />
            <span className="text-xs uppercase tracking-wider font-medium">
              Recommendations ({warnings.length})
            </span>
          </div>
          <div className="space-y-2">
            {warnings.map((warning, index) => (
              <div
                key={index}
                className={`flex items-start gap-3 p-3 rounded-lg ${
                  warning.severity === 'critical' ? 'bg-red-900/20 border border-red-700/30' :
                  warning.severity === 'warning' ? 'bg-amber-900/20 border border-amber-700/30' :
                  'bg-blue-900/20 border border-blue-700/30'
                }`}
              >
                <div className={`mt-0.5 ${
                  warning.severity === 'critical' ? 'text-red-400' :
                  warning.severity === 'warning' ? 'text-amber-400' :
                  'text-blue-400'
                }`}>
                  {warning.severity === 'critical' ? <XCircle size={16} /> :
                   warning.severity === 'warning' ? <AlertTriangle size={16} /> :
                   <CheckCircle2 size={16} />}
                </div>
                <div>
                  <div className={`text-sm font-medium ${
                    warning.severity === 'critical' ? 'text-red-300' :
                    warning.severity === 'warning' ? 'text-amber-300' :
                    'text-blue-300'
                  }`}>
                    {warning.message}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* All Clear Message */}
      {warnings.length === 0 && (
        <div className="bg-emerald-900/20 rounded-xl p-6 border border-emerald-700/30 text-center">
          <CheckCircle2 className="text-emerald-400 mx-auto mb-2" size={32} />
          <div className="text-lg font-semibold text-emerald-400">All Checks Passed</div>
          <div className="text-sm text-emerald-300/70 mt-1">
            Proposed configuration meets all capacity and redundancy requirements
          </div>
        </div>
      )}
    </div>
  );
};

export default ScenarioResults;
