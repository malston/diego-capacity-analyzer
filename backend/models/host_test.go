// ABOUTME: Tests for host-level model fields and calculations
// ABOUTME: Validates host count, cores per host, memory per host, and HA admission control

package models

import (
	"encoding/json"
	"testing"
)

func TestClusterInput_HAAdmissionControlPercentage(t *testing.T) {
	input := ClusterInput{
		Name:                          "HA Cluster",
		HostCount:                     4,
		MemoryGBPerHost:               1024,
		CPUCoresPerHost:               64,
		HAAdmissionControlPercentage:  25, // Reserve 25% for HA
		DiegoCellCount:                100,
		DiegoCellMemoryGB:             32,
		DiegoCellCPU:                  4,
	}

	if input.HAAdmissionControlPercentage != 25 {
		t.Errorf("Expected HAAdmissionControlPercentage 25, got %d", input.HAAdmissionControlPercentage)
	}
}

func TestClusterState_HostLevelFields(t *testing.T) {
	mi := ManualInput{
		Name: "Host Level Test",
		Clusters: []ClusterInput{
			{
				Name:                          "cluster-01",
				HostCount:                     4,
				MemoryGBPerHost:               1024,
				CPUCoresPerHost:               64,
				HAAdmissionControlPercentage:  25,
				DiegoCellCount:                100,
				DiegoCellMemoryGB:             32,
				DiegoCellCPU:                  4,
				DiegoCellDiskGB:               100,
			},
		},
	}

	state := mi.ToInfrastructureState()
	cluster := state.Clusters[0]

	// Verify host-level fields are populated
	if cluster.MemoryGBPerHost != 1024 {
		t.Errorf("Expected MemoryGBPerHost 1024, got %d", cluster.MemoryGBPerHost)
	}

	if cluster.CPUCoresPerHost != 64 {
		t.Errorf("Expected CPUCoresPerHost 64, got %d", cluster.CPUCoresPerHost)
	}

	if cluster.HAAdmissionControlPercentage != 25 {
		t.Errorf("Expected HAAdmissionControlPercentage 25, got %d", cluster.HAAdmissionControlPercentage)
	}
}

func TestClusterState_HAUsableCapacity(t *testing.T) {
	tests := []struct {
		name                 string
		hostCount            int
		memoryPerHost        int
		haPercentage         int
		expectedUsableMemory int
	}{
		{
			name:                 "25% HA reservation with 4 hosts",
			hostCount:            4,
			memoryPerHost:        1024,
			haPercentage:         25,
			expectedUsableMemory: 3072, // 4 * 1024 * 0.75 = 3072
		},
		{
			name:                 "0% HA reservation (no HA)",
			hostCount:            4,
			memoryPerHost:        1024,
			haPercentage:         0,
			expectedUsableMemory: 4096, // 4 * 1024 = 4096
		},
		{
			name:                 "50% HA reservation",
			hostCount:            4,
			memoryPerHost:        1024,
			haPercentage:         50,
			expectedUsableMemory: 2048, // 4 * 1024 * 0.50 = 2048
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name: "HA Test",
				Clusters: []ClusterInput{
					{
						Name:                          "cluster-01",
						HostCount:                     tt.hostCount,
						MemoryGBPerHost:               tt.memoryPerHost,
						CPUCoresPerHost:               64,
						HAAdmissionControlPercentage:  tt.haPercentage,
						DiegoCellCount:                10,
						DiegoCellMemoryGB:             32,
						DiegoCellCPU:                  4,
					},
				},
			}

			state := mi.ToInfrastructureState()
			cluster := state.Clusters[0]

			if cluster.HAUsableMemoryGB != tt.expectedUsableMemory {
				t.Errorf("Expected HAUsableMemoryGB %d, got %d", tt.expectedUsableMemory, cluster.HAUsableMemoryGB)
			}
		})
	}
}

func TestClusterState_HAUsableCPU(t *testing.T) {
	tests := []struct {
		name              string
		hostCount         int
		coresPerHost      int
		haPercentage      int
		expectedUsableCPU int
	}{
		{
			name:              "25% HA reservation with 4 hosts",
			hostCount:         4,
			coresPerHost:      64,
			haPercentage:      25,
			expectedUsableCPU: 192, // 4 * 64 * 0.75 = 192
		},
		{
			name:              "0% HA reservation",
			hostCount:         4,
			coresPerHost:      64,
			haPercentage:      0,
			expectedUsableCPU: 256, // 4 * 64 = 256
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name: "HA CPU Test",
				Clusters: []ClusterInput{
					{
						Name:                          "cluster-01",
						HostCount:                     tt.hostCount,
						MemoryGBPerHost:               1024,
						CPUCoresPerHost:               tt.coresPerHost,
						HAAdmissionControlPercentage:  tt.haPercentage,
						DiegoCellCount:                10,
						DiegoCellMemoryGB:             32,
						DiegoCellCPU:                  4,
					},
				},
			}

			state := mi.ToInfrastructureState()
			cluster := state.Clusters[0]

			if cluster.HAUsableCPUCores != tt.expectedUsableCPU {
				t.Errorf("Expected HAUsableCPUCores %d, got %d", tt.expectedUsableCPU, cluster.HAUsableCPUCores)
			}
		})
	}
}

