// ABOUTME: Data models for what-if scenario input, output, and comparison
// ABOUTME: Supports capacity planning with proposed VM size and cell count changes

package models

import "fmt"

// ScenarioInput represents proposed changes for what-if analysis
type ScenarioInput struct {
	ProposedCellMemoryGB int    `json:"proposed_cell_memory_gb"`
	ProposedCellCPU      int    `json:"proposed_cell_cpu"`
	ProposedCellCount    int    `json:"proposed_cell_count"`
	TargetCluster        string `json:"target_cluster"` // Empty = all clusters
}

// ScenarioResult represents computed metrics for a scenario
type ScenarioResult struct {
	CellCount        int     `json:"cell_count"`
	CellMemoryGB     int     `json:"cell_memory_gb"`
	CellCPU          int     `json:"cell_cpu"`
	AppCapacityGB    int     `json:"app_capacity_gb"`
	UtilizationPct   float64 `json:"utilization_pct"`
	FreeChunks       int     `json:"free_chunks"`
	N1UtilizationPct float64 `json:"n1_utilization_pct"`
	FaultImpact      int     `json:"fault_impact"`
	InstancesPerCell float64 `json:"instances_per_cell"`
}

// CellSize returns formatted cell size string like "4×32"
func (r *ScenarioResult) CellSize() string {
	return fmt.Sprintf("%d×%d", r.CellCPU, r.CellMemoryGB)
}

// ScenarioWarning represents a tradeoff warning
type ScenarioWarning struct {
	Severity string `json:"severity"` // "info", "warning", "critical"
	Message  string `json:"message"`
}

// ScenarioDelta represents changes between current and proposed
type ScenarioDelta struct {
	CapacityChangeGB     int     `json:"capacity_change_gb"`
	UtilizationChangePct float64 `json:"utilization_change_pct"`
	RedundancyChange     string  `json:"redundancy_change"` // "improved", "reduced", "unchanged"
}

// ScenarioComparison represents full comparison response
type ScenarioComparison struct {
	Current  ScenarioResult    `json:"current"`
	Proposed ScenarioResult    `json:"proposed"`
	Warnings []ScenarioWarning `json:"warnings"`
	Delta    ScenarioDelta     `json:"delta"`
}
