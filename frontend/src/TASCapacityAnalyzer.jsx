import { useState, useMemo } from 'react';
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, Legend, ResponsiveContainer } from 'recharts';
import { Server, Activity, Zap, TrendingUp, AlertTriangle, CheckCircle, Layers, LogOut, User, RefreshCw, Database } from 'lucide-react';
import { useAuth } from './contexts/AuthContext';
import { cfApi } from './services/cfApi';
import ScenarioAnalyzer from './components/ScenarioAnalyzer';
import InfrastructurePlanning from './components/InfrastructurePlanning';
import Tooltip from './components/Tooltip';

const DASHBOARD_TOOLTIPS = {
  totalCells: "Number of Diego cells (VMs that run app containers). More cells = more capacity for workloads.",
  utilization: "Percentage of total memory actively consumed by running apps. Below 50% = consolidation opportunity. Above 80% = running hot.",
  avgCpu: "Average processor load across all cells. Sustained >70% means apps are competing for CPU cycles.",
  unusedMemory: "Memory apps reserved but aren't actually using. May be an optimization opportunity for right-sizing.",
};

// Mock data - replace with real CF API calls
const mockData = {
  cells: [
    { id: 'cell-01', name: 'diego_cell/0', memory_mb: 16384, allocated_mb: 12288, used_mb: 9830, cpu_percent: 45, isolation_segment: 'default' },
    { id: 'cell-02', name: 'diego_cell/1', memory_mb: 16384, allocated_mb: 14336, used_mb: 11200, cpu_percent: 62, isolation_segment: 'default' },
    { id: 'cell-03', name: 'diego_cell/2', memory_mb: 16384, allocated_mb: 8192, used_mb: 6400, cpu_percent: 28, isolation_segment: 'default' },
    { id: 'cell-04', name: 'diego_cell/3', memory_mb: 32768, allocated_mb: 24576, used_mb: 19660, cpu_percent: 55, isolation_segment: 'production' },
    { id: 'cell-05', name: 'diego_cell/4', memory_mb: 32768, allocated_mb: 28672, used_mb: 22100, cpu_percent: 71, isolation_segment: 'production' },
    { id: 'cell-06', name: 'diego_cell/5', memory_mb: 8192, allocated_mb: 6144, used_mb: 4800, cpu_percent: 38, isolation_segment: 'development' },
  ],
  apps: [
    { name: 'api-gateway', instances: 4, requested_mb: 1024, actual_mb: 780, isolation_segment: 'production' },
    { name: 'auth-service', instances: 3, requested_mb: 512, actual_mb: 420, isolation_segment: 'production' },
    { name: 'payment-processor', instances: 2, requested_mb: 2048, actual_mb: 1650, isolation_segment: 'production' },
    { name: 'web-ui', instances: 6, requested_mb: 768, actual_mb: 580, isolation_segment: 'default' },
    { name: 'background-jobs', instances: 2, requested_mb: 1536, actual_mb: 980, isolation_segment: 'default' },
    { name: 'analytics-engine', instances: 1, requested_mb: 4096, actual_mb: 3200, isolation_segment: 'production' },
    { name: 'notification-service', instances: 3, requested_mb: 512, actual_mb: 380, isolation_segment: 'default' },
    { name: 'dev-app-1', instances: 1, requested_mb: 1024, actual_mb: 450, isolation_segment: 'development' },
    { name: 'dev-app-2', instances: 1, requested_mb: 512, actual_mb: 280, isolation_segment: 'development' },
  ]
};

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
      alert(`✅ CF API Connected!\n\nAPI Version: ${info.links?.self?.href || 'Unknown'}\n\nNow try loading data again.`);
    } catch (err) {
      console.error('Connection test failed:', err);
      const errorDetails = `
❌ Cannot reach CF API

Error: ${err.message}

Possible causes:
• CORS blocking (most common)
• Invalid CF_API_URL in .env
• Network/firewall issues
• CF API is down

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
    const potentialInstances = Math.floor(newCapacity / 512); // Assume avg 512MB per instance
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
          recommendedMb: Math.ceil(app.actual_mb * 1.2), // 20% buffer
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
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;600;700&family=Space+Grotesk:wght@400;600;700&display=swap');
        
        body {
          font-family: 'JetBrains Mono', monospace;
        }
        
        .title-font {
          font-family: 'Space Grotesk', sans-serif;
        }

        .metric-card {
          background: linear-gradient(135deg, rgba(30, 41, 59, 0.4) 0%, rgba(15, 23, 42, 0.6) 100%);
          border: 1px solid rgba(71, 85, 105, 0.3);
          backdrop-filter: blur(10px);
        }

        .metric-card:hover {
          border-color: rgba(59, 130, 246, 0.5);
          transform: translateY(-2px);
          transition: all 0.3s ease;
        }

        .cell-row {
          background: rgba(30, 41, 59, 0.3);
          border-left: 3px solid transparent;
          transition: all 0.2s ease;
        }

        .cell-row:hover {
          background: rgba(30, 41, 59, 0.5);
          border-left-color: #3b82f6;
        }

        .progress-bar {
          background: linear-gradient(90deg, rgba(59, 130, 246, 0.2) 0%, rgba(59, 130, 246, 0.05) 100%);
          overflow: hidden;
          position: relative;
        }

        .progress-fill {
          background: linear-gradient(90deg, #3b82f6 0%, #2563eb 100%);
          transition: width 0.5s ease;
          position: relative;
        }

        .progress-fill::after {
          content: '';
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: linear-gradient(90deg, transparent 0%, rgba(255,255,255,0.3) 50%, transparent 100%);
          animation: shimmer 2s infinite;
        }

        @keyframes shimmer {
          0% { transform: translateX(-100%); }
          100% { transform: translateX(100%); }
        }

        .status-badge {
          padding: 0.25rem 0.75rem;
          border-radius: 9999px;
          font-size: 0.75rem;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .segment-chip {
          background: rgba(59, 130, 246, 0.15);
          border: 1px solid rgba(59, 130, 246, 0.3);
          padding: 0.25rem 0.75rem;
          border-radius: 0.5rem;
          font-size: 0.75rem;
          display: inline-block;
        }
      `}</style>

      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-blue-500/20 rounded-lg border border-blue-500/30">
              <Server className="w-8 h-8 text-blue-400" />
            </div>
            <div>
              <h1 className="text-4xl font-bold title-font bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent">
                TAS Capacity Analyzer
              </h1>
              <p className="text-slate-400 text-sm mt-1">Real-time diego cell capacity and density optimization</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            {/* User Info */}
            {user && (
              <div className="flex items-center gap-3 px-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg">
                <User className="w-4 h-4 text-slate-400" />
                <span className="text-sm text-slate-300">{user.username}</span>
                <button
                  onClick={logout}
                  className="ml-2 text-slate-400 hover:text-red-400 transition-colors"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => setActiveTab('dashboard')}
            className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              activeTab === 'dashboard'
                ? 'bg-blue-500 text-white'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700 hover:border-blue-500'
            }`}
          >
            <Activity className="w-4 h-4 inline mr-2" />
            Dashboard
          </button>
          <button
            onClick={() => setActiveTab('scenarios')}
            className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              activeTab === 'scenarios'
                ? 'bg-blue-500 text-white'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700 hover:border-blue-500'
            }`}
          >
            <Zap className="w-4 h-4 inline mr-2" />
            Scenario Analysis
          </button>
          <button
            onClick={() => setActiveTab('planning')}
            className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              activeTab === 'planning'
                ? 'bg-blue-500 text-white'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700 hover:border-blue-500'
            }`}
          >
            <Server className="w-4 h-4 inline mr-2" />
            Infrastructure Planning
          </button>
        </div>

        {/* Controls Row (only show on dashboard) */}
        {activeTab === 'dashboard' && (
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              {/* Data Source Toggle */}
              <button
                onClick={toggleDataSource}
                disabled={loading}
                className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all flex items-center gap-2 ${
                  useMockData
                    ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30 hover:bg-amber-500/30'
                    : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 hover:bg-emerald-500/30'
                } disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                <Database className="w-4 h-4" />
                {useMockData ? 'Using Mock Data' : 'Live CF Data'}
              </button>

            {/* Test Connection Button */}
            {useMockData && (
              <button
                onClick={testConnection}
                disabled={loading}
                className="px-4 py-2 bg-blue-500/20 text-blue-300 border border-blue-500/30 rounded-lg text-sm hover:bg-blue-500/30 transition-all flex items-center gap-2 disabled:opacity-50"
                title="Test CF API connection"
              >
                <CheckCircle className="w-4 h-4" />
                Test Connection
              </button>
            )}

            {/* Refresh Button (only for live data) */}
            {!useMockData && (
              <button
                onClick={loadCFData}
                disabled={loading}
                className="px-4 py-2 bg-slate-800/50 text-slate-300 border border-slate-700 rounded-lg text-sm hover:border-blue-500 transition-all flex items-center gap-2 disabled:opacity-50"
                title="Refresh data"
              >
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            )}

            {/* Last Refresh Time */}
            {lastRefresh && (
              <span className="text-xs text-slate-500">
                Last updated: {lastRefresh.toLocaleTimeString()}
              </span>
            )}
          </div>

              <div className="flex gap-2">
                <select
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
                >
                  <Zap className="w-4 h-4 inline mr-2" />
                  What-If Mode
                </button>
              </div>
            </div>
          )
        }

        {/* Error Message */}
        {activeTab === 'dashboard' && error && (
          <div className="mt-4 p-4 bg-red-500/10 border border-red-500/30 rounded-lg flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
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
      </div>

      {/* Dashboard Tab Content */}
      {activeTab === 'dashboard' && (
        <>
          {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <div className="metric-card p-6 rounded-xl">
          <div className="flex items-center justify-between mb-2">
            <Tooltip text={DASHBOARD_TOOLTIPS.totalCells} position="bottom" showIcon>
              <span className="text-slate-400 text-sm uppercase tracking-wide">Total Cells</span>
            </Tooltip>
            <Server className="w-5 h-5 text-blue-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-1">{metrics.totalCells}</div>
          <div className="text-xs text-slate-400">{(metrics.totalMemory / 1024).toFixed(1)} GB capacity</div>
        </div>

        <div className="metric-card p-6 rounded-xl">
          <div className="flex items-center justify-between mb-2">
            <Tooltip text={DASHBOARD_TOOLTIPS.utilization} position="bottom" showIcon>
              <span className="text-slate-400 text-sm uppercase tracking-wide">Utilization</span>
            </Tooltip>
            <Activity className="w-5 h-5 text-emerald-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-1">{metrics.utilizationPercent.toFixed(1)}%</div>
          <div className="text-xs text-slate-400">
            {(metrics.totalUsed / 1024).toFixed(1)} GB / {(metrics.totalMemory / 1024).toFixed(1)} GB
          </div>
        </div>

        <div className="metric-card p-6 rounded-xl">
          <div className="flex items-center justify-between mb-2">
            <Tooltip text={DASHBOARD_TOOLTIPS.avgCpu} position="bottom" showIcon>
              <span className="text-slate-400 text-sm uppercase tracking-wide">Avg CPU</span>
            </Tooltip>
            <TrendingUp className="w-5 h-5 text-amber-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-1">{metrics.avgCpu.toFixed(1)}%</div>
          <div className="text-xs text-slate-400">Across all cells</div>
        </div>

        <div className="metric-card p-6 rounded-xl">
          <div className="flex items-center justify-between mb-2">
            <Tooltip text={DASHBOARD_TOOLTIPS.unusedMemory} position="bottom" showIcon>
              <span className="text-slate-400 text-sm uppercase tracking-wide">Unused Memory</span>
            </Tooltip>
            <AlertTriangle className="w-5 h-5 text-amber-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-1">{(metrics.unusedMemory / 1024).toFixed(1)} GB</div>
          <div className="text-xs text-slate-400">{metrics.unusedPercent.toFixed(1)}% over-allocated</div>
        </div>
      </div>

      {/* What-If Scenario */}
      {showWhatIf && (
        <div className="metric-card p-6 rounded-xl mb-8 border-2 border-blue-500/50">
          <div className="flex items-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-blue-400" />
            <h2 className="text-xl font-bold title-font text-white">What-If Scenario</h2>
          </div>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm text-slate-400 mb-3">
                Memory Overcommit Ratio: <span className={`font-bold ${
                  overcommitRatio <= 1.5 ? 'text-emerald-400' :
                  overcommitRatio <= 2.0 ? 'text-yellow-400' :
                  overcommitRatio <= 3.0 ? 'text-orange-400' :
                  'text-red-400'
                }`}>{overcommitRatio.toFixed(1)}x</span>
                <span className={`ml-2 text-xs px-2 py-0.5 rounded ${
                  overcommitRatio <= 1.5 ? 'bg-emerald-900/50 text-emerald-400' :
                  overcommitRatio <= 2.0 ? 'bg-yellow-900/50 text-yellow-400' :
                  overcommitRatio <= 3.0 ? 'bg-orange-900/50 text-orange-400' :
                  'bg-red-900/50 text-red-400'
                }`}>
                  {overcommitRatio <= 1.5 ? 'Safe' :
                   overcommitRatio <= 2.0 ? 'Caution' :
                   overcommitRatio <= 3.0 ? 'High Risk' :
                   'Labs Only'}
                </span>
              </label>
              <input
                type="range"
                min="1.0"
                max="4.0"
                step="0.1"
                value={overcommitRatio}
                onChange={(e) => setOvercommitRatio(parseFloat(e.target.value))}
                className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
              />
              <div className="flex justify-between text-xs text-slate-500 mt-1">
                <span>1.0x (None)</span>
                <span className="text-yellow-500">2.0x</span>
                <span className="text-red-500">4.0x (Labs)</span>
              </div>
            </div>

            <div className="space-y-3">
              <div className="flex justify-between items-center p-3 bg-slate-800/50 rounded-lg">
                <span className="text-slate-400">New Capacity:</span>
                <span className="text-white font-bold">{(metrics.newCapacity / 1024).toFixed(1)} GB</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-slate-800/50 rounded-lg">
                <span className="text-slate-400">Current Instances:</span>
                <span className="text-white font-bold">{metrics.totalInstances}</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-green-500/10 border border-green-500/30 rounded-lg">
                <span className="text-green-400">Additional Capacity:</span>
                <span className="text-green-400 font-bold">+{metrics.additionalInstances} instances</span>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Cell Capacity Chart */}
        <div className="metric-card p-6 rounded-xl">
          <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
            <Server className="w-5 h-5 text-blue-400" />
            Cell Capacity Overview
          </h2>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={cellChartData}>
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
            <Layers className="w-5 h-5 text-purple-400" />
            Isolation Segments
          </h2>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
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
                {segmentData.map((entry, index) => (
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

      {/* Detailed Cell View */}
      <div className="metric-card p-6 rounded-xl mb-8">
        <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
          <Server className="w-5 h-5 text-blue-400" />
          Diego Cells Detail
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700 text-slate-400">
                <th className="text-left py-3 px-4">Cell</th>
                <th className="text-left py-3 px-4">Segment</th>
                <th className="text-right py-3 px-4">Capacity</th>
                <th className="text-right py-3 px-4">Allocated</th>
                <th className="text-right py-3 px-4">Used</th>
                <th className="text-right py-3 px-4">CPU</th>
                <th className="text-left py-3 px-4">Utilization</th>
              </tr>
            </thead>
            <tbody>
              {data.cells
                .filter(c => selectedSegment === 'all' || c.isolation_segment === selectedSegment)
                .map((cell) => {
                  const utilizationPercent = (cell.used_mb / cell.memory_mb) * 100;
                  const status = utilizationPercent > 80 ? 'high' : utilizationPercent > 60 ? 'medium' : 'low';
                  
                  return (
                    <tr key={cell.id} className="cell-row border-b border-slate-800">
                      <td className="py-3 px-4 font-semibold text-white">{cell.name}</td>
                      <td className="py-3 px-4">
                        <span className="segment-chip">{cell.isolation_segment}</span>
                      </td>
                      <td className="py-3 px-4 text-right text-slate-300">{cell.memory_mb} MB</td>
                      <td className="py-3 px-4 text-right text-slate-300">{cell.allocated_mb} MB</td>
                      <td className="py-3 px-4 text-right text-white font-semibold">{cell.used_mb} MB</td>
                      <td className="py-3 px-4 text-right">
                        <span className={`${cell.cpu_percent > 70 ? 'text-red-400' : cell.cpu_percent > 50 ? 'text-amber-400' : 'text-emerald-400'} font-semibold`}>
                          {cell.cpu_percent}%
                        </span>
                      </td>
                      <td className="py-3 px-4">
                        <div className="flex items-center gap-3">
                          <div className="progress-bar w-32 h-2 rounded-full">
                            <div 
                              className={`progress-fill h-full rounded-full ${
                                status === 'high' ? 'bg-red-500' : status === 'medium' ? 'bg-amber-500' : 'bg-emerald-500'
                              }`}
                              style={{ width: `${utilizationPercent}%` }}
                            />
                          </div>
                          <span className="text-slate-300 font-semibold">{utilizationPercent.toFixed(1)}%</span>
                        </div>
                      </td>
                    </tr>
                  );
                })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Right-Sizing Recommendations */}
      {recommendations.length > 0 && (
        <div className="metric-card p-6 rounded-xl border-2 border-amber-500/30">
          <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-amber-400" />
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
                    app.overheadPercent > 30 ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
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
        </>
      )}

      {/* Scenario Analysis Tab Content */}
      {activeTab === 'scenarios' && (
        <div className="mt-8">
          <ScenarioAnalyzer />
        </div>
      )}

      {/* Infrastructure Planning Tab Content */}
      {activeTab === 'planning' && (
        <div className="mt-8">
          <InfrastructurePlanning />
        </div>
      )}

      {/* Footer */}
      <div className="mt-8 text-center text-slate-500 text-xs">
        <p>TAS Capacity Analyzer v1.0 | {useMockData ? 'Mock Data Mode' : 'Live CF API Data'}</p>
        <p className="mt-1">Built for platform engineers by platform engineers</p>
      </div>
    </div>
  );
};

export default TASCapacityAnalyzer;