func TestClusterState_VMsPerHost(t *testing.T) {
	mi := ManualInput{
		Name: "VMs Per Host Test",
		Clusters: []ClusterInput{
			{
				Name:                          "cluster-01",
				HostCount:                     4,
				MemoryGBPerHost:               1024,
				CPUCoresPerHost:               64,
				HAAdmissionControlPercentage:  0,
				DiegoCellCount:                100, // 100 cells across 4 hosts
				DiegoCellMemoryGB:             32,
				DiegoCellCPU:                  4,
			},
		},
	}

	state := mi.ToInfrastructureState()
	cluster := state.Clusters[0]

	// Expected: 100 cells / 4 hosts = 25 VMs per host
	expectedVMsPerHost := 25.0
	if cluster.VMsPerHost != expectedVMsPerHost {
		t.Errorf("Expected VMsPerHost %.1f, got %.1f", expectedVMsPerHost, cluster.VMsPerHost)
	}
}

func TestInfrastructureState_TotalHAUsableCapacity(t *testing.T) {
	mi := ManualInput{
		Name: "Multi-Cluster HA Test",
		Clusters: []ClusterInput{
			{
				Name:                          "cluster-01",
				HostCount:                     4,
				MemoryGBPerHost:               1024,
				CPUCoresPerHost:               64,
				HAAdmissionControlPercentage:  25,
				DiegoCellCount:                100,
				DiegoCellMemoryGB:             32,
				DiegoCellCPU:                  4,
			},
			{
				Name:                          "cluster-02",
				HostCount:                     3,
				MemoryGBPerHost:               512,
				CPUCoresPerHost:               48,
				HAAdmissionControlPercentage:  25,
				DiegoCellCount:                50,
				DiegoCellMemoryGB:             32,
				DiegoCellCPU:                  4,
			},
		},
	}

	state := mi.ToInfrastructureState()

	// cluster-01 HA usable memory: 4 * 1024 * 0.75 = 3072
	// cluster-02 HA usable memory: 3 * 512 * 0.75 = 1152
	// Total: 4224
	expectedTotalHAMemory := 4224
	if state.TotalHAUsableMemoryGB != expectedTotalHAMemory {
		t.Errorf("Expected TotalHAUsableMemoryGB %d, got %d", expectedTotalHAMemory, state.TotalHAUsableMemoryGB)
	}

	// cluster-01 HA usable CPU: 4 * 64 * 0.75 = 192
	// cluster-02 HA usable CPU: 3 * 48 * 0.75 = 108
	// Total: 300
	expectedTotalHACPU := 300
	if state.TotalHAUsableCPUCores != expectedTotalHACPU {
		t.Errorf("Expected TotalHAUsableCPUCores %d, got %d", expectedTotalHACPU, state.TotalHAUsableCPUCores)
	}
}

func TestHostLevelFieldsSerialization(t *testing.T) {
	state := ClusterState{
		Name:                         "Serialization Test",
		HostCount:                    4,
		MemoryGB:                     4096,
		CPUCores:                     256,
		MemoryGBPerHost:              1024,
		CPUCoresPerHost:              64,
		HAAdmissionControlPercentage: 25,
		HAUsableMemoryGB:             3072,
		HAUsableCPUCores:             192,
		VMsPerHost:                   25.0,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal ClusterState: %v", err)
	}

	var decoded ClusterState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ClusterState: %v", err)
	}

	if decoded.MemoryGBPerHost != state.MemoryGBPerHost {
		t.Errorf("MemoryGBPerHost mismatch: got %d, want %d", decoded.MemoryGBPerHost, state.MemoryGBPerHost)
	}
	if decoded.CPUCoresPerHost != state.CPUCoresPerHost {
		t.Errorf("CPUCoresPerHost mismatch: got %d, want %d", decoded.CPUCoresPerHost, state.CPUCoresPerHost)
	}
	if decoded.HAAdmissionControlPercentage != state.HAAdmissionControlPercentage {
		t.Errorf("HAAdmissionControlPercentage mismatch: got %d, want %d", decoded.HAAdmissionControlPercentage, state.HAAdmissionControlPercentage)
	}
	if decoded.HAUsableMemoryGB != state.HAUsableMemoryGB {
		t.Errorf("HAUsableMemoryGB mismatch: got %d, want %d", decoded.HAUsableMemoryGB, state.HAUsableMemoryGB)
	}
	if decoded.HAUsableCPUCores != state.HAUsableCPUCores {
		t.Errorf("HAUsableCPUCores mismatch: got %d, want %d", decoded.HAUsableCPUCores, state.HAUsableCPUCores)
	}
	if decoded.VMsPerHost != state.VMsPerHost {
		t.Errorf("VMsPerHost mismatch: got %.1f, want %.1f", decoded.VMsPerHost, state.VMsPerHost)
	}
}

func TestInfrastructureState_HostLevelSerialization(t *testing.T) {
	state := InfrastructureState{
		Source:               "manual",
		Name:                 "HA Serialization Test",
		TotalHAUsableMemoryGB: 4224,
		TotalHAUsableCPUCores: 300,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal InfrastructureState: %v", err)
	}

	var decoded InfrastructureState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InfrastructureState: %v", err)
	}

	if decoded.TotalHAUsableMemoryGB != state.TotalHAUsableMemoryGB {
		t.Errorf("TotalHAUsableMemoryGB mismatch: got %d, want %d", decoded.TotalHAUsableMemoryGB, state.TotalHAUsableMemoryGB)
	}
	if decoded.TotalHAUsableCPUCores != state.TotalHAUsableCPUCores {
		t.Errorf("TotalHAUsableCPUCores mismatch: got %d, want %d", decoded.TotalHAUsableCPUCores, state.TotalHAUsableCPUCores)
	}
}
