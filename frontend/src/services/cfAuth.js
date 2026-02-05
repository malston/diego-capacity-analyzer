// ABOUTME: Cloud Foundry authentication service using BFF OAuth pattern
// ABOUTME: Authenticates via backend endpoints, never exposes tokens to JavaScript

import { withCSRFToken } from "../utils/csrf";

/**
 * Cloud Foundry Authentication Service (BFF Pattern)
 * Handles authentication through backend endpoints with httpOnly cookies.
 * Tokens are never exposed to JavaScript - backend manages all token operations.
 */
class CFAuthService {
  constructor() {
    this._cachedUser = null;
  }

  /**
   * Authenticate with CF UAA via backend BFF endpoint
   * @param {string} username - CF username
   * @param {string} password - CF password
   * @returns {Promise<Object>} Authentication response with user info
   */
  async login(username, password) {
    try {
      const response = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          username,
          password,
        }),
      });

      const data = await response.json();

      if (!response.ok || !data.success) {
        throw new Error(data.error || "Authentication failed");
      }

      // Cache user info
      this._cachedUser = {
        username: data.username,
        userId: data.user_id,
      };

      return {
        success: true,
        username: data.username,
        userId: data.user_id,
      };
    } catch (error) {
      console.error("CF Authentication error:", error);
      throw error;
    }
  }

  /**
   * Logout and clear session
   */
  async logout() {
    try {
      await fetch("/api/v1/auth/logout", {
        method: "POST",
        headers: withCSRFToken(),
        credentials: "include",
      });
    } finally {
      // Always clear cached user, even on network error
      this._cachedUser = null;
    }
  }

  /**
   * Check if user is currently authenticated
   * @returns {Promise<boolean>} Authentication status
   */
  async isAuthenticated() {
    try {
      const response = await fetch("/api/v1/auth/me", {
        method: "GET",
        credentials: "include",
      });

      if (!response.ok) {
        return false;
      }

      const data = await response.json();
      return data.authenticated === true;
    } catch {
      return false;
    }
  }

  /**
   * Get user info from session
   * @returns {Promise<Object|null>} User info or null if not authenticated
   */
  async getUserInfo() {
    try {
      const response = await fetch("/api/v1/auth/me", {
        method: "GET",
        credentials: "include",
      });

      if (!response.ok) {
        return null;
      }

      const data = await response.json();

      if (!data.authenticated) {
        return null;
      }

      return {
        username: data.username,
        userId: data.user_id,
      };
    } catch {
      return null;
    }
  }
}

// Export singleton instance
export const cfAuth = new CFAuthService();
