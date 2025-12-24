// ABOUTME: Step 2 of scenario wizard - resource type selection
// ABOUTME: Handles memory/CPU/disk toggle and disk size input

import { ArrowRight } from 'lucide-react';
import { RESOURCE_TYPES } from '../../../config/resourceConfig';

const ResourceTypesStep = ({
  selectedResources,
  toggleResource,
  customDisk,
  setCustomDisk,
  onContinue,
  onSkip,
}) => {
  const showDiskInput = selectedResources.includes('disk');

  return (
    <div className="space-y-6">
      <div>
        <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-3">
          Which resources to analyze?
        </label>
        <div className="flex flex-wrap gap-2">
          {RESOURCE_TYPES.map((resource) => {
            const Icon = resource.icon;
            const isSelected = selectedResources.includes(resource.id);
            return (
              <button
                type="button"
                key={resource.id}
                onClick={() => toggleResource(resource.id)}
                aria-pressed={isSelected}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg border transition-all ${
                  isSelected
                    ? 'bg-cyan-600/30 border-cyan-500 text-cyan-300'
                    : 'bg-slate-700/50 border-slate-600 text-gray-400 hover:border-slate-500'
                }`}
              >
                <Icon size={16} />
                {resource.label}
              </button>
            );
          })}
        </div>
      </div>

      {showDiskInput && (
        <div>
          <label
            htmlFor="disk-per-cell"
            className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
          >
            Disk per Cell (GB)
          </label>
          <input
            id="disk-per-cell"
            type="number"
            value={customDisk}
            onChange={(e) => setCustomDisk(Number(e.target.value))}
            min={32}
            className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors max-w-xs"
          />
        </div>
      )}

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
    </div>
  );
};

export default ResourceTypesStep;
