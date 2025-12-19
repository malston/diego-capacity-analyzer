# Phase 1: Manual Input + Scenario Calculator Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable what-if capacity analysis using manually-provided infrastructure data, without requiring vCenter access.

**Architecture:** New models for infrastructure and scenario data, a scenario calculator service that computes metrics and warnings, and two new API endpoints for manual data input and scenario comparison.

**Tech Stack:** Go 1.21+, standard library only, existing handler patterns from handlers.go

---

## Task 1: Infrastructure Models

**Files:**
- Create: `backend/models/infrastructure.go`
- Test: `backend/models/infrastructure_test.go`

**Step 1: Write the failing test**

```go
// backend/models/infrastructure_test.go
package models

import (
	"encoding/json"
	"testing"
)

func TestManualInputParsing(t *testing.T) {
	input := `{
		"name": "Customer ACME Production",
		"clusters": [
			{
				"name": "cluster-01",
				"host_count": 8,
				"memory_gb_per_host": 2048,
				"cpu_cores_per_host": 64,
				"diego_cell_count": 250,
				"diego_cell_memory_gb": 32,
				"diego_cell_cpu": 4
			}
		],
		"platform_vms_gb": 4800,
		"total_app_memory_gb": 10500,
		"total_app_instances": 7500
	}`

	var mi ManualInput
	err := json.Unmarshal([]byte(input), &mi)
	if err != nil {
		t.Fatalf("Failed to parse ManualInput: %v", err)
	}

	if mi.Name != "Customer ACME Production" {
		t.Errorf("Expected name 'Customer ACME Production', got '%s'", mi.Name)
	}
	if len(mi.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(mi.Clusters))
	}
	if mi.Clusters[0].HostCount != 8 {
		t.Errorf("Expected host_count 8, got %d", mi.Clusters[0].HostCount)
	}
	if mi.TotalAppMemoryGB != 10500 {
		t.Errorf("Expected total_app_memory_gb 10500, got %d", mi.TotalAppMemoryGB)
	}
}

func TestInfrastructureStateCalculation(t *testing.T) {
	mi := ManualInput{
		Name: "Test Env",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         8,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    250,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         7,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    220,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
		PlatformVMsGB:      4800,
		TotalAppMemoryGB:   10500,
		TotalAppInstances:  7500,
	}

	state := mi.ToInfrastructureState()

	// 8 + 7 = 15 hosts
	if state.TotalHostCount != 15 {
		t.Errorf("Expected TotalHostCount 15, got %d", state.TotalHostCount)
	}

	// (8 * 2048) + (7 * 2048) = 30720 GB
	if state.TotalMemoryGB != 30720 {
		t.Errorf("Expected TotalMemoryGB 30720, got %d", state.TotalMemoryGB)
	}

	// N-1 per cluster: (7 * 2048) + (6 * 2048) = 14336 + 12288 = 26624 GB
	if state.TotalN1MemoryGB != 26624 {
		t.Errorf("Expected TotalN1MemoryGB 26624, got %d", state.TotalN1MemoryGB)
	}

	// 250 + 220 = 470 cells
	if state.TotalCellCount != 470 {
		t.Errorf("Expected TotalCellCount 470, got %d", state.TotalCellCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./models/... -run TestManualInput -v`
Expected: FAIL with "undefined: ManualInput"

**Step 3: Write minimal implementation**

```go
// backend/models/infrastructure.go
// ABOUTME: Data models for infrastructure state and manual input
// ABOUTME: Supports what-if capacity analysis with user-provided data

package models

import "time"

// ClusterInput represents user-provided cluster configuration
type ClusterInput struct {
	Name              string `json:"name"`
	HostCount         int    `json:"host_count"`
	MemoryGBPerHost   int    `json:"memory_gb_per_host"`
	CPUCoresPerHost   int    `json:"cpu_cores_per_host"`
	DiegoCellCount    int    `json:"diego_cell_count"`
	DiegoCellMemoryGB int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU      int    `json:"diego_cell_cpu"`
}

// ManualInput represents user-provided infrastructure data
type ManualInput struct {
	Name              string         `json:"name"`
	Clusters          []ClusterInput `json:"clusters"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
}

