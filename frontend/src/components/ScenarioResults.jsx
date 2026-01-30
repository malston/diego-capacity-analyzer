// ABOUTME: Complete scenario results dashboard with scorecards and gauges
// ABOUTME: Replaces basic ComparisonTable with visual capacity analysis

import {
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Zap,
  Cpu,
  Server,
  Shield,
  Activity,
  Database,
  Gauge,
} from "lucide-react";
import CapacityGauge from "./CapacityGauge";
import MetricScorecard from "./MetricScorecard";
import Tooltip from "./Tooltip";
import {
  TPS_STATUS_COLORS,
  TPS_STATUS_BG_COLORS,
} from "../config/resourceConfig";

const TOOLTIPS = {
  n1Capacity:
    "Infrastructure utilization vs N-1 host capacity. If one ESXi host fails, can the remaining hosts run all Diego cell and platform VMs? Below 75% = safe. Above 85% = at risk of capacity exhaustion after host failure.",
  memoryUtilization:
    "App memory divided by total cell capacity. Below 80% = healthy headroom. Above 90% = near capacity exhaustion.",
  diskUtilization:
    "App disk usage divided by total cell disk capacity. Same thresholds as memory.",
  stagingCapacity:
    "Available chunks for staging new apps. Chunk size is auto-detected from your average app instance size, or defaults to 4GB. When you cf push, Diego needs a chunk to build your app. Low chunks = deployment queues.",
  tps: "Tasks Per Second - modeled estimate of scheduler throughput based on cell count. Not a live measurement. Actual TPS varies by infrastructure. Customize curve in Advanced Options.",
  tpsStatus:
    "Optimal (≥80% of peak): scheduler performing well. Degraded (50-79%): noticeable slowdown. Critical (<50%): severe delays.",
  cellCount: "Number of Diego cell VMs running your apps.",
  appCapacity: "Total memory available for apps after system overhead.",
  faultImpact:
    "Average app instances displaced if one cell fails. Lower = smaller blast radius.",
  instancesPerCell:
    "Average app instances per cell. Lower = more distributed workload.",
  cpuRatio:
    "vCPU:pCPU ratio measures CPU oversubscription. Conservative (≤4:1): safe for production. Moderate (4-8:1): monitor CPU Ready time. Aggressive (>8:1): expect contention.",
};

