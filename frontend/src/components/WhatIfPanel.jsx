// ABOUTME: What-If scenario panel for memory overcommit analysis
// ABOUTME: Allows users to simulate different overcommit ratios and see capacity impact

import { Zap } from "lucide-react";

const WhatIfPanel = ({ overcommitRatio, setOvercommitRatio, metrics }) => {
  // Thresholds aligned with docs/UI-GUIDE.md and VMware best practices:
  // 1.0-1.3x = Low (Production), 1.3-2.0x = Medium (Dev/Test), 2.0-3.0x = High (Labs)
  const getRatioColor = (ratio) => {
    if (ratio <= 1.3) return "text-emerald-400";
    if (ratio <= 2.0) return "text-yellow-400";
    if (ratio <= 3.0) return "text-orange-400";
    return "text-red-400";
  };

  const getRatioBadge = (ratio) => {
    if (ratio <= 1.3)
      return { bg: "bg-emerald-900/50 text-emerald-400", label: "Low Risk" };
    if (ratio <= 2.0)
      return { bg: "bg-yellow-900/50 text-yellow-400", label: "Medium" };
    if (ratio <= 3.0)
      return { bg: "bg-orange-900/50 text-orange-400", label: "High Risk" };
    return { bg: "bg-red-900/50 text-red-400", label: "Labs Only" };
  };

  const badge = getRatioBadge(overcommitRatio);

  return (
    <div className="metric-card p-6 rounded-xl mb-8 border-2 border-blue-500/50">
      <div className="flex items-center gap-2 mb-4">
        <Zap className="w-5 h-5 text-blue-400" aria-hidden="true" />
        <h2 className="text-xl font-bold title-font text-white">
          What-If Scenario
        </h2>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div>
          <label
            htmlFor="overcommit-slider"
            className="block text-sm text-slate-400 mb-3"
          >
            Memory Overcommit Ratio:{" "}
            <span className={`font-bold ${getRatioColor(overcommitRatio)}`}>
              {overcommitRatio.toFixed(1)}x
            </span>
            <span className={`ml-2 text-xs px-2 py-0.5 rounded ${badge.bg}`}>
              {badge.label}
            </span>
          </label>
          <input
            id="overcommit-slider"
            type="range"
            min="1.0"
            max="4.0"
            step="0.1"
            value={overcommitRatio}
            onChange={(e) => setOvercommitRatio(parseFloat(e.target.value))}
            className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
            aria-valuemin={1.0}
            aria-valuemax={4.0}
            aria-valuenow={overcommitRatio}
            aria-valuetext={`${overcommitRatio.toFixed(1)}x overcommit ratio, ${badge.label}`}
          />
          <div className="flex justify-between text-xs text-slate-500 mt-1">
            <span>1.0x (None)</span>
            <span className="text-yellow-500">2.0x</span>
            <span className="text-red-500">4.0x (Labs)</span>
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex justify-between items-center p-3 bg-slate-800/50 rounded-lg">
            <span className="text-slate-400">New Capacity:</span>
            <span className="text-white font-bold">
              {(metrics.newCapacity / 1024).toFixed(1)} GB
            </span>
          </div>
          <div className="flex justify-between items-center p-3 bg-slate-800/50 rounded-lg">
            <span className="text-slate-400">Current Instances:</span>
            <span className="text-white font-bold">
              {metrics.totalInstances}
            </span>
          </div>
          <div className="flex justify-between items-center p-3 bg-slate-800/50 rounded-lg">
            <span className="text-slate-400">Avg Instance Size:</span>
            <span className="text-white font-bold">
              {metrics.avgInstanceSize} MB
            </span>
          </div>
          <div
            className={`flex justify-between items-center p-3 rounded-lg ${
              metrics.additionalInstances >= 0
                ? "bg-green-500/10 border border-green-500/30"
                : "bg-red-500/10 border border-red-500/30"
            }`}
          >
            <span
              className={
                metrics.additionalInstances >= 0
                  ? "text-green-400"
                  : "text-red-400"
              }
            >
              Additional Capacity:
            </span>
            <span
              className={`font-bold ${
                metrics.additionalInstances >= 0
                  ? "text-green-400"
                  : "text-red-400"
              }`}
            >
              {metrics.additionalInstances >= 0 ? "+" : ""}
              {metrics.additionalInstances} instances
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default WhatIfPanel;
