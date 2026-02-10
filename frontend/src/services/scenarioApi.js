// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison with BFF auth

import { withCSRFToken } from "../utils/csrf";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export const scenarioApi = {
  /**
   * Submit manual infrastructure data
   * @param {Object} data - ManualInput object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setManualInfrastructure(data) {
    const response = await fetch(`${API_URL}/api/v1/infrastructure/manual`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || "Failed to set infrastructure");
    }
    return response.json();
  },

  /**
   * Compare current vs proposed scenario
   * @param {Object} input - ScenarioInput object
   * @returns {Promise<Object>} ScenarioComparison
   */
  async compareScenario(input) {
    const response = await fetch(`${API_URL}/api/v1/scenario/compare`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || "Failed to compare scenario");
    }
    return response.json();
  },

  /**
   * Fetch live infrastructure data from vSphere
   * @returns {Promise<Object>} InfrastructureState
   */
  async getLiveInfrastructure() {
    const response = await fetch(`${API_URL}/api/v1/infrastructure`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || "Failed to fetch live infrastructure");
    }
    return response.json();
  },

  /**
   * Get current infrastructure status
   * @returns {Promise<Object>} Status including vsphere_configured, has_data, source
   */
  async getInfrastructureStatus() {
    const response = await fetch(`${API_URL}/api/v1/infrastructure/status`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
    if (!response.ok) {
      return { vsphere_configured: false, has_data: false };
    }
    return response.json();
  },

  /**
   * Set infrastructure state directly (for vSphere data loaded from cache)
   * @param {Object} state - InfrastructureState object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setInfrastructureState(state) {
    const response = await fetch(`${API_URL}/api/v1/infrastructure/state`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(state),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || "Failed to set infrastructure state");
    }
    return response.json();
  },

  /**
   * Calculate max deployable cells given IaaS capacity
   * @param {Object} input - PlanningInput with cell_memory_gb, cell_cpu, overhead_pct
   * @returns {Promise<Object>} PlanningResponse with result and recommendations
   */
  async calculatePlanning(input) {
    const response = await fetch(`${API_URL}/api/v1/infrastructure/planning`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || "Failed to calculate planning");
    }
    return response.json();
  },
};
