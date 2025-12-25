// ABOUTME: Collapsible host configuration section for advanced capacity planning
// ABOUTME: Contains host count, cores/memory per host, and HA admission control

import { useState } from 'react';
import { ChevronDown, ChevronUp, Server, Cpu, HardDrive, Shield } from 'lucide-react';

const HostConfigSection = ({
  hostCount,
  setHostCount,
  coresPerHost,
  setCoresPerHost,
  memoryPerHost,
  setMemoryPerHost,
  haAdmissionPct,
  setHaAdmissionPct,
  defaultExpanded = false,
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  const totalCores = hostCount * coresPerHost;
  const totalMemoryGB = hostCount * memoryPerHost;

  return (
    <div className="bg-slate-700/30 rounded-lg border border-slate-600/30 overflow-hidden">
      {/* Header - Always visible */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-slate-700/50 transition-colors"
        aria-expanded={isExpanded}
        aria-label="Host Configuration"
      >
        <div className="flex items-center gap-2">
          <Server size={16} className="text-cyan-400" />
          <span className="text-sm font-medium text-gray-200">Host Configuration</span>
          <span className="text-xs text-gray-500 px-2 py-0.5 bg-slate-600/50 rounded">
            Optional
          </span>
        </div>
        {isExpanded ? (
          <ChevronUp size={16} className="text-gray-400" />
        ) : (
          <ChevronDown size={16} className="text-gray-400" />
        )}
      </button>

      {/* Expandable Content */}
      {isExpanded && (
        <div className="px-4 pb-4 space-y-4 border-t border-slate-600/30 pt-4">
          {/* Host Count and Cores Row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="host-count"
                className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
              >
                <span className="flex items-center gap-1">
                  <Server size={12} />
                  Number of Hosts
                </span>
              </label>
              <input
                id="host-count"
                type="number"
                value={hostCount}
                onChange={(e) => setHostCount(Number(e.target.value))}
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
              />
            </div>

            <div>
              <label
                htmlFor="cores-per-host"
                className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
              >
                <span className="flex items-center gap-1">
                  <Cpu size={12} />
                  Cores per Host
                </span>
              </label>
              <input
                id="cores-per-host"
                type="number"
                value={coresPerHost}
                onChange={(e) => setCoresPerHost(Number(e.target.value))}
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
              />
            </div>
          </div>

          {/* Memory and HA Row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="memory-per-host"
                className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
              >
                <span className="flex items-center gap-1">
                  <HardDrive size={12} />
                  Memory per Host (GB)
                </span>
              </label>
              <input
                id="memory-per-host"
                type="number"
                value={memoryPerHost}
                onChange={(e) => setMemoryPerHost(Number(e.target.value))}
                min={64}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
              />
            </div>

            <div>
              <label
                htmlFor="ha-admission"
                className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
              >
                <span className="flex items-center gap-1">
                  <Shield size={12} />
                  HA Admission (%)
                </span>
              </label>
              <input
                id="ha-admission"
                type="number"
                value={haAdmissionPct}
                onChange={(e) => setHaAdmissionPct(Number(e.target.value))}
                min={0}
                max={100}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
              />
              <div className="text-xs text-gray-500 mt-1">
                Capacity reserved for HA failover
              </div>
            </div>
          </div>

          {/* Summary */}
          <div className="bg-slate-800/50 rounded-lg p-3 border border-slate-600/30">
            <div className="text-xs uppercase tracking-wider text-gray-500 mb-2">
              Total Host Capacity
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="flex items-center justify-between">
                <span className="text-gray-400 text-sm">Total Cores:</span>
                <span className="text-cyan-400 font-mono font-bold">{totalCores}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-gray-400 text-sm">Total Memory:</span>
                <span className="text-cyan-400 font-mono font-bold">{totalMemoryGB} GB</span>
              </div>
            </div>
          </div>

          {/* Help text */}
          <div className="text-xs text-gray-500 bg-slate-700/20 rounded p-3 border border-slate-600/20">
            <strong className="text-gray-400">Tip:</strong> Host configuration enables HA capacity analysis.
            These values should match your vSphere cluster configuration. The HA admission control percentage
            determines how much capacity is reserved for VM failover in case of host failure.
          </div>
        </div>
      )}
    </div>
  );
};

export default HostConfigSection;
