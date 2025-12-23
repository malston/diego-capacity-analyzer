/**
 * Authentication Context
 * Provides authentication state and methods to React components
 */

import { createContext, useContext, useState, useEffect } from 'react';
import { cfAuth } from '../services/cfAuth';
import { cfApi } from '../services/cfApi';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Check authentication status on mount
  useEffect(() => {
    const checkAuth = () => {
      const authenticated = cfAuth.isAuthenticated();
      setIsAuthenticated(authenticated);
      
      if (authenticated) {
        const userInfo = cfAuth.getUserInfo();
        setUser(userInfo);
      }
      
      setLoading(false);
    };

    checkAuth();
  }, []);

  /**
   * Login with username and password
   * @param {string} username 
   * @param {string} password 
   */
  const login = async (username, password) => {
    try {
      setLoading(true);
      setError(null);
      
      await cfAuth.login(username, password);
      
      const userInfo = cfAuth.getUserInfo();
      setUser(userInfo);
      setIsAuthenticated(true);
      
      return { success: true };
    } catch (err) {
      setError(err.message);
      setIsAuthenticated(false);
      setUser(null);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  /**
   * Logout and clear authentication
   */
  const logout = () => {
    cfAuth.logout();
    setIsAuthenticated(false);
    setUser(null);
    setError(null);
  };

  /**
   * Verify connection to CF API
   */
  const verifyConnection = async () => {
    try {
      await cfApi.getInfo();
      return { success: true };
    } catch (err) {
      throw new Error('Could not connect to Cloud Foundry API: ' + err.message);
    }
  };

  const value = {
    isAuthenticated,
    user,
    loading,
    error,
    login,
    logout,
    verifyConnection,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

/**
 * Hook to use authentication context
 * @returns {Object} Authentication context
 */
export const useAuth = () => {
  const context = useContext(AuthContext);
  
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  
  return context;
};
