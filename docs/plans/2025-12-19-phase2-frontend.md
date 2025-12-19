# Phase 2: Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add what-if scenario analysis UI to the existing TAS Capacity Analyzer dashboard with manual data input, comparison table, and warning display.

**Architecture:** New React components integrated into existing TASCapacityAnalyzer.jsx. Uses existing Tailwind CSS styling and Recharts for visualization. Calls Phase 1 backend APIs.

**Tech Stack:** React 18, Vite 5, Tailwind CSS, lucide-react icons

---

## Task 1: API Service for Scenario Analysis

**Files:**
- Create: `frontend/src/services/scenarioApi.js`

**Step 1: Create API service**

```javascript
// frontend/src/services/scenarioApi.js
// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const scenarioApi = {
  /**
   * Submit manual infrastructure data
   * @param {Object} data - ManualInput object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setManualInfrastructure(data) {
    const response = await fetch(`${API_URL}/api/infrastructure/manual`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to set infrastructure');
    }
    return response.json();
  },

  /**
   * Compare current vs proposed scenario
   * @param {Object} input - ScenarioInput object
   * @returns {Promise<Object>} ScenarioComparison
   */
  async compareScenario(input) {
    const response = await fetch(`${API_URL}/api/scenario/compare`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to compare scenario');
    }
    return response.json();
  },
};
```

**Step 2: Verify file created**

Run: `cat frontend/src/services/scenarioApi.js`

**Step 3: Commit**

```bash
git add frontend/src/services/scenarioApi.js
git commit -m "feat: add scenario API service"
```

---

## Task 2: VM Size Presets Configuration

**Files:**
- Create: `frontend/src/config/vmPresets.js`

**Step 1: Create presets configuration**

```javascript
// frontend/src/config/vmPresets.js
// ABOUTME: VM size presets for Diego cell what-if analysis
// ABOUTME: Common configurations used in TAS deployments

export const VM_SIZE_PRESETS = [
  { label: '4 vCPU × 32 GB', cpu: 4, memoryGB: 32 },
  { label: '4 vCPU × 64 GB', cpu: 4, memoryGB: 64 },
  { label: '8 vCPU × 64 GB', cpu: 8, memoryGB: 64 },
  { label: '8 vCPU × 128 GB', cpu: 8, memoryGB: 128 },
  { label: 'Custom...', cpu: null, memoryGB: null },
];

export const DEFAULT_PRESET_INDEX = 0; // 4×32

export const formatCellSize = (cpu, memoryGB) => `${cpu}×${memoryGB}`;
```

**Step 2: Commit**

```bash
git add frontend/src/config/vmPresets.js
git commit -m "feat: add VM size presets configuration"
```

---

## Task 3: DataSourceSelector Component

**Files:**
- Create: `frontend/src/components/DataSourceSelector.jsx`

**Step 1: Create component**

```jsx
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
```

**Step 2: Commit**

```bash
git add frontend/src/components/DataSourceSelector.jsx
git commit -m "feat: add DataSourceSelector component"
```

---

## Task 4: ComparisonTable Component

**Files:**
- Create: `frontend/src/components/ComparisonTable.jsx`

**Step 1: Create component**

