// frontend/src/services/scenarioApi.js
// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const scenarioApi = {
  /**
   * Submit manual infrastructure data
   * @param {Object} data - ManualInput object
   * @returns {Promise<Object>} InfrastructureState
   */
  async setManualInfrastructure(data) {
    const response = await fetch(`${API_URL}/api/infrastructure/manual`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to set infrastructure');
    }
    return response.json();
  },

  /**
   * Compare current vs proposed scenario
   * @param {Object} input - ScenarioInput object
   * @returns {Promise<Object>} ScenarioComparison
   */
  async compareScenario(input) {
    const response = await fetch(`${API_URL}/api/scenario/compare`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to compare scenario');
    }
    return response.json();
  },
};
