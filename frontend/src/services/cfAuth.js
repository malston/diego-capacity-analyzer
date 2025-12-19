/**
 * Cloud Foundry UAA Authentication Service
 * Handles OAuth2 authentication with CF UAA
 */

const UAA_URL = import.meta.env.VITE_CF_UAA_URL || 'https://login.sys.example.com';
const CF_CLIENT_ID = 'cf';
const CF_CLIENT_SECRET = '';

class CFAuthService {
  constructor() {
    this.token = null;
    this.refreshToken = null;
    this.tokenExpiry = null;
  }

  /**
   * Authenticate with CF UAA using password grant
   * @param {string} username - CF username
   * @param {string} password - CF password
   * @returns {Promise<Object>} Authentication response with access token
   */
  async login(username, password) {
    try {
      const response = await fetch(`${UAA_URL}/oauth/token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Authorization': `Basic ${btoa(`${CF_CLIENT_ID}:${CF_CLIENT_SECRET}`)}`,
          'Accept': 'application/json',
        },
        body: new URLSearchParams({
          grant_type: 'password',
          username,
          password,
          response_type: 'token',
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error_description || 'Authentication failed');
      }

      const data = await response.json();
      
      // Store tokens
      this.token = data.access_token;
      this.refreshToken = data.refresh_token;
      this.tokenExpiry = Date.now() + (data.expires_in * 1000);

      // Store in sessionStorage for persistence across page refreshes
      sessionStorage.setItem('cf_token', this.token);
      sessionStorage.setItem('cf_refresh_token', this.refreshToken);
      sessionStorage.setItem('cf_token_expiry', this.tokenExpiry.toString());

      return {
        success: true,
        token: this.token,
        expiresIn: data.expires_in,
      };
    } catch (error) {
      console.error('CF Authentication error:', error);
      throw error;
    }
  }

  /**
   * Refresh the access token using refresh token
   * @returns {Promise<Object>} New access token
   */
  async refreshAccessToken() {
    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await fetch(`${UAA_URL}/oauth/token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Authorization': `Basic ${btoa(`${CF_CLIENT_ID}:${CF_CLIENT_SECRET}`)}`,
          'Accept': 'application/json',
        },
        body: new URLSearchParams({
          grant_type: 'refresh_token',
          refresh_token: this.refreshToken,
        }),
      });

      if (!response.ok) {
        // If refresh fails, clear tokens and require re-login
        this.logout();
        throw new Error('Token refresh failed. Please login again.');
      }

      const data = await response.json();
      
      this.token = data.access_token;
      this.refreshToken = data.refresh_token;
      this.tokenExpiry = Date.now() + (data.expires_in * 1000);

      // Update sessionStorage
      sessionStorage.setItem('cf_token', this.token);
      sessionStorage.setItem('cf_refresh_token', this.refreshToken);
      sessionStorage.setItem('cf_token_expiry', this.tokenExpiry.toString());

      return {
        success: true,
        token: this.token,
      };
    } catch (error) {
      console.error('Token refresh error:', error);
      throw error;
    }
  }

  /**
   * Get current access token, refreshing if necessary
   * @returns {Promise<string>} Valid access token
   */
  async getToken() {
    // Try to restore from sessionStorage if not in memory
    if (!this.token) {
      this.token = sessionStorage.getItem('cf_token');
      this.refreshToken = sessionStorage.getItem('cf_refresh_token');
      const expiry = sessionStorage.getItem('cf_token_expiry');
      this.tokenExpiry = expiry ? parseInt(expiry) : null;
    }

    // If no token at all, user needs to login
    if (!this.token) {
      throw new Error('Not authenticated. Please login.');
    }

    // Check if token is expired (with 60 second buffer)
    if (this.tokenExpiry && Date.now() > (this.tokenExpiry - 60000)) {
      // Token expired or about to expire, refresh it
      await this.refreshAccessToken();
    }

    return this.token;
  }

  /**
   * Check if user is currently authenticated
   * @returns {boolean} Authentication status
   */
  isAuthenticated() {
    const token = this.token || sessionStorage.getItem('cf_token');
    const expiry = this.tokenExpiry || parseInt(sessionStorage.getItem('cf_token_expiry') || '0');
    
    return !!(token && expiry && Date.now() < expiry);
  }

  /**
   * Logout and clear all tokens
   */
  logout() {
    this.token = null;
    this.refreshToken = null;
    this.tokenExpiry = null;
    
    sessionStorage.removeItem('cf_token');
    sessionStorage.removeItem('cf_refresh_token');
    sessionStorage.removeItem('cf_token_expiry');
  }

  /**
   * Get user info from token
   * @returns {Object|null} Decoded user info
   */
  getUserInfo() {
    if (!this.token) {
      return null;
    }

    try {
      // Decode JWT token (simple base64 decode of payload)
      const payload = this.token.split('.')[1];
      const decoded = JSON.parse(atob(payload));
      
      return {
        username: decoded.user_name,
        userId: decoded.user_id,
        email: decoded.email,
        scopes: decoded.scope,
      };
    } catch (error) {
      console.error('Error decoding token:', error);
      return null;
    }
  }
}

// Export singleton instance
export const cfAuth = new CFAuthService();