const ScenarioResults = ({
  comparison,
  warnings,
  selectedResources = ["memory"],
}) => {
  if (!comparison) return null;

  const { current, proposed, delta } = comparison;

  // Ensure warnings is always an array (backend may return null)
  const safeWarnings = warnings ?? [];

  // Check for over-capacity (utilization > 100%)
  const isOverCapacity = proposed.utilization_pct > 100;

  // Overall status
  const criticalCount = safeWarnings.filter(
    (w) => w.severity === "critical",
  ).length;
  const warningCount = safeWarnings.filter(
    (w) => w.severity === "warning",
  ).length;

  let overallStatus = "good";
  let statusMessage = "Configuration looks healthy";
  let StatusIcon = CheckCircle2;
  let statusColor = "text-emerald-400";
  let statusBg = "bg-emerald-900/20 border-emerald-700/30";
  let statusAnswer = "✓ YES";

  // Over capacity is the most critical issue
  if (isOverCapacity) {
    overallStatus = "critical";
    statusMessage = `Insufficient capacity - needs ${(proposed.utilization_pct - 100).toFixed(0)}% more space`;
    StatusIcon = XCircle;
    statusColor = "text-red-400";
    statusBg = "bg-red-900/30 border-red-600/50";
    statusAnswer = "✗ NO";
  } else if (criticalCount > 0) {
    overallStatus = "critical";
    statusMessage = `${criticalCount} critical issue${criticalCount > 1 ? "s" : ""} detected`;
    StatusIcon = XCircle;
    statusColor = "text-red-400";
    statusBg = "bg-red-900/20 border-red-700/30";
    statusAnswer = "✗ NO";
  } else if (warningCount > 0) {
    overallStatus = "warning";
    statusMessage = `${warningCount} warning${warningCount > 1 ? "s" : ""} to review`;
    StatusIcon = AlertTriangle;
    statusColor = "text-amber-400";
    statusBg = "bg-amber-900/20 border-amber-700/30";
    statusAnswer = "⚠ MAYBE";
  }

  // Format helpers
  const formatGB = (gb) =>
    gb >= 1000 ? `${(gb / 1000).toFixed(1)}T` : `${gb}G`;
  const formatNum = (n) =>
    n >= 1000 ? `${(n / 1000).toFixed(1)}K` : n.toString();
  const formatFieldName = (field) => {
    const names = {
      cell_count: "Cell count",
      cell_memory_gb: "Cell memory",
      cell_cpu: "Cell CPU",
      cell_disk_gb: "Cell disk",
      host_count: "Host count",
    };
    return names[field] || field;
  };

  return (
    <div className="space-y-6">
      {/* Overall Status Banner */}
      <div
        className={`${statusBg} border rounded-xl p-4 flex items-center justify-between`}
      >
        <div className="flex items-center gap-3">
          <StatusIcon className={`${statusColor}`} size={24} />
          <div>
            <div className={`font-semibold ${statusColor}`}>
              {overallStatus === "good" ? "Will It Fit?" : "Capacity Check"}
            </div>
            <div className="text-sm text-gray-400">{statusMessage}</div>
          </div>
        </div>
        <div className={`text-3xl font-mono font-bold ${statusColor}`}>
          {statusAnswer}
        </div>
      </div>

      {/* Constraint Callout - shows which constraint is limiting */}
      {comparison.constraints && (
        <div
          className={`rounded-xl p-4 border ${
            comparison.constraints.limiting_constraint === "ha_admission"
              ? "bg-cyan-900/20 border-cyan-700/30"
              : "bg-amber-900/20 border-amber-700/30"
          }`}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Shield
                className={`${
                  comparison.constraints.limiting_constraint === "ha_admission"
                    ? "text-cyan-400"
                    : "text-amber-400"
                }`}
                size={20}
              />
              <div>
                <div className="text-sm font-medium text-gray-200">
                  Constrained by: {comparison.constraints.limiting_label}
                </div>
                <div className="text-xs text-gray-400">
                  Reserves{" "}
                  {formatGB(
                    comparison.constraints[
                      comparison.constraints.limiting_constraint
                    ].reserved_gb,
                  )}
                  B
                  {comparison.constraints.limiting_constraint ===
                    "ha_admission" && (
                    <span>
                      {" "}
                      (equivalent to N-
                      {comparison.constraints.ha_admission.n_equivalent}{" "}
                      tolerance)
                    </span>
                  )}
                </div>
              </div>
            </div>
            <div className="text-right">
              <div className="text-lg font-mono font-bold text-gray-200">
                {formatGB(
                  comparison.constraints[
                    comparison.constraints.limiting_constraint
                  ].usable_gb,
                )}
                B
              </div>
              <div className="text-xs text-gray-500">usable capacity</div>
            </div>
          </div>
          {comparison.constraints.insufficient_ha_warning && (
            <div className="mt-3 pt-3 border-t border-amber-700/30 flex items-center gap-2 text-amber-400">
              <AlertTriangle size={14} />
              <span className="text-xs">
                HA% may be insufficient for N-1 host failure protection
              </span>
            </div>
          )}
        </div>
      )}

      {/* Key Gauges Row */}
      <div
        className={`grid gap-6 ${(() => {
          // Base gauges: N-1 Capacity + Staging Capacity = 2
          // Optional: Memory, Disk, CPU based on selectedResources
          const hasMemory = selectedResources.includes("memory");
          const hasDisk =
            selectedResources.includes("disk") && proposed.disk_capacity_gb > 0;
          const hasCpu =
            selectedResources.includes("cpu") && proposed.total_pcpus > 0;
          const count =
            2 + (hasMemory ? 1 : 0) + (hasDisk ? 1 : 0) + (hasCpu ? 1 : 0);
          if (count >= 5) return "grid-cols-2 lg:grid-cols-5";
          if (count === 4) return "grid-cols-2 lg:grid-cols-4";
          if (count === 3) return "grid-cols-3";
          return "grid-cols-2";
        })()}`}
      >
        {/* N-1 / Constraint Utilization Gauge */}
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Shield size={16} />
            <Tooltip text={TOOLTIPS.n1Capacity} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">
                {comparison.constraints?.limiting_label
                  ? `Capacity (${comparison.constraints.limiting_label})`
                  : "N-1 Capacity"}
              </span>
            </Tooltip>
          </div>
          <CapacityGauge
            value={
              comparison.constraints
                ? comparison.constraints[
                    comparison.constraints.limiting_constraint
                  ].utilization_pct
                : proposed.n1_utilization_pct
            }
            label="Utilization"
            thresholds={{ warning: 75, critical: 85 }}
            inverse={true}
          />
          <div className="mt-4 text-center text-xs text-gray-500">
            Safe under host failure
          </div>
        </div>

        {/* Cell Utilization Gauge - only if memory selected */}
        {selectedResources.includes("memory") && (
          <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center gap-2 mb-4 text-gray-400">
              <Activity size={16} />
              <Tooltip
                text={TOOLTIPS.memoryUtilization}
                position="bottom"
                showIcon
              >
                <span className="text-xs uppercase tracking-wider font-medium">
                  Memory Utilization
                </span>
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
        )}

        {/* Disk Utilization Gauge - only if disk selected and has data */}
        {selectedResources.includes("disk") &&
          proposed.disk_capacity_gb > 0 && (
            <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
              <div className="flex items-center gap-2 mb-4 text-gray-400">
                <Database size={16} />
                <Tooltip
                  text={TOOLTIPS.diskUtilization}
                  position="bottom"
                  showIcon
                >
                  <span className="text-xs uppercase tracking-wider font-medium">
                    Disk Utilization
                  </span>
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

        {/* Free Chunks - threshold-based display */}
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Zap size={16} />
            <Tooltip text={TOOLTIPS.stagingCapacity} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">
                Staging Capacity
              </span>
            </Tooltip>
          </div>
          <div className="flex flex-col items-center justify-center h-[120px]">
            <div
              className={`text-4xl font-mono font-bold ${
                proposed.free_chunks >= 20
                  ? "text-emerald-400"
                  : proposed.free_chunks >= 10
                    ? "text-amber-400"
                    : "text-red-400"
              }`}
            >
              {formatNum(proposed.free_chunks)}
            </div>
            <div className="text-sm text-gray-400 mt-2">free chunks</div>
            <div
              className={`text-xs mt-2 px-2 py-0.5 rounded ${
                proposed.free_chunks >= 20
                  ? "bg-emerald-900/30 text-emerald-400"
                  : proposed.free_chunks >= 10
                    ? "bg-amber-900/30 text-amber-400"
                    : "bg-red-900/30 text-red-400"
              }`}
            >
              {proposed.free_chunks >= 20
                ? "Healthy"
                : proposed.free_chunks >= 10
                  ? "Limited"
                  : "Constrained"}
            </div>
          </div>
          <div className="mt-4 text-center text-xs text-gray-500">
            {proposed.chunk_size_mb
              ? `${(proposed.chunk_size_mb / 1024).toFixed(1)}GB chunks for staging`
              : "4GB chunks for staging"}
          </div>
        </div>

        {/* CPU Ratio Gauge - only if cpu selected and data available */}
        {selectedResources.includes("cpu") && proposed.total_pcpus > 0 && (
          <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center gap-2 mb-4 text-gray-400">
              <Cpu size={16} />
              <Tooltip text={TOOLTIPS.cpuRatio} position="bottom" showIcon>
                <span className="text-xs uppercase tracking-wider font-medium">
                  vCPU:pCPU Ratio
                </span>
              </Tooltip>
            </div>
            <div className="flex flex-col items-center justify-center h-[120px]">
              <div
                className={`text-4xl font-mono font-bold ${
                  proposed.cpu_risk_level === "conservative"
                    ? "text-emerald-400"
                    : proposed.cpu_risk_level === "moderate"
                      ? "text-amber-400"
                      : "text-red-400"
                }`}
              >
                {proposed.vcpu_ratio.toFixed(1)}:1
              </div>
              <div className="text-sm text-gray-400 mt-2">
                {proposed.total_vcpus.toLocaleString()} vCPU /{" "}
                {proposed.total_pcpus.toLocaleString()} pCPU
              </div>
              <div
                className={`text-xs mt-2 px-2 py-0.5 rounded ${
                  proposed.cpu_risk_level === "conservative"
                    ? "bg-emerald-900/30 text-emerald-400"
                    : proposed.cpu_risk_level === "moderate"
                      ? "bg-amber-900/30 text-amber-400"
                      : "bg-red-900/30 text-red-400"
                }`}
              >
                {proposed.cpu_risk_level}
              </div>
            </div>
            {proposed?.cpu_headroom_cells !== undefined && (
              <div className="mt-2 text-center">
                <span
                  className={`text-sm font-medium ${
                    proposed.cpu_headroom_cells > 0
                      ? "text-emerald-400"
                      : proposed.cpu_headroom_cells < 0
                        ? "text-red-400"
                        : "text-gray-400"
                  }`}
                >
                  Headroom: {proposed.cpu_headroom_cells > 0 ? "+" : ""}
                  {proposed.cpu_headroom_cells} cells
                </span>
                <p className="text-xs text-gray-500 mt-1">
                  {proposed.cpu_headroom_cells >= 0
                    ? "before reaching target ratio"
                    : "over target ratio"}
                </p>
              </div>
            )}
            <div className="mt-4 text-center text-xs text-gray-500">
              Physical CPU oversubscription
            </div>
          </div>
        )}
      </div>

      {/* TPS Performance Indicator (hidden when TPS model is disabled) */}
      {proposed.estimated_tps > 0 && proposed.tps_status !== "disabled" && (
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Gauge size={16} />
            <Tooltip text={TOOLTIPS.tps} position="bottom" showIcon>
              <span className="text-xs uppercase tracking-wider font-medium">
                Scheduling Performance (TPS)
              </span>
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
                <div
                  className={`text-xs mt-1 px-2 py-0.5 rounded ${TPS_STATUS_BG_COLORS[current.tps_status] || "bg-gray-500/20"} ${TPS_STATUS_COLORS[current.tps_status] || "text-gray-400"}`}
                >
                  {current.tps_status}
                </div>
              </div>

              {/* Arrow */}
              <div className="text-3xl text-cyan-500">→</div>

              {/* Proposed TPS */}
              <div className="text-center">
                <div className="text-xs text-gray-500 mb-1">Proposed</div>
                <div
                  className={`text-2xl font-mono font-bold ${TPS_STATUS_COLORS[proposed.tps_status] || "text-gray-300"}`}
                >
                  {proposed.estimated_tps.toLocaleString()}
                </div>
                <div
                  className={`text-xs mt-1 px-2 py-0.5 rounded ${TPS_STATUS_BG_COLORS[proposed.tps_status] || "bg-gray-500/20"} ${TPS_STATUS_COLORS[proposed.tps_status] || "text-gray-400"}`}
                >
                  {proposed.tps_status}
                </div>
              </div>
            </div>

            {/* TPS Change Indicator */}
            <div className="text-right">
              <div className="text-xs text-gray-500 mb-1">Change</div>
              <div
                className={`text-xl font-mono font-bold ${
                  proposed.estimated_tps >= current.estimated_tps
                    ? "text-emerald-400"
                    : "text-amber-400"
                }`}
              >
                {proposed.estimated_tps >= current.estimated_tps ? "+" : ""}
                {(
                  ((proposed.estimated_tps - current.estimated_tps) /
                    Math.max(1, current.estimated_tps)) *
                  100
                ).toFixed(0)}
                %
              </div>
              <div className="text-xs text-gray-500">
                {proposed.estimated_tps - current.estimated_tps >= 0 ? "+" : ""}
                {(
                  proposed.estimated_tps - current.estimated_tps
                ).toLocaleString()}{" "}
                TPS
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
          <span className="text-xs uppercase tracking-wider font-medium">
            Cell Configuration Change
          </span>
        </div>
        <div className="flex items-center justify-center gap-8">
          {/* Current */}
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-2">CURRENT</div>
            <div className="bg-slate-700/50 rounded-lg px-6 py-4 border border-slate-600/50">
              <div className="text-3xl font-mono font-bold text-gray-300">
                {current.cell_cpu} <span className="text-gray-500">×</span>{" "}
                {current.cell_memory_gb}
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
            <div
              className={`text-xs mt-1 ${
                delta.resilience_change === "low"
                  ? "text-emerald-400"
                  : delta.resilience_change === "moderate"
                    ? "text-amber-400"
                    : delta.resilience_change === "high"
                      ? "text-red-400"
                      : "text-gray-500"
              }`}
            >
              {delta.resilience_change === "low"
                ? "✓ Low risk"
                : delta.resilience_change === "moderate"
                  ? "⚠ Moderate risk"
                  : delta.resilience_change === "high"
                    ? "⚠ High risk"
                    : "No change"}
            </div>
          </div>

          {/* Proposed */}
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-2">PROPOSED</div>
            <div className="bg-cyan-900/20 rounded-lg px-6 py-4 border border-cyan-700/30">
              <div className="text-3xl font-mono font-bold text-cyan-400">
                {proposed.cell_cpu} <span className="text-cyan-600">×</span>{" "}
                {proposed.cell_memory_gb}
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
            <div
              className={`text-lg font-mono font-bold ${
                delta.capacity_change_gb >= 0
                  ? "text-emerald-400"
                  : "text-red-400"
              }`}
            >
              {delta.capacity_change_gb >= 0 ? "+" : ""}
              {formatGB(delta.capacity_change_gb)}B
            </div>
          </div>
          <div className="text-center">
            <div className="text-xs text-gray-500">Utilization Change</div>
            <div
              className={`text-lg font-mono font-bold ${
                delta.utilization_change_pct <= 0
                  ? "text-emerald-400"
                  : "text-amber-400"
              }`}
            >
              {delta.utilization_change_pct >= 0 ? "+" : ""}
              {delta.utilization_change_pct.toFixed(1)}%
            </div>
          </div>
        </div>
      </div>

      {/* Capacity Constraints Section - shows max cells by resource with bottleneck */}
      {((selectedResources.includes("cpu") && proposed.max_cells_by_cpu > 0) ||
        (selectedResources.includes("memory") && comparison.constraints)) &&
        comparison.constraints && (
          <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
            <div className="flex items-center gap-2 mb-4 text-gray-400">
              <Server size={16} />
              <span className="text-xs uppercase tracking-wider font-medium">
                Maximum Deployable Cells
              </span>
            </div>

            {(() => {
              const memorySelected = selectedResources.includes("memory");
              const cpuSelected = selectedResources.includes("cpu");

              // Calculate max cells by memory from constraint analysis
              const constraint = comparison.constraints;
              const limitingConstraint =
                constraint.limiting_constraint === "ha_admission"
                  ? constraint.ha_admission
                  : constraint.n_minus_x;
              const maxCellsByMemory =
                proposed.cell_memory_gb > 0
                  ? Math.floor(
                      limitingConstraint.usable_gb / proposed.cell_memory_gb,
                    )
                  : 0;

              const maxCellsByCPU = proposed.max_cells_by_cpu;

              // Determine bottleneck only when both resources are selected
              const bothSelected = memorySelected && cpuSelected;
              const isMemoryBottleneck =
                bothSelected && maxCellsByMemory <= maxCellsByCPU;
              const isCPUBottleneck =
                bothSelected && maxCellsByCPU < maxCellsByMemory;

              // Calculate headroom for each (cells remaining beyond current)
              const currentCells = proposed.cell_count;
              const memoryHeadroom = maxCellsByMemory - currentCells;
              const cpuHeadroom = proposed.cpu_headroom_cells;

              return (
                <div className="space-y-3">
                  {/* Memory Constraint - only show if memory is selected */}
                  {memorySelected && (
                    <div
                      className={`flex items-center justify-between p-3 rounded-lg ${
                        isMemoryBottleneck
                          ? "bg-amber-500/10 border border-amber-500/30"
                          : "bg-slate-700/30"
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <Database
                          size={16}
                          className={
                            isMemoryBottleneck
                              ? "text-amber-400"
                              : "text-gray-400"
                          }
                        />
                        <span
                          className={`font-medium ${isMemoryBottleneck ? "text-amber-300" : "text-gray-300"}`}
                        >
                          Memory
                        </span>
                        {isMemoryBottleneck && (
                          <span className="text-xs text-amber-400 px-2 py-0.5 bg-amber-500/20 rounded">
                            ← BOTTLENECK
                          </span>
                        )}
                      </div>
                      <div className="text-right">
                        <span
                          className={`font-mono font-bold ${isMemoryBottleneck ? "text-amber-400" : "text-gray-300"}`}
                        >
                          {maxCellsByMemory} cells
                        </span>
                        {!isMemoryBottleneck && memoryHeadroom > 0 && (
                          <span className="text-xs text-gray-500 ml-2">
                            (+{memoryHeadroom} headroom)
                          </span>
                        )}
                      </div>
                    </div>
                  )}

                  {/* CPU Constraint - only show if cpu is selected */}
                  {cpuSelected && maxCellsByCPU > 0 && (
                    <div
                      className={`flex items-center justify-between p-3 rounded-lg ${
                        isCPUBottleneck
                          ? "bg-amber-500/10 border border-amber-500/30"
                          : "bg-slate-700/30"
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <Cpu
                          size={16}
                          className={
                            isCPUBottleneck ? "text-amber-400" : "text-gray-400"
                          }
                        />
                        <span
                          className={`font-medium ${isCPUBottleneck ? "text-amber-300" : "text-gray-300"}`}
                        >
                          CPU
                        </span>
                        {isCPUBottleneck && (
                          <span className="text-xs text-amber-400 px-2 py-0.5 bg-amber-500/20 rounded">
                            ← BOTTLENECK
                          </span>
                        )}
                      </div>
                      <div className="text-right">
                        <span
                          className={`font-mono font-bold ${isCPUBottleneck ? "text-amber-400" : "text-gray-300"}`}
                        >
                          {maxCellsByCPU} cells
                        </span>
                        {!isCPUBottleneck && cpuHeadroom > 0 && (
                          <span className="text-xs text-gray-500 ml-2">
                            (+{cpuHeadroom} headroom)
                          </span>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              );
            })()}

            <div className="mt-4 pt-3 border-t border-slate-700/50 text-center">
              <p className="text-xs text-gray-500">
                Max cells limited by{" "}
                {selectedResources.includes("memory") &&
                selectedResources.includes("cpu")
                  ? `${comparison.constraints.limiting_constraint === "ha_admission" ? "HA Admission" : "N-1"} memory and target CPU ratio`
                  : selectedResources.includes("memory")
                    ? `${comparison.constraints.limiting_constraint === "ha_admission" ? "HA Admission" : "N-1"} memory`
                    : "target CPU ratio"}
              </p>
            </div>
          </div>
        )}

      {/* Warnings Section */}
      {safeWarnings.length > 0 && (
        <div className="bg-slate-800/30 rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <AlertTriangle size={16} />
            <span className="text-xs uppercase tracking-wider font-medium">
              Recommendations ({safeWarnings.length})
            </span>
          </div>
          <div className="space-y-3">
            {safeWarnings.map((warning, index) => (
              <div
                key={index}
                className={`p-4 rounded-lg ${
                  warning.severity === "critical"
                    ? "bg-red-900/20 border border-red-700/30"
                    : warning.severity === "warning"
                      ? "bg-amber-900/20 border border-amber-700/30"
                      : "bg-blue-900/20 border border-blue-700/30"
                }`}
              >
                <div className="flex items-start gap-3">
                  <div
                    className={`mt-0.5 ${
                      warning.severity === "critical"
                        ? "text-red-400"
                        : warning.severity === "warning"
                          ? "text-amber-400"
                          : "text-blue-400"
                    }`}
                  >
                    {warning.severity === "critical" ? (
                      <XCircle size={16} />
                    ) : warning.severity === "warning" ? (
                      <AlertTriangle size={16} />
                    ) : (
                      <CheckCircle2 size={16} />
                    )}
                  </div>
                  <div className="flex-1">
                    <div
                      className={`text-sm font-medium ${
                        warning.severity === "critical"
                          ? "text-red-300"
                          : warning.severity === "warning"
                            ? "text-amber-300"
                            : "text-blue-300"
                      }`}
                    >
                      {warning.message}
                    </div>

                    {/* Change context */}
                    {warning.change && (
                      <div className="mt-2 text-xs text-gray-400 font-mono">
                        <span className="text-gray-500">Change: </span>
                        {formatFieldName(warning.change.field)}:{" "}
                        {warning.change.previous_val} →{" "}
                        {warning.change.proposed_val}
                        <span className="ml-2 text-gray-500">
                          ({warning.change.delta >= 0 ? "+" : ""}
                          {warning.change.delta},{" "}
                          {warning.change.delta_pct >= 0 ? "+" : ""}
                          {warning.change.delta_pct.toFixed(1)}%)
                        </span>
                      </div>
                    )}

                    {/* Fix suggestions */}
                    {warning.fixes && warning.fixes.length > 0 && (
                      <div className="mt-3 space-y-1">
                        <div className="text-xs text-gray-500 uppercase tracking-wider">
                          Suggested fixes:
                        </div>
                        {warning.fixes.map((fix, fixIdx) => (
                          <div
                            key={fixIdx}
                            className="flex items-center gap-2 text-sm"
                          >
                            <span className="text-cyan-400">→</span>
                            <span className="text-cyan-300">
                              {fix.description}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* All Clear Message */}
      {safeWarnings.length === 0 && (
        <div className="bg-emerald-900/20 rounded-xl p-6 border border-emerald-700/30 text-center">
          <CheckCircle2 className="text-emerald-400 mx-auto mb-2" size={32} />
          <div className="text-lg font-semibold text-emerald-400">
            All Checks Passed
          </div>
          <div className="text-sm text-emerald-300/70 mt-1">
            Proposed configuration meets all capacity and redundancy
            requirements
          </div>
        </div>
      )}
    </div>
  );
};

export default ScenarioResults;