```jsx
// frontend/src/components/ComparisonTable.jsx
// ABOUTME: Side-by-side comparison table for current vs proposed scenarios
// ABOUTME: Shows metrics with change indicators

import React from 'react';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

const formatNumber = (num) => {
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toFixed(1);
};

const formatGB = (gb) => {
  if (gb >= 1000) return `${(gb / 1000).toFixed(1)} TB`;
  return `${gb} GB`;
};

const ChangeIndicator = ({ current, proposed, inverse = false }) => {
  const diff = proposed - current;
  if (Math.abs(diff) < 0.1) {
    return <span className="text-gray-400"><Minus size={16} /></span>;
  }
  const isPositive = inverse ? diff < 0 : diff > 0;
  return isPositive ? (
    <span className="text-green-600 flex items-center gap-1">
      <TrendingUp size={16} />
      {diff > 0 ? '+' : ''}{formatNumber(diff)}
    </span>
  ) : (
    <span className="text-red-600 flex items-center gap-1">
      <TrendingDown size={16} />
      {diff > 0 ? '+' : ''}{formatNumber(diff)}
    </span>
  );
};

const ComparisonTable = ({ comparison }) => {
  if (!comparison) return null;

  const { current, proposed, delta } = comparison;

  const metrics = [
    {
      label: 'Cell Count',
      current: current.cell_count,
      proposed: proposed.cell_count,
      format: (v) => v,
      inverse: false,
    },
    {
      label: 'Cell Size',
      current: `${current.cell_cpu}×${current.cell_memory_gb}`,
      proposed: `${proposed.cell_cpu}×${proposed.cell_memory_gb}`,
      noChange: true,
    },
    {
      label: 'App Capacity',
      current: current.app_capacity_gb,
      proposed: proposed.app_capacity_gb,
      format: formatGB,
      inverse: false,
    },
    {
      label: 'Utilization',
      current: current.utilization_pct,
      proposed: proposed.utilization_pct,
      format: (v) => `${v.toFixed(1)}%`,
      inverse: true, // Lower is better
    },
    {
      label: 'Free Chunks',
      current: current.free_chunks,
      proposed: proposed.free_chunks,
      format: (v) => v,
      inverse: false,
    },
    {
      label: 'N-1 Utilization',
      current: current.n1_utilization_pct,
      proposed: proposed.n1_utilization_pct,
      format: (v) => `${v.toFixed(1)}%`,
      inverse: true, // Lower is better
    },
    {
      label: 'Fault Impact',
      current: current.fault_impact,
      proposed: proposed.fault_impact,
      format: (v) => `${v} apps/cell`,
      inverse: true, // Lower is better
    },
  ];

  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <table className="w-full">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700">
              Metric
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-700">
              Current ({current.cell_cpu}×{current.cell_memory_gb})
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-700">
              Proposed ({proposed.cell_cpu}×{proposed.cell_memory_gb})
            </th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-700">
              Change
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {metrics.map((m) => (
            <tr key={m.label} className="hover:bg-gray-50">
              <td className="px-4 py-3 text-sm text-gray-900">{m.label}</td>
              <td className="px-4 py-3 text-sm text-right text-gray-900">
                {m.format ? m.format(m.current) : m.current}
              </td>
              <td className="px-4 py-3 text-sm text-right text-gray-900">
                {m.format ? m.format(m.proposed) : m.proposed}
              </td>
              <td className="px-4 py-3 text-sm text-right">
                {m.noChange ? (
                  <span className="text-gray-400">—</span>
                ) : (
                  <ChangeIndicator
                    current={m.current}
                    proposed={m.proposed}
                    inverse={m.inverse}
                  />
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default ComparisonTable;
```

**Step 2: Commit**

```bash
git add frontend/src/components/ComparisonTable.jsx
git commit -m "feat: add ComparisonTable component"
```

---

## Task 5: WarningsList Component

**Files:**
- Create: `frontend/src/components/WarningsList.jsx`

**Step 1: Create component**

```jsx
// frontend/src/components/WarningsList.jsx
// ABOUTME: Displays scenario warnings with severity colors
// ABOUTME: Critical warnings in red, warnings in yellow

import React from 'react';
import { AlertTriangle, AlertCircle, Info } from 'lucide-react';

const severityConfig = {
  critical: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    text: 'text-red-800',
    icon: AlertCircle,
  },
  warning: {
    bg: 'bg-yellow-50',
    border: 'border-yellow-200',
    text: 'text-yellow-800',
    icon: AlertTriangle,
  },
  info: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    text: 'text-blue-800',
    icon: Info,
  },
};

const WarningsList = ({ warnings }) => {
  if (!warnings || warnings.length === 0) {
    return (
      <div className="bg-green-50 border border-green-200 rounded-lg p-4 text-green-800">
        ✓ No warnings - proposed configuration looks good
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {warnings.map((warning, index) => {
        const config = severityConfig[warning.severity] || severityConfig.info;
        const Icon = config.icon;
        return (
          <div
            key={index}
            className={`${config.bg} ${config.border} border rounded-lg p-3 flex items-start gap-3`}
          >
            <Icon className={`${config.text} flex-shrink-0 mt-0.5`} size={18} />
            <span className={config.text}>{warning.message}</span>
          </div>
        );
      })}
    </div>
  );
};

export default WarningsList;
```

