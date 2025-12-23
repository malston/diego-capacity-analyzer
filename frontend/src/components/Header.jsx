// ABOUTME: Header component with title, user info, settings, and tab navigation
// ABOUTME: Handles logout, tab switching, and settings panel for TAS Capacity Analyzer

import { Server, Activity, Zap, LogOut, User } from 'lucide-react';
import SettingsPanel from './SettingsPanel';

const Header = ({
  user,
  logout,
  activeTab,
  setActiveTab,
  useMockData,
  loading,
  lastRefresh,
  onToggleDataSource,
  onTestConnection,
  onRefresh,
}) => {
  const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: Activity },
    { id: 'planning', label: 'Infrastructure Planning', icon: Server },
    { id: 'scenarios', label: 'Scenario Analysis', icon: Zap },
  ];

  return (
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
          {/* Settings Panel */}
          <SettingsPanel
            useMockData={useMockData}
            loading={loading}
            lastRefresh={lastRefresh}
            onToggleDataSource={onToggleDataSource}
            onTestConnection={onTestConnection}
            onRefresh={onRefresh}
          />

          {/* User Info */}
          {user && (
            <div className="flex items-center gap-3 px-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg">
              <User className="w-4 h-4 text-slate-400" aria-hidden="true" />
              <span className="text-sm text-slate-300">{user.username}</span>
              <button
                onClick={logout}
                className="ml-2 text-slate-400 hover:text-red-400 transition-colors"
                aria-label="Logout"
              >
                <LogOut className="w-4 h-4" aria-hidden="true" />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Tab Navigation */}
      <nav className="flex gap-2" role="tablist" aria-label="Main navigation">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            role="tab"
            aria-selected={activeTab === id}
            aria-controls={`${id}-panel`}
            className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              activeTab === id
                ? 'bg-blue-500 text-white'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700 hover:border-blue-500'
            }`}
          >
            <Icon className="w-4 h-4 inline mr-2" aria-hidden="true" />
            {label}
          </button>
        ))}
      </nav>
    </div>
  );
};

export default Header;
