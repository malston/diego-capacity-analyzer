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
		CPUThreadsPerHost:               64,
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
				CPUThreadsPerHost:               64,
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

	if cluster.CPUThreadsPerHost != 64 {
		t.Errorf("Expected CPUThreadsPerHost 64, got %d", cluster.CPUThreadsPerHost)
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
						CPUThreadsPerHost:               64,
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
						CPUThreadsPerHost:               tt.coresPerHost,
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
				CPUThreadsPerHost:               64,
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
				CPUThreadsPerHost:               64,
				HAAdmissionControlPercentage:  25,
				DiegoCellCount:                100,
				DiegoCellMemoryGB:             32,
				DiegoCellCPU:                  4,
			},
			{
				Name:                          "cluster-02",
				HostCount:                     3,
				MemoryGBPerHost:               512,
				CPUThreadsPerHost:               48,
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
		CPUThreadsPerHost:              64,
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
	if decoded.CPUThreadsPerHost != state.CPUThreadsPerHost {
		t.Errorf("CPUThreadsPerHost mismatch: got %d, want %d", decoded.CPUThreadsPerHost, state.CPUThreadsPerHost)
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
		Source:                "manual",
		Name:                  "HA Serialization Test",
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

func TestClusterState_HostMemoryUtilization(t *testing.T) {
	tests := []struct {
		name                       string
		hostCount                  int
		memoryPerHost              int
		cellCount                  int
		cellMemory                 int
		expectedUtilizationPercent float64
	}{
		{
			name:          "50% memory utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     64,
			cellMemory:    32,
			// Total host memory: 4 * 1024 = 4096 GB
			// Total cell memory: 64 * 32 = 2048 GB
			// Utilization: 2048 / 4096 = 50%
			expectedUtilizationPercent: 50.0,
		},
		{
			name:          "75% memory utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     96,
			cellMemory:    32,
			// Total host memory: 4096 GB
			// Total cell memory: 96 * 32 = 3072 GB
			// Utilization: 3072 / 4096 = 75%
			expectedUtilizationPercent: 75.0,
		},
		{
			name:          "100% memory utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     128,
			cellMemory:    32,
			// Total host memory: 4096 GB
			// Total cell memory: 128 * 32 = 4096 GB
			// Utilization: 4096 / 4096 = 100%
			expectedUtilizationPercent: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name: "Memory Utilization Test",
				Clusters: []ClusterInput{
					{
						Name:              "cluster-01",
						HostCount:         tt.hostCount,
						MemoryGBPerHost:   tt.memoryPerHost,
						CPUThreadsPerHost:   64,
						DiegoCellCount:    tt.cellCount,
						DiegoCellMemoryGB: tt.cellMemory,
						DiegoCellCPU:      4,
					},
				},
			}

			state := mi.ToInfrastructureState()
			cluster := state.Clusters[0]

			if cluster.HostMemoryUtilizationPercent != tt.expectedUtilizationPercent {
				t.Errorf("Expected HostMemoryUtilizationPercent %.1f, got %.1f",
					tt.expectedUtilizationPercent, cluster.HostMemoryUtilizationPercent)
			}
		})
	}
}

