import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Server, Lock, User, AlertCircle, Loader } from 'lucide-react';

const Login = () => {
  const { login, loading, error } = useAuth();
  const [credentials, setCredentials] = useState({
    username: '',
    password: '',
  });
  const [localError, setLocalError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLocalError('');

    if (!credentials.username || !credentials.password) {
      setLocalError('Please enter both username and password');
      return;
    }

    try {
      setIsSubmitting(true);
      await login(credentials.username, credentials.password);
    } catch (err) {
      setLocalError(err.message || 'Authentication failed');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (e) => {
    setCredentials({
      ...credentials,
      [e.target.name]: e.target.value,
    });
    setLocalError('');
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-6 font-mono">
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;600;700&family=Space+Grotesk:wght@400;600;700&display=swap');
        
        body {
          font-family: 'JetBrains Mono', monospace;
        }
        
        .title-font {
          font-family: 'Space Grotesk', sans-serif;
        }
      `}</style>

      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-500/20 rounded-2xl border border-blue-500/30 mb-4">
            <Server className="w-8 h-8 text-blue-400" />
          </div>
          <h1 className="text-3xl font-bold title-font bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-2">
            TAS Capacity Analyzer
          </h1>
          <p className="text-slate-400 text-sm">
            Sign in to analyze your diego cell capacity
          </p>
        </div>

        {/* Login Form */}
        <div className="bg-gradient-to-br from-slate-800/40 to-slate-900/40 backdrop-blur-lg border border-slate-700/50 rounded-2xl p-8 shadow-2xl">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Username Field */}
            <div>
              <label htmlFor="username" className="block text-sm font-semibold text-slate-300 mb-2">
                Cloud Foundry Username
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-slate-500" />
                <input
                  id="username"
                  name="username"
                  type="text"
                  value={credentials.username}
                  onChange={handleChange}
                  disabled={isSubmitting}
                  className="w-full pl-11 pr-4 py-3 bg-slate-900/50 border border-slate-700 rounded-lg text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all disabled:opacity-50"
                  placeholder="user@example.com"
                  autoComplete="username"
                />
              </div>
            </div>

            {/* Password Field */}
            <div>
              <label htmlFor="password" className="block text-sm font-semibold text-slate-300 mb-2">
                Password
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-slate-500" />
                <input
                  id="password"
                  name="password"
                  type="password"
                  value={credentials.password}
                  onChange={handleChange}
                  disabled={isSubmitting}
                  className="w-full pl-11 pr-4 py-3 bg-slate-900/50 border border-slate-700 rounded-lg text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all disabled:opacity-50"
                  placeholder="••••••••"
                  autoComplete="current-password"
                />
              </div>
            </div>

            {/* Error Message */}
            {(error || localError) && (
              <div className="flex items-start gap-3 p-4 bg-red-500/10 border border-red-500/30 rounded-lg">
                <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-red-300">
                  {localError || error}
                </div>
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isSubmitting || loading}
              className="w-full py-3 px-4 bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-500 hover:to-blue-400 text-white font-semibold rounded-lg transition-all transform hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none flex items-center justify-center gap-2"
            >
              {isSubmitting ? (
                <>
                  <Loader className="w-5 h-5 animate-spin" />
                  <span>Authenticating...</span>
                </>
              ) : (
                <span>Sign In</span>
              )}
            </button>
          </form>

          {/* Info */}
          <div className="mt-6 pt-6 border-t border-slate-700/50">
            <p className="text-xs text-slate-500 text-center">
              Your credentials are used to authenticate with Cloud Foundry UAA.
              <br />
              Tokens are stored securely in your browser session.
            </p>
          </div>
        </div>

        {/* Configuration Notice - only show if not configured */}
        {(!import.meta.env.VITE_CF_API_URL || !import.meta.env.VITE_CF_UAA_URL) && (
          <div className="mt-6 p-4 bg-amber-500/10 border border-amber-500/30 rounded-lg">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-amber-300">
                <p className="font-semibold mb-1">Configuration Required</p>
                <p className="text-amber-400/80">
                  Set <span className="font-mono bg-slate-900/50 px-1.5 py-0.5 rounded">VITE_CF_API_URL</span> and{' '}
                  <span className="font-mono bg-slate-900/50 px-1.5 py-0.5 rounded">VITE_CF_UAA_URL</span> in your .env file
                </p>
                <p className="text-amber-400/60 text-xs mt-2">
                  Missing: {!import.meta.env.VITE_CF_API_URL && 'VITE_CF_API_URL'} {!import.meta.env.VITE_CF_UAA_URL && 'VITE_CF_UAA_URL'}
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;
