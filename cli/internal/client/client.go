// ABOUTME: HTTP client for Diego Capacity Analyzer API
// ABOUTME: Wraps API calls with proper error handling for CLI usage

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is the API client for Diego Capacity Analyzer backend
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new API client with the given base URL
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthResponse represents the /api/health endpoint response
type HealthResponse struct {
	CFAPI       string      `json:"cf_api"`
	BOSHAPI     string      `json:"bosh_api"`
	CacheStatus CacheStatus `json:"cache_status"`
}

// CacheStatus represents cache state in health response
type CacheStatus struct {
	CellsCached bool `json:"cells_cached"`
	AppsCached  bool `json:"apps_cached"`
}

// InfrastructureStatus represents the /api/infrastructure/status endpoint response
type InfrastructureStatus struct {
	HasData               bool    `json:"has_data"`
	Source                string  `json:"source,omitempty"`
	Name                  string  `json:"name,omitempty"`
	ClusterCount          int     `json:"cluster_count,omitempty"`
	HostCount             int     `json:"host_count,omitempty"`
	CellCount             int     `json:"cell_count,omitempty"`
	ConstrainingResource  string  `json:"constraining_resource,omitempty"`
	BottleneckSummary     string  `json:"bottleneck_summary,omitempty"`
	VSphereConfigured     bool    `json:"vsphere_configured"`
	MemoryUtilization     float64 `json:"memory_utilization,omitempty"`
	N1CapacityPercent     float64 `json:"n1_capacity_percent,omitempty"`
	N1Status              string  `json:"n1_status,omitempty"`
	HAMinFailuresSurvived int     `json:"ha_min_host_failures_survived,omitempty"`
	HAStatus              string  `json:"ha_status,omitempty"`
	Timestamp             string  `json:"timestamp,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code"`
}

// ClusterState represents computed metrics for a single cluster
type ClusterState struct {
	Name                         string  `json:"name"`
	HostCount                    int     `json:"host_count"`
	MemoryGB                     int     `json:"memory_gb"`
	CPUCores                     int     `json:"cpu_cores"`
	MemoryGBPerHost              int     `json:"memory_gb_per_host"`
	CPUCoresPerHost              int     `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int     `json:"ha_admission_control_percentage"`
	HAUsableMemoryGB             int     `json:"ha_usable_memory_gb"`
	HAHostFailuresSurvived       int     `json:"ha_host_failures_survived"`
	HAStatus                     string  `json:"ha_status"`
	N1MemoryGB                   int     `json:"n1_memory_gb"`
	DiegoCellCount               int     `json:"diego_cell_count"`
	DiegoCellMemoryGB            int     `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int     `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int     `json:"diego_cell_disk_gb"`
	TotalVCPUs                   int     `json:"total_vcpus"`
	TotalCellMemoryGB            int     `json:"total_cell_memory_gb"`
	VCPURatio                    float64 `json:"vcpu_ratio"`
}

// InfrastructureState represents the full infrastructure data
type InfrastructureState struct {
	Source                       string         `json:"source"`
	Name                         string         `json:"name"`
	Clusters                     []ClusterState `json:"clusters"`
	TotalMemoryGB                int            `json:"total_memory_gb"`
	TotalN1MemoryGB              int            `json:"total_n1_memory_gb"`
	TotalHAUsableMemoryGB        int            `json:"total_ha_usable_memory_gb"`
	HAMinHostFailuresSurvived    int            `json:"ha_min_host_failures_survived"`
	HAStatus                     string         `json:"ha_status"`
	TotalCellMemoryGB            int            `json:"total_cell_memory_gb"`
	HostMemoryUtilizationPercent float64        `json:"host_memory_utilization_percent"`
	HostCPUUtilizationPercent    float64        `json:"host_cpu_utilization_percent"`
	TotalHostCount               int            `json:"total_host_count"`
	TotalCellCount               int            `json:"total_cell_count"`
	TotalCPUCores                int            `json:"total_cpu_cores"`
	TotalVCPUs                   int            `json:"total_vcpus"`
	VCPURatio                    float64        `json:"vcpu_ratio"`
	CPURiskLevel                 string         `json:"cpu_risk_level"`
	PlatformVMsGB                int            `json:"platform_vms_gb"`
	TotalAppMemoryGB             int            `json:"total_app_memory_gb"`
	TotalAppDiskGB               int            `json:"total_app_disk_gb"`
	TotalAppInstances            int            `json:"total_app_instances"`
	Timestamp                    string         `json:"timestamp"`
	Cached                       bool           `json:"cached"`
}

// Health calls the /api/health endpoint
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("request canceled")
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timed out")
		}
		return nil, fmt.Errorf("cannot connect to backend at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("backend error: %s", errResp.Error)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &health, nil
}

// InfrastructureStatus calls the /api/infrastructure/status endpoint
func (c *Client) InfrastructureStatus(ctx context.Context) (*InfrastructureStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/infrastructure/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("request canceled")
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timed out")
		}
		return nil, fmt.Errorf("cannot connect to backend at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("backend error: %s", errResp.Error)
	}

	var status InfrastructureStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &status, nil
}

// GetInfrastructure calls GET /api/infrastructure
func (c *Client) GetInfrastructure(ctx context.Context) (*InfrastructureState, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/infrastructure", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.handleRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var infra InfrastructureState
	if err := json.NewDecoder(resp.Body).Decode(&infra); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &infra, nil
}

// handleRequestError converts context errors to user-friendly messages
func (c *Client) handleRequestError(ctx context.Context, err error) error {
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("request canceled")
	}
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("request timed out")
	}
	return fmt.Errorf("cannot connect to backend at %s: %w", c.baseURL, err)
}

// handleErrorResponse parses API error responses
func (c *Client) handleErrorResponse(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}
	return fmt.Errorf("backend error: %s", errResp.Error)
}
