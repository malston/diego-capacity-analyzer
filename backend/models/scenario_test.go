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

func TestScenarioInput_CPUFields(t *testing.T) {
	input := ScenarioInput{
		PhysicalCoresPerHost: 32,
		TargetVCPURatio:      4,
	}

	if input.PhysicalCoresPerHost != 32 {
		t.Errorf("PhysicalCoresPerHost = %d, want 32", input.PhysicalCoresPerHost)
	}
	if input.TargetVCPURatio != 4 {
		t.Errorf("TargetVCPURatio = %d, want 4", input.TargetVCPURatio)
	}
}
