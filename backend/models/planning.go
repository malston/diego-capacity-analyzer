// ABOUTME: Data models for infrastructure planning calculations
// ABOUTME: Computes max deployable cells given IaaS capacity constraints

package models

// PlanningInput represents input for infrastructure planning calculation
type PlanningInput struct {
	CellMemoryGB int     `json:"cell_memory_gb"` // Desired memory per cell
	CellCPU      int     `json:"cell_cpu"`       // Desired vCPUs per cell
	OverheadPct  float64 `json:"overhead_pct"`   // Memory overhead % (default 7)
}

// PlanningResult represents the output of infrastructure planning
type PlanningResult struct {
	MaxCellsByMemory int     `json:"max_cells_by_memory"` // Cells constrained by memory
	MaxCellsByCPU    int     `json:"max_cells_by_cpu"`    // Cells constrained by CPU
	DeployableCells  int     `json:"deployable_cells"`    // MIN(memory, cpu)
	Bottleneck       string  `json:"bottleneck"`          // "memory", "cpu", or "balanced"
	MemoryUsedGB     int     `json:"memory_used_gb"`      // Total memory consumed by cells
	MemoryAvailGB    int     `json:"memory_avail_gb"`     // Total available memory
	CPUUsed          int     `json:"cpu_used"`            // Total vCPUs consumed by cells
	CPUAvail         int     `json:"cpu_avail"`           // Total available vCPUs
	MemoryUtilPct    float64 `json:"memory_util_pct"`     // Memory utilization %
	CPUUtilPct       float64 `json:"cpu_util_pct"`        // CPU utilization %
	HeadroomCells    int     `json:"headroom_cells"`      // Unused capacity in cells
}

// SizingRecommendation represents a cell sizing alternative
type SizingRecommendation struct {
	CellCPU         int     `json:"cell_cpu"`
	CellMemoryGB    int     `json:"cell_memory_gb"`
	DeployableCells int     `json:"deployable_cells"`
	Bottleneck      string  `json:"bottleneck"`
	MemoryUtilPct   float64 `json:"memory_util_pct"`
	CPUUtilPct      float64 `json:"cpu_util_pct"`
	Label           string  `json:"label"` // e.g. "4×32 GB"
}

// PlanningResponse is the full response including recommendations
type PlanningResponse struct {
	Result          PlanningResult         `json:"result"`
	Recommendations []SizingRecommendation `json:"recommendations"`
}
