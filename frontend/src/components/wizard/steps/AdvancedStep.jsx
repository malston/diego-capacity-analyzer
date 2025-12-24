// ABOUTME: Step 3 of scenario wizard - advanced configuration options
// ABOUTME: Handles memory overhead, hypothetical app, and TPS curve

import { ArrowRight, Plus, X, CheckCircle2 } from 'lucide-react';
import { DEFAULT_TPS_CURVE } from '../../../config/resourceConfig';

const AdvancedStep = ({
  overheadPct,
  setOverheadPct,
  useAdditionalApp,
  setUseAdditionalApp,
  additionalApp,
  setAdditionalApp,
  tpsCurve,
  setTPSCurve,
  onContinue,
  onSkip,
  isLastStep = false,
}) => {
  const updateTPSPoint = (index, field, value) => {
    setTPSCurve((prev) =>
      prev.map((pt, i) => (i === index ? { ...pt, [field]: parseInt(value) || 0 } : pt))
    );
  };

  const addTPSPoint = () => {
    const lastPt = tpsCurve[tpsCurve.length - 1] || { cells: 0, tps: 0 };
    setTPSCurve([...tpsCurve, { cells: lastPt.cells + 50, tps: Math.max(50, lastPt.tps - 100) }]);
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
      {/* Memory Overhead Slider */}
      <div>
        <label
          htmlFor="overhead-slider"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          Memory Overhead: {overheadPct}%
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

      {/* Hypothetical App Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <h4 className="text-sm font-medium text-gray-300 mb-3">Hypothetical App</h4>
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
              <label htmlFor="app-name" className="block text-xs text-gray-400 mb-1">
                App Name
              </label>
              <input
                id="app-name"
                type="text"
                value={additionalApp.name}
                onChange={(e) => setAdditionalApp({ ...additionalApp, name: e.target.value })}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-instances" className="block text-xs text-gray-400 mb-1">
                Instances
              </label>
              <input
                id="app-instances"
                type="number"
                value={additionalApp.instances}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, instances: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-memory" className="block text-xs text-gray-400 mb-1">
                Memory/Instance (GB)
              </label>
              <input
                id="app-memory"
                type="number"
                value={additionalApp.memoryGB}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, memoryGB: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-disk" className="block text-xs text-gray-400 mb-1">
                Disk/Instance (GB)
              </label>
              <input
                id="app-disk"
                type="number"
                value={additionalApp.diskGB}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, diskGB: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
          </div>
        )}
      </div>

      {/* TPS Curve Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <h4 className="text-sm font-medium text-gray-300 mb-3">TPS Performance Curve</h4>
        <p className="text-xs text-gray-400 mb-3">
          Customize to match your observed scheduler performance.
        </p>

        <div className="space-y-2 mb-3">
          {tpsCurve.map((pt, i) => (
            <div key={i} className="flex items-center gap-2">
              <input
                type="number"
                value={pt.cells}
                onChange={(e) => updateTPSPoint(i, 'cells', e.target.value)}
                aria-label={`TPS point ${i + 1} cells`}
                className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
              <span className="text-gray-500">cells →</span>
              <input
                type="number"
                value={pt.tps}
                onChange={(e) => updateTPSPoint(i, 'tps', e.target.value)}
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
