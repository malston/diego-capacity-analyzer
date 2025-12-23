// ABOUTME: Configuration for resource types and thresholds
// ABOUTME: Enables multi-select resource analysis with configurable TPS curve

import { HardDrive, Cpu, Database } from 'lucide-react';

// Resource types available for analysis
export const RESOURCE_TYPES = [
  { id: 'memory', label: 'Memory', unit: 'GB', icon: HardDrive, defaultSelected: true },
  { id: 'cpu', label: 'CPU', unit: 'vCPU', icon: Cpu, defaultSelected: false },
  { id: 'disk', label: 'Disk', unit: 'GB', icon: Database, defaultSelected: false },
];

// Default selected resources for backward compatibility
export const DEFAULT_SELECTED_RESOURCES = ['memory'];

// Utilization thresholds for warnings
export const UTILIZATION_THRESHOLDS = {
  memory: { warning: 80, critical: 90 },
  cpu: { warning: 70, critical: 85 },
  disk: { warning: 80, critical: 90 },
};

// Default overhead percentages
export const OVERHEAD_DEFAULTS = {
  memoryPct: 7.0,   // 7% memory overhead for Garden/system
  diskPct: 0.01,    // 0.01% disk overhead (negligible)
};

// Default TPS curve - baseline estimates, user can override in Advanced Options
export const DEFAULT_TPS_CURVE = [
  { cells: 1, tps: 284 },
  { cells: 3, tps: 1964 },    // Peak efficiency
  { cells: 9, tps: 1932 },
  { cells: 100, tps: 1389 },
  { cells: 210, tps: 104 },   // Severe degradation
];

// TPS status thresholds (as % of peak)
export const TPS_STATUS_THRESHOLDS = {
  optimal: 80,    // >= 80% of peak TPS
  degraded: 50,   // 50-79% of peak TPS
  critical: 0,    // < 50% of peak TPS
};

// Find peak TPS in curve
export const getPeakTPS = (curve = DEFAULT_TPS_CURVE) => {
  return Math.max(...curve.map(pt => pt.tps));
};

// Estimate TPS for a cell count using linear interpolation
export const estimateTPS = (cellCount, curve = DEFAULT_TPS_CURVE) => {
  if (cellCount <= 0 || !curve || curve.length === 0) {
    return { tps: 0, status: 'unknown' };
  }

  let tps = 0;

  // Exact match
  const exactMatch = curve.find(pt => pt.cells === cellCount);
  if (exactMatch) {
    tps = exactMatch.tps;
  } else {
    // Interpolation
    for (let i = 0; i < curve.length - 1; i++) {
      if (cellCount >= curve[i].cells && cellCount <= curve[i + 1].cells) {
        const ratio = (cellCount - curve[i].cells) / (curve[i + 1].cells - curve[i].cells);
        tps = Math.round(curve[i].tps + ratio * (curve[i + 1].tps - curve[i].tps));
        break;
      }
    }

    // Beyond last data point - extrapolate degradation
    if (tps === 0 && cellCount > curve[curve.length - 1].cells) {
      const lastPt = curve[curve.length - 1];
      tps = Math.max(1, Math.round(lastPt.tps * lastPt.cells / cellCount));
    }

    // Before first data point
    if (tps === 0 && cellCount < curve[0].cells) {
      tps = curve[0].tps;
    }
  }

  // Determine status
  const peakTPS = getPeakTPS(curve);
  const pctOfPeak = (tps / peakTPS) * 100;

  let status = 'critical';
  if (pctOfPeak >= TPS_STATUS_THRESHOLDS.optimal) {
    status = 'optimal';
  } else if (pctOfPeak >= TPS_STATUS_THRESHOLDS.degraded) {
    status = 'degraded';
  }

  return { tps, status };
};

// Color mapping for TPS status
export const TPS_STATUS_COLORS = {
  optimal: 'text-emerald-400',
  degraded: 'text-amber-400',
  critical: 'text-red-400',
  unknown: 'text-gray-400',
};

// Background colors for TPS status
export const TPS_STATUS_BG_COLORS = {
  optimal: 'bg-emerald-500/20',
  degraded: 'bg-amber-500/20',
  critical: 'bg-red-500/20',
  unknown: 'bg-gray-500/20',
};
