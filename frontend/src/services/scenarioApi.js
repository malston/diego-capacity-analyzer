// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison with BFF auth

import { withCSRFToken } from "../utils/csrf";
import { apiFetch } from "./apiClient";

const API_URL = import.meta.env.VITE_API_URL || "";

export const scenarioApi = {
  async setManualInfrastructure(data) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/manual`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(data),
    });
  },

  async compareScenario(input) {
    return apiFetch(`${API_URL}/api/v1/scenario/compare`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },

  async getLiveInfrastructure() {
    return apiFetch(`${API_URL}/api/v1/infrastructure`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
  },

  async getInfrastructureStatus() {
    try {
      return await apiFetch(`${API_URL}/api/v1/infrastructure/status`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });
    } catch {
      return { vsphere_configured: false, has_data: false };
    }
  },

  async setInfrastructureState(state) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/state`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(state),
    });
  },

  async calculatePlanning(input) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/planning`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },
};
