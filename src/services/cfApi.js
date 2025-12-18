/**
 * Cloud Foundry API Service
 * Handles all CF API interactions for cell and app data
 */

import { cfAuth } from './cfAuth';

const CF_API_URL = import.meta.env.VITE_CF_API_URL || 'https://api.sys.example.com';

class CFApiService {
  /**
   * Make authenticated request to CF API
   * @param {string} endpoint - API endpoint
   * @param {Object} options - Fetch options
   * @returns {Promise<Object>} API response
   */
  async request(endpoint, options = {}) {
    try {
      const token = await cfAuth.getToken();
      
      const response = await fetch(`${CF_API_URL}${endpoint}`, {
        ...options,
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
          ...options.headers,
        },
      });

      if (!response.ok) {
        if (response.status === 401) {
          // Token might be invalid, try to refresh
          await cfAuth.refreshAccessToken();
          // Retry the request
          return this.request(endpoint, options);
        }
        
        const error = await response.json();
        throw new Error(error.description || `API Error: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('CF API request error:', error);
      throw error;
    }
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
      nextUrl = data.pagination?.next?.href ? 
        data.pagination.next.href.replace(CF_API_URL, '') : null;
    }

    return allResources;
  }

  /**
   * Get all isolation segments
   * @returns {Promise<Array>} Isolation segments
   */
  async getIsolationSegments() {
    try {
      const data = await this.request('/v3/isolation_segments');
      return data.resources || [];
    } catch (error) {
      console.error('Error fetching isolation segments:', error);
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
      const apps = await this.fetchAllPages('/v3/apps');
      
      // Fetch processes for each app
      const appsWithProcesses = await Promise.all(
        apps.map(async (app) => {
          try {
            const processes = await this.request(`/v3/apps/${app.guid}/processes`);
            const webProcess = processes.resources?.find(p => p.type === 'web');
            
            // Get process stats for memory usage
            let stats = null;
            if (webProcess) {
              try {
                stats = await this.request(`/v3/processes/${webProcess.guid}/stats`);
              } catch (e) {
                console.warn(`Could not fetch stats for ${app.name}:`, e.message);
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
              actual_mb: stats?.resources ? 
                Math.round(
                  stats.resources.reduce((sum, s) => sum + (s.usage?.mem || 0), 0) / 
                  (stats.resources.length || 1) / (1024 * 1024)
                ) : webProcess?.memory_in_mb || 0,
              // Get isolation segment from relationships
              isolation_segment: app.relationships?.space?.data?.guid || 'default',
            };
          } catch (error) {
            console.warn(`Error processing app ${app.name}:`, error.message);
            return null;
          }
        })
      );

      return appsWithProcesses.filter(app => app !== null);
    } catch (error) {
      console.error('Error fetching applications:', error);
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
      // Try to get cell info from internal endpoint (requires cloud_controller.admin scope)
      const data = await this.request('/v3/info');
      
      // CF v3 API doesn't directly expose diego cell info
      // You'll need to get this from BOSH or a custom metrics endpoint
      console.warn('Diego cell info not available via standard CF API');
      console.warn('Consider using BOSH API or Tanzu Hub metrics');
      
      return [];
    } catch (error) {
      console.error('Error fetching diego cells:', error);
      return [];
    }
  }

  /**
   * Get CF info (API version, etc.)
   * @returns {Promise<Object>} CF info
   */
  async getInfo() {
    try {
      // This endpoint doesn't require authentication
      const response = await fetch(`${CF_API_URL}/v3/info`);
      return await response.json();
    } catch (error) {
      console.error('Error fetching CF info:', error);
      throw error;
    }
  }

  /**
   * BOSH Integration - Get diego cell data from BOSH director
   * This requires BOSH credentials and is typically done server-side
   * @param {string} boshUrl - BOSH director URL
   * @param {string} boshToken - BOSH auth token
   * @param {string} deploymentName - CF deployment name
   * @returns {Promise<Array>} Diego cell VMs
   */
  async getDiegoCellsFromBOSH(boshUrl, boshToken, deploymentName) {
    try {
      const response = await fetch(`${boshUrl}/deployments/${deploymentName}/vms`, {
        headers: {
          'Authorization': `Bearer ${boshToken}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`BOSH API error: ${response.status}`);
      }

      const vms = await response.json();
      
      // Filter for diego_cell VMs and extract capacity info
      return vms
        .filter(vm => vm.job_name === 'diego_cell' || vm.job_name === 'compute')
        .map(vm => ({
          id: vm.id,
          name: `${vm.job_name}/${vm.index}`,
          memory_mb: vm.vitals?.mem?.percent ? 
            Math.round((vm.vitals.mem.kb * 1024) / (1024 * 1024)) : 16384,
          cpu_percent: vm.vitals?.cpu?.sys || 0,
          disk_mb: vm.vitals?.disk?.system?.percent || 0,
          isolation_segment: vm.az || 'default', // Use AZ as proxy for segment
          state: vm.state,
        }));
    } catch (error) {
      console.error('Error fetching BOSH data:', error);
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
        return 'default';
      }

      const segment = await this.request(`/v3/isolation_segments/${segmentGuid}`);
      return segment.name || 'default';
    } catch (error) {
      console.warn('Could not determine isolation segment:', error.message);
      return 'default';
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
      const spaceGuids = [...new Set(apps.map(app => app.isolation_segment))];
      
      // Fetch isolation segments for each space
      const segmentMap = {};
      await Promise.all(
        spaceGuids.map(async (spaceGuid) => {
          if (spaceGuid && spaceGuid !== 'default') {
            try {
              const space = await this.request(`/v3/spaces/${spaceGuid}`);
              const segmentGuid = space.relationships?.isolation_segment?.data?.guid;
              
              if (segmentGuid) {
                const segment = await this.request(`/v3/isolation_segments/${segmentGuid}`);
                segmentMap[spaceGuid] = segment.name;
              }
            } catch (e) {
              console.warn(`Could not fetch segment for space ${spaceGuid}`);
            }
          }
        })
      );

      // Map apps to their isolation segment names
      return apps.map(app => ({
        ...app,
        isolation_segment: segmentMap[app.isolation_segment] || 'default',
      }));
    } catch (error) {
      console.error('Error fetching apps with segments:', error);
      throw error;
    }
  }
}

// Export singleton instance
export const cfApi = new CFApiService();