// ClusterState represents computed cluster metrics
type ClusterState struct {
	Name              string `json:"name"`
	HostCount         int    `json:"host_count"`
	MemoryGB          int    `json:"memory_gb"`
	CPUCores          int    `json:"cpu_cores"`
	N1MemoryGB        int    `json:"n1_memory_gb"`
	UsableMemoryGB    int    `json:"usable_memory_gb"`
	DiegoCellCount    int    `json:"diego_cell_count"`
	DiegoCellMemoryGB int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU      int    `json:"diego_cell_cpu"`
}

// InfrastructureState represents computed infrastructure metrics
type InfrastructureState struct {
	Source            string         `json:"source"` // "manual" or "vsphere"
	Name              string         `json:"name"`
	Clusters          []ClusterState `json:"clusters"`
	TotalMemoryGB     int            `json:"total_memory_gb"`
	TotalN1MemoryGB   int            `json:"total_n1_memory_gb"`
	TotalHostCount    int            `json:"total_host_count"`
	TotalCellCount    int            `json:"total_cell_count"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
	Timestamp         time.Time      `json:"timestamp"`
	Cached            bool           `json:"cached"`
}

// ToInfrastructureState converts manual input to computed state
func (mi *ManualInput) ToInfrastructureState() InfrastructureState {
	state := InfrastructureState{
		Source:            "manual",
		Name:              mi.Name,
		Clusters:          make([]ClusterState, len(mi.Clusters)),
		PlatformVMsGB:     mi.PlatformVMsGB,
		TotalAppMemoryGB:  mi.TotalAppMemoryGB,
		TotalAppInstances: mi.TotalAppInstances,
		Timestamp:         time.Now(),
		Cached:            false,
	}

	for i, c := range mi.Clusters {
		clusterMemory := c.HostCount * c.MemoryGBPerHost
		clusterCPU := c.HostCount * c.CPUCoresPerHost
		n1Memory := (c.HostCount - 1) * c.MemoryGBPerHost
		usableMemory := int(float64(n1Memory) * 0.9) // 10% overhead

		state.Clusters[i] = ClusterState{
			Name:              c.Name,
			HostCount:         c.HostCount,
			MemoryGB:          clusterMemory,
			CPUCores:          clusterCPU,
			N1MemoryGB:        n1Memory,
			UsableMemoryGB:    usableMemory,
			DiegoCellCount:    c.DiegoCellCount,
			DiegoCellMemoryGB: c.DiegoCellMemoryGB,
			DiegoCellCPU:      c.DiegoCellCPU,
		}

		state.TotalMemoryGB += clusterMemory
		state.TotalN1MemoryGB += n1Memory
		state.TotalHostCount += c.HostCount
		state.TotalCellCount += c.DiegoCellCount
	}

	return state
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./models/... -run TestManualInput -v && go test ./models/... -run TestInfrastructure -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/infrastructure.go backend/models/infrastructure_test.go
git commit -m "feat: add infrastructure models for manual input"
```

---

## Task 2: Scenario Models

**Files:**
- Create: `backend/models/scenario.go`
- Test: `backend/models/scenario_test.go`

**Step 1: Write the failing test**

```go
// backend/models/scenario_test.go
package models

import (
	"encoding/json"
	"testing"
)

func TestScenarioInputParsing(t *testing.T) {
	input := `{
		"proposed_cell_memory_gb": 64,
		"proposed_cell_cpu": 4,
		"proposed_cell_count": 235,
		"target_cluster": ""
	}`

	var si ScenarioInput
	err := json.Unmarshal([]byte(input), &si)
	if err != nil {
		t.Fatalf("Failed to parse ScenarioInput: %v", err)
	}

	if si.ProposedCellMemoryGB != 64 {
		t.Errorf("Expected proposed_cell_memory_gb 64, got %d", si.ProposedCellMemoryGB)
	}
	if si.ProposedCellCPU != 4 {
		t.Errorf("Expected proposed_cell_cpu 4, got %d", si.ProposedCellCPU)
	}
	if si.ProposedCellCount != 235 {
		t.Errorf("Expected proposed_cell_count 235, got %d", si.ProposedCellCount)
	}
}

func TestScenarioResultCellSize(t *testing.T) {
	result := ScenarioResult{
		CellCount:     470,
		CellMemoryGB:  32,
		CellCPU:       4,
	}

	if result.CellSize() != "4×32" {
		t.Errorf("Expected CellSize '4×32', got '%s'", result.CellSize())
	}
}

func TestScenarioWarningSeverity(t *testing.T) {
	warning := ScenarioWarning{
		Severity: "critical",
		Message:  "Exceeds N-1 capacity",
	}

	if warning.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", warning.Severity)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./models/... -run TestScenario -v`
Expected: FAIL with "undefined: ScenarioInput"

**Step 3: Write minimal implementation**

```go
// backend/models/scenario.go
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
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./models/... -run TestScenario -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/models/scenario.go backend/models/scenario_test.go
git commit -m "feat: add scenario models for what-if comparison"
```

---

## Task 3: Scenario Calculator - Core Formulas

**Files:**
- Create: `backend/services/scenario.go`
- Test: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

Test the formulas against the capacity doc example (470 cells, 4×32, 7500 instances, 10.5 TB apps):

```go
// backend/services/scenario_test.go
package services

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestCalculateCurrentScenario(t *testing.T) {
	// Based on capacity doc: 470 cells (4×32), 7500 instances, 10.5 TB apps
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624, // 14 hosts * 2048 - simulated N-1
		TotalCellCount:    470,
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateCurrent(state)

	// Cell count
	if result.CellCount != 470 {
		t.Errorf("Expected CellCount 470, got %d", result.CellCount)
	}

	// App capacity: 470 × (32 - 5) = 470 × 27 = 12690 GB
	if result.AppCapacityGB != 12690 {
		t.Errorf("Expected AppCapacityGB 12690, got %d", result.AppCapacityGB)
	}

	// Utilization: 10500 / 12690 × 100 = 82.7%
	if result.UtilizationPct < 82 || result.UtilizationPct > 83 {
		t.Errorf("Expected UtilizationPct ~82.7%%, got %.1f%%", result.UtilizationPct)
	}

	// Free chunks: (12690 - 10500) / 4 = 547
	if result.FreeChunks != 547 {
		t.Errorf("Expected FreeChunks 547, got %d", result.FreeChunks)
	}

	// Instances per cell: 7500 / 470 = 15.96
	if result.InstancesPerCell < 15.9 || result.InstancesPerCell > 16.1 {
		t.Errorf("Expected InstancesPerCell ~16, got %.1f", result.InstancesPerCell)
	}

	// Fault impact (apps per cell): 7500 / 470 = 16
	if result.FaultImpact != 16 {
		t.Errorf("Expected FaultImpact 16, got %d", result.FaultImpact)
	}

	// N-1 utilization: (470 × 32 + 4800) / 26624 × 100 = 74.5%
	// Cell memory: 470 × 32 = 15040
	// Total: 15040 + 4800 = 19840
	// 19840 / 26624 = 74.5%
	if result.N1UtilizationPct < 74 || result.N1UtilizationPct > 75 {
		t.Errorf("Expected N1UtilizationPct ~74.5%%, got %.1f%%", result.N1UtilizationPct)
	}
}

func TestCalculateProposedScenario(t *testing.T) {
	// Same infrastructure, but proposing 4×64 cells with 235 cells
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    470,
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    235,
	}

	calc := NewScenarioCalculator()
	result := calc.CalculateProposed(state, input)

	// Cell count
	if result.CellCount != 235 {
		t.Errorf("Expected CellCount 235, got %d", result.CellCount)
	}

	// App capacity: 235 × (64 - 5) = 235 × 59 = 13865 GB
	if result.AppCapacityGB != 13865 {
		t.Errorf("Expected AppCapacityGB 13865, got %d", result.AppCapacityGB)
	}

	// Utilization: 10500 / 13865 × 100 = 75.7%
	if result.UtilizationPct < 75 || result.UtilizationPct > 76 {
		t.Errorf("Expected UtilizationPct ~75.7%%, got %.1f%%", result.UtilizationPct)
	}

	// Fault impact: 7500 / 235 = 32
	if result.FaultImpact != 32 {
		t.Errorf("Expected FaultImpact 32, got %d", result.FaultImpact)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services/... -run TestCalculate -v`
Expected: FAIL with "undefined: NewScenarioCalculator"

**Step 3: Write minimal implementation**

```go
// backend/services/scenario.go
// ABOUTME: Scenario calculator for what-if capacity analysis
// ABOUTME: Computes metrics and warnings for current vs proposed configurations

package services

import (
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const (
	// GardenOverheadGB is memory reserved per cell for Garden/system
	GardenOverheadGB = 5
	// ChunkSizeGB is the size of a free chunk for staging
	ChunkSizeGB = 4
)

// ScenarioCalculator computes capacity metrics for scenarios
type ScenarioCalculator struct{}

// NewScenarioCalculator creates a new calculator
func NewScenarioCalculator() *ScenarioCalculator {
	return &ScenarioCalculator{}
}

// CalculateCurrent computes metrics for current infrastructure state
func (c *ScenarioCalculator) CalculateCurrent(state models.InfrastructureState) models.ScenarioResult {
	// Get cell config from first cluster (assumes uniform cells)
	var cellMemoryGB, cellCPU int
	for _, cluster := range state.Clusters {
		if cluster.DiegoCellMemoryGB > 0 {
			cellMemoryGB = cluster.DiegoCellMemoryGB
			cellCPU = cluster.DiegoCellCPU
			break
		}
	}

	return c.calculate(
		state.TotalCellCount,
		cellMemoryGB,
		cellCPU,
		state.TotalAppMemoryGB,
		state.TotalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
	)
}

// CalculateProposed computes metrics for a proposed scenario
func (c *ScenarioCalculator) CalculateProposed(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioResult {
	return c.calculate(
		input.ProposedCellCount,
		input.ProposedCellMemoryGB,
		input.ProposedCellCPU,
		state.TotalAppMemoryGB,
		state.TotalAppInstances,
		state.PlatformVMsGB,
		state.TotalN1MemoryGB,
	)
}

// calculate performs the core metric calculations
func (c *ScenarioCalculator) calculate(
	cellCount int,
	cellMemoryGB int,
	cellCPU int,
	totalAppMemoryGB int,
	totalAppInstances int,
	platformVMsGB int,
	n1MemoryGB int,
) models.ScenarioResult {
	// App capacity: cells × (cellMemory - overhead)
	appCapacityGB := cellCount * (cellMemoryGB - GardenOverheadGB)

	// Utilization: appMemory / capacity × 100
	var utilizationPct float64
	if appCapacityGB > 0 {
		utilizationPct = float64(totalAppMemoryGB) / float64(appCapacityGB) * 100
	}

	// Free chunks: (capacity - used) / chunkSize
	freeChunks := (appCapacityGB - totalAppMemoryGB) / ChunkSizeGB
	if freeChunks < 0 {
		freeChunks = 0
	}

	// Instances per cell
	var instancesPerCell float64
	if cellCount > 0 {
		instancesPerCell = float64(totalAppInstances) / float64(cellCount)
	}

	// Fault impact (rounded)
	faultImpact := int(instancesPerCell)

	// N-1 utilization: (cellMemory + platformVMs) / n1Memory × 100
	totalCellMemoryGB := cellCount * cellMemoryGB
	var n1UtilizationPct float64
	if n1MemoryGB > 0 {
		n1UtilizationPct = float64(totalCellMemoryGB+platformVMsGB) / float64(n1MemoryGB) * 100
	}

	return models.ScenarioResult{
		CellCount:        cellCount,
		CellMemoryGB:     cellMemoryGB,
		CellCPU:          cellCPU,
		AppCapacityGB:    appCapacityGB,
		UtilizationPct:   utilizationPct,
		FreeChunks:       freeChunks,
		N1UtilizationPct: n1UtilizationPct,
		FaultImpact:      faultImpact,
		InstancesPerCell: instancesPerCell,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services/... -run TestCalculate -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat: add scenario calculator with core formulas"
```

---

## Task 4: Scenario Calculator - Warnings

**Files:**
- Modify: `backend/services/scenario.go`
- Modify: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

```go
// Add to backend/services/scenario_test.go

func TestGenerateWarnings_CriticalN1(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 90, // > 85% = critical
		FreeChunks:       500,
		CellCount:        100,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Exceeds N-1 capacity safety margin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for N-1 > 85%")
	}
}

func TestGenerateWarnings_LowFreeChunks(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       150, // < 200 = critical
		CellCount:        100,
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed)

	found := false
	for _, w := range warnings {
		if w.Severity == "critical" && w.Message == "Critical: Low staging capacity" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected critical warning for free chunks < 200")
	}
}

func TestGenerateWarnings_RedundancyReduction(t *testing.T) {
	current := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        100,
	}
	proposed := models.ScenarioResult{
		N1UtilizationPct: 70,
		FreeChunks:       500,
		CellCount:        40, // 60% reduction
	}

	calc := NewScenarioCalculator()
	warnings := calc.GenerateWarnings(current, proposed)

	found := false
	for _, w := range warnings {
		if w.Severity == "warning" && w.Message == "Significant redundancy reduction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning for > 50% cell count reduction")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services/... -run TestGenerateWarnings -v`
Expected: FAIL with "calc.GenerateWarnings undefined"

**Step 3: Write minimal implementation**

Add to `backend/services/scenario.go`:

```go
// GenerateWarnings produces warnings based on proposed scenario
func (c *ScenarioCalculator) GenerateWarnings(current, proposed models.ScenarioResult) []models.ScenarioWarning {
	var warnings []models.ScenarioWarning

	// N-1 utilization warnings
	if proposed.N1UtilizationPct > 85 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Exceeds N-1 capacity safety margin",
		})
	} else if proposed.N1UtilizationPct > 75 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Approaching N-1 capacity limits",
		})
	}

	// Free chunks warnings
	if proposed.FreeChunks < 200 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Critical: Low staging capacity",
		})
	} else if proposed.FreeChunks < 400 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Low staging capacity",
		})
	}

	// Cell utilization warnings
	if proposed.UtilizationPct > 90 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "critical",
			Message:  "Cell utilization critically high",
		})
	} else if proposed.UtilizationPct > 80 {
		warnings = append(warnings, models.ScenarioWarning{
			Severity: "warning",
			Message:  "Cell utilization elevated",
		})
	}

	// Redundancy reduction warning
	if current.CellCount > 0 {
		reduction := float64(current.CellCount-proposed.CellCount) / float64(current.CellCount) * 100
		if reduction > 50 {
			warnings = append(warnings, models.ScenarioWarning{
				Severity: "warning",
				Message:  "Significant redundancy reduction",
			})
		}
	}

	return warnings
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services/... -run TestGenerateWarnings -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat: add warning generation to scenario calculator"
```

---

## Task 5: Scenario Calculator - Compare Method

**Files:**
- Modify: `backend/services/scenario.go`
- Modify: `backend/services/scenario_test.go`

**Step 1: Write the failing test**

```go
// Add to backend/services/scenario_test.go

func TestCompare(t *testing.T) {
	state := models.InfrastructureState{
		TotalN1MemoryGB:   26624,
		TotalCellCount:    470,
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
		Clusters: []models.ClusterState{
			{
				DiegoCellCount:    470,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	input := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    235,
	}

	calc := NewScenarioCalculator()
	comparison := calc.Compare(state, input)

	// Current should match
	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}

	// Proposed should match
	if comparison.Proposed.CellCount != 235 {
		t.Errorf("Expected Proposed.CellCount 235, got %d", comparison.Proposed.CellCount)
	}

	// Delta - capacity increased
	if comparison.Delta.CapacityChangeGB <= 0 {
		t.Errorf("Expected positive capacity change, got %d", comparison.Delta.CapacityChangeGB)
	}

	// Delta - redundancy reduced (fewer cells)
	if comparison.Delta.RedundancyChange != "reduced" {
		t.Errorf("Expected RedundancyChange 'reduced', got '%s'", comparison.Delta.RedundancyChange)
	}

	// Should have warning about redundancy
	if len(comparison.Warnings) == 0 {
		t.Error("Expected at least one warning")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services/... -run TestCompare -v`
Expected: FAIL with "calc.Compare undefined"

**Step 3: Write minimal implementation**

Add to `backend/services/scenario.go`:

```go
// Compare computes full comparison between current and proposed scenarios
func (c *ScenarioCalculator) Compare(state models.InfrastructureState, input models.ScenarioInput) models.ScenarioComparison {
	current := c.CalculateCurrent(state)
	proposed := c.CalculateProposed(state, input)
	warnings := c.GenerateWarnings(current, proposed)

	// Calculate delta
	capacityChange := proposed.AppCapacityGB - current.AppCapacityGB
	utilizationChange := proposed.UtilizationPct - current.UtilizationPct

	var redundancyChange string
	if proposed.CellCount > current.CellCount {
		redundancyChange = "improved"
	} else if proposed.CellCount < current.CellCount {
		redundancyChange = "reduced"
	} else {
		redundancyChange = "unchanged"
	}

	return models.ScenarioComparison{
		Current:  current,
		Proposed: proposed,
		Warnings: warnings,
		Delta: models.ScenarioDelta{
			CapacityChangeGB:     capacityChange,
			UtilizationChangePct: utilizationChange,
			RedundancyChange:     redundancyChange,
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services/... -run TestCompare -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/scenario.go backend/services/scenario_test.go
git commit -m "feat: add Compare method for full scenario comparison"
```

---

## Task 6: API Handlers

**Files:**
- Modify: `backend/handlers/handlers.go`
- Modify: `backend/handlers/handlers_test.go`

**Step 1: Write the failing test**

```go
// Add to backend/handlers/handlers_test.go

func TestHandleManualInfrastructure(t *testing.T) {
	body := `{
		"name": "Test Env",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 8,
			"memory_gb_per_host": 2048,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 250,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4
		}],
		"platform_vms_gb": 4800,
		"total_app_memory_gb": 10500,
		"total_app_instances": 7500
	}`

	req := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := NewHandler(nil, nil)
	handler.HandleManualInfrastructure(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.InfrastructureState
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Source != "manual" {
		t.Errorf("Expected source 'manual', got '%s'", response.Source)
	}
	if response.TotalHostCount != 8 {
		t.Errorf("Expected TotalHostCount 8, got %d", response.TotalHostCount)
	}
}

func TestHandleScenarioCompare(t *testing.T) {
	// First, set up manual infrastructure
	manualBody := `{
		"name": "Test Env",
		"clusters": [{
			"name": "cluster-01",
			"host_count": 15,
			"memory_gb_per_host": 2048,
			"cpu_cores_per_host": 64,
			"diego_cell_count": 470,
			"diego_cell_memory_gb": 32,
			"diego_cell_cpu": 4
		}],
		"platform_vms_gb": 4800,
		"total_app_memory_gb": 10500,
		"total_app_instances": 7500
	}`

	handler := NewHandler(nil, nil)

	// Set manual infrastructure
	req1 := httptest.NewRequest("POST", "/api/infrastructure/manual", strings.NewReader(manualBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleManualInfrastructure(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set manual infrastructure: %s", w1.Body.String())
	}

	// Now compare scenario
	compareBody := `{
		"proposed_cell_memory_gb": 64,
		"proposed_cell_cpu": 4,
		"proposed_cell_count": 235
	}`

	req2 := httptest.NewRequest("POST", "/api/scenario/compare", strings.NewReader(compareBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleScenarioCompare(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var comparison models.ScenarioComparison
	if err := json.NewDecoder(w2.Body).Decode(&comparison); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}
	if comparison.Proposed.CellCount != 235 {
		t.Errorf("Expected Proposed.CellCount 235, got %d", comparison.Proposed.CellCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./handlers/... -run TestHandleManual -v && go test ./handlers/... -run TestHandleScenario -v`
