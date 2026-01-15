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
	// Test JSON unmarshaling (actual API contract)
	jsonInput := `{"physical_cores_per_host": 32, "target_vcpu_ratio": 4}`
	var input ScenarioInput
	err := json.Unmarshal([]byte(jsonInput), &input)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if input.PhysicalCoresPerHost != 32 {
		t.Errorf("PhysicalCoresPerHost = %d, want 32", input.PhysicalCoresPerHost)
	}
	if input.TargetVCPURatio != 4 {
		t.Errorf("TargetVCPURatio = %d, want 4", input.TargetVCPURatio)
	}
}

func TestScenarioResult_CPUFields(t *testing.T) {
	jsonInput := `{"total_vcpus": 160, "total_pcpus": 96, "vcpu_ratio": 1.67, "cpu_risk_level": "conservative"}`
	var result ScenarioResult
	err := json.Unmarshal([]byte(jsonInput), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.TotalVCPUs != 160 {
		t.Errorf("TotalVCPUs = %d, want 160", result.TotalVCPUs)
	}
	if result.TotalPCPUs != 96 {
		t.Errorf("TotalPCPUs = %d, want 96", result.TotalPCPUs)
	}
	if result.VCPURatio != 1.67 {
		t.Errorf("VCPURatio = %f, want 1.67", result.VCPURatio)
	}
	if result.CPURiskLevel != "conservative" {
		t.Errorf("CPURiskLevel = %s, want conservative", result.CPURiskLevel)
	}
}

func TestScenarioDelta_VCPURatioChange(t *testing.T) {
	jsonInput := `{"vcpu_ratio_change": 1.5}`
	var delta ScenarioDelta
	err := json.Unmarshal([]byte(jsonInput), &delta)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if delta.VCPURatioChange != 1.5 {
		t.Errorf("VCPURatioChange = %f, want 1.5", delta.VCPURatioChange)
	}
}
