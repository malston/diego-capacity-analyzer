// ABOUTME: Step 3 of scenario wizard - advanced configuration options
// ABOUTME: Handles memory overhead, hypothetical app, host config, and TPS curve

import { ArrowRight, Plus, X, CheckCircle2 } from "lucide-react";
import { DEFAULT_TPS_CURVE } from "../../../config/resourceConfig";
import HostConfigSection from "./HostConfigSection";
import Tooltip from "../../Tooltip";

const AdvancedStep = ({
  overheadPct,
  setOverheadPct,
  useAdditionalApp,
  setUseAdditionalApp,
  additionalApp,
  setAdditionalApp,
  tpsCurve,
  setTPSCurve,
  enableTPS,
  setEnableTPS,
  // Host config props
  hostCount,
  setHostCount,
  coresPerHost,
  setCoresPerHost,
  memoryPerHost,
  setMemoryPerHost,
  haAdmissionPct,
  setHaAdmissionPct,
  // Chunk size props
  chunkSizeMB,
  setChunkSizeMB,
  autoDetectedChunkSizeMB,
  onContinue,
  onSkip,
  isLastStep = false,
}) => {
  const updateTPSPoint = (index, field, value) => {
    setTPSCurve((prev) =>
      prev.map((pt, i) =>
        i === index ? { ...pt, [field]: parseInt(value) || 0 } : pt,
      ),
    );
  };

  const addTPSPoint = () => {
    const lastPt = tpsCurve[tpsCurve.length - 1] || { cells: 0, tps: 0 };
    setTPSCurve([
      ...tpsCurve,
      { cells: lastPt.cells + 50, tps: Math.max(50, lastPt.tps - 100) },
    ]);
  };

  const removeTPSPoint = (index) => {
    if (tpsCurve.length > 2) {
      setTPSCurve((prev) => prev.filter((_, i) => i !== index));
    }
  };

  const resetTPSCurve = () => {
    setTPSCurve(DEFAULT_TPS_CURVE);
  };

  return (
    <div className="space-y-6">
      {/* Host Configuration Section - Collapsible, expanded by default */}
      <HostConfigSection
        hostCount={hostCount}
        setHostCount={setHostCount}
        coresPerHost={coresPerHost}
        setCoresPerHost={setCoresPerHost}
        memoryPerHost={memoryPerHost}
        setMemoryPerHost={setMemoryPerHost}
        haAdmissionPct={haAdmissionPct}
        setHaAdmissionPct={setHaAdmissionPct}
        defaultExpanded={true}
      />

      {/* Memory Overhead Slider */}
      <div>
        <label
          htmlFor="overhead-slider"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          <Tooltip
            text="Reserved memory for Diego agent, Garden runtime, and OS processes (default 7%). Applications cannot use this capacity -- it's system-level overhead."
            position="right"
            showIcon
          >
            <span>Memory Overhead: {overheadPct}%</span>
          </Tooltip>
        </label>
        <input
          id="overhead-slider"
          type="range"
          value={overheadPct}
          onChange={(e) => setOverheadPct(Number(e.target.value))}
          min={1}
          max={20}
          step={0.5}
          className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-500"
        />
        <div className="flex justify-between text-xs text-gray-500 mt-1">
          <span>1%</span>
          <span>Default: 7%</span>
          <span>20%</span>
        </div>
      </div>

      {/* Staging Chunk Size Override */}
      <div>
        <label
          htmlFor="chunk-size-input"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          <Tooltip
            text="Contiguous memory needed to stage (compile) apps. Auto-detected from max app memory limit in your environment. Override if you anticipate deploying larger apps than currently exist."
            position="right"
            showIcon
          >
            <span>Staging Chunk Size (MB)</span>
          </Tooltip>
        </label>
        <div className="flex items-center gap-3">
          <input
            id="chunk-size-input"
            type="number"
            value={chunkSizeMB || ""}
            onChange={(e) => {
              const value = e.target.value;
              setChunkSizeMB(value === "" ? null : Number(value));
            }}
            placeholder={
              autoDetectedChunkSizeMB > 0
                ? `${autoDetectedChunkSizeMB} (auto-detected)`
                : "4096 (default)"
            }
            min={256}
            max={16384}
            step={256}
            className="w-48 bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none placeholder-gray-500"
          />
          <span className="text-xs text-gray-500">
            {chunkSizeMB
              ? `${(chunkSizeMB / 1024).toFixed(1)} GB`
              : autoDetectedChunkSizeMB > 0
                ? `${(autoDetectedChunkSizeMB / 1024).toFixed(1)} GB (auto)`
                : "4 GB (default)"}
          </span>
        </div>
        <p className="text-xs text-gray-500 mt-1">
          Based on max app memory limit. Increase for large Java apps or if
          staging fails with memory errors.
        </p>
      </div>

      {/* Hypothetical App Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <h4 className="text-sm font-medium text-gray-300 mb-2">
          <Tooltip
            text="Simulate deploying a new app to see capacity impact before actually deploying. Useful for capacity planning: 'Can my foundation handle this workload?'"
            position="right"
            showIcon
          >
            <span>Hypothetical App</span>
          </Tooltip>
        </h4>
        <p className="text-xs text-gray-400 mb-3">
          Model a new workload to see if it fits (e.g., a 50-instance app with
          2GB each).
        </p>
        <div className="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            id="use-app"
            checked={useAdditionalApp}
            onChange={(e) => setUseAdditionalApp(e.target.checked)}
            className="rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500"
          />
          <label htmlFor="use-app" className="text-sm text-gray-300">
            Include in analysis
          </label>
        </div>

        {useAdditionalApp && (
          <div className="grid grid-cols-2 gap-3">
            <div className="col-span-2">
              <label
                htmlFor="app-name"
                className="block text-xs text-gray-400 mb-1"
              >
                App Name
              </label>
              <input
                id="app-name"
                type="text"
                value={additionalApp.name}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, name: e.target.value })
                }
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label
                htmlFor="app-instances"
                className="block text-xs text-gray-400 mb-1"
              >
                Instances
              </label>
              <input
                id="app-instances"
                type="number"
                value={additionalApp.instances}
                onChange={(e) =>
                  setAdditionalApp({
                    ...additionalApp,
                    instances: Number(e.target.value),
                  })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label
                htmlFor="app-memory"
                className="block text-xs text-gray-400 mb-1"
              >
                Memory/Instance (GB)
              </label>
              <input
                id="app-memory"
                type="number"
                value={additionalApp.memoryGB}
                onChange={(e) =>
                  setAdditionalApp({
                    ...additionalApp,
                    memoryGB: Number(e.target.value),
                  })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label
                htmlFor="app-disk"
                className="block text-xs text-gray-400 mb-1"
              >
                Disk/Instance (GB)
              </label>
              <input
                id="app-disk"
                type="number"
                value={additionalApp.diskGB}
                onChange={(e) =>
                  setAdditionalApp({
                    ...additionalApp,
                    diskGB: Number(e.target.value),
                  })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
          </div>
        )}
      </div>

      {/* TPS Performance Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-medium text-gray-300">
            <Tooltip
              text="Estimates Diego scheduler throughput (Tasks Per Second) based on cell count. More cells = more coordination overhead = lower TPS. Values are modeled from internal benchmarks."
              position="right"
              showIcon
            >
              <span>TPS Performance Model</span>
            </Tooltip>
          </h4>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">
              {enableTPS ? "Enabled" : "Disabled"}
            </span>
            <button
              type="button"
              onClick={() => setEnableTPS(!enableTPS)}
              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                enableTPS ? "bg-cyan-600" : "bg-slate-600"
              }`}
              aria-label="Toggle TPS model"
            >
              <span
                className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                  enableTPS ? "translate-x-5" : "translate-x-1"
                }`}
              />
            </button>
          </div>
        </div>
        <p className="text-xs text-gray-400 mb-3">
          {enableTPS
            ? "Customize to match your observed scheduler performance."
            : "TPS modeling is experimental and may not be accurate for all environments."}
        </p>

        {enableTPS && (
          <>
            <div className="space-y-2 mb-3">
              {tpsCurve.map((pt, i) => (
                <div
                  key={`tps-${pt.cells}-${i}`}
                  className="flex items-center gap-2"
                >
                  <input
                    type="number"
                    value={pt.cells}
                    onChange={(e) => updateTPSPoint(i, "cells", e.target.value)}
                    aria-label={`TPS point ${i + 1} cells`}
                    className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                  />
                  <span className="text-gray-500">cells →</span>
                  <input
                    type="number"
                    value={pt.tps}
                    onChange={(e) => updateTPSPoint(i, "tps", e.target.value)}
                    aria-label={`TPS point ${i + 1} tps`}
                    className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                  />
                  <span className="text-gray-500">TPS</span>
                  {tpsCurve.length > 2 && (
                    <button
                      type="button"
                      onClick={() => removeTPSPoint(i)}
                      className="text-red-400 hover:text-red-300 p-1"
                      aria-label={`Remove TPS point ${i + 1}`}
                    >
                      <X size={14} />
                    </button>
                  )}
                </div>
              ))}
            </div>

            <div className="flex gap-2">
              <button
                type="button"
                onClick={addTPSPoint}
                className="text-xs text-cyan-400 hover:text-cyan-300 flex items-center gap-1"
              >
                <Plus size={12} /> Add Point
              </button>
              <button
                type="button"
                onClick={resetTPSCurve}
                className="text-xs text-gray-400 hover:text-gray-300"
              >
                Reset to Default
              </button>
            </div>
          </>
        )}
      </div>

      {isLastStep ? (
        <div className="flex items-center gap-3 pt-4 text-emerald-400">
          <CheckCircle2 size={20} />
          <span className="text-sm font-medium">
            Configuration complete — click Run Analysis below
          </span>
        </div>
      ) : (
        <div className="flex justify-end gap-3 pt-4">
          <button
            type="button"
            onClick={onSkip}
            className="px-6 py-2.5 text-gray-400 hover:text-gray-300 transition-colors font-medium"
          >
            Skip
          </button>
          <button
            type="button"
            onClick={onContinue}
            className="flex items-center gap-2 px-6 py-2.5 bg-cyan-600 text-white rounded-lg hover:bg-cyan-500 transition-colors font-medium"
          >
            Continue
            <ArrowRight size={16} />
          </button>
        </div>
      )}
    </div>
  );
};

export default AdvancedStep;