Expected: FAIL with "handler.HandleManualInfrastructure undefined"

**Step 3: Read existing handlers.go to understand structure**

Run: `head -100 backend/handlers/handlers.go`

**Step 4: Write minimal implementation**

Add to `backend/handlers/handlers.go`:

```go
// Add field to Handler struct
type Handler struct {
	// ... existing fields ...
	infrastructureState *models.InfrastructureState
	scenarioCalc        *services.ScenarioCalculator
	infraMutex          sync.RWMutex
}

// Update NewHandler
func NewHandler(boshClient *services.BOSHClient, cfClient *services.CFClient) *Handler {
	return &Handler{
		// ... existing fields ...
		scenarioCalc: services.NewScenarioCalculator(),
	}
}

// HandleManualInfrastructure handles POST /api/infrastructure/manual
func (h *Handler) HandleManualInfrastructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input models.ManualInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	state := input.ToInfrastructureState()

	h.infraMutex.Lock()
	h.infrastructureState = &state
	h.infraMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// HandleScenarioCompare handles POST /api/scenario/compare
func (h *Handler) HandleScenarioCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.infraMutex.RLock()
	state := h.infrastructureState
	h.infraMutex.RUnlock()

	if state == nil {
		writeError(w, "No infrastructure data. Set via /api/infrastructure/manual first.", http.StatusBadRequest)
		return
	}

	var input models.ScenarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	comparison := h.scenarioCalc.Compare(*state, input)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comparison)
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: message,
		Code:  code,
	})
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./handlers/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/handlers/handlers.go backend/handlers/handlers_test.go
git commit -m "feat: add API handlers for manual infrastructure and scenario compare"
```

