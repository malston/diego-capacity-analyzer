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
	CFAPI   string `json:"cf_api"`
	BOSHAPI string `json:"bosh_api"`
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
