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
		PlatformVMsGB:     4800,
		TotalAppMemoryGB:  10500,
		TotalAppInstances: 7500,
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

func TestAvgInstanceMemoryMB(t *testing.T) {
	mi := ManualInput{
		Name:              "test",
		Clusters:          []ClusterInput{{Name: "c1", HostCount: 4, MemoryGBPerHost: 512, CPUCoresPerHost: 32, DiegoCellCount: 10, DiegoCellMemoryGB: 32, DiegoCellCPU: 4}},
		PlatformVMsGB:     200,
		TotalAppMemoryGB:  150,
		TotalAppInstances: 50,
	}
	state := mi.ToInfrastructureState()

	// 150 GB * 1024 MB/GB / 50 instances = 3072 MB
	expected := 3072
	if state.AvgInstanceMemoryMB != expected {
		t.Errorf("Expected AvgInstanceMemoryMB %d, got %d", expected, state.AvgInstanceMemoryMB)
	}
}

func TestAvgInstanceMemoryMB_ZeroInstances(t *testing.T) {
	mi := ManualInput{
		Name:              "test",
		Clusters:          []ClusterInput{{Name: "c1", HostCount: 4, MemoryGBPerHost: 512, CPUCoresPerHost: 32, DiegoCellCount: 10, DiegoCellMemoryGB: 32, DiegoCellCPU: 4}},
		TotalAppMemoryGB:  150,
		TotalAppInstances: 0,
	}
	state := mi.ToInfrastructureState()

	if state.AvgInstanceMemoryMB != 0 {
		t.Errorf("Expected AvgInstanceMemoryMB 0 for zero instances, got %d", state.AvgInstanceMemoryMB)
	}
}

func TestMaxInstanceMemoryMB_FromManualInput(t *testing.T) {
	// Test that MaxInstanceMemoryMB is properly copied from ManualInput to InfrastructureState
	mi := ManualInput{
		Name:                "test",
		Clusters:            []ClusterInput{{Name: "c1", HostCount: 4, MemoryGBPerHost: 512, CPUCoresPerHost: 32, DiegoCellCount: 10, DiegoCellMemoryGB: 32, DiegoCellCPU: 4}},
		PlatformVMsGB:       200,
		TotalAppMemoryGB:    150,
		TotalAppInstances:   50,
		MaxInstanceMemoryMB: 2048, // Explicitly set max instance memory
	}
	state := mi.ToInfrastructureState()

	if state.MaxInstanceMemoryMB != 2048 {
		t.Errorf("Expected MaxInstanceMemoryMB 2048, got %d", state.MaxInstanceMemoryMB)
	}
}

func TestMaxInstanceMemoryMB_JSONParsing(t *testing.T) {
	// Test that MaxInstanceMemoryMB is properly parsed from JSON (simulating sample file)
	input := `{
		"name": "Sample Foundation",
		"clusters": [
			{
				"name": "cluster-01",
				"host_count": 8,
				"memory_gb_per_host": 1024,
				"cpu_cores_per_host": 64,
				"diego_cell_count": 50,
				"diego_cell_memory_gb": 64,
				"diego_cell_cpu": 8
			}
		],
		"platform_vms_gb": 800,
		"total_app_memory_gb": 2000,
		"total_app_disk_gb": 2500,
		"total_app_instances": 500,
		"max_instance_memory_mb": 4096
	}`

	var mi ManualInput
	err := json.Unmarshal([]byte(input), &mi)
	if err != nil {
		t.Fatalf("Failed to parse ManualInput: %v", err)
	}

	if mi.MaxInstanceMemoryMB != 4096 {
		t.Errorf("Expected MaxInstanceMemoryMB 4096 from JSON, got %d", mi.MaxInstanceMemoryMB)
	}

	state := mi.ToInfrastructureState()
	if state.MaxInstanceMemoryMB != 4096 {
		t.Errorf("Expected state.MaxInstanceMemoryMB 4096, got %d", state.MaxInstanceMemoryMB)
	}
}
