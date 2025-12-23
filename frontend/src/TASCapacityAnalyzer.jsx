// ABOUTME: Main TAS Capacity Analyzer dashboard component
// ABOUTME: Orchestrates header, metrics, charts, and tabbed content views

import { useState, useMemo } from 'react';
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, Legend, ResponsiveContainer } from 'recharts';
import { Server, Zap, TrendingUp, AlertTriangle, Layers } from 'lucide-react';
import { useAuth } from './contexts/AuthContext';
import { cfApi } from './services/cfApi';
import ScenarioAnalyzer from './components/ScenarioAnalyzer';
import InfrastructurePlanning from './components/InfrastructurePlanning';
import Header from './components/Header';
import MetricCards from './components/MetricCards';
import WhatIfPanel from './components/WhatIfPanel';
import CellDetailTable from './components/CellDetailTable';
import { mockData } from './data/mockData';
import './TASCapacityAnalyzer.css';

const TASCapacityAnalyzer = () => {
  const { user, logout } = useAuth();
  const [activeTab, setActiveTab] = useState('dashboard');
  const [overcommitRatio, setOvercommitRatio] = useState(1.0);
  const [selectedSegment, setSelectedSegment] = useState('all');
  const [showWhatIf, setShowWhatIf] = useState(false);
  const [useMockData, setUseMockData] = useState(true);
  const [data, setData] = useState(mockData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(null);

  // Load real CF data from backend
  const loadCFData = async () => {
    setLoading(true);
    setError(null);

    try {
      const apiURL = import.meta.env.VITE_API_URL || 'http://localhost:8080';
      const response = await fetch(`${apiURL}/api/dashboard`);

      if (!response.ok) {
        throw new Error(`Backend returned ${response.status}`);
      }

      const dashboardData = await response.json();

      setData({
        cells: dashboardData.cells,
        apps: dashboardData.apps,
      });

      setUseMockData(false);
      setLastRefresh(new Date(dashboardData.metadata.timestamp));
    } catch (err) {
      console.error('Error loading data:', err);
      setError(err.message);
      setData(mockData);
      setUseMockData(true);
    } finally {
      setLoading(false);
    }
  };

  // Test CF API connection
  const testConnection = async () => {
    setLoading(true);
    setError(null);

    try {
      const info = await cfApi.getInfo();
      alert(`CF API Connected!\n\nAPI Version: ${info.links?.self?.href || 'Unknown'}\n\nNow try loading data again.`);
    } catch (err) {
      console.error('Connection test failed:', err);
      const errorDetails = `
Cannot reach CF API

Error: ${err.message}

Possible causes:
- CORS blocking (most common)
- Invalid CF_API_URL in .env
- Network/firewall issues
- CF API is down

Check browser console (F12) for details.`;
      alert(errorDetails);
    } finally {
      setLoading(false);
    }
  };

  // Toggle between mock and real data
  const toggleDataSource = () => {
    if (useMockData) {
      loadCFData();
    } else {
      setData(mockData);
      setUseMockData(true);
      setError(null);
    }
  };

  // Calculate metrics
  const metrics = useMemo(() => {
    const filteredCells = selectedSegment === 'all'
      ? data.cells
      : data.cells.filter(c => c.isolation_segment === selectedSegment);

    const totalMemory = filteredCells.reduce((sum, c) => sum + c.memory_mb, 0);
    const totalAllocated = filteredCells.reduce((sum, c) => sum + c.allocated_mb, 0);
    const totalUsed = filteredCells.reduce((sum, c) => sum + c.used_mb, 0);
    const avgCpu = filteredCells.reduce((sum, c) => sum + c.cpu_percent, 0) / filteredCells.length;

    const filteredApps = selectedSegment === 'all'
      ? data.apps
      : data.apps.filter(a => a.isolation_segment === selectedSegment);

    const totalAppMemoryRequested = filteredApps.reduce((sum, a) => sum + (a.requested_mb * a.instances), 0);
    const totalAppMemoryUsed = filteredApps.reduce((sum, a) => sum + (a.actual_mb * a.instances), 0);
    const unusedMemory = totalAppMemoryRequested - totalAppMemoryUsed;

    // What-if calculations
    const newCapacity = totalMemory * overcommitRatio;
    const potentialInstances = Math.floor(newCapacity / 512);
    const currentInstances = filteredApps.reduce((sum, a) => sum + a.instances, 0);

    return {
      totalCells: filteredCells.length,
      totalMemory,
      totalAllocated,
      totalUsed,
      avgCpu,
      utilizationPercent: (totalUsed / totalMemory) * 100,
      allocationPercent: (totalAllocated / totalMemory) * 100,
      unusedMemory,
      unusedPercent: (unusedMemory / totalAppMemoryRequested) * 100,
      totalApps: filteredApps.length,
      totalInstances: currentInstances,
      newCapacity,
      potentialInstances,
      additionalInstances: potentialInstances - currentInstances,
    };
  }, [overcommitRatio, selectedSegment, data]);

  // Right-sizing recommendations
  const recommendations = useMemo(() => {
    const filtered = selectedSegment === 'all'
      ? data.apps
      : data.apps.filter(a => a.isolation_segment === selectedSegment);

    return filtered
      .map(app => {
        const overhead = app.requested_mb - app.actual_mb;
        const overheadPercent = (overhead / app.requested_mb) * 100;
        return {
          ...app,
          overhead,
          overheadPercent,
          recommendedMb: Math.ceil(app.actual_mb * 1.2),
          potentialSavings: (overhead * app.instances),
        };
      })
      .filter(app => app.overheadPercent > 15)
      .sort((a, b) => b.potentialSavings - a.potentialSavings);
  }, [selectedSegment, data]);

  // Cell utilization data for chart
  const cellChartData = data.cells
    .filter(c => selectedSegment === 'all' || c.isolation_segment === selectedSegment)
    .map(cell => ({
      name: cell.name.split('/')[1],
      allocated: Math.round((cell.allocated_mb / cell.memory_mb) * 100),
      used: Math.round((cell.used_mb / cell.memory_mb) * 100),
      available: Math.round(((cell.memory_mb - cell.allocated_mb) / cell.memory_mb) * 100),
    }));

  // Isolation segment distribution
  const segmentData = Object.entries(
    data.cells.reduce((acc, cell) => {
      acc[cell.isolation_segment] = (acc[cell.isolation_segment] || 0) + 1;
      return acc;
    }, {})
  ).map(([name, value]) => ({ name, value }));

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'];

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-slate-100 p-6 font-mono">
      <Header
        user={user}
        logout={logout}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        useMockData={useMockData}
        loading={loading}
        lastRefresh={lastRefresh}
        onToggleDataSource={toggleDataSource}
        onTestConnection={testConnection}
        onRefresh={loadCFData}
      />

      {/* Dashboard Controls (segment filter and What-If toggle) */}
      {activeTab === 'dashboard' && (
        <div className="flex items-center justify-end gap-2 mb-6">
          <label htmlFor="segment-filter" className="sr-only">Filter by segment</label>
          <select
            id="segment-filter"
            value={selectedSegment}
            onChange={(e) => setSelectedSegment(e.target.value)}
            className="px-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-sm focus:outline-none focus:border-blue-500"
          >
            <option value="all">All Segments</option>
            <option value="default">Default</option>
            <option value="production">Production</option>
            <option value="development">Development</option>
          </select>

          <button
            onClick={() => setShowWhatIf(!showWhatIf)}
            className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              showWhatIf
                ? 'bg-blue-500 text-white'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700 hover:border-blue-500'
            }`}
            aria-pressed={showWhatIf}
            aria-label="Toggle What-If mode"
          >
            <Zap className="w-4 h-4 inline mr-2" aria-hidden="true" />
            What-If Mode
          </button>
        </div>
      )}

      {/* Error Message */}
      {activeTab === 'dashboard' && error && (
        <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-lg flex items-start gap-3" role="alert">
          <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" aria-hidden="true" />
          <div className="flex-1">
            <p className="text-red-300 text-sm font-semibold">Error loading CF data</p>
            <p className="text-red-400/80 text-xs mt-1">{error}</p>
            <p className="text-red-400/60 text-xs mt-2">Falling back to mock data.</p>
            {error.includes('CORS') && (
              <div className="mt-3 p-3 bg-slate-900/50 rounded text-xs text-slate-300">
                <p className="font-semibold text-amber-400 mb-2">CORS Issue - Solutions:</p>
                <ul className="space-y-1 list-disc list-inside text-slate-400">
                  <li>Configure CF/HAProxy to allow localhost in CORS headers</li>
                  <li>Create a backend proxy to handle CF API requests</li>
                  <li>Deploy this app to the same domain as your CF API</li>
                </ul>
                <p className="mt-2 text-slate-500">Check browser DevTools Console (F12) for detailed error</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Dashboard Tab Content */}
      {activeTab === 'dashboard' && (
        <div role="tabpanel" id="dashboard-panel" aria-labelledby="dashboard-tab">
          <MetricCards metrics={metrics} />

          {showWhatIf && (
            <WhatIfPanel
              overcommitRatio={overcommitRatio}
              setOvercommitRatio={setOvercommitRatio}
              metrics={metrics}
            />
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            {/* Cell Capacity Chart */}
            <div className="metric-card p-6 rounded-xl">
              <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
                <Server className="w-5 h-5 text-blue-400" aria-hidden="true" />
                Cell Capacity Overview
              </h2>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={cellChartData} aria-label="Cell capacity bar chart">
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(71, 85, 105, 0.3)" />
                  <XAxis dataKey="name" stroke="#94a3b8" tick={{ fill: '#94a3b8', fontSize: 12 }} />
                  <YAxis stroke="#94a3b8" tick={{ fill: '#94a3b8', fontSize: 12 }} />
                  <RechartsTooltip
                    contentStyle={{
                      backgroundColor: 'rgba(15, 23, 42, 0.95)',
                      border: '1px solid rgba(59, 130, 246, 0.3)',
                      borderRadius: '8px',
                      color: '#fff'
                    }}
                  />
                  <Legend wrapperStyle={{ color: '#94a3b8' }} />
                  <Bar dataKey="used" stackId="a" fill="#3b82f6" name="Used %" />
                  <Bar dataKey="allocated" stackId="a" fill="#10b981" name="Allocated (unused) %" />
                  <Bar dataKey="available" stackId="a" fill="#64748b" name="Available %" />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Isolation Segments */}
            <div className="metric-card p-6 rounded-xl">
              <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
                <Layers className="w-5 h-5 text-purple-400" aria-hidden="true" />
                Isolation Segments
              </h2>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart aria-label="Isolation segments pie chart">
                  <Pie
                    data={segmentData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                    outerRadius={100}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {segmentData.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <RechartsTooltip
                    contentStyle={{
                      backgroundColor: 'rgba(15, 23, 42, 0.95)',
                      border: '1px solid rgba(59, 130, 246, 0.3)',
                      borderRadius: '8px',
                      color: '#fff'
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>

          <CellDetailTable cells={data.cells} selectedSegment={selectedSegment} />

          {/* Right-Sizing Recommendations */}
          {recommendations.length > 0 && (
            <div className="metric-card p-6 rounded-xl border-2 border-amber-500/30">
              <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
                <TrendingUp className="w-5 h-5 text-amber-400" aria-hidden="true" />
                Right-Sizing Recommendations
                <span className="ml-auto text-sm font-normal text-slate-400">
                  Potential savings: {(recommendations.reduce((sum, r) => sum + r.potentialSavings, 0) / 1024).toFixed(1)} GB
                </span>
              </h2>
              <div className="space-y-3">
                {recommendations.map((app, idx) => (
                  <div key={idx} className="p-4 bg-slate-800/30 rounded-lg border border-slate-700 hover:border-amber-500/50 transition-all">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        <span className="text-white font-semibold">{app.name}</span>
                        <span className="ml-3 text-slate-400 text-sm">({app.instances} instances)</span>
                        <span className="ml-3 segment-chip">{app.isolation_segment}</span>
                      </div>
                      <span className={`status-badge ${
                        app.overheadPercent > 30
                          ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                          : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                      }`}>
                        {app.overheadPercent.toFixed(0)}% overhead
                      </span>
                    </div>
                    <div className="grid grid-cols-4 gap-4 text-sm">
                      <div>
                        <div className="text-slate-400">Requested</div>
                        <div className="text-white font-semibold">{app.requested_mb} MB</div>
                      </div>
                      <div>
                        <div className="text-slate-400">Actual Usage</div>
                        <div className="text-white font-semibold">{app.actual_mb} MB</div>
                      </div>
                      <div>
                        <div className="text-slate-400">Recommended</div>
                        <div className="text-emerald-400 font-semibold">{app.recommendedMb} MB</div>
                      </div>
                      <div>
                        <div className="text-slate-400">Savings/Instance</div>
                        <div className="text-emerald-400 font-semibold">-{app.overhead} MB</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Infrastructure Planning Tab Content */}
      {activeTab === 'planning' && (
        <div role="tabpanel" id="planning-panel" aria-labelledby="planning-tab" className="mt-8">
          <InfrastructurePlanning />
        </div>
      )}

      {/* Scenario Analysis Tab Content */}
      {activeTab === 'scenarios' && (
        <div role="tabpanel" id="scenarios-panel" aria-labelledby="scenarios-tab" className="mt-8">
          <ScenarioAnalyzer />
        </div>
      )}

      {/* Footer */}
      <footer className="mt-8 text-center text-slate-500 text-xs">
        <p>TAS Capacity Analyzer v1.0 | {useMockData ? 'Mock Data Mode' : 'Live CF API Data'}</p>
        <p className="mt-1">Built for platform engineers by platform engineers</p>
      </footer>
    </div>
  );
};

export default TASCapacityAnalyzer;
