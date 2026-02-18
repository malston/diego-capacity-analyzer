// ABOUTME: Shared fetch wrapper with structured error handling
// ABOUTME: Catches network errors and throws user-friendly ApiConnectionError

/**
 * Structured error for network/connection failures.
 * `message` contains a user-friendly summary.
 * `detail` contains diagnostic information for troubleshooting.
 */
export class ApiConnectionError extends Error {
  constructor(url) {
    super("Unable to reach the server");
    this.name = "ApiConnectionError";

    let origin;
    try {
      origin = new URL(url, window.location.origin).origin;
    } catch {
      origin = "the configured backend";
    }

    this.detail =
      `The backend at ${origin} is not responding. ` +
      "Common causes: the server isn't running, a firewall is blocking " +
      "the connection, or CORS is misconfigured. Check that the backend " +
      "is running and accessible.";
  }
}

/**
 * Fetch wrapper that classifies errors into user-friendly categories.
 *
 * - Network failures (TypeError) become ApiConnectionError
 * - HTTP errors parse the JSON body for the server's error message
 * - Successful responses return parsed JSON
 *
 * @param {string} url - Request URL
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<any>} Parsed JSON response
 */
export async function apiFetch(url, options) {
  let response;
  try {
    response = await fetch(url, options);
  } catch (err) {
    if (err instanceof TypeError) {
      throw new ApiConnectionError(url);
    }
    throw err;
  }

  if (!response.ok) {
    let message;
    try {
      const body = await response.json();
      message = body.error || body.description || body.message;
    } catch {
      // Response body is not JSON
    }
    throw new Error(
      message || `Server error: ${response.status} ${response.statusText}`,
    );
  }

  return response.json();
}
