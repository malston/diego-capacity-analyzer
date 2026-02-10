// ABOUTME: CSRF token utility for reading token from cookie
// ABOUTME: Used by API services to add X-CSRF-Token header to requests

/**
 * Get CSRF token from cookie
 * @returns {string|null} CSRF token or null if not found
 */
export function getCSRFToken() {
  const match = document.cookie.match(/DIEGO_CSRF=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : null;
}

/**
 * Get headers with CSRF token included
 * @param {Object} headers - Existing headers object
 * @returns {Object} Headers with X-CSRF-Token added if available
 */
export function withCSRFToken(headers = {}) {
  const csrfToken = getCSRFToken();
  if (csrfToken) {
    return {
      ...headers,
      "X-CSRF-Token": csrfToken,
    };
  }
  return headers;
}
