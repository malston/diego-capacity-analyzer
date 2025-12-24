// frontend/src/components/ScenarioAnalyzer.jsx
// ABOUTME: Main what-if scenario analyzer component
// ABOUTME: Combines data source, comparison table, and warnings

import { useState, useEffect, useMemo } from 'react';
import { Calculator, RefreshCw, FileDown, Sparkles, ChevronDown, ChevronUp, Plus, X, Settings2, Server, HardDrive, Cpu, Database, AlertCircle } from 'lucide-react';
import DataSourceSelector from './DataSourceSelector';
import ScenarioResults from './ScenarioResults';
import { scenarioApi } from '../services/scenarioApi';
import { VM_SIZE_PRESETS, DEFAULT_PRESET_INDEX } from '../config/vmPresets';
import { generateMarkdownReport, downloadMarkdown } from '../utils/exportMarkdown';
import {
  RESOURCE_TYPES,
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
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showAppSection, setShowAppSection] = useState(false);
  const [showTPSEditor, setShowTPSEditor] = useState(false);

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

      // Set initial cell count from data
      const totalCells = data.clusters.reduce(
        (sum, c) => sum + c.diego_cell_count,
        0
      );
      setCellCount(totalCells);

      // Set initial disk from first cluster if available
      if (data.clusters[0]?.diego_cell_disk_gb) {
        setCustomDisk(data.clusters[0].diego_cell_disk_gb);
      }
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

  // Calculate equivalent cell count suggestion when VM size changes
  // Only show when user might accidentally reduce capacity
  const equivalentCellSuggestion = useMemo(() => {
    if (!currentConfig || currentConfig.totalMemoryGB === 0) return null;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const proposedMemoryGB = preset.memoryGB || customMemory;

    // If proposed memory is same as current, no suggestion needed
    if (proposedMemoryGB === currentConfig.cellMemoryGB) return null;

    // Calculate equivalent cells to maintain same total capacity
    const equivalentCells = Math.round(currentConfig.totalMemoryGB / proposedMemoryGB);

    // Only show if user's cell count is BELOW equivalent (might be reducing capacity)
    // Don't show if user is clearly expanding (cellCount >= equivalentCells)
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
  const isCustom = preset.cpu === null;

  const handleExportMarkdown = () => {
    if (!comparison || !infrastructureData) return;
    const markdown = generateMarkdownReport(comparison, infrastructureData);
    const filename = `${infrastructureData.name || 'capacity'}-analysis-${new Date().toISOString().split('T')[0]}.md`;
    downloadMarkdown(markdown, filename);
  };

  const updateTPSPoint = (index, field, value) => {
    setTPSCurve(prev => prev.map((pt, i) =>
      i === index ? { ...pt, [field]: parseInt(value) || 0 } : pt
    ));
  };

  const addTPSPoint = () => {
    const lastPt = tpsCurve[tpsCurve.length - 1] || { cells: 0, tps: 0 };
    setTPSCurve([...tpsCurve, { cells: lastPt.cells + 50, tps: Math.max(50, lastPt.tps - 100) }]);
  };

  const removeTPSPoint = (index) => {
    if (tpsCurve.length > 2) {
      setTPSCurve(prev => prev.filter((_, i) => i !== index));
    }
  };

  const resetTPSCurve = () => {
    setTPSCurve(DEFAULT_TPS_CURVE);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-100 flex items-center gap-3">
          <div className="p-2 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-lg">
            <Sparkles className="text-white" size={20} />
          </div>
          What-If Scenario Analysis
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

      {/* IaaS Quick Reference - only shows when IaaS data is available */}
      {maxCellsEstimate && infrastructureState && (
        <div className="bg-slate-800/30 rounded-lg p-4 border border-slate-600/30 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Server size={18} className="text-cyan-400" />
            <div>
              <span className="text-gray-400 text-sm">IaaS Capacity:</span>
              <span className="ml-2 text-cyan-400 font-mono font-bold">{maxCellsEstimate.maxCells}</span>
              <span className="text-gray-400 text-sm ml-1">max cells</span>
              <span className="text-gray-500 text-xs ml-2">
                ({maxCellsEstimate.bottleneck}-limited)
              </span>
            </div>
          </div>
          {cellCount > maxCellsEstimate.maxCells && (
            <div className="flex items-center gap-2 text-amber-400 text-sm">
              <AlertCircle size={16} />
              <span>Exceeds IaaS capacity by {cellCount - maxCellsEstimate.maxCells} cells</span>
            </div>
          )}
        </div>
      )}

      {infrastructureState && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
            <Calculator size={18} className="text-cyan-400" />
            Proposed Configuration
          </h3>

          {/* Resource Type Selection */}
          <div className="mb-6">
            <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
              Resource Types to Analyze
            </label>
            <div className="flex flex-wrap gap-2">
              {RESOURCE_TYPES.map(resource => {
                const Icon = resource.icon;
                const isSelected = selectedResources.includes(resource.id);
                return (
                  <button
                    type="button"
                    key={resource.id}
                    onClick={() => toggleResource(resource.id)}
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

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                VM Size
              </label>
              <select
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
              <>
                <div>
                  <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                    vCPU
                  </label>
                  <input
                    type="number"
                    value={customCPU}
                    onChange={(e) => setCustomCPU(Number(e.target.value))}
                    min={1}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
                  />
                </div>
                <div>
                  <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                    Memory (GB)
                  </label>
                  <input
                    type="number"
                    value={customMemory}
                    onChange={(e) => setCustomMemory(Number(e.target.value))}
                    min={8}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
                  />
                </div>
              </>
            )}

            {/* Disk input - only when disk is selected */}
            {selectedResources.includes('disk') && (
              <div>
                <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                  Disk per Cell (GB)
                </label>
                <input
                  type="number"
                  value={customDisk}
                  onChange={(e) => setCustomDisk(Number(e.target.value))}
                  min={32}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
                />
              </div>
            )}

            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Cell Count
              </label>
              <input
                type="number"
                value={cellCount}
                onChange={(e) => setCellCount(Number(e.target.value))}
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
              />
              {equivalentCellSuggestion && (
                <button
                  type="button"
                  onClick={() => setCellCount(equivalentCellSuggestion.equivalentCells)}
                  className="mt-2 text-xs text-amber-400 hover:text-amber-300 flex items-center gap-1 transition-colors"
                >
                  <Sparkles size={12} />
                  For equivalent capacity ({equivalentCellSuggestion.currentTotalGB}GB): use {equivalentCellSuggestion.equivalentCells} cells
                </button>
              )}
            </div>
          </div>

          {/* Advanced Options Toggle */}
          <div className="border-t border-slate-700 pt-4 mb-4">
            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-2 text-gray-400 hover:text-gray-300 text-sm"
            >
              <Settings2 size={16} />
              Advanced Options
              {showAdvanced ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </button>
          </div>

          {showAdvanced && (
            <div className="space-y-4 mb-6 pl-4 border-l-2 border-slate-700">
              {/* Overhead Percentage */}
              <div>
                <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                  Memory Overhead: {overheadPct}%
                </label>
                <input
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
                <button
                  onClick={() => setShowAppSection(!showAppSection)}
                  className="flex items-center gap-2 text-gray-300 hover:text-gray-200 text-sm font-medium w-full justify-between"
                >
                  <span className="flex items-center gap-2">
                    <Plus size={16} />
                    Add Hypothetical App
                  </span>
                  {showAppSection ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </button>

                {showAppSection && (
                  <div className="mt-4 space-y-3">
                    <div className="flex items-center gap-2 mb-3">
                      <input
                        type="checkbox"
                        id="useApp"
                        checked={useAdditionalApp}
                        onChange={(e) => setUseAdditionalApp(e.target.checked)}
                        className="rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500"
                      />
                      <label htmlFor="useApp" className="text-sm text-gray-300">
                        Include this app in analysis
                      </label>
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                      <div className="col-span-2">
                        <label className="block text-xs text-gray-400 mb-1">App Name</label>
                        <input
                          type="text"
                          value={additionalApp.name}
                          onChange={(e) => setAdditionalApp({ ...additionalApp, name: e.target.value })}
                          className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm focus:border-cyan-500 outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-400 mb-1">Instances</label>
                        <input
                          type="number"
                          value={additionalApp.instances}
                          onChange={(e) => setAdditionalApp({ ...additionalApp, instances: Number(e.target.value) })}
                          min={1}
                          className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-400 mb-1">Memory/Instance (GB)</label>
                        <input
                          type="number"
                          value={additionalApp.memoryGB}
                          onChange={(e) => setAdditionalApp({ ...additionalApp, memoryGB: Number(e.target.value) })}
                          min={1}
                          className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-400 mb-1">Disk/Instance (GB)</label>
                        <input
                          type="number"
                          value={additionalApp.diskGB}
                          onChange={(e) => setAdditionalApp({ ...additionalApp, diskGB: Number(e.target.value) })}
                          min={1}
                          className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                        />
                      </div>
                    </div>

                    {useAdditionalApp && additionalApp.name && (
                      <div className="text-xs text-cyan-400 mt-2">
                        Adding: {additionalApp.instances} × {additionalApp.memoryGB}GB RAM, {additionalApp.diskGB}GB disk
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* TPS Curve Editor */}
              <div className="bg-slate-700/30 rounded-lg p-4">
                <button
                  onClick={() => setShowTPSEditor(!showTPSEditor)}
                  className="flex items-center gap-2 text-gray-300 hover:text-gray-200 text-sm font-medium w-full justify-between"
                >
                  <span className="flex items-center gap-2">
                    <Settings2 size={16} />
                    TPS Performance Curve
                  </span>
                  {showTPSEditor ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </button>

                {showTPSEditor && (
                  <div className="mt-4">
                    <p className="text-xs text-gray-400 mb-3">
                      Default values are baseline estimates. Actual TPS varies by infrastructure, network, and database backend. Customize to match your observed scheduler performance.
                    </p>

                    <div className="space-y-2 mb-3">
                      {tpsCurve.map((pt, i) => (
                        <div key={i} className="flex items-center gap-2">
                          <input
                            type="number"
                            value={pt.cells}
                            onChange={(e) => updateTPSPoint(i, 'cells', e.target.value)}
                            placeholder="Cells"
                            className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                          />
                          <span className="text-gray-500">cells →</span>
                          <input
                            type="number"
                            value={pt.tps}
                            onChange={(e) => updateTPSPoint(i, 'tps', e.target.value)}
                            placeholder="TPS"
                            className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
                          />
                          <span className="text-gray-500">TPS</span>
                          {tpsCurve.length > 2 && (
                            <button
                              onClick={() => removeTPSPoint(i)}
                              className="text-red-400 hover:text-red-300 p-1"
                            >
                              <X size={14} />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>

                    <div className="flex gap-2">
                      <button
                        onClick={addTPSPoint}
                        className="text-xs text-cyan-400 hover:text-cyan-300 flex items-center gap-1"
                      >
                        <Plus size={12} /> Add Point
                      </button>
                      <button
                        onClick={resetTPSCurve}
                        className="text-xs text-gray-400 hover:text-gray-300"
                      >
                        Reset to Default
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
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
      )}

      {error && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 text-red-300">
          Error: {error}
        </div>
      )}

      {comparison && (
        <>
          <div className="flex justify-end mb-4">
            <button
              onClick={handleExportMarkdown}
              className="flex items-center gap-2 px-4 py-2 bg-slate-700 text-gray-200 rounded-lg hover:bg-slate-600 transition-colors border border-slate-600"
            >
              <FileDown size={16} />
              Export Report
            </button>
          </div>
          <ScenarioResults
            comparison={comparison}
            warnings={comparison.warnings}
            selectedResources={selectedResources}
          />
        </>
      )}
    </div>
  );
};

export default ScenarioAnalyzer;
