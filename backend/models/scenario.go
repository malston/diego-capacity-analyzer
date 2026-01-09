// ABOUTME: Data models for what-if scenario input, output, and comparison
// ABOUTME: Supports capacity planning with proposed VM size and cell count changes

package models

import "fmt"

// ScenarioInput represents proposed changes for what-if analysis
type ScenarioInput struct {
	ProposedCellMemoryGB int      `json:"proposed_cell_memory_gb"`
	ProposedCellCPU      int      `json:"proposed_cell_cpu"`
	ProposedCellDiskGB   int      `json:"proposed_cell_disk_gb"`
	ProposedCellCount    int      `json:"proposed_cell_count"`
	TargetCluster        string   `json:"target_cluster"`        // Empty = all clusters
	SelectedResources    []string `json:"selected_resources"`    // ["cpu", "memory", "disk"]
	OverheadPct          float64  `json:"overhead_pct"`          // Memory overhead % (default 7)
	AdditionalApp        *AppSpec `json:"additional_app"`        // Optional app to add
	TPSCurve             []TPSPt  `json:"tps_curve"`             // Custom TPS curve (only used if EnableTPS is true)
	// Host configuration for constraint analysis
	HostCount       int `json:"host_count"`
	MemoryPerHostGB int `json:"memory_per_host_gb"`
	HAAdmissionPct  int `json:"ha_admission_pct"`
}

// EnableTPS returns true if TPS analysis should be performed.
// TPS is only calculated when tps_curve is explicitly provided.
func (s *ScenarioInput) EnableTPS() bool {
	return len(s.TPSCurve) > 0
}

// AppSpec represents a hypothetical app for capacity planning
type AppSpec struct {
	Name      string `json:"name"`
	Instances int    `json:"instances"`
	MemoryGB  int    `json:"memory_gb"`
	DiskGB    int    `json:"disk_gb"`
}

// TPSPt represents a data point in the TPS performance curve
type TPSPt struct {
	Cells int `json:"cells"`
	TPS   int `json:"tps"`
}

// ScenarioResult represents computed metrics for a scenario
type ScenarioResult struct {
	CellCount          int     `json:"cell_count"`
	CellMemoryGB       int     `json:"cell_memory_gb"`
	CellCPU            int     `json:"cell_cpu"`
	CellDiskGB         int     `json:"cell_disk_gb"`
	AppCapacityGB      int     `json:"app_capacity_gb"`
	DiskCapacityGB     int     `json:"disk_capacity_gb"`
	UtilizationPct     float64 `json:"utilization_pct"`
	DiskUtilizationPct float64 `json:"disk_utilization_pct"`
	FreeChunks         int     `json:"free_chunks"`
	N1UtilizationPct   float64 `json:"n1_utilization_pct"`
	FaultImpact        int     `json:"fault_impact"`
	InstancesPerCell   float64 `json:"instances_per_cell"`
	EstimatedTPS       int     `json:"estimated_tps"`
	TPSStatus          string  `json:"tps_status"`      // "optimal", "degraded", "critical"
	BlastRadiusPct     float64 `json:"blast_radius_pct"` // % of capacity lost per single cell failure
}

// CellSize returns formatted cell size string like "4×32"
func (r *ScenarioResult) CellSize() string {
	return fmt.Sprintf("%d×%d", r.CellCPU, r.CellMemoryGB)
}

// ConfigChange describes what configuration the user modified that triggered a warning
type ConfigChange struct {
	Field       string  `json:"field"`        // "cell_count", "cell_memory_gb", "host_count", etc.
	PreviousVal int     `json:"previous_val"` // Current/baseline value
	ProposedVal int     `json:"proposed_val"` // User's proposed value
	Delta       int     `json:"delta"`        // Difference (proposed - previous)
	DeltaPct    float64 `json:"delta_pct"`    // Percentage change
}

// FixSuggestion describes how to resolve a warning
type FixSuggestion struct {
	Description string `json:"description"` // Human-readable fix description
	Field       string `json:"field"`       // Which field to change
	Value       int    `json:"value"`       // Suggested value
}

// ScenarioWarning represents a tradeoff warning with optional context
type ScenarioWarning struct {
	Severity string          `json:"severity"`         // "info", "warning", "critical"
	Message  string          `json:"message"`          // Warning message
	Change   *ConfigChange   `json:"change,omitempty"` // What caused this warning
	Fixes    []FixSuggestion `json:"fixes,omitempty"`  // How to fix (max 2)
}

// ScenarioDelta represents changes between current and proposed
type ScenarioDelta struct {
	CapacityChangeGB         int     `json:"capacity_change_gb"`
	DiskCapacityChangeGB     int     `json:"disk_capacity_change_gb"`
	UtilizationChangePct     float64 `json:"utilization_change_pct"`
	DiskUtilizationChangePct float64 `json:"disk_utilization_change_pct"`
	ResilienceChange         string  `json:"resilience_change"` // "low", "moderate", "high" based on blast radius
}

// ScenarioComparison represents full comparison response
type ScenarioComparison struct {
	Current         ScenarioResult      `json:"current"`
	Proposed        ScenarioResult      `json:"proposed"`
	Warnings        []ScenarioWarning   `json:"warnings"`
	Delta           ScenarioDelta       `json:"delta"`
	Recommendations []Recommendation    `json:"recommendations,omitempty"`
	Constraints     *ConstraintAnalysis `json:"constraints,omitempty"`
}

// CapacityConstraint represents a single constraint calculation (HA% or N-X)
type CapacityConstraint struct {
	Type           string  `json:"type"`            // "ha_admission" or "n_minus_x"
	ReservedGB     int     `json:"reserved_gb"`     // Memory reserved by this constraint
	ReservedPct    float64 `json:"reserved_pct"`    // Percentage of total reserved
	UsableGB       int     `json:"usable_gb"`       // Memory available after reserve
	NEquivalent    int     `json:"n_equivalent"`    // Hosts worth of capacity reserved
	IsLimiting     bool    `json:"is_limiting"`     // True if this is the limiting constraint
	UtilizationPct float64 `json:"utilization_pct"` // How full the usable capacity is
}

// ConstraintAnalysis compares HA Admission Control vs N-X tolerance
type ConstraintAnalysis struct {
	HAAdmission           CapacityConstraint `json:"ha_admission"`
	NMinusX               CapacityConstraint `json:"n_minus_x"`
	LimitingConstraint    string             `json:"limiting_constraint"`     // "ha_admission" or "n_minus_x"
	LimitingLabel         string             `json:"limiting_label"`          // "HA 25% (≈N-4)"
	InsufficientHAWarning bool               `json:"insufficient_ha_warning"` // True if HA% < N-1
}
