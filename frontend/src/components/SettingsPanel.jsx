// ABOUTME: Settings dropdown panel for data source and connection controls
// ABOUTME: Provides unobtrusive access to mock/live data toggle and API testing

import { useState, useRef, useEffect } from 'react';
import { Settings, Database, CheckCircle, RefreshCw, X } from 'lucide-react';

const SettingsPanel = ({
  useMockData,
  loading,
  lastRefresh,
  onToggleDataSource,
  onTestConnection,
  onRefresh,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const panelRef = useRef(null);

  // Close panel when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (panelRef.current && !panelRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  // Close on escape key
  useEffect(() => {
    const handleEscape = (event) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen]);

  return (
    <div className="relative" ref={panelRef}>
      {/* Settings trigger button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`p-2 rounded-lg transition-all ${
          isOpen
            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
            : 'bg-slate-800/50 text-slate-400 border border-slate-700 hover:text-slate-300 hover:border-slate-600'
        }`}
        aria-label="Settings"
        aria-expanded={isOpen}
        aria-haspopup="true"
      >
        <Settings className="w-5 h-5" aria-hidden="true" />
      </button>

      {/* Status indicator dot */}
      <span
        className={`absolute -top-1 -right-1 w-2.5 h-2.5 rounded-full border-2 border-slate-950 ${
          useMockData ? 'bg-amber-400' : 'bg-emerald-400'
        }`}
        aria-hidden="true"
      />

      {/* Dropdown panel */}
      {isOpen && (
        <div
          className="absolute right-0 mt-2 w-72 bg-slate-900 border border-slate-700 rounded-xl shadow-xl z-50"
          role="dialog"
          aria-label="Settings panel"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
            <h3 className="text-sm font-semibold text-white">Data Settings</h3>
            <button
              onClick={() => setIsOpen(false)}
              className="text-slate-400 hover:text-slate-300 transition-colors"
              aria-label="Close settings"
            >
              <X className="w-4 h-4" aria-hidden="true" />
            </button>
          </div>

          {/* Content */}
          <div className="p-4 space-y-4">
            {/* Data Source Status */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Database className="w-4 h-4 text-slate-400" aria-hidden="true" />
                <span className="text-sm text-slate-300">Data Source</span>
              </div>
              <span
                className={`text-xs font-semibold px-2 py-1 rounded ${
                  useMockData
                    ? 'bg-amber-500/20 text-amber-300'
                    : 'bg-emerald-500/20 text-emerald-300'
                }`}
              >
                {useMockData ? 'Mock' : 'Live'}
              </span>
            </div>

            {/* Toggle Button */}
            <button
              onClick={() => {
                onToggleDataSource();
                if (useMockData) {
                  // Keep panel open when switching to live to show loading state
                } else {
                  setIsOpen(false);
                }
              }}
              disabled={loading}
              className={`w-full px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center justify-center gap-2 ${
                useMockData
                  ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 hover:bg-emerald-500/30'
                  : 'bg-amber-500/20 text-amber-300 border border-amber-500/30 hover:bg-amber-500/30'
              } disabled:opacity-50 disabled:cursor-not-allowed`}
              aria-label={useMockData ? 'Switch to live CF data' : 'Switch to mock data'}
            >
              {loading ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" aria-hidden="true" />
                  Connecting...
                </>
              ) : useMockData ? (
                'Connect to Live CF Data'
              ) : (
                'Switch to Mock Data'
              )}
            </button>

            {/* Test Connection (only when using mock) */}
            {useMockData && (
              <button
                onClick={onTestConnection}
                disabled={loading}
                className="w-full px-4 py-2.5 bg-slate-800 text-slate-300 border border-slate-700 rounded-lg text-sm hover:bg-slate-700 transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                aria-label="Test CF API connection"
              >
                <CheckCircle className="w-4 h-4" aria-hidden="true" />
                Test API Connection
              </button>
            )}

            {/* Refresh (only when using live data) */}
            {!useMockData && (
              <button
                onClick={onRefresh}
                disabled={loading}
                className="w-full px-4 py-2.5 bg-slate-800 text-slate-300 border border-slate-700 rounded-lg text-sm hover:bg-slate-700 transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                aria-label="Refresh data from CF API"
              >
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
                Refresh Data
              </button>
            )}

            {/* Last Refresh Time */}
            {lastRefresh && (
              <div className="text-xs text-slate-500 text-center pt-2 border-t border-slate-800">
                Last updated: {lastRefresh.toLocaleTimeString()}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default SettingsPanel;
