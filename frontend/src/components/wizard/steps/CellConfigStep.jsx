// ABOUTME: Step 1 of scenario wizard - Diego cell configuration
// ABOUTME: Handles VM size preset selection and cell count input

import { Sparkles, ArrowRight } from "lucide-react";
import { VM_SIZE_PRESETS } from "../../../config/vmPresets";
import Tooltip from "../../Tooltip";

const CellConfigStep = ({
  selectedPreset,
  setSelectedPreset,
  customCPU,
  setCustomCPU,
  customMemory,
  setCustomMemory,
  cellCount,
  setCellCount,
  equivalentCellSuggestion,
  onContinue,
}) => {
  const preset = VM_SIZE_PRESETS[selectedPreset];
  const isCustom = preset.cpu === null;
  const canContinue = cellCount > 0;

  return (
    <div className="space-y-6">
      <div>
        <label
          htmlFor="vm-size"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          <Tooltip
            text="Preset VM configurations for Diego cells. Larger VMs support more app instances but reduce fault tolerance (more impact per cell failure)."
            position="right"
            showIcon
          >
            <span>VM Size</span>
          </Tooltip>
        </label>
        <select
          id="vm-size"
          value={selectedPreset}
          onChange={(e) => setSelectedPreset(Number(e.target.value))}
          className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
        >
          {VM_SIZE_PRESETS.map((p, i) => (
            <option key={i} value={i}>
              {p.label}
            </option>
          ))}
        </select>
      </div>

      {isCustom && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="custom-cpu"
              className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
            >
              vCPU
            </label>
            <input
              id="custom-cpu"
              type="number"
              value={customCPU}
              onChange={(e) => setCustomCPU(Number(e.target.value))}
              min={1}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
            />
          </div>
          <div>
            <label
              htmlFor="custom-memory"
              className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
            >
              Memory (GB)
            </label>
            <input
              id="custom-memory"
              type="number"
              value={customMemory}
              onChange={(e) => setCustomMemory(Number(e.target.value))}
              min={8}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
            />
          </div>
        </div>
      )}

      <div>
        <label
          htmlFor="cell-count"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          Cell Count
        </label>
        <input
          id="cell-count"
          type="number"
          value={cellCount}
          onChange={(e) => setCellCount(Number(e.target.value))}
          min={1}
          className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
        />
        {equivalentCellSuggestion && (
          <Tooltip
            text="Calculates how many cells of the new size match your current total capacity. Useful when resizing cells without changing overall capacity."
            position="right"
          >
            <button
              type="button"
              onClick={() =>
                setCellCount(equivalentCellSuggestion.equivalentCells)
              }
              className="mt-2 text-xs text-amber-400 hover:text-amber-300 flex items-center gap-1 transition-colors"
            >
              <Sparkles size={12} />
              For equivalent capacity ({equivalentCellSuggestion.currentTotalGB}
              GB): use {equivalentCellSuggestion.equivalentCells} cells
            </button>
          </Tooltip>
        )}
      </div>

      <div className="flex justify-end pt-4">
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

export default CellConfigStep;
