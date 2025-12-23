// frontend/src/components/DataSourceSelector.jsx
// ABOUTME: Data source selector for infrastructure input
// ABOUTME: Supports live vSphere, JSON upload, and manual form entry

import { useState, useRef, useEffect } from 'react';
import { Upload, FileText, Edit3, RefreshCw, Server, FolderOpen } from 'lucide-react';
import { scenarioApi } from '../services/scenarioApi';

const SAMPLE_FILES = [
  { name: 'Small Foundation (Dev/Test)', file: 'small-foundation.json' },
  { name: 'Medium Foundation (Staging)', file: 'medium-foundation.json' },
  { name: 'Large Foundation (Production)', file: 'large-foundation.json' },
  { name: 'Enterprise Multi-Cluster', file: 'multi-cluster-enterprise.json' },
  { name: 'Diego Benchmark 50K', file: 'diego-benchmark-50k.json' },
  { name: 'Diego Benchmark 250K', file: 'diego-benchmark-250k.json' },
];

const DataSourceSelector = ({ onDataLoaded, currentData }) => {
  const [mode, setMode] = useState('upload'); // 'live' | 'upload' | 'manual'
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [vsphereConfigured, setVsphereConfigured] = useState(false);
  const fileInputRef = useRef(null);

  // Check if vSphere is configured on mount
  useEffect(() => {
    const checkVsphereStatus = async () => {
      try {
        const status = await scenarioApi.getInfrastructureStatus();
        setVsphereConfigured(status.vsphere_configured);
      } catch (err) {
        console.warn('Could not check vSphere status:', err);
      }
    };
    checkVsphereStatus();
  }, []);

  // Handle mode selection - auto-fetch for live mode
  const handleModeSelect = async (newMode) => {
    setMode(newMode);
    setError(null);

    // Auto-fetch when selecting live mode
    if (newMode === 'live') {
      await handleFetchLive();
    }
  };

  // Manual entry form state
  const [formData, setFormData] = useState({
    name: '',
    hostCount: '',
    ramPerHost: '',
    cpuCoresPerHost: '64',
    diegoCellCount: '',
    cellMemory: '64',
    cellVCpu: '8',
    platformVMs: '',
    totalAppMemory: '',
    appInstances: ''
  });

  const handleFileUpload = (event) => {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const data = JSON.parse(e.target.result);
        validateManualInput(data);
        onDataLoaded(data);
        setError(null);
        // Store in localStorage for persistence
        localStorage.setItem('scenario-infrastructure', JSON.stringify(data));
      } catch (err) {
        setError(`Invalid JSON: ${err.message}`);
      }
    };
    reader.readAsText(file);
  };

  const validateManualInput = (data) => {
    if (!data.name) throw new Error('Missing "name" field');
    if (!data.clusters || !Array.isArray(data.clusters)) {
      throw new Error('Missing or invalid "clusters" array');
    }
    if (data.clusters.length === 0) {
      throw new Error('At least one cluster required');
    }
    for (const c of data.clusters) {
      if (!c.host_count || !c.diego_cell_count || !c.diego_cell_memory_gb) {
        throw new Error('Cluster missing required fields');
      }
    }
  };

  const handleExport = () => {
    if (!currentData) return;
    const blob = new Blob([JSON.stringify(currentData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${currentData.name || 'infrastructure'}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleFetchLive = async () => {
    setLoading(true);
    setError(null);
    try {
      const state = await scenarioApi.getLiveInfrastructure();
      onDataLoaded(state);
      // Store in localStorage for persistence
      localStorage.setItem('scenario-infrastructure', JSON.stringify(state));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleLoadSample = async (filename) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`/samples/${filename}`);
      if (!response.ok) {
        throw new Error('Failed to load sample file');
      }
      const data = await response.json();
      validateManualInput(data);
      onDataLoaded(data);
      localStorage.setItem('scenario-infrastructure', JSON.stringify(data));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    setError(null);
  };

  const handleManualSubmit = (e) => {
    e.preventDefault();
    try {
      // Validate required fields
      if (!formData.name.trim()) throw new Error('Environment name is required');
      if (!formData.hostCount || parseInt(formData.hostCount) <= 0) {
        throw new Error('Host count must be greater than 0');
      }
      if (!formData.ramPerHost || parseInt(formData.ramPerHost) <= 0) {
        throw new Error('RAM per host must be greater than 0');
      }
      if (!formData.diegoCellCount || parseInt(formData.diegoCellCount) <= 0) {
        throw new Error('Diego cell count must be greater than 0');
      }

      // Create ManualInput JSON matching backend model expectations
      const manualInput = {
        name: formData.name.trim(),
        clusters: [
          {
            name: formData.name.trim(),
            host_count: parseInt(formData.hostCount),
            memory_gb_per_host: parseInt(formData.ramPerHost),
            cpu_cores_per_host: parseInt(formData.cpuCoresPerHost) || 64,
            diego_cell_count: parseInt(formData.diegoCellCount),
            diego_cell_memory_gb: parseInt(formData.cellMemory),
            diego_cell_cpu: parseInt(formData.cellVCpu),
            diego_cell_disk_gb: 0
          }
        ],
        platform_vms_gb: formData.platformVMs ? parseInt(formData.platformVMs) : 0,
        total_app_memory_gb: formData.totalAppMemory ? parseInt(formData.totalAppMemory) : 0,
        total_app_disk_gb: 0,
        total_app_instances: formData.appInstances ? parseInt(formData.appInstances) : 0
      };

      validateManualInput(manualInput);
      onDataLoaded(manualInput);
      setError(null);
      // Store in localStorage for persistence
      localStorage.setItem('scenario-infrastructure', JSON.stringify(manualInput));
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 mb-4">
      <h3 className="text-lg font-semibold mb-4 text-gray-200">Infrastructure Data Source</h3>

      <div className="flex gap-3 mb-4 flex-wrap">
        {vsphereConfigured && (
          <button
            onClick={() => handleModeSelect('live')}
            disabled={loading && mode === 'live'}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-lg transition-colors disabled:opacity-70 ${
              mode === 'live'
                ? 'bg-emerald-600 text-white'
                : 'bg-slate-700 text-gray-300 hover:bg-slate-600 border border-slate-600'
            }`}
          >
            {loading && mode === 'live' ? (
              <RefreshCw size={16} className="animate-spin" />
            ) : (
              <Server size={16} />
            )}
            {loading && mode === 'live' ? 'Connecting...' : 'Live (vSphere)'}
          </button>
        )}
        <button
          onClick={() => handleModeSelect('upload')}
          className={`flex items-center gap-2 px-4 py-2.5 rounded-lg transition-colors ${
            mode === 'upload'
              ? 'bg-cyan-600 text-white'
              : 'bg-slate-700 text-gray-300 hover:bg-slate-600 border border-slate-600'
          }`}
        >
          <Upload size={16} />
          Upload JSON
        </button>
        <button
          onClick={() => handleModeSelect('manual')}
          className={`flex items-center gap-2 px-4 py-2.5 rounded-lg transition-colors ${
            mode === 'manual'
              ? 'bg-cyan-600 text-white'
              : 'bg-slate-700 text-gray-300 hover:bg-slate-600 border border-slate-600'
          }`}
        >
          <Edit3 size={16} />
          Manual Entry
        </button>
        {currentData && (
          <button
            onClick={handleExport}
            className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-slate-700 text-gray-300 hover:bg-slate-600 border border-slate-600 transition-colors ml-auto"
          >
            <FileText size={16} />
            Export
          </button>
        )}
      </div>

      {mode === 'live' && loading && (
        <div className="border border-emerald-700/30 bg-emerald-900/20 rounded-lg p-6">
          <div className="flex items-center justify-center gap-3 text-emerald-300">
            <RefreshCw size={20} className="animate-spin" />
            <span>Connecting to vSphere...</span>
          </div>
          <p className="text-sm text-gray-500 mt-2 text-center">
            Discovering clusters, hosts, and Diego cells
          </p>
        </div>
      )}

      {mode === 'upload' && (
        <div className="space-y-4">
          <div className="border-2 border-dashed border-slate-600 rounded-lg p-6 text-center bg-slate-700/30">
            <input
              type="file"
              accept=".json"
              onChange={handleFileUpload}
              ref={fileInputRef}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              className="text-cyan-400 hover:text-cyan-300 font-medium"
            >
              Click to upload JSON file
            </button>
            <p className="text-sm text-gray-500 mt-2">
              or drag and drop
            </p>
          </div>

          <div className="border-t border-slate-700 pt-4">
            <p className="text-sm font-medium text-gray-400 mb-2 flex items-center gap-2">
              <FolderOpen size={16} />
              Or load a sample configuration:
            </p>
            <div className="grid grid-cols-2 gap-2">
              {SAMPLE_FILES.map((sample) => (
                <button
                  key={sample.file}
                  onClick={() => handleLoadSample(sample.file)}
                  disabled={loading}
                  className="text-left px-3 py-2 text-sm bg-slate-700/50 text-gray-300 hover:bg-slate-600/50 rounded border border-slate-600 disabled:opacity-50 transition-colors"
                >
                  {sample.name}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {mode === 'manual' && (
        <form onSubmit={handleManualSubmit} className="space-y-4">
          <div>
            <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
              Environment Name *
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={handleInputChange}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
              placeholder="e.g., Production"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Host Count *
              </label>
              <input
                type="number"
                name="hostCount"
                value={formData.hostCount}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="e.g., 10"
                min="1"
                required
              />
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                RAM per Host (GB) *
              </label>
              <input
                type="number"
                name="ramPerHost"
                value={formData.ramPerHost}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="e.g., 512"
                min="1"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                CPU Cores per Host
              </label>
              <input
                type="number"
                name="cpuCoresPerHost"
                value={formData.cpuCoresPerHost}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="Default: 64"
                min="1"
              />
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Diego Cell Count *
              </label>
              <input
                type="number"
                name="diegoCellCount"
                value={formData.diegoCellCount}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="e.g., 30"
                min="1"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Cell Memory (GB) *
              </label>
              <select
                name="cellMemory"
                value={formData.cellMemory}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                required
              >
                <option value="32">32 GB</option>
                <option value="64">64 GB</option>
                <option value="128">128 GB</option>
              </select>
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Cell vCPU *
              </label>
              <select
                name="cellVCpu"
                value={formData.cellVCpu}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                required
              >
                <option value="4">4 vCPU</option>
                <option value="8">8 vCPU</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Platform VMs Memory (GB)
              </label>
              <input
                type="number"
                name="platformVMs"
                value={formData.platformVMs}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="Optional"
                min="0"
              />
            </div>

            <div>
              <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
                Total App Memory (GB)
              </label>
              <input
                type="number"
                name="totalAppMemory"
                value={formData.totalAppMemory}
                onChange={handleInputChange}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                placeholder="Optional"
                min="0"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2">
              App Instances
            </label>
            <input
              type="number"
              name="appInstances"
              value={formData.appInstances}
              onChange={handleInputChange}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono placeholder-gray-500 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
              placeholder="Optional"
              min="0"
            />
          </div>

          <button
            type="submit"
            className="w-full bg-gradient-to-r from-cyan-600 to-blue-600 text-white px-4 py-3 rounded-lg hover:from-cyan-500 hover:to-blue-500 transition-all font-medium"
          >
            Create Environment
          </button>
        </form>
      )}

      {error && (
        <p className="text-red-400 text-sm mt-2">{error}</p>
      )}

      {currentData && (
        <div className="mt-4 p-4 bg-slate-700/50 rounded-lg border border-slate-600">
          <div className="flex items-center justify-between">
            <p className="font-medium text-gray-200">{currentData.name}</p>
            {currentData.source && (
              <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                currentData.source === 'vsphere'
                  ? 'bg-emerald-900/50 text-emerald-400'
                  : 'bg-cyan-900/50 text-cyan-400'
              }`}>
                {currentData.source === 'vsphere' ? 'Live' : 'Manual'}
              </span>
            )}
          </div>
          <p className="text-gray-400 text-sm mt-1">
            {currentData.clusters?.length || 0} clusters, {' '}
            {currentData.clusters?.reduce((sum, c) => sum + c.host_count, 0) || 0} hosts, {' '}
            {currentData.clusters?.reduce((sum, c) => sum + c.diego_cell_count, 0) || 0} cells
          </p>
          {currentData.cached && (
            <p className="text-xs text-gray-500 mt-1">Cached data</p>
          )}
        </div>
      )}
    </div>
  );
};

export default DataSourceSelector;