---

## Task 7: Wire Up Routes

**Files:**
- Modify: `backend/main.go`

**Step 1: Read existing main.go**

Run: `cat backend/main.go`

**Step 2: Add new routes**

Add to route setup in `main.go`:

```go
// Add after existing routes
http.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
http.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)
```

**Step 3: Verify build and existing tests**

Run: `cd backend && go build && go test ./...`
Expected: Build succeeds, all tests pass

**Step 4: Commit**

```bash
git add backend/main.go
git commit -m "feat: wire up scenario analysis API routes"
```

---

## Task 8: End-to-End Test

**Files:**
- Create: `backend/e2e_test.go`

**Step 1: Write end-to-end test**

```go
// backend/e2e_test.go
// ABOUTME: End-to-end test for scenario analysis API
// ABOUTME: Tests full flow from manual input to scenario comparison

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestScenarioAnalysisE2E(t *testing.T) {
	handler := handlers.NewHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/infrastructure/manual", handler.HandleManualInfrastructure)
	mux.HandleFunc("/api/scenario/compare", handler.HandleScenarioCompare)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Step 1: Set manual infrastructure (based on capacity doc)
	manualInput := models.ManualInput{
		Name: "Customer ACME Production",
		Clusters: []models.ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         8,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    250,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         7,
				MemoryGBPerHost:   2048,
				CPUCoresPerHost:   64,
				DiegoCellCount:    220,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
	}

	body1, _ := json.Marshal(manualInput)
	resp1, err := http.Post(server.URL+"/api/infrastructure/manual", "application/json", bytes.NewReader(body1))
	if err != nil {
		t.Fatalf("Failed to post manual infrastructure: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp1.StatusCode)
	}

	var infraState models.InfrastructureState
	json.NewDecoder(resp1.Body).Decode(&infraState)

	if infraState.TotalCellCount != 470 {
		t.Errorf("Expected TotalCellCount 470, got %d", infraState.TotalCellCount)
	}

	// Step 2: Compare scenario (4×32 current → 4×64 proposed)
	scenarioInput := models.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      4,
		ProposedCellCount:    235,
	}

	body2, _ := json.Marshal(scenarioInput)
	resp2, err := http.Post(server.URL+"/api/scenario/compare", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("Failed to post scenario compare: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp2.StatusCode)
	}

	var comparison models.ScenarioComparison
	json.NewDecoder(resp2.Body).Decode(&comparison)

	// Validate current state
	if comparison.Current.CellCount != 470 {
		t.Errorf("Expected Current.CellCount 470, got %d", comparison.Current.CellCount)
	}
	if comparison.Current.CellSize() != "4×32" {
		t.Errorf("Expected Current.CellSize '4×32', got '%s'", comparison.Current.CellSize())
	}

	// Validate proposed state
	if comparison.Proposed.CellCount != 235 {
		t.Errorf("Expected Proposed.CellCount 235, got %d", comparison.Proposed.CellCount)
	}
	if comparison.Proposed.CellSize() != "4×64" {
		t.Errorf("Expected Proposed.CellSize '4×64', got '%s'", comparison.Proposed.CellSize())
	}

	// Validate delta
	if comparison.Delta.RedundancyChange != "reduced" {
		t.Errorf("Expected RedundancyChange 'reduced', got '%s'", comparison.Delta.RedundancyChange)
	}

	// Should have warning about redundancy reduction (50% cell reduction)
	hasRedundancyWarning := false
	for _, w := range comparison.Warnings {
		if w.Message == "Significant redundancy reduction" {
			hasRedundancyWarning = true
			break
		}
	}
	if !hasRedundancyWarning {
		t.Error("Expected warning about redundancy reduction")
	}

	t.Logf("Comparison: Current %s (%d cells) → Proposed %s (%d cells)",
		comparison.Current.CellSize(), comparison.Current.CellCount,
		comparison.Proposed.CellSize(), comparison.Proposed.CellCount)
	t.Logf("Capacity: %d GB → %d GB (change: %+d GB)",
		comparison.Current.AppCapacityGB, comparison.Proposed.AppCapacityGB,
		comparison.Delta.CapacityChangeGB)
	t.Logf("Warnings: %d", len(comparison.Warnings))
}
```