**Step 2: Commit**

```bash
git add frontend/src/components/WarningsList.jsx
git commit -m "feat: add WarningsList component"
```

---

## Task 6: ScenarioAnalyzer Component

**Files:**
- Create: `frontend/src/components/ScenarioAnalyzer.jsx`

**Step 1: Create main component**

```jsx
// frontend/src/components/ScenarioAnalyzer.jsx
// ABOUTME: Main what-if scenario analyzer component
// ABOUTME: Combines data source, comparison table, and warnings

import React, { useState, useEffect } from 'react';
import { Calculator, RefreshCw } from 'lucide-react';
import DataSourceSelector from './DataSourceSelector';
import ComparisonTable from './ComparisonTable';
import WarningsList from './WarningsList';
import { scenarioApi } from '../services/scenarioApi';
import { VM_SIZE_PRESETS, DEFAULT_PRESET_INDEX } from '../config/vmPresets';

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
          <ComparisonTable comparison={comparison} />
          <WarningsList warnings={comparison.warnings} />
        </>
      )}
    </div>
  );
};

export default ScenarioAnalyzer;
```

**Step 2: Commit**

```bash
git add frontend/src/components/ScenarioAnalyzer.jsx
git commit -m "feat: add ScenarioAnalyzer main component"
```

---

## Task 7: Integrate into TASCapacityAnalyzer

**Files:**
- Modify: `frontend/src/TASCapacityAnalyzer.jsx`

**Step 1: Read current file**

Run: `head -50 frontend/src/TASCapacityAnalyzer.jsx`

**Step 2: Add import and tab for ScenarioAnalyzer**

Add import at top:
```jsx
import ScenarioAnalyzer from './components/ScenarioAnalyzer';
```

Add new state for active tab and render ScenarioAnalyzer in a tab.

**Step 3: Test integration**

Run: `cd frontend && npm run dev` and verify the component renders.

**Step 4: Commit**

```bash
git add frontend/src/TASCapacityAnalyzer.jsx
git commit -m "feat: integrate ScenarioAnalyzer into dashboard"
```

---

## Task 8: Manual Entry Form

**Files:**
- Modify: `frontend/src/components/DataSourceSelector.jsx`

**Step 1: Add manual entry form**

Add a form for quick single-cluster entry when mode === 'manual':

```jsx
{mode === 'manual' && (
  <ManualEntryForm onSubmit={handleManualSubmit} />
)}
```

Create the form with fields:
- Environment name
- Host count
- RAM per host
- Diego cell count
- Cell size (dropdown)
- Platform VMs (GB)
- Total app memory (GB)
- App instances

**Step 2: Commit**

```bash
git add frontend/src/components/DataSourceSelector.jsx
git commit -m "feat: add manual entry form to DataSourceSelector"
```

---

## Task 9: Final Integration Test

**Step 1: Start backend**

```bash
cd backend && go build && ./capacity-backend &
```

**Step 2: Start frontend**

```bash
cd frontend && npm run dev
```

**Step 3: Manual test checklist**

- [ ] Upload JSON file with infrastructure data
- [ ] See infrastructure summary displayed
- [ ] Select VM size preset (4×64)
- [ ] Change cell count (e.g., 235)
- [ ] Click "Compare Scenarios"
- [ ] Verify comparison table shows current vs proposed
- [ ] Verify warnings display correctly
- [ ] Export configuration works

**Step 4: Final commit**

```bash
git add -A
git commit -m "Phase 2 complete: Frontend scenario analyzer"
```

---

## Summary

Phase 2 implements:
- API service for scenario endpoints
- DataSourceSelector with JSON upload + manual form
- ComparisonTable with change indicators
- WarningsList with severity colors
- ScenarioAnalyzer main component
- Integration into existing dashboard

**Files Created:**
- `frontend/src/services/scenarioApi.js`
- `frontend/src/config/vmPresets.js`
- `frontend/src/components/DataSourceSelector.jsx`
- `frontend/src/components/ComparisonTable.jsx`
- `frontend/src/components/WarningsList.jsx`
- `frontend/src/components/ScenarioAnalyzer.jsx`

**Files Modified:**
- `frontend/src/TASCapacityAnalyzer.jsx`
