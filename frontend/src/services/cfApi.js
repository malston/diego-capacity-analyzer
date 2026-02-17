// ABOUTME: Cloud Foundry API service using BFF proxy pattern
// ABOUTME: All CF API calls go through backend proxy - tokens never exposed to JavaScript

import { ApiConnectionError } from "./apiClient";

const API_URL = import.meta.env.VITE_API_URL || "";

class CFApiService {
  /**
   * Make authenticated request to CF API via backend proxy
   * @param {string} endpoint - API endpoint (e.g., "/v3/apps")
   * @param {Object} options - Fetch options
   * @returns {Promise<Object>} API response
   */
  async request(endpoint, options = {}) {
    // Map CF API path to our backend proxy path
    const proxyPath = this.mapToProxyPath(endpoint);

    let response;
    try {
      response = await fetch(`${API_URL}${proxyPath}`, {
        ...options,
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
        credentials: "include",
      });
    } catch (err) {
      if (err instanceof TypeError) {
        throw new ApiConnectionError(`${API_URL}${proxyPath}`);
      }
      throw err;
    }

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Authentication required. Please login.");
      }

      let errorMsg = `API Error: ${response.status}`;
      try {
        const error = await response.json();
        errorMsg = error.description || error.error || errorMsg;
      } catch {
        // Use default error message if JSON parsing fails
      }
      throw new Error(errorMsg);
    }

    return await response.json();
  }

  /**
   * Map CF API endpoints to our backend proxy endpoints
   * @param {string} cfPath - CF API path
   * @returns {string} Backend proxy path
   */
  mapToProxyPath(cfPath) {
    // /v3/isolation_segments -> /api/v1/cf/isolation-segments
    // /v3/isolation_segments/{guid} -> /api/v1/cf/isolation-segments/{guid}
    // /v3/apps -> /api/v1/cf/apps
    // /v3/apps/{guid}/processes -> /api/v1/cf/apps/{guid}/processes
    // /v3/processes/{guid}/stats -> /api/v1/cf/processes/{guid}/stats
    // /v3/spaces/{guid} -> /api/v1/cf/spaces/{guid}

    // Handle pagination URLs that include full CF API URL
    const v3Index = cfPath.indexOf("/v3/");
    if (v3Index !== -1) {
      cfPath = cfPath.substring(v3Index);
    }

    // Use startsWith to avoid unintended replacements in edge cases
    // (e.g., paths like "/v3/apps/v3/apps-test" would be incorrectly replaced by chained .replace())
    const mappings = [
      ["/v3/isolation_segments", "/api/v1/cf/isolation-segments"],
      ["/v3/apps", "/api/v1/cf/apps"],
      ["/v3/processes", "/api/v1/cf/processes"],
      ["/v3/spaces", "/api/v1/cf/spaces"],
      ["/v3/info", "/api/v1/cf/info"],
    ];

    for (const [cfPrefix, proxyPrefix] of mappings) {
      if (cfPath.startsWith(cfPrefix)) {
        return proxyPrefix + cfPath.substring(cfPrefix.length);
      }
    }

    // If no mapping applies, return the path unchanged
    return cfPath;
  }

  /**
   * Fetch all pages of a paginated CF API endpoint
   * @param {string} endpoint - API endpoint
   * @returns {Promise<Array>} All resources from all pages
   */
  async fetchAllPages(endpoint) {
    let allResources = [];
    let nextUrl = endpoint;

    while (nextUrl) {
      const data = await this.request(nextUrl);
      allResources = allResources.concat(data.resources || []);

      // Check for next page
      nextUrl = data.pagination?.next?.href || null;
    }

    return allResources;
  }

  /**
   * Get all isolation segments
   * @returns {Promise<Array>} Isolation segments
   */
  async getIsolationSegments() {
    try {
      const data = await this.request("/v3/isolation_segments");
      return data.resources || [];
    } catch (error) {
      console.error("Error fetching isolation segments:", error);
      return [];
    }
  }

  /**
   * Get all applications with their processes
   * @returns {Promise<Array>} Apps with process information
   */
  async getApplications() {
    try {
      // Fetch all apps
      const apps = await this.fetchAllPages("/v3/apps");

      // Fetch processes for each app
      const appsWithProcesses = await Promise.all(
        apps.map(async (app) => {
          try {
            const processes = await this.request(
              `/v3/apps/${app.guid}/processes`,
            );
            const webProcess = processes.resources?.find(
              (p) => p.type === "web",
            );

            // Get process stats for memory usage
            let stats = null;
            if (webProcess) {
              try {
                stats = await this.request(
                  `/v3/processes/${webProcess.guid}/stats`,
                );
              } catch (e) {
                console.warn(
                  `Could not fetch stats for ${app.name}:`,
                  e.message,
                );
              }
            }

            return {
              name: app.name,
              guid: app.guid,
              state: app.state,
              instances: webProcess?.instances || 0,
              requested_mb: webProcess?.memory_in_mb || 0,
              disk_mb: webProcess?.disk_in_mb || 0,
              // Calculate actual usage from stats
              actual_mb: stats?.resources
                ? Math.round(
                    stats.resources.reduce(
                      (sum, s) => sum + (s.usage?.mem || 0),
                      0,
                    ) /
                      (stats.resources.length || 1) /
                      (1024 * 1024),
                  )
                : webProcess?.memory_in_mb || 0,
              // Get isolation segment from relationships
              isolation_segment:
                app.relationships?.space?.data?.guid || "default",
            };
          } catch (error) {
            console.warn(`Error processing app ${app.name}:`, error.message);
            return null;
          }
        }),
      );

      return appsWithProcesses.filter((app) => app !== null);
    } catch (error) {
      console.error("Error fetching applications:", error);
      throw error;
    }
  }

  /**
   * Get diego cell information from CF API
   * Note: This requires admin privileges and access to diego endpoint
   * @returns {Promise<Array>} Diego cell information
   */
  async getDiegoCells() {
    try {
      // CF v3 API doesn't directly expose diego cell info
      // You'll need to get this from BOSH or a custom metrics endpoint
      console.warn("Diego cell info not available via standard CF API");
      console.warn("Consider using BOSH API or Tanzu Hub metrics");

      return [];
    } catch (error) {
      console.error("Error fetching diego cells:", error);
      return [];
    }
  }

  /**
   * Get CF info (API version, etc.)
   * @returns {Promise<Object>} CF info
   */
  async getInfo() {
    try {
      let response;
      try {
        response = await fetch(`${API_URL}/api/v1/health`, {
          credentials: "include",
        });
      } catch (err) {
        if (err instanceof TypeError) {
          throw new ApiConnectionError(`${API_URL}/api/v1/health`);
        }
        throw err;
      }
      return await response.json();
    } catch (error) {
      console.error("Error fetching CF info:", error);
      throw error;
    }
  }

  /**
   * Helper: Map space GUID to isolation segment name
   * @param {string} spaceGuid - Space GUID
   * @returns {Promise<string>} Isolation segment name
   */
  async getIsolationSegmentForSpace(spaceGuid) {
    try {
      const space = await this.request(`/v3/spaces/${spaceGuid}`);
      const segmentGuid = space.relationships?.isolation_segment?.data?.guid;

      if (!segmentGuid) {
        return "default";
      }

      const segment = await this.request(
        `/v3/isolation_segments/${segmentGuid}`,
      );
      return segment.name || "default";
    } catch (error) {
      console.warn("Could not determine isolation segment:", error.message);
      return "default";
    }
  }

  /**
   * Get enriched app data with isolation segment names
   * @returns {Promise<Array>} Apps with isolation segment names
   */
  async getAppsWithSegments() {
    try {
      const apps = await this.getApplications();

      // Get unique space GUIDs
      const spaceGuids = [...new Set(apps.map((app) => app.isolation_segment))];

      // Fetch isolation segments for each space
      const segmentMap = {};
      await Promise.all(
        spaceGuids.map(async (spaceGuid) => {
          if (spaceGuid && spaceGuid !== "default") {
            try {
              const space = await this.request(`/v3/spaces/${spaceGuid}`);
              const segmentGuid =
                space.relationships?.isolation_segment?.data?.guid;

              if (segmentGuid) {
                const segment = await this.request(
                  `/v3/isolation_segments/${segmentGuid}`,
                );
                segmentMap[spaceGuid] = segment.name;
              }
            } catch {
              console.warn(`Could not fetch segment for space ${spaceGuid}`);
            }
          }
        }),
      );

      // Map apps to their isolation segment names
      return apps.map((app) => ({
        ...app,
        isolation_segment: segmentMap[app.isolation_segment] || "default",
      }));
    } catch (error) {
      console.error("Error fetching apps with segments:", error);
      throw error;
    }
  }
}

// Export singleton instance
export const cfApi = new CFApiService();
