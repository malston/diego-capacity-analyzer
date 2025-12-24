// frontend/src/components/ScenarioAnalyzer.jsx
// ABOUTME: Main what-if scenario analyzer component
// ABOUTME: Combines data source, comparison table, and warnings

import { useState, useEffect, useMemo } from 'react';
import { Calculator, RefreshCw, FileDown, Sparkles, Server, HardDrive, Cpu, Database, AlertCircle } from 'lucide-react';
import DataSourceSelector from './DataSourceSelector';
import ScenarioResults from './ScenarioResults';
import ScenarioWizard from './wizard/ScenarioWizard';
import { scenarioApi } from '../services/scenarioApi';
import { VM_SIZE_PRESETS, DEFAULT_PRESET_INDEX } from '../config/vmPresets';
import { generateMarkdownReport, downloadMarkdown } from '../utils/exportMarkdown';
import {
  DEFAULT_SELECTED_RESOURCES,
  OVERHEAD_DEFAULTS,
  DEFAULT_TPS_CURVE,
} from '../config/resourceConfig';

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
  const [customDisk, setCustomDisk] = useState(128);
  const [cellCount, setCellCount] = useState(0);

  // New feature state
  const [selectedResources, setSelectedResources] = useState(DEFAULT_SELECTED_RESOURCES);
  const [overheadPct, setOverheadPct] = useState(OVERHEAD_DEFAULTS.memoryPct);

  // Wizard step completion tracking
  const [step1Completed, setStep1Completed] = useState(false);

  // Additional app state
  const [additionalApp, setAdditionalApp] = useState({
    name: 'hypothetical-app',
    instances: 1,
    memoryGB: 1,
    diskGB: 1,
  });
  const [useAdditionalApp, setUseAdditionalApp] = useState(false);

  // TPS curve state
  const [tpsCurve, setTPSCurve] = useState(DEFAULT_TPS_CURVE);

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

      // Set initial disk from first cluster if available
      if (data.clusters[0]?.diego_cell_disk_gb) {
        setCustomDisk(data.clusters[0].diego_cell_disk_gb);
      }
      // Note: cellCount is auto-set by the useEffect that calculates equivalent capacity
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const toggleResource = (resourceId) => {
    setSelectedResources(prev =>
      prev.includes(resourceId)
        ? prev.filter(r => r !== resourceId)
        : [...prev, resourceId]
    );
  };

  const handleStepComplete = (stepIndex) => {
    if (stepIndex === 0) {
      setStep1Completed(true);
    }
  };

  // Compute current configuration summary from loaded data
  const currentConfig = useMemo(() => {
    if (!infrastructureData?.clusters?.length) return null;

    const clusters = infrastructureData.clusters;
    const totalCells = clusters.reduce((sum, c) => sum + (c.diego_cell_count || 0), 0);
    const totalHosts = clusters.reduce((sum, c) => sum + (c.host_count || 0), 0);

    // Get cell specs from first cluster (assume uniform)
    const firstCluster = clusters[0];
    const cellCpu = firstCluster.diego_cell_cpu || firstCluster.diego_cell_vcpu || 0;
    const cellMemoryGB = firstCluster.diego_cell_memory_gb || 0;
    const cellDiskGB = firstCluster.diego_cell_disk_gb || 0;

    const totalMemoryGB = totalCells * cellMemoryGB;
    const totalDiskGB = totalCells * cellDiskGB;

    return {
      name: infrastructureData.name || 'Loaded Infrastructure',
      totalCells,
      totalHosts,
      cellCpu,
      cellMemoryGB,
      cellDiskGB,
      totalMemoryGB,
      totalDiskGB,
      clusterCount: clusters.length,
    };
  }, [infrastructureData]);

  // Auto-update cell count to equivalent capacity when VM size changes
  useEffect(() => {
    if (!currentConfig || currentConfig.totalMemoryGB === 0) return;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const proposedMemoryGB = preset.memoryGB || customMemory;

    // Calculate equivalent cells to maintain same total capacity
    const equivalentCells = Math.round(currentConfig.totalMemoryGB / proposedMemoryGB);

    // Auto-set cell count to equivalent capacity
    setCellCount(equivalentCells);
  }, [selectedPreset, customMemory, currentConfig]);

  // Show suggestion when user manually reduces cell count below equivalent capacity
  const equivalentCellSuggestion = useMemo(() => {
    if (!currentConfig || currentConfig.totalMemoryGB === 0) return null;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const proposedMemoryGB = preset.memoryGB || customMemory;

    // Calculate equivalent cells to maintain same total capacity
    const equivalentCells = Math.round(currentConfig.totalMemoryGB / proposedMemoryGB);

    // Only show if user's cell count is BELOW equivalent (manually reduced)
    if (cellCount >= equivalentCells) return null;

    return {
      equivalentCells,
      proposedMemoryGB,
      currentTotalGB: currentConfig.totalMemoryGB,
    };
  }, [currentConfig, selectedPreset, customMemory, cellCount]);

  // Compute IaaS capacity from loaded data (if available)
  const iaasCapacity = useMemo(() => {
    if (!infrastructureData?.clusters?.length) return null;

    const clusters = infrastructureData.clusters;
    const totalHosts = clusters.reduce((sum, c) => sum + (c.host_count || 0), 0);

    // Only show if we have IaaS-level data (hosts with memory)
    if (totalHosts === 0) return null;

    // Handle both formats: vSphere has memory_gb (total), manual has memory_gb_per_host
    const totalMemoryGB = clusters.reduce((sum, c) => {
      if (c.memory_gb) return sum + c.memory_gb;
      return sum + (c.host_count || 0) * (c.memory_gb_per_host || 0);
    }, 0);

    if (totalMemoryGB === 0) return null;

    const totalCPUCores = clusters.reduce((sum, c) => {
      if (c.cpu_cores) return sum + c.cpu_cores;
      return sum + (c.host_count || 0) * (c.cpu_cores_per_host || 64);
    }, 0);

    // N-1 memory for HA
    const n1MemoryGB = clusters.reduce((sum, c) => {
      if (c.n1_memory_gb) return sum + c.n1_memory_gb;
      const hostCount = c.host_count || 0;
      const memPerHost = c.memory_gb_per_host || (c.memory_gb / hostCount) || 0;
      return sum + ((hostCount - 1) * memPerHost);
    }, 0);

    return {
      totalHosts,
      totalMemoryGB,
      totalCPUCores,
      n1MemoryGB,
    };
  }, [infrastructureData]);

  // Compute max deployable cells based on proposed cell size and IaaS capacity
  const maxCellsEstimate = useMemo(() => {
    if (!iaasCapacity) return null;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const proposedMemoryGB = preset.memoryGB || customMemory;
    const proposedCPU = preset.cpu || customCPU;

    const byMemory = Math.floor(iaasCapacity.n1MemoryGB / proposedMemoryGB);
    const byCPU = Math.floor(iaasCapacity.totalCPUCores / proposedCPU);
    const maxCells = Math.min(byMemory, byCPU);
    const bottleneck = byMemory <= byCPU ? 'memory' : 'cpu';

    return {
      maxCells,
      byMemory,
      byCPU,
      bottleneck,
    };
  }, [iaasCapacity, selectedPreset, customMemory, customCPU]);

  const handleCompare = async () => {
    if (!infrastructureState) return;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const cpu = preset.cpu || customCPU;
    const memory = preset.memoryGB || customMemory;

    setLoading(true);
    setError(null);

    try {
      const scenarioInput = {
        proposed_cell_memory_gb: memory,
        proposed_cell_cpu: cpu,
        proposed_cell_disk_gb: selectedResources.includes('disk') ? customDisk : 0,
        proposed_cell_count: cellCount,
        selected_resources: selectedResources,
        overhead_pct: overheadPct,
        tps_curve: tpsCurve,
      };

      // Add hypothetical app if enabled
      if (useAdditionalApp && additionalApp.name) {
        scenarioInput.additional_app = {
          name: additionalApp.name,
          instances: additionalApp.instances,
          memory_gb: additionalApp.memoryGB,
          disk_gb: additionalApp.diskGB,
        };
      }

      const result = await scenarioApi.compareScenario(scenarioInput);
      setComparison(result);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const preset = VM_SIZE_PRESETS[selectedPreset];

  const handleExportMarkdown = () => {
    if (!comparison || !infrastructureData) return;
    const markdown = generateMarkdownReport(comparison, infrastructureData);
    const filename = `${infrastructureData.name || 'capacity'}-analysis-${new Date().toISOString().split('T')[0]}.md`;
    downloadMarkdown(markdown, filename);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-100 flex items-center gap-3">
          <div className="p-2 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-lg">
            <Sparkles className="text-white" size={20} />
          </div>
          Capacity Planning
        </h2>
      </div>

      <DataSourceSelector
        onDataLoaded={handleDataLoaded}
        currentData={infrastructureData}
      />

      {!infrastructureState && (
        <div className="text-center py-8 text-gray-500">
          <p className="text-sm">Load infrastructure data above to start analyzing scenarios</p>
        </div>
      )}

      {/* IaaS Capacity Section */}
      {iaasCapacity && infrastructureState && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 mb-6">
          <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
            <Server size={18} className="text-cyan-400" />
            IaaS Capacity
            {maxCellsEstimate && (
              <span className="ml-auto text-sm font-normal">
                <span className="text-gray-400">Max Cells:</span>
                <span className="ml-2 text-cyan-400 font-mono font-bold">{maxCellsEstimate.maxCells}</span>
                <span className="text-gray-500 text-xs ml-1">({maxCellsEstimate.bottleneck}-limited)</span>
              </span>
            )}
          </h3>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Server size={14} />
                Hosts
              </div>
              <div className="text-2xl font-mono font-bold text-cyan-400">
                {iaasCapacity.totalHosts}
              </div>
              {infrastructureData?.clusters?.length > 1 && (
                <div className="text-xs text-gray-500 mt-1">
                  across {infrastructureData.clusters.length} clusters
                </div>
              )}
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <HardDrive size={14} />
                Total Memory
              </div>
              <div className="text-2xl font-mono font-bold text-cyan-400">
                {iaasCapacity.totalMemoryGB >= 1000
                  ? `${(iaasCapacity.totalMemoryGB / 1000).toFixed(1)}T`
                  : `${iaasCapacity.totalMemoryGB}G`}
              </div>
              <div className="text-xs text-gray-500 mt-1">
                N-1: {iaasCapacity.n1MemoryGB >= 1000
                  ? `${(iaasCapacity.n1MemoryGB / 1000).toFixed(1)}T`
                  : `${iaasCapacity.n1MemoryGB}G`}
              </div>
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Cpu size={14} />
                Total vCPUs
              </div>
              <div className="text-2xl font-mono font-bold text-cyan-400">
                {iaasCapacity.totalCPUCores}
              </div>
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Calculator size={14} />
                Max Cells
              </div>
              <div className="text-2xl font-mono font-bold text-cyan-400">
                {maxCellsEstimate?.maxCells || '—'}
              </div>
              {maxCellsEstimate && cellCount > maxCellsEstimate.maxCells && (
                <div className="text-xs text-amber-400 mt-1 flex items-center gap-1">
                  <AlertCircle size={12} />
                  Proposed exceeds by {cellCount - maxCellsEstimate.maxCells}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Current Configuration Summary */}
      {currentConfig && infrastructureState && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 mb-6">
          <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
            <Server size={18} className="text-emerald-400" />
            Current Configuration
            <span className="text-xs font-normal text-gray-500 ml-2">
              (from loaded data)
            </span>
          </h3>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Server size={14} />
                Cells
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {currentConfig.totalCells}
              </div>
              {currentConfig.clusterCount > 1 && (
                <div className="text-xs text-gray-500 mt-1">
                  across {currentConfig.clusterCount} clusters
                </div>
              )}
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Cpu size={14} />
                Cell Size
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {currentConfig.cellCpu} <span className="text-gray-500">×</span> {currentConfig.cellMemoryGB}
              </div>
              <div className="text-xs text-gray-500 mt-1">vCPU × GB</div>
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <HardDrive size={14} />
                Total Memory
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {currentConfig.totalMemoryGB >= 1000
                  ? `${(currentConfig.totalMemoryGB / 1000).toFixed(1)}T`
                  : `${currentConfig.totalMemoryGB}G`}
              </div>
              <div className="text-xs text-gray-500 mt-1">
                {currentConfig.totalCells} × {currentConfig.cellMemoryGB}GB
              </div>
            </div>

            {currentConfig.cellDiskGB > 0 && (
              <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
                <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                  <Database size={14} />
                  Total Disk
                </div>
                <div className="text-2xl font-mono font-bold text-emerald-400">
                  {currentConfig.totalDiskGB >= 1000
                    ? `${(currentConfig.totalDiskGB / 1000).toFixed(1)}T`
                    : `${currentConfig.totalDiskGB}G`}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {currentConfig.totalCells} × {currentConfig.cellDiskGB}GB
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Scenario Configuration Wizard */}
      {infrastructureState && (
        <ScenarioWizard
          selectedPreset={selectedPreset}
          setSelectedPreset={setSelectedPreset}
          customCPU={customCPU}
          setCustomCPU={setCustomCPU}
          customMemory={customMemory}
          setCustomMemory={setCustomMemory}
          cellCount={cellCount}
          setCellCount={setCellCount}
          equivalentCellSuggestion={equivalentCellSuggestion}
          selectedResources={selectedResources}
          toggleResource={toggleResource}
          customDisk={customDisk}
          setCustomDisk={setCustomDisk}
          overheadPct={overheadPct}
          setOverheadPct={setOverheadPct}
          useAdditionalApp={useAdditionalApp}
          setUseAdditionalApp={setUseAdditionalApp}
          additionalApp={additionalApp}
          setAdditionalApp={setAdditionalApp}
          tpsCurve={tpsCurve}
          setTPSCurve={setTPSCurve}
          onStepComplete={handleStepComplete}
        />
      )}

      {/* Run Analysis Section - appears after Step 1 completed */}
      {infrastructureState && step1Completed && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-semibold text-gray-200 flex items-center gap-2">
                <Sparkles size={18} className="text-cyan-400" />
                Ready to Analyze
              </h3>
              <p className="text-sm text-gray-400 mt-1">
                {preset.label}, {cellCount} cells | {selectedResources.join(', ')}
                {overheadPct !== 7 && ` | ${overheadPct}% overhead`}
              </p>
            </div>
            <div className="flex items-center gap-3">
              {comparison && (
                <button
                  onClick={handleExportMarkdown}
                  className="flex items-center gap-2 px-4 py-2 bg-slate-700 text-gray-200 rounded-lg hover:bg-slate-600 transition-colors border border-slate-600"
                >
                  <FileDown size={16} />
                  Export
                </button>
              )}
              <button
                onClick={handleCompare}
                disabled={loading}
                className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-cyan-600 to-blue-600 text-white rounded-lg hover:from-cyan-500 hover:to-blue-500 disabled:opacity-50 transition-all font-medium shadow-lg shadow-cyan-500/20"
              >
                {loading ? (
                  <RefreshCw className="animate-spin" size={18} />
                ) : (
                  <Sparkles size={18} />
                )}
                Run Analysis
              </button>
            </div>
          </div>
        </div>
      )}

      {error && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 text-red-300">
          Error: {error}
        </div>
      )}

      {comparison && (
        <ScenarioResults
          comparison={comparison}
          warnings={comparison.warnings}
          selectedResources={selectedResources}
        />
      )}
    </div>
  );
};

export default ScenarioAnalyzer;
