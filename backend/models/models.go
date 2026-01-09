// ABOUTME: Data models for Diego cells, apps, and API responses
// ABOUTME: JSON-serializable structures matching frontend expectations

package models

import "time"

// DiegoCell represents a Diego cell VM with capacity metrics
type DiegoCell struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	MemoryMB         int    `json:"memory_mb"`
	AllocatedMB      int    `json:"allocated_mb"`
	UsedMB           int    `json:"used_mb"`
	CPUPercent       int    `json:"cpu_percent"`
	IsolationSegment string `json:"isolation_segment"`
}

// App represents a Cloud Foundry application with memory and disk metrics
type App struct {
	Name             string `json:"name"`
	GUID             string `json:"guid,omitempty"`
	Instances        int    `json:"instances"`
	RequestedMB      int    `json:"requested_mb"`
	ActualMB         int    `json:"actual_mb"`
	RequestedDiskMB  int    `json:"requested_disk_mb"`
	IsolationSegment string `json:"isolation_segment"`
}

// IsolationSegment represents a CF isolation segment
type IsolationSegment struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// DashboardResponse is the unified API response
type DashboardResponse struct {
	Cells    []DiegoCell        `json:"cells"`
	Apps     []App              `json:"apps"`
	Segments []IsolationSegment `json:"segments"`
	Metadata Metadata           `json:"metadata"`
}

// Metadata contains response metadata
type Metadata struct {
	Timestamp     time.Time `json:"timestamp"`
	Cached        bool      `json:"cached"`
	BOSHAvailable bool      `json:"bosh_available"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code"`
}