func TestClusterState_HostCPUUtilization(t *testing.T) {
	tests := []struct {
		name                       string
		hostCount                  int
		coresPerHost               int
		cellCount                  int
		cellCPU                    int
		expectedUtilizationPercent float64
	}{
		{
			name:         "50% CPU utilization (2:1 ratio)",
			hostCount:    4,
			coresPerHost: 64,
			cellCount:    32,
			cellCPU:      4,
			// Total host cores: 4 * 64 = 256
			// Total cell vCPU: 32 * 4 = 128
			// Utilization: 128 / 256 = 50%
			expectedUtilizationPercent: 50.0,
		},
		{
			name:         "200% CPU utilization (2:1 overcommit)",
			hostCount:    4,
			coresPerHost: 64,
			cellCount:    128,
			cellCPU:      4,
			// Total host cores: 256
			// Total cell vCPU: 128 * 4 = 512
			// Utilization: 512 / 256 = 200%
			expectedUtilizationPercent: 200.0,
		},
		{
			name:         "100% CPU utilization (1:1 ratio)",
			hostCount:    4,
			coresPerHost: 64,
			cellCount:    64,
			cellCPU:      4,
			// Total host cores: 256
			// Total cell vCPU: 64 * 4 = 256
			// Utilization: 256 / 256 = 100%
			expectedUtilizationPercent: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name: "CPU Utilization Test",
				Clusters: []ClusterInput{
					{
						Name:              "cluster-01",
						HostCount:         tt.hostCount,
						MemoryGBPerHost:   1024,
						CPUThreadsPerHost:   tt.coresPerHost,
						DiegoCellCount:    tt.cellCount,
						DiegoCellMemoryGB: 32,
						DiegoCellCPU:      tt.cellCPU,
					},
				},
			}

			state := mi.ToInfrastructureState()
			cluster := state.Clusters[0]

			if cluster.HostCPUUtilizationPercent != tt.expectedUtilizationPercent {
				t.Errorf("Expected HostCPUUtilizationPercent %.1f, got %.1f",
					tt.expectedUtilizationPercent, cluster.HostCPUUtilizationPercent)
			}
		})
	}
}

