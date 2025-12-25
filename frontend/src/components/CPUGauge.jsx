// ABOUTME: CPU utilization gauge component with vCPU:pCPU ratio indicator
// ABOUTME: Combines circular gauge with color-coded ratio risk level

import { AlertTriangle, CheckCircle } from 'lucide-react';
import CapacityGauge from './CapacityGauge';

// vCPU:pCPU ratio risk level thresholds per spec
const getRatioRiskLevel = (ratio) => {
  if (ratio <= 4) {
    return {
      level: 'low',
      label: 'Low',
      color: 'text-emerald-400',
      bgColor: 'bg-emerald-500/20',
      icon: CheckCircle,
    };
  } else if (ratio <= 8) {
    return {
      level: 'medium',
      label: 'Medium',
      color: 'text-amber-400',
      bgColor: 'bg-amber-500/20',
      icon: AlertTriangle,
    };
  } else {
    return {
      level: 'high',
      label: 'High',
      color: 'text-red-400',
      bgColor: 'bg-red-500/20',
      icon: AlertTriangle,
    };
  }
};

const CPUGauge = ({
  cpuUtilization,
  vcpuRatio,
  size = 120,
}) => {
  const riskLevel = getRatioRiskLevel(vcpuRatio);
  const RiskIcon = riskLevel.icon;

  // CPU utilization thresholds: warning at 70%, critical at 85%
  const cpuThresholds = { warning: 70, critical: 85 };

  return (
    <div className="flex flex-col items-center gap-3">
      {/* CPU Utilization Gauge */}
      <CapacityGauge
        value={cpuUtilization}
        label="CPU"
        size={size}
        thresholds={cpuThresholds}
        inverse={true}
        suffix="%"
      />

      {/* vCPU:pCPU Ratio Indicator */}
      <div
        className={`flex items-center gap-2 px-3 py-1.5 rounded-lg ${riskLevel.bgColor}`}
      >
        <RiskIcon size={14} className={riskLevel.color} />
        <span className={`text-sm font-mono ${riskLevel.color}`}>
          {vcpuRatio}:1
        </span>
        <span className={`text-xs ${riskLevel.color}`}>
          {riskLevel.label}
        </span>
      </div>
    </div>
  );
};

export default CPUGauge;