**Step 2: Run end-to-end test**

Run: `cd backend && go test -run TestScenarioAnalysisE2E -v`
Expected: PASS with comparison output logged

**Step 3: Commit**

```bash
git add backend/e2e_test.go
git commit -m "test: add end-to-end test for scenario analysis"
```

---

## Task 9: Final Verification

**Step 1: Run all backend tests**

Run: `cd backend && go test ./... -v`
Expected: All tests pass

**Step 2: Build and verify**

Run: `cd backend && go build -o capacity-backend`
Expected: Build succeeds

**Step 3: Test manually with curl**

```bash
# Start server in background
./capacity-backend &

# Set manual infrastructure
curl -X POST http://localhost:8080/api/infrastructure/manual \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test",
    "clusters": [{
      "name": "cluster-01",
      "host_count": 15,
      "memory_gb_per_host": 2048,
      "cpu_cores_per_host": 64,
      "diego_cell_count": 470,
      "diego_cell_memory_gb": 32,
      "diego_cell_cpu": 4
    }],
    "platform_vms_gb": 4800,
    "total_app_memory_gb": 10500,
    "total_app_instances": 7500
  }'

# Compare scenario
curl -X POST http://localhost:8080/api/scenario/compare \
  -H "Content-Type: application/json" \
  -d '{
    "proposed_cell_memory_gb": 64,
    "proposed_cell_cpu": 4,
    "proposed_cell_count": 235
  }'

# Stop server
pkill capacity-backend
```

**Step 4: Final commit**

```bash
git add -A
git commit -m "Phase 1 complete: Manual input + scenario calculator"
```

---

## Summary

Phase 1 implements:
- Infrastructure models for manual input and computed state
- Scenario models for input, result, and comparison
- Scenario calculator with formulas from capacity doc
- Warning generation for capacity/redundancy tradeoffs
- API endpoints: `POST /api/infrastructure/manual` and `POST /api/scenario/compare`
- End-to-end test validating the full flow

**Next Phase:** Frontend with DataSourceSelector, ScenarioAnalyzer, and ComparisonTable components.
