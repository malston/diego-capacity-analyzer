// ABOUTME: CPU configuration step for scenario wizard
// ABOUTME: Handles physical cores, host count, and vCPU:pCPU ratio inputs

import { ArrowRight, Cpu, Server, AlertTriangle, CheckCircle } from 'lucide-react';

// vCPU:pCPU ratio risk level thresholds per spec
const getRatioRiskLevel = (ratio) => {
  if (ratio <= 4) {
    return {
      level: 'low',
      label: 'Low - Production safe',
      color: 'text-emerald-400',
      bgColor: 'bg-emerald-500/20',
      icon: CheckCircle,
    };
  } else if (ratio <= 8) {
    return {
      level: 'medium',
      label: 'Medium - Monitor CPU ready',
      color: 'text-amber-400',
      bgColor: 'bg-amber-500/20',
      icon: AlertTriangle,
    };
  } else {
    return {
      level: 'high',
      label: 'High - Expect contention',
      color: 'text-red-400',
      bgColor: 'bg-red-500/20',
      icon: AlertTriangle,
    };
  }
};

const CPUConfigStep = ({
  physicalCoresPerHost,
  setPhysicalCoresPerHost,
  hostCount,
  setHostCount,
  targetVCPURatio,
  setTargetVCPURatio,
  onContinue,
  onSkip,
}) => {
  const totalCores = physicalCoresPerHost * hostCount;
  const riskLevel = getRatioRiskLevel(targetVCPURatio);
  const RiskIcon = riskLevel.icon;

  const canContinue = hostCount > 0 && physicalCoresPerHost > 0;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label
            htmlFor="physical-cores"
            className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
          >
            <span className="flex items-center gap-1">
              <Cpu size={12} />
              Physical Cores per Host
            </span>
          </label>
          <input
            id="physical-cores"
            type="number"
            value={physicalCoresPerHost}
            onChange={(e) => setPhysicalCoresPerHost(Number(e.target.value))}
            min={1}
            className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
          />
        </div>

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
      </div>

      {/* Total cores calculation display */}
      <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
        <div className="flex items-center justify-between">
          <span className="text-gray-400 text-sm">Total Physical Cores</span>
          <span className="text-2xl font-mono font-bold text-cyan-400">
            {totalCores}
          </span>
        </div>
        <div className="text-xs text-gray-500 mt-1">
          {physicalCoresPerHost} cores × {hostCount} hosts = {totalCores} total cores
        </div>
      </div>

      <div>
        <label
          htmlFor="vcpu-ratio"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          Target vCPU:pCPU Ratio
        </label>
        <div className="flex items-center gap-4">
          <input
            id="vcpu-ratio"
            type="number"
            value={targetVCPURatio}
            onChange={(e) => setTargetVCPURatio(Number(e.target.value))}
            min={1}
            max={16}
            step={1}
            className="w-24 bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
          />
          <span className="text-gray-400">: 1</span>

          {/* Risk level indicator */}
          <div
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg ${riskLevel.bgColor}`}
          >
            <RiskIcon size={14} className={riskLevel.color} />
            <span className={`text-sm ${riskLevel.color}`}>{riskLevel.label}</span>
          </div>
        </div>
        <div className="text-xs text-gray-500 mt-2">
          Typical recommendations: ≤4:1 for production, 4-8:1 for dev/test
        </div>
      </div>

      <div className="flex justify-end gap-3 pt-4">
        {onSkip && (
          <button
            type="button"
            onClick={onSkip}
            className="px-6 py-2.5 text-gray-400 hover:text-gray-300 transition-colors font-medium"
          >
            Skip
          </button>
        )}
        <button
          type="button"
          onClick={onContinue}
          disabled={!canContinue}
          className="flex items-center gap-2 px-6 py-2.5 bg-cyan-600 text-white rounded-lg hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
        >
          Continue
          <ArrowRight size={16} />
        </button>
      </div>
    </div>
  );
};

export default CPUConfigStep;
