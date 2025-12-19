// frontend/src/components/ScenarioAnalyzer.jsx
// ABOUTME: Main what-if scenario analyzer component
// ABOUTME: Combines data source, comparison table, and warnings

import React, { useState, useEffect } from 'react';
import { Calculator, RefreshCw, FileDown } from 'lucide-react';
import DataSourceSelector from './DataSourceSelector';
import ComparisonTable from './ComparisonTable';
import WarningsList from './WarningsList';
import { scenarioApi } from '../services/scenarioApi';
import { VM_SIZE_PRESETS, DEFAULT_PRESET_INDEX } from '../config/vmPresets';
import { generateMarkdownReport, downloadMarkdown } from '../utils/exportMarkdown';

const ScenarioAnalyzer = () => {
  const [infrastructureData, setInfrastructureData] = useState(null);
  const [infrastructureState, setInfrastructureState] = useState(null);
  const [comparison, setComparison] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Scenario input state
  const [selectedPreset, setSelectedPreset] = useState(DEFAULT_PRESET_INDEX);
  const [customCPU, setCustomCPU] = useState(4);
  const [customMemory, setCustomMemory] = useState(32);
  const [cellCount, setCellCount] = useState(0);

  // Load from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem('scenario-infrastructure');
    if (saved) {
      try {
        const data = JSON.parse(saved);
        setInfrastructureData(data);
        handleDataLoaded(data);
      } catch (e) {
        console.error('Failed to load saved infrastructure:', e);
      }
    }
  }, []);

  const handleDataLoaded = async (data) => {
    setInfrastructureData(data);
    setLoading(true);
    setError(null);

    try {
      const state = await scenarioApi.setManualInfrastructure(data);
      setInfrastructureState(state);

      // Set initial cell count from data
      const totalCells = data.clusters.reduce(
        (sum, c) => sum + c.diego_cell_count,
        0
      );
      setCellCount(totalCells);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCompare = async () => {
    if (!infrastructureState) return;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const cpu = preset.cpu || customCPU;
    const memory = preset.memoryGB || customMemory;

    setLoading(true);
    setError(null);

    try {
      const result = await scenarioApi.compareScenario({
        proposed_cell_memory_gb: memory,
        proposed_cell_cpu: cpu,
        proposed_cell_count: cellCount,
      });
      setComparison(result);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const preset = VM_SIZE_PRESETS[selectedPreset];
  const isCustom = preset.cpu === null;

  const handleExportMarkdown = () => {
    if (!comparison || !infrastructureData) return;
    const markdown = generateMarkdownReport(comparison, infrastructureData);
    const filename = `${infrastructureData.name || 'capacity'}-analysis-${new Date().toISOString().split('T')[0]}.md`;
    downloadMarkdown(markdown, filename);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
          <Calculator className="text-blue-600" />
          What-If Scenario Analysis
        </h2>
      </div>

      <DataSourceSelector
        onDataLoaded={handleDataLoaded}
        currentData={infrastructureData}
      />

      {infrastructureState && (
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-lg font-semibold mb-4">Proposed Configuration</h3>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                VM Size
              </label>
              <select
                value={selectedPreset}
                onChange={(e) => setSelectedPreset(Number(e.target.value))}
                className="w-full border rounded-md px-3 py-2"
              >
                {VM_SIZE_PRESETS.map((p, i) => (
                  <option key={i} value={i}>
                    {p.label}
                  </option>
                ))}
              </select>
            </div>

            {isCustom && (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    vCPU
                  </label>
                  <input
                    type="number"
                    value={customCPU}
                    onChange={(e) => setCustomCPU(Number(e.target.value))}
                    min={1}
                    className="w-full border rounded-md px-3 py-2"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Memory (GB)
                  </label>
                  <input
                    type="number"
                    value={customMemory}
                    onChange={(e) => setCustomMemory(Number(e.target.value))}
                    min={8}
                    className="w-full border rounded-md px-3 py-2"
                  />
                </div>
              </>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Cell Count
              </label>
              <input
                type="number"
                value={cellCount}
                onChange={(e) => setCellCount(Number(e.target.value))}
                min={1}
                className="w-full border rounded-md px-3 py-2"
              />
            </div>
          </div>

          <button
            onClick={handleCompare}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? (
              <RefreshCw className="animate-spin" size={16} />
            ) : (
              <Calculator size={16} />
            )}
            Compare Scenarios
          </button>
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-800">
          Error: {error}
        </div>
      )}

      {comparison && (
        <>
          <div className="flex justify-end mb-2">
            <button
              onClick={handleExportMarkdown}
              className="flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200"
            >
              <FileDown size={16} />
              Export to Markdown
            </button>
          </div>
          <ComparisonTable comparison={comparison} />
          <WarningsList warnings={comparison.warnings} />
        </>
      )}
    </div>
  );
};

export default ScenarioAnalyzer;
