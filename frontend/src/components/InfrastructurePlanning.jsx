// frontend/src/components/InfrastructurePlanning.jsx
// ABOUTME: Infrastructure planning component for calculating max deployable cells
// ABOUTME: Shows cell count constrained by memory and CPU with recommendations

import { useState, useEffect, useMemo } from 'react';
import { Server, Cpu, HardDrive, Calculator, RefreshCw, AlertCircle, CheckCircle, Lightbulb } from 'lucide-react';
import DataSourceSelector from './DataSourceSelector';
import { scenarioApi } from '../services/scenarioApi';
import { VM_SIZE_PRESETS, DEFAULT_PRESET_INDEX } from '../config/vmPresets';

const InfrastructurePlanning = () => {
  const [infrastructureData, setInfrastructureData] = useState(null);
  const [infrastructureState, setInfrastructureState] = useState(null);
  const [planningResult, setPlanningResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Input state
  const [selectedPreset, setSelectedPreset] = useState(DEFAULT_PRESET_INDEX);
  const [customCPU, setCustomCPU] = useState(4);
  const [customMemory, setCustomMemory] = useState(32);

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
    setPlanningResult(null);

    try {
      // If data is already InfrastructureState (from vSphere), use the state endpoint
      // For manual input, use the manual endpoint which converts to state
      if (data.source === 'vsphere' || data.source === 'manual') {
        // Data is already InfrastructureState format
        const state = await scenarioApi.setInfrastructureState(data);
        setInfrastructureState(state);
      } else {
        // Data is ManualInput format, convert to state on backend
        const state = await scenarioApi.setManualInfrastructure(data);
        setInfrastructureState(state);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Compute IaaS capacity from loaded data
  // Handles both manual input (memory_gb_per_host) and vSphere (memory_gb total)
  const iaasCapacity = useMemo(() => {
    if (!infrastructureData?.clusters?.length) return null;

    const clusters = infrastructureData.clusters;
    const totalHosts = clusters.reduce((sum, c) => sum + (c.host_count || 0), 0);

    // Handle both formats: vSphere has memory_gb (total), manual has memory_gb_per_host
    const totalMemoryGB = clusters.reduce((sum, c) => {
      if (c.memory_gb) return sum + c.memory_gb;
      return sum + (c.host_count || 0) * (c.memory_gb_per_host || 0);
    }, 0);

    const totalCPUCores = clusters.reduce((sum, c) => {
      if (c.cpu_cores) return sum + c.cpu_cores;
      return sum + (c.host_count || 0) * (c.cpu_cores_per_host || 64);
    }, 0);

    // N-1 memory: use n1_memory_gb if available (vSphere), otherwise calculate
    const n1MemoryGB = clusters.reduce((sum, c) => {
      if (c.n1_memory_gb) return sum + c.n1_memory_gb;
      const hostCount = c.host_count || 0;
      const memPerHost = c.memory_gb_per_host || 0;
      return sum + ((hostCount - 1) * memPerHost);
    }, 0);

    return {
      totalHosts,
      totalMemoryGB,
      totalCPUCores,
      n1MemoryGB,
      clusterCount: clusters.length,
    };
  }, [infrastructureData]);

  const handleCalculate = async () => {
    if (!infrastructureState) return;

    const preset = VM_SIZE_PRESETS[selectedPreset];
    const cpu = preset.cpu || customCPU;
    const memory = preset.memoryGB || customMemory;

    setLoading(true);
    setError(null);

    try {
      const result = await scenarioApi.calculatePlanning({
        cell_memory_gb: memory,
        cell_cpu: cpu,
      });
      setPlanningResult(result);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const preset = VM_SIZE_PRESETS[selectedPreset];
  const isCustom = preset.cpu === null;

  const getBottleneckColor = (bottleneck) => {
    switch (bottleneck) {
      case 'memory': return 'text-amber-400';
      case 'cpu': return 'text-orange-400';
      case 'balanced': return 'text-emerald-400';
      default: return 'text-gray-400';
    }
  };

  const getBottleneckIcon = (bottleneck) => {
    switch (bottleneck) {
      case 'memory': return <HardDrive size={16} />;
      case 'cpu': return <Cpu size={16} />;
      case 'balanced': return <CheckCircle size={16} />;
      default: return <AlertCircle size={16} />;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-100 flex items-center gap-3">
          <div className="p-2 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg">
            <Server className="text-white" size={20} />
          </div>
          Infrastructure Planning
        </h2>
      </div>

      <DataSourceSelector
        onDataLoaded={handleDataLoaded}
        currentData={infrastructureData}
      />

      {!infrastructureState && (
        <div className="text-center py-8 text-gray-500">
          <p className="text-sm">Load infrastructure data above to calculate cell capacity</p>
        </div>
      )}

      {/* IaaS Capacity Summary */}
      {iaasCapacity && infrastructureState && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
            <Server size={18} className="text-emerald-400" />
            IaaS Capacity
          </h3>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Server size={14} />
                Hosts
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {iaasCapacity.totalHosts}
              </div>
              {iaasCapacity.clusterCount > 1 && (
                <div className="text-xs text-gray-500 mt-1">
                  across {iaasCapacity.clusterCount} clusters
                </div>
              )}
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <HardDrive size={14} />
                Total Memory
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
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
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {iaasCapacity.totalCPUCores}
              </div>
            </div>

            <div className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30">
              <div className="flex items-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-2">
                <Calculator size={14} />
                Clusters
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-400">
                {iaasCapacity.clusterCount}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cell Configuration */}
      {infrastructureState && (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
          <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
            <Calculator size={18} className="text-teal-400" />
            Cell Configuration
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                VM Size
              </label>
              <select
                value={selectedPreset}
                onChange={(e) => setSelectedPreset(Number(e.target.value))}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 outline-none transition-colors"
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
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 outline-none transition-colors"
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
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 outline-none transition-colors"
                  />
                </div>
              </>
            )}
          </div>

          <button
            onClick={handleCalculate}
            disabled={loading}
            className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-emerald-600 to-teal-600 text-white rounded-lg hover:from-emerald-500 hover:to-teal-500 disabled:opacity-50 transition-all font-medium shadow-lg shadow-emerald-500/20"
          >
            {loading ? (
              <RefreshCw className="animate-spin" size={18} />
            ) : (
              <Calculator size={18} />
            )}
            Calculate Cell Capacity
          </button>
        </div>
      )}

      {error && (
        <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 text-red-300">
          Error: {error}
        </div>
      )}

      {/* Results */}
      {planningResult && (
        <div className="space-y-6">
          {/* Main Result */}
          <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
            <h3 className="text-lg font-semibold mb-6 text-gray-200">Calculation Results</h3>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
              {/* By Memory */}
              <div className="bg-slate-700/30 rounded-lg p-5 border border-slate-600/30 text-center">
                <div className="flex items-center justify-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-3">
                  <HardDrive size={14} />
                  By Memory
                </div>
                <div className="text-3xl font-mono font-bold text-blue-400">
                  {planningResult.result.max_cells_by_memory}
                </div>
                <div className="text-sm text-gray-500 mt-2">cells possible</div>
              </div>

              {/* By CPU */}
              <div className="bg-slate-700/30 rounded-lg p-5 border border-slate-600/30 text-center">
                <div className="flex items-center justify-center gap-2 text-gray-400 text-xs uppercase tracking-wider mb-3">
                  <Cpu size={14} />
                  By CPU
                </div>
                <div className="text-3xl font-mono font-bold text-purple-400">
                  {planningResult.result.max_cells_by_cpu}
                </div>
                <div className="text-sm text-gray-500 mt-2">cells possible</div>
              </div>

              {/* Deployable */}
              <div className="bg-gradient-to-br from-emerald-900/30 to-teal-900/30 rounded-lg p-5 border border-emerald-600/30 text-center">
                <div className="flex items-center justify-center gap-2 text-emerald-400 text-xs uppercase tracking-wider mb-3">
                  <CheckCircle size={14} />
                  Deployable
                </div>
                <div className="text-4xl font-mono font-bold text-emerald-400">
                  {planningResult.result.deployable_cells}
                </div>
                <div className={`text-sm mt-2 flex items-center justify-center gap-1 ${getBottleneckColor(planningResult.result.bottleneck)}`}>
                  {getBottleneckIcon(planningResult.result.bottleneck)}
                  {planningResult.result.bottleneck === 'balanced' ? 'Balanced' : `${planningResult.result.bottleneck}-constrained`}
                </div>
              </div>
            </div>

            {/* Utilization Bars */}
            <div className="space-y-4">
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-gray-400">Memory Utilization</span>
                  <span className="text-gray-300 font-mono">{planningResult.result.memory_util_pct.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-slate-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-500 to-blue-400 rounded-full transition-all duration-500"
                    style={{ width: `${Math.min(planningResult.result.memory_util_pct, 100)}%` }}
                  />
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {planningResult.result.memory_used_gb} GB / {planningResult.result.memory_avail_gb} GB
                </div>
              </div>

              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-gray-400">CPU Utilization</span>
                  <span className="text-gray-300 font-mono">{planningResult.result.cpu_util_pct.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-slate-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-purple-500 to-purple-400 rounded-full transition-all duration-500"
                    style={{ width: `${Math.min(planningResult.result.cpu_util_pct, 100)}%` }}
                  />
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {planningResult.result.cpu_used} vCPU / {planningResult.result.cpu_avail} vCPU
                </div>
              </div>
            </div>

            {planningResult.result.headroom_cells > 0 && (
              <div className="mt-4 p-3 bg-slate-700/30 rounded-lg border border-slate-600/30">
                <p className="text-sm text-gray-400">
                  <span className="text-amber-400 font-medium">{planningResult.result.headroom_cells}</span> cells worth of {planningResult.result.bottleneck === 'memory' ? 'CPU' : 'memory'} capacity unused
                </p>
              </div>
            )}
          </div>

          {/* Sizing Recommendations */}
          {planningResult.recommendations && planningResult.recommendations.length > 0 && (
            <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
              <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
                <Lightbulb size={18} className="text-amber-400" />
                Sizing Recommendations
              </h3>

              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-gray-400 text-xs uppercase tracking-wider border-b border-slate-700">
                      <th className="py-3 px-4 text-left">Cell Size</th>
                      <th className="py-3 px-4 text-right">Cells</th>
                      <th className="py-3 px-4 text-left">Bottleneck</th>
                      <th className="py-3 px-4 text-right">Memory %</th>
                      <th className="py-3 px-4 text-right">CPU %</th>
                    </tr>
                  </thead>
                  <tbody>
                    {planningResult.recommendations.map((rec, i) => (
                      <tr key={i} className="border-b border-slate-700/50 hover:bg-slate-700/20">
                        <td className="py-3 px-4 font-mono text-gray-200">{rec.label}</td>
                        <td className="py-3 px-4 text-right font-mono font-bold text-emerald-400">
                          {rec.deployable_cells}
                        </td>
                        <td className={`py-3 px-4 ${getBottleneckColor(rec.bottleneck)}`}>
                          <span className="flex items-center gap-1">
                            {getBottleneckIcon(rec.bottleneck)}
                            {rec.bottleneck}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-right font-mono text-gray-300">
                          {rec.memory_util_pct.toFixed(0)}%
                        </td>
                        <td className="py-3 px-4 text-right font-mono text-gray-300">
                          {rec.cpu_util_pct.toFixed(0)}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <p className="text-xs text-gray-500 mt-4">
                Choose smaller cells for more fault tolerance and granular scaling, or larger cells for fewer VMs to manage.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default InfrastructurePlanning;
