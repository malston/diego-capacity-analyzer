// ABOUTME: HTTP client for Diego Capacity Analyzer API
// ABOUTME: Wraps API calls with proper error handling for CLI usage

package client

import (
	"bytes"
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

// ClusterInput represents user-provided cluster configuration
type ClusterInput struct {
	Name                         string `json:"name"`
	HostCount                    int    `json:"host_count"`
	MemoryGBPerHost              int    `json:"memory_gb_per_host"`
	CPUCoresPerHost              int    `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int    `json:"ha_admission_control_percentage"`
	DiegoCellCount               int    `json:"diego_cell_count"`
	DiegoCellMemoryGB            int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int    `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int    `json:"diego_cell_disk_gb"`
}

// ManualInput represents user-provided infrastructure data
type ManualInput struct {
	Name              string         `json:"name"`
	Clusters          []ClusterInput `json:"clusters"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppDiskGB    int            `json:"total_app_disk_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
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

// SetManualInfrastructure calls POST /api/infrastructure/manual
func (c *Client) SetManualInfrastructure(ctx context.Context, input *ManualInput) (*InfrastructureState, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/infrastructure/manual", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

// ScenarioInput represents proposed changes for what-if analysis
type ScenarioInput struct {
	ProposedCellMemoryGB int      `json:"proposed_cell_memory_gb"`
	ProposedCellCPU      int      `json:"proposed_cell_cpu"`
	ProposedCellDiskGB   int      `json:"proposed_cell_disk_gb"`
	ProposedCellCount    int      `json:"proposed_cell_count"`
	TargetCluster        string   `json:"target_cluster"`
	SelectedResources    []string `json:"selected_resources"`
	OverheadPct          float64  `json:"overhead_pct"`
	HostCount            int      `json:"host_count"`
	MemoryPerHostGB      int      `json:"memory_per_host_gb"`
	HAAdmissionPct       int      `json:"ha_admission_pct"`
	PhysicalCoresPerHost int      `json:"physical_cores_per_host"`
	TargetVCPURatio      int      `json:"target_vcpu_ratio"`
	PlatformVMsCPU       int      `json:"platform_vms_cpu"`
}

// ScenarioResult represents computed metrics for a scenario
type ScenarioResult struct {
	CellCount        int     `json:"cell_count"`
	CellMemoryGB     int     `json:"cell_memory_gb"`
	CellCPU          int     `json:"cell_cpu"`
	CellDiskGB       int     `json:"cell_disk_gb"`
	AppCapacityGB    int     `json:"app_capacity_gb"`
	UtilizationPct   float64 `json:"utilization_pct"`
	FreeChunks       int     `json:"free_chunks"`
	N1UtilizationPct float64 `json:"n1_utilization_pct"`
	FaultImpact      int     `json:"fault_impact"`
	BlastRadiusPct   float64 `json:"blast_radius_pct"`
	TotalVCPUs       int     `json:"total_vcpus"`
	TotalPCPUs       int     `json:"total_pcpus"`
	VCPURatio        float64 `json:"vcpu_ratio"`
	CPURiskLevel     string  `json:"cpu_risk_level"`
}

// ScenarioDelta represents changes between current and proposed
type ScenarioDelta struct {
	CapacityChangeGB     int     `json:"capacity_change_gb"`
	UtilizationChangePct float64 `json:"utilization_change_pct"`
	ResilienceChange     string  `json:"resilience_change"`
	VCPURatioChange      float64 `json:"vcpu_ratio_change"`
}

// ScenarioWarning represents a tradeoff warning
type ScenarioWarning struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ScenarioComparison represents full comparison response
type ScenarioComparison struct {
	Current  ScenarioResult    `json:"current"`
	Proposed ScenarioResult    `json:"proposed"`
	Delta    ScenarioDelta     `json:"delta"`
	Warnings []ScenarioWarning `json:"warnings"`
}

// CompareScenario calls POST /api/scenario/compare
func (c *Client) CompareScenario(ctx context.Context, input *ScenarioInput) (*ScenarioComparison, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/scenario/compare", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.handleRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var comparison ScenarioComparison
	if err := json.NewDecoder(resp.Body).Decode(&comparison); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &comparison, nil
}
