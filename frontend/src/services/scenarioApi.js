// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison with BFF auth

import { withCSRFToken } from "../utils/csrf";
import { apiFetch, ApiConnectionError } from "./apiClient";

const API_URL = import.meta.env.VITE_API_URL || "";

export const scenarioApi = {
  /**
   * Submit manual infrastructure data
   * @param {Object} data - ManualInput object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setManualInfrastructure(data) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/manual`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(data),
    });
  },

  /**
   * Compare current vs proposed scenario
   * @param {Object} input - ScenarioInput object
   * @returns {Promise<Object>} ScenarioComparison
   */
  async compareScenario(input) {
    return apiFetch(`${API_URL}/api/v1/scenario/compare`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },

  /**
   * Fetch live infrastructure data from vSphere
   * @returns {Promise<Object>} InfrastructureState
   */
  async getLiveInfrastructure() {
    return apiFetch(`${API_URL}/api/v1/infrastructure`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
  },

  /**
   * Get current infrastructure status
   * @returns {Promise<Object>} Status including vsphere_configured, has_data, source
   */
  async getInfrastructureStatus() {
    try {
      return await apiFetch(`${API_URL}/api/v1/infrastructure/status`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });
    } catch (err) {
      if (err instanceof ApiConnectionError) throw err;
      return { vsphere_configured: false, has_data: false };
    }
  },

  /**
   * Set infrastructure state directly (for vSphere data loaded from cache)
   * @param {Object} state - InfrastructureState object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setInfrastructureState(state) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/state`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(state),
    });
  },

  /**
   * Calculate max deployable cells given IaaS capacity
   * @param {Object} input - PlanningInput with cell_memory_gb, cell_cpu, overhead_pct
   * @returns {Promise<Object>} PlanningResponse with result and recommendations
   */
  async calculatePlanning(input) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/planning`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },
};
