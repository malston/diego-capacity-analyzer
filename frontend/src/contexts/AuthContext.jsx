// ABOUTME: Authentication context providing auth state to React components
// ABOUTME: Uses BFF OAuth pattern with backend session management

import { createContext, useContext, useState, useEffect } from "react";
import { cfAuth } from "../services/cfAuth";
import { cfApi } from "../services/cfApi";

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Check authentication status on mount via backend /api/v1/auth/me
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const authenticated = await cfAuth.isAuthenticated();
        setIsAuthenticated(authenticated);

        if (authenticated) {
          const userInfo = await cfAuth.getUserInfo();
          setUser(userInfo);
        }
      } catch (err) {
        console.warn("Auth check failed:", err.message);
        setIsAuthenticated(false);
        setUser(null);
      } finally {
        setLoading(false);
      }
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

      const userInfo = await cfAuth.getUserInfo();
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
  const logout = async () => {
    await cfAuth.logout();
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
      throw new Error("Could not connect to Cloud Foundry API: " + err.message);
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

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

/**
 * Hook to use authentication context
 * @returns {Object} Authentication context
 */
export const useAuth = () => {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return context;
};
