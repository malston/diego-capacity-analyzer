// frontend/src/components/DataSourceSelector.jsx
// ABOUTME: Data source selector for infrastructure input
// ABOUTME: Supports JSON upload and manual form entry

import React, { useState, useRef } from 'react';
import { Upload, FileText, Edit3 } from 'lucide-react';

const DataSourceSelector = ({ onDataLoaded, currentData }) => {
  const [mode, setMode] = useState('upload'); // 'upload' | 'manual'
  const [error, setError] = useState(null);
  const fileInputRef = useRef(null);

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

      // Create ManualInput JSON
      const manualInput = {
        name: formData.name.trim(),
        clusters: [
          {
            name: formData.name.trim(),
            host_count: parseInt(formData.hostCount),
            ram_per_host_gb: parseInt(formData.ramPerHost),
            cpu_cores_per_host: parseInt(formData.cpuCoresPerHost) || 64,
            diego_cell_count: parseInt(formData.diegoCellCount),
            diego_cell_memory_gb: parseInt(formData.cellMemory),
            diego_cell_vcpu: parseInt(formData.cellVCpu),
            platform_vms_memory_gb: formData.platformVMs ? parseInt(formData.platformVMs) : 0,
            total_app_memory_gb: formData.totalAppMemory ? parseInt(formData.totalAppMemory) : 0,
            app_instance_count: formData.appInstances ? parseInt(formData.appInstances) : 0
          }
        ]
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
    <div className="bg-white rounded-lg shadow p-4 mb-4">
      <h3 className="text-lg font-semibold mb-3">Infrastructure Data Source</h3>

      <div className="flex gap-4 mb-4">
        <button
          onClick={() => setMode('upload')}
          className={`flex items-center gap-2 px-4 py-2 rounded ${
            mode === 'upload' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          <Upload size={16} />
          Upload JSON
        </button>
        <button
          onClick={() => setMode('manual')}
          className={`flex items-center gap-2 px-4 py-2 rounded ${
            mode === 'manual' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          <Edit3 size={16} />
          Manual Entry
        </button>
        {currentData && (
          <button
            onClick={handleExport}
            className="flex items-center gap-2 px-4 py-2 rounded bg-gray-100 ml-auto"
          >
            <FileText size={16} />
            Export
          </button>
        )}
      </div>

      {mode === 'upload' && (
        <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
          <input
            type="file"
            accept=".json"
            onChange={handleFileUpload}
            ref={fileInputRef}
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            className="text-blue-600 hover:text-blue-800"
          >
            Click to upload JSON file
          </button>
          <p className="text-sm text-gray-500 mt-2">
            or drag and drop
          </p>
        </div>
      )}

      {mode === 'manual' && (
        <form onSubmit={handleManualSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Environment Name *
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={handleInputChange}
              className="w-full border border-gray-300 rounded px-3 py-2"
              placeholder="e.g., Production"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Host Count *
              </label>
              <input
                type="number"
                name="hostCount"
                value={formData.hostCount}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="e.g., 10"
                min="1"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                RAM per Host (GB) *
              </label>
              <input
                type="number"
                name="ramPerHost"
                value={formData.ramPerHost}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="e.g., 512"
                min="1"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                CPU Cores per Host
              </label>
              <input
                type="number"
                name="cpuCoresPerHost"
                value={formData.cpuCoresPerHost}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="Default: 64"
                min="1"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Diego Cell Count *
              </label>
              <input
                type="number"
                name="diegoCellCount"
                value={formData.diegoCellCount}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="e.g., 30"
                min="1"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Cell Memory (GB) *
              </label>
              <select
                name="cellMemory"
                value={formData.cellMemory}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                required
              >
                <option value="32">32 GB</option>
                <option value="64">64 GB</option>
                <option value="128">128 GB</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Cell vCPU *
              </label>
              <select
                name="cellVCpu"
                value={formData.cellVCpu}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                required
              >
                <option value="4">4 vCPU</option>
                <option value="8">8 vCPU</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Platform VMs Memory (GB)
              </label>
              <input
                type="number"
                name="platformVMs"
                value={formData.platformVMs}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="Optional"
                min="0"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Total App Memory (GB)
              </label>
              <input
                type="number"
                name="totalAppMemory"
                value={formData.totalAppMemory}
                onChange={handleInputChange}
                className="w-full border border-gray-300 rounded px-3 py-2"
                placeholder="Optional"
                min="0"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              App Instances
            </label>
            <input
              type="number"
              name="appInstances"
              value={formData.appInstances}
              onChange={handleInputChange}
              className="w-full border border-gray-300 rounded px-3 py-2"
              placeholder="Optional"
              min="0"
            />
          </div>

          <button
            type="submit"
            className="w-full bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
          >
            Create Environment
          </button>
        </form>
      )}

      {error && (
        <p className="text-red-600 text-sm mt-2">{error}</p>
      )}

      {currentData && (
        <div className="mt-4 p-3 bg-gray-50 rounded text-sm">
          <p className="font-medium">{currentData.name}</p>
          <p className="text-gray-600">
            {currentData.clusters?.length || 0} clusters, {' '}
            {currentData.clusters?.reduce((sum, c) => sum + c.host_count, 0) || 0} hosts, {' '}
            {currentData.clusters?.reduce((sum, c) => sum + c.diego_cell_count, 0) || 0} cells
          </p>
        </div>
      )}
    </div>
  );
};

export default DataSourceSelector;
