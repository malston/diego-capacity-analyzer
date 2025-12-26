// ABOUTME: Host analysis display component showing infrastructure metrics
// ABOUTME: Displays host count, VMs per host, utilization, and HA status

import { Server, Cpu, HardDrive, Shield, AlertTriangle, CheckCircle } from 'lucide-react';

// Calculate HA survivability
const calculateHASurvivability = (hostCount, haAdmissionPct) => {
  if (hostCount === 0) return { hosts: 0, status: 'unknown' };

  // HA reservation percentage determines how many hosts can fail
  // 25% = 1/4 of cluster can fail = N-1 for 4 hosts
  // 33% = 1/3 of cluster can fail = N-1 for 3 hosts
  // 50% = half can fail = N-2 for 4 hosts
  const hostsSurvivable = Math.floor((haAdmissionPct / 100) * hostCount);

  // Minimum N-1 threshold: need at least 25% for 4 hosts, 33% for 3 hosts
  const minHAForN1 = 100 / hostCount;

  if (haAdmissionPct < minHAForN1) {
    return { hosts: hostsSurvivable, status: 'warning' };
  }

  return { hosts: hostsSurvivable, status: 'good' };
};

// Get utilization status
const getUtilizationStatus = (utilization) => {
  if (utilization >= 90) {
    return { color: 'text-red-400', bgColor: 'bg-red-500/20', status: 'critical' };
  } else if (utilization >= 75) {
    return { color: 'text-amber-400', bgColor: 'bg-amber-500/20', status: 'warning' };
  }
  return { color: 'text-cyan-400', bgColor: 'bg-cyan-500/20', status: 'good' };
};

const HostAnalysisCard = ({
  hostCount,
  coresPerHost,
  memoryPerHost,
  totalCells,
  haAdmissionPct,
  memoryUtilization,
  cpuUtilization,
}) => {
  const totalCores = hostCount * coresPerHost;
  const totalMemoryGB = hostCount * memoryPerHost;
  const vmsPerHost = hostCount > 0 ? (totalCells / hostCount).toFixed(1) : 0;

  const haStatus = calculateHASurvivability(hostCount, haAdmissionPct);
  const memStatus = getUtilizationStatus(memoryUtilization);
  const cpuStatus = getUtilizationStatus(cpuUtilization);

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
        <Server size={18} className="text-cyan-400" />
        Host Analysis
      </h3>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {/* Host Count */}
        <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
          <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
            <Server size={14} />
            Hosts
          </div>
          <div className="text-2xl font-mono font-bold text-cyan-400">
            {hostCount === 0 ? 'N/A' : hostCount}
          </div>
        </div>

        {/* Total Cores */}
        <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
          <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
            <Cpu size={14} />
            Total Cores
          </div>
          <div className="text-2xl font-mono font-bold text-cyan-400">
            {totalCores}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {coresPerHost} per host
          </div>
        </div>

        {/* Total Memory */}
        <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
          <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
            <HardDrive size={14} />
            Total Memory
          </div>
          <div className="text-2xl font-mono font-bold text-cyan-400">
            {totalMemoryGB >= 1000
              ? `${(totalMemoryGB / 1000).toFixed(1)}T`
              : `${totalMemoryGB}G`}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {memoryPerHost} GB per host
          </div>
        </div>

        {/* VMs Per Host */}
        <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
          <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
            <Server size={14} />
            VMs per Host
          </div>
          <div className="text-2xl font-mono font-bold text-cyan-400">
            {vmsPerHost}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {totalCells} cells total
          </div>
        </div>
      </div>

      {/* Utilization and HA Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
        {/* Memory Utilization */}
        <div className={`rounded-lg p-4 border border-slate-600/30 ${memStatus.bgColor}`}>
          <div className="flex items-center justify-between">
            <span className="text-gray-400 text-sm">Memory Utilization</span>
            <span className={`text-lg font-mono font-bold ${memStatus.color}`}>
              {memoryUtilization}%
            </span>
          </div>
        </div>

        {/* CPU Utilization */}
        <div className={`rounded-lg p-4 border border-slate-600/30 ${cpuStatus.bgColor}`}>
          <div className="flex items-center justify-between">
            <span className="text-gray-400 text-sm">CPU Utilization</span>
            <span className={`text-lg font-mono font-bold ${cpuStatus.color}`}>
              {cpuUtilization}%
            </span>
          </div>
        </div>

        {/* HA Status */}
        <div className={`rounded-lg p-4 border border-slate-600/30 ${
          haStatus.status === 'warning' ? 'bg-amber-500/20' : 'bg-emerald-500/20'
        }`}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Shield size={16} className={
                haStatus.status === 'warning' ? 'text-amber-400' : 'text-emerald-400'
              } />
              <span className="text-gray-400 text-sm">HA {haAdmissionPct}%</span>
            </div>
            <div className="flex items-center gap-2">
              {haStatus.status === 'warning' ? (
                <>
                  <AlertTriangle size={14} className="text-amber-400" />
                  <span className="text-amber-400 text-sm font-medium">Risk</span>
                </>
              ) : (
                <>
                  <CheckCircle size={14} className="text-emerald-400" />
                  <span className="text-emerald-400 text-sm font-medium">
                    N-{haStatus.hosts}
                  </span>
                </>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default HostAnalysisCard;