func TestClusterState_ZeroHostsHandled(t *testing.T) {
	mi := ManualInput{
		Name: "Zero Hosts Test",
		Clusters: []ClusterInput{
			{
				Name:              "empty-cluster",
				HostCount:         0,
				MemoryGBPerHost:   1024,
				CPUThreadsPerHost:   64,
				DiegoCellCount:    10,
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	state := mi.ToInfrastructureState()
	cluster := state.Clusters[0]

	// Should not panic or have NaN/Inf values
	if cluster.VMsPerHost != 0 {
		t.Errorf("Expected VMsPerHost 0 with zero hosts, got %.1f", cluster.VMsPerHost)
	}
	if cluster.HostMemoryUtilizationPercent != 0 {
		t.Errorf("Expected HostMemoryUtilizationPercent 0 with zero hosts, got %.1f", cluster.HostMemoryUtilizationPercent)
	}
	if cluster.HostCPUUtilizationPercent != 0 {
		t.Errorf("Expected HostCPUUtilizationPercent 0 with zero hosts, got %.1f", cluster.HostCPUUtilizationPercent)
	}
}

func TestInfrastructureState_AggregateHostUtilization(t *testing.T) {
	mi := ManualInput{
		Name: "Multi-Cluster Utilization Test",
		Clusters: []ClusterInput{
			{
				Name:              "cluster-01",
				HostCount:         4,
				MemoryGBPerHost:   1024,
				CPUThreadsPerHost:   64,
				DiegoCellCount:    64, // 50% memory usage (2048/4096), 100% CPU (256/256)
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
			{
				Name:              "cluster-02",
				HostCount:         2,
				MemoryGBPerHost:   512,
				CPUThreadsPerHost:   32,
				DiegoCellCount:    16, // 50% memory usage (512/1024), 100% CPU (64/64)
				DiegoCellMemoryGB: 32,
				DiegoCellCPU:      4,
			},
		},
	}

	state := mi.ToInfrastructureState()

	// Total host memory: 4*1024 + 2*512 = 5120 GB
	// Total cell memory: 64*32 + 16*32 = 2560 GB
	// Aggregate utilization: 2560 / 5120 = 50%
	expectedMemoryUtil := 50.0
	if state.HostMemoryUtilizationPercent != expectedMemoryUtil {
		t.Errorf("Expected aggregate HostMemoryUtilizationPercent %.1f, got %.1f",
			expectedMemoryUtil, state.HostMemoryUtilizationPercent)
	}

	// Total host cores: 4*64 + 2*32 = 320
	// Total vCPUs: 64*4 + 16*4 = 320
	// Aggregate utilization: 320 / 320 = 100%
	expectedCPUUtil := 100.0
	if state.HostCPUUtilizationPercent != expectedCPUUtil {
		t.Errorf("Expected aggregate HostCPUUtilizationPercent %.1f, got %.1f",
			expectedCPUUtil, state.HostCPUUtilizationPercent)
	}
}

func TestHostUtilizationSerialization(t *testing.T) {
	state := ClusterState{
		Name:                         "Utilization Serialization Test",
		HostMemoryUtilizationPercent: 75.5,
		HostCPUUtilizationPercent:    125.0,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal ClusterState: %v", err)
	}

	var decoded ClusterState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ClusterState: %v", err)
	}

	if decoded.HostMemoryUtilizationPercent != state.HostMemoryUtilizationPercent {
		t.Errorf("HostMemoryUtilizationPercent mismatch: got %.1f, want %.1f",
			decoded.HostMemoryUtilizationPercent, state.HostMemoryUtilizationPercent)
	}
	if decoded.HostCPUUtilizationPercent != state.HostCPUUtilizationPercent {
		t.Errorf("HostCPUUtilizationPercent mismatch: got %.1f, want %.1f",
			decoded.HostCPUUtilizationPercent, state.HostCPUUtilizationPercent)
	}
}

func TestClusterState_HAHostFailureCapacity(t *testing.T) {
	tests := []struct {
		name                         string
		hostCount                    int
		memoryPerHost                int
		cellCount                    int
		cellMemory                   int
		haPercentage                 int
		expectedHostFailuresSurvived int
		expectedHAStatus             string
	}{
		{
			name:          "Can survive 2 host failures - low utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     48, // 48 * 32 = 1536 GB (37.5% of 4096)
			cellMemory:    32,
			haPercentage:  25, // 75% usable = 3072 GB, need 1536 GB
			// With 3 hosts (one failed), 75% of 3072 = 2304 GB usable >= 1536 GB -> OK
			// With 2 hosts (two failed), 75% of 2048 = 1536 GB usable >= 1536 GB -> OK
			expectedHostFailuresSurvived: 2,
			expectedHAStatus:             "ok",
		},
		{
			name:          "Cannot survive host failure - high utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     96, // 96 * 32 = 3072 GB (75% of 4096)
			cellMemory:    32,
			haPercentage:  25, // 75% usable = 3072 GB, need 3072 GB
			// With 3 hosts (one failed), 75% of 3072 = 2304 GB usable < 3072 GB -> FAIL
			expectedHostFailuresSurvived: 0,
			expectedHAStatus:             "at-risk",
		},
		{
			name:          "Can survive 3 host failures - very low utilization",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     24, // 24 * 32 = 768 GB (18.75% of 4096)
			cellMemory:    32,
			haPercentage:  25, // 75% usable = 3072 GB, need 768 GB
			// With 1 host (three failed), 75% of 1024 = 768 GB usable >= 768 GB -> OK
			expectedHostFailuresSurvived: 3,
			expectedHAStatus:             "ok",
		},
		{
			name:          "No HA reservation - can survive 2 failures",
			hostCount:     4,
			memoryPerHost: 1024,
			cellCount:     64, // 64 * 32 = 2048 GB (50% of 4096)
			cellMemory:    32,
			haPercentage:  0, // 100% usable = 4096 GB, need 2048 GB
			// With 2 hosts (two failed), 100% of 2048 = 2048 GB usable >= 2048 GB -> OK
			expectedHostFailuresSurvived: 2,
			expectedHAStatus:             "ok",
		},
		{
			name:                         "Single host cluster - cannot survive any failure",
			hostCount:                    1,
			memoryPerHost:                1024,
			cellCount:                    16,
			cellMemory:                   32,
			haPercentage:                 0,
			expectedHostFailuresSurvived: 0,
			expectedHAStatus:             "at-risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name: "HA Capacity Test",
				Clusters: []ClusterInput{
					{
						Name:                         "cluster-01",
						HostCount:                    tt.hostCount,
						MemoryGBPerHost:              tt.memoryPerHost,
						CPUThreadsPerHost:              64,
						HAAdmissionControlPercentage: tt.haPercentage,
						DiegoCellCount:               tt.cellCount,
						DiegoCellMemoryGB:            tt.cellMemory,
						DiegoCellCPU:                 4,
					},
				},
			}

			state := mi.ToInfrastructureState()
			cluster := state.Clusters[0]

			if cluster.HAHostFailuresSurvived != tt.expectedHostFailuresSurvived {
				t.Errorf("Expected HAHostFailuresSurvived %d, got %d",
					tt.expectedHostFailuresSurvived, cluster.HAHostFailuresSurvived)
			}

			if cluster.HAStatus != tt.expectedHAStatus {
				t.Errorf("Expected HAStatus '%s', got '%s'",
					tt.expectedHAStatus, cluster.HAStatus)
			}
		})
	}
}

func TestInfrastructureState_AggregateHAStatus(t *testing.T) {
	tests := []struct {
		name                         string
		clusters                     []ClusterInput
		expectedHAStatus             string
		expectedMinHostFailures      int
	}{
		{
			name: "All clusters healthy - reports minimum failures",
			clusters: []ClusterInput{
				{
					Name:              "cluster-01",
					HostCount:         4,
					MemoryGBPerHost:   1024,
					CPUThreadsPerHost:   64,
					DiegoCellCount:    24, // 768 GB needed, can survive 3 failures
					DiegoCellMemoryGB: 32,
					DiegoCellCPU:      4,
				},
				{
					Name:              "cluster-02",
					HostCount:         3,
					MemoryGBPerHost:   1024,
					CPUThreadsPerHost:   64,
					DiegoCellCount:    24, // 768 GB needed, 3 hosts, can survive 2 failures
					DiegoCellMemoryGB: 32,
					DiegoCellCPU:      4,
				},
			},
			expectedHAStatus:        "ok",
			expectedMinHostFailures: 2, // Minimum across clusters (cluster-02)
		},
		{
			name: "One cluster at risk - reports at-risk",
			clusters: []ClusterInput{
				{
					Name:              "cluster-01",
					HostCount:         4,
					MemoryGBPerHost:   1024,
					CPUThreadsPerHost:   64,
					DiegoCellCount:    24,
					DiegoCellMemoryGB: 32,
					DiegoCellCPU:      4,
				},
				{
					Name:                         "cluster-02",
					HostCount:                    4,
					MemoryGBPerHost:              1024,
					CPUThreadsPerHost:              64,
					HAAdmissionControlPercentage: 25,
					DiegoCellCount:               96, // High utilization
					DiegoCellMemoryGB:            32,
					DiegoCellCPU:                 4,
				},
			},
			expectedHAStatus:        "at-risk",
			expectedMinHostFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := ManualInput{
				Name:     "Multi-Cluster HA Test",
				Clusters: tt.clusters,
			}

			state := mi.ToInfrastructureState()

			if state.HAStatus != tt.expectedHAStatus {
				t.Errorf("Expected HAStatus '%s', got '%s'",
					tt.expectedHAStatus, state.HAStatus)
			}

			if state.HAMinHostFailuresSurvived != tt.expectedMinHostFailures {
				t.Errorf("Expected HAMinHostFailuresSurvived %d, got %d",
					tt.expectedMinHostFailures, state.HAMinHostFailuresSurvived)
			}
		})
	}
}

func TestHAFieldsSerialization(t *testing.T) {
	state := ClusterState{
		Name:                   "HA Serialization Test",
		HAHostFailuresSurvived: 2,
		HAStatus:               "ok",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal ClusterState: %v", err)
	}

	var decoded ClusterState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ClusterState: %v", err)
	}

	if decoded.HAHostFailuresSurvived != state.HAHostFailuresSurvived {
		t.Errorf("HAHostFailuresSurvived mismatch: got %d, want %d",
			decoded.HAHostFailuresSurvived, state.HAHostFailuresSurvived)
	}
	if decoded.HAStatus != state.HAStatus {
		t.Errorf("HAStatus mismatch: got '%s', want '%s'",
			decoded.HAStatus, state.HAStatus)
	}
}
