// frontend/src/components/DataSourceSelector.jsx
// ABOUTME: Data source selector for infrastructure input
// ABOUTME: Supports JSON upload and manual form entry

import React, { useState, useRef } from 'react';
import { Upload, FileText, Edit3 } from 'lucide-react';

const DataSourceSelector = ({ onDataLoaded, currentData }) => {
  const [mode, setMode] = useState('upload'); // 'upload' | 'manual'
  const [error, setError] = useState(null);
  const fileInputRef = useRef(null);

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
