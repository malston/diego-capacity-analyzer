// ABOUTME: Tests for CPU-related model fields and calculations
// ABOUTME: Validates vCPU:pCPU ratio, CPU risk levels, and infrastructure CPU metrics

package models

import (
	"encoding/json"
	"testing"
)

func TestInfrastructureState_TotalCPUCores(t *testing.T) {
	mi := ManualInput{
		Name: "CPU Test Env",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         4,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   64,
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         3,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   48,
				DiegoCellCount:    75,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	state := mi.ToInfrastructureState()

	// Expected: (4 * 64) + (3 * 48) = 256 + 144 = 400 total CPU cores
	expectedCores := 400
	if state.TotalCPUCores != expectedCores {
		t.Errorf("Expected TotalCPUCores %d, got %d", expectedCores, state.TotalCPUCores)
	}
}

func TestInfrastructureState_TotalVCPUs(t *testing.T) {
	mi := ManualInput{
		Name: "vCPU Test Env",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         4,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   64,
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         3,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   48,
				DiegoCellCount:    75,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      8,
			},
		},
	}

	state := mi.ToInfrastructureState()

	// Expected: (100 * 4) + (75 * 8) = 400 + 600 = 1000 total vCPUs
	expectedVCPUs := 1000
	if state.TotalVCPUs != expectedVCPUs {
		t.Errorf("Expected TotalVCPUs %d, got %d", expectedVCPUs, state.TotalVCPUs)
	}
}

func TestInfrastructureState_VCPURatio(t *testing.T) {
	mi := ManualInput{
		Name: "Ratio Test Env",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         4,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   64,
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	state := mi.ToInfrastructureState()

	// Total pCPU: 4 * 64 = 256
	// Total vCPU: 100 * 4 = 400
	// Ratio: 400 / 256 = 1.5625
	expectedRatio := 1.5625
	if state.VCPURatio != expectedRatio {
		t.Errorf("Expected VCPURatio %.4f, got %.4f", expectedRatio, state.VCPURatio)
	}
}

func TestCPURiskLevel(t *testing.T) {
	tests := []struct {
		name     string
		ratio    float64
		expected string
	}{
		{"low risk - ratio 2:1", 2.0, "low"},
		{"low risk - ratio 4:1", 4.0, "low"},
		{"medium risk - ratio 5:1", 5.0, "medium"},
		{"medium risk - ratio 8:1", 8.0, "medium"},
		{"high risk - ratio 9:1", 9.0, "high"},
		{"high risk - ratio 12:1", 12.0, "high"},
		{"edge case - exactly 4:1", 4.0, "low"},
		{"edge case - just over 4:1", 4.1, "medium"},
		{"edge case - exactly 8:1", 8.0, "medium"},
		{"edge case - just over 8:1", 8.1, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CPURiskLevel(tt.ratio)
			if result != tt.expected {
				t.Errorf("CPURiskLevel(%.1f) = %s; want %s", tt.ratio, result, tt.expected)
			}
		})
	}
}

func TestInfrastructureState_CPURiskLevel(t *testing.T) {
	mi := ManualInput{
		Name: "High Ratio Test",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         2,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   32,
				DiegoCellCount:    50,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      16, // High vCPU per cell
			},
		},
	}

	state := mi.ToInfrastructureState()

	// Total pCPU: 2 * 32 = 64
	// Total vCPU: 50 * 16 = 800
	// Ratio: 800 / 64 = 12.5 -> high risk
	if state.CPURiskLevel != "high" {
		t.Errorf("Expected CPURiskLevel 'high', got '%s'", state.CPURiskLevel)
	}
}

func TestInfrastructureState_CPUFieldsSerialization(t *testing.T) {
	state := InfrastructureState{
		Source:        "manual",
		Name:          "Serialization Test",
		TotalCPUCores: 256,
		TotalVCPUs:    1024,
		VCPURatio:     4.0,
		CPURiskLevel:  "low",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal InfrastructureState: %v", err)
	}

	var decoded InfrastructureState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InfrastructureState: %v", err)
	}

	if decoded.TotalCPUCores != state.TotalCPUCores {
		t.Errorf("TotalCPUCores mismatch: got %d, want %d", decoded.TotalCPUCores, state.TotalCPUCores)
	}
	if decoded.TotalVCPUs != state.TotalVCPUs {
		t.Errorf("TotalVCPUs mismatch: got %d, want %d", decoded.TotalVCPUs, state.TotalVCPUs)
	}
	if decoded.VCPURatio != state.VCPURatio {
		t.Errorf("VCPURatio mismatch: got %.4f, want %.4f", decoded.VCPURatio, state.VCPURatio)
	}
	if decoded.CPURiskLevel != state.CPURiskLevel {
		t.Errorf("CPURiskLevel mismatch: got %s, want %s", decoded.CPURiskLevel, state.CPURiskLevel)
	}
}

func TestClusterState_CPUFields(t *testing.T) {
	mi := ManualInput{
		Name: "Cluster CPU Test",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         4,
				MemoryGBPerHost:   1024,
				CPUCoresPerHost:   64,
				DiegoCellCount:    100,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	state := mi.ToInfrastructureState()

	if len(state.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(state.Clusters))
	}

	cluster := state.Clusters[0]

	// TotalVCPUs for this cluster: 100 * 4 = 400
	expectedVCPUs := 400
	if cluster.TotalVCPUs != expectedVCPUs {
		t.Errorf("Cluster TotalVCPUs: got %d, want %d", cluster.TotalVCPUs, expectedVCPUs)
	}

	// VCPURatio for this cluster: 400 / 256 = 1.5625
	expectedRatio := 1.5625
	if cluster.VCPURatio != expectedRatio {
		t.Errorf("Cluster VCPURatio: got %.4f, want %.4f", cluster.VCPURatio, expectedRatio)
	}
}
