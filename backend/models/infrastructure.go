// ABOUTME: Data models for infrastructure state and manual input
// ABOUTME: Supports what-if capacity analysis with user-provided data

package models

import "time"

// ClusterInput represents user-provided cluster configuration
type ClusterInput struct {
	Name                         string `json:"name"`
	HostCount                    int    `json:"host_count"`
	MemoryGBPerHost              int    `json:"memory_gb_per_host"`
	CPUCoresPerHost              int    `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int    `json:"ha_admission_control_percentage"`
	DiegoCellCount               int    `json:"diego_cell_count"`
	DiegoCellMemoryGB            int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int    `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int    `json:"diego_cell_disk_gb"`
}

// ManualInput represents user-provided infrastructure data
type ManualInput struct {
	Name              string         `json:"name"`
	Clusters          []ClusterInput `json:"clusters"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppDiskGB    int            `json:"total_app_disk_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
}

// ClusterState represents computed cluster metrics
type ClusterState struct {
	Name                         string  `json:"name"`
	HostCount                    int     `json:"host_count"`
	MemoryGB                     int     `json:"memory_gb"`
	CPUCores                     int     `json:"cpu_cores"`
	MemoryGBPerHost              int     `json:"memory_gb_per_host"`
	CPUCoresPerHost              int     `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int     `json:"ha_admission_control_percentage"`
	HAUsableMemoryGB             int     `json:"ha_usable_memory_gb"`
	HAUsableCPUCores             int     `json:"ha_usable_cpu_cores"`
	HAHostFailuresSurvived       int     `json:"ha_host_failures_survived"`
	HAStatus                     string  `json:"ha_status"`
	VMsPerHost                   float64 `json:"vms_per_host"`
	HostMemoryUtilizationPercent float64 `json:"host_memory_utilization_percent"`
	HostCPUUtilizationPercent    float64 `json:"host_cpu_utilization_percent"`
	N1MemoryGB                   int     `json:"n1_memory_gb"`
	UsableMemoryGB               int     `json:"usable_memory_gb"`
	DiegoCellCount               int     `json:"diego_cell_count"`
	DiegoCellMemoryGB            int     `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int     `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int     `json:"diego_cell_disk_gb"`
	TotalVCPUs                   int     `json:"total_vcpus"`
	TotalCellMemoryGB            int     `json:"total_cell_memory_gb"`
	VCPURatio                    float64 `json:"vcpu_ratio"`
}

// InfrastructureState represents computed infrastructure metrics
type InfrastructureState struct {
	Source                       string         `json:"source"` // "manual" or "vsphere"
	Name                         string         `json:"name"`
	Clusters                     []ClusterState `json:"clusters"`
	TotalMemoryGB                int            `json:"total_memory_gb"`
	TotalN1MemoryGB              int            `json:"total_n1_memory_gb"`
	TotalHAUsableMemoryGB        int            `json:"total_ha_usable_memory_gb"`
	TotalHAUsableCPUCores        int            `json:"total_ha_usable_cpu_cores"`
	HAMinHostFailuresSurvived    int            `json:"ha_min_host_failures_survived"`
	HAStatus                     string         `json:"ha_status"`
	TotalCellMemoryGB            int            `json:"total_cell_memory_gb"`
	HostMemoryUtilizationPercent float64        `json:"host_memory_utilization_percent"`
	HostCPUUtilizationPercent    float64        `json:"host_cpu_utilization_percent"`
	TotalHostCount               int            `json:"total_host_count"`
	TotalCellCount               int            `json:"total_cell_count"`
	TotalCPUCores                int            `json:"total_cpu_cores"`
	TotalVCPUs                   int            `json:"total_vcpus"`
	VCPURatio                    float64        `json:"vcpu_ratio"`
	CPURiskLevel                 string         `json:"cpu_risk_level"`
	PlatformVMsGB                int            `json:"platform_vms_gb"`
	TotalAppMemoryGB             int            `json:"total_app_memory_gb"`
	TotalAppDiskGB               int            `json:"total_app_disk_gb"`
	TotalAppInstances            int            `json:"total_app_instances"`
	AvgInstanceMemoryMB          int            `json:"avg_instance_memory_mb"`
	Timestamp                    time.Time      `json:"timestamp"`
	Cached                       bool           `json:"cached"`
}

// CPURiskLevel returns the risk level based on vCPU:pCPU ratio
// Thresholds: ≤4:1 = low, 4:1-8:1 = medium, >8:1 = high
func CPURiskLevel(ratio float64) string {
	if ratio <= 4.0 {
		return "low"
	}
	if ratio <= 8.0 {
		return "medium"
	}
	return "high"
}

// CalculateHAHostFailures determines how many host failures a cluster can survive
// based on its current capacity utilization and HA admission control policy.
// Returns (hostFailuresSurvived, haStatus)
func CalculateHAHostFailures(hostCount, memoryPerHost, haPercentage, requiredMemory int) (int, string) {
	if hostCount <= 1 {
		return 0, "at-risk"
	}

	haMultiplier := float64(100-haPercentage) / 100.0

	// Test how many hosts can fail while still meeting capacity requirements
	failuresSurvived := 0
	for failedHosts := 1; failedHosts < hostCount; failedHosts++ {
		remainingHosts := hostCount - failedHosts
		remainingMemory := remainingHosts * memoryPerHost
		usableMemory := int(float64(remainingMemory) * haMultiplier)

		if usableMemory >= requiredMemory {
			failuresSurvived = failedHosts
		} else {
			break
		}
	}

	status := "at-risk"
	if failuresSurvived >= 1 {
		status = "ok"
	}

	return failuresSurvived, status
}

// ToInfrastructureState converts manual input to computed state
func (mi *ManualInput) ToInfrastructureState() InfrastructureState {
	state := InfrastructureState{
		Source:            "manual",
		Name:              mi.Name,
		Clusters:          make([]ClusterState, len(mi.Clusters)),
		PlatformVMsGB:     mi.PlatformVMsGB,
		TotalAppMemoryGB:  mi.TotalAppMemoryGB,
		TotalAppDiskGB:    mi.TotalAppDiskGB,
		TotalAppInstances: mi.TotalAppInstances,
		Timestamp:         time.Now(),
		Cached:            false,
	}

	for i, c := range mi.Clusters {
		clusterMemory := c.HostCount * c.MemoryGBPerHost
		clusterCPU := c.HostCount * c.CPUCoresPerHost
		clusterVCPUs := c.DiegoCellCount * c.DiegoCellCPU
		clusterCellMemory := c.DiegoCellCount * c.DiegoCellMemoryGB
		n1Memory := (c.HostCount - 1) * c.MemoryGBPerHost
		usableMemory := int(float64(n1Memory) * 0.9) // 10% overhead

		// Calculate HA-aware usable capacity
		haMultiplier := float64(100-c.HAAdmissionControlPercentage) / 100.0
		haUsableMemory := int(float64(clusterMemory) * haMultiplier)
		haUsableCPU := int(float64(clusterCPU) * haMultiplier)

		// Calculate VMs per host
		var vmsPerHost float64
		if c.HostCount > 0 {
			vmsPerHost = float64(c.DiegoCellCount) / float64(c.HostCount)
		}

		// Calculate host utilization percentages
		var hostMemoryUtil, hostCPUUtil float64
		if clusterMemory > 0 {
			hostMemoryUtil = (float64(clusterCellMemory) / float64(clusterMemory)) * 100.0
		}
		if clusterCPU > 0 {
			hostCPUUtil = (float64(clusterVCPUs) / float64(clusterCPU)) * 100.0
		}

		var clusterVCPURatio float64
		if clusterCPU > 0 {
			clusterVCPURatio = float64(clusterVCPUs) / float64(clusterCPU)
		}

		// Calculate HA host failure capacity
		haFailures, haStatus := CalculateHAHostFailures(
			c.HostCount, c.MemoryGBPerHost, c.HAAdmissionControlPercentage, clusterCellMemory)

		state.Clusters[i] = ClusterState{
			Name:                         c.Name,
			HostCount:                    c.HostCount,
			MemoryGB:                     clusterMemory,
			CPUCores:                     clusterCPU,
			MemoryGBPerHost:              c.MemoryGBPerHost,
			CPUCoresPerHost:              c.CPUCoresPerHost,
			HAAdmissionControlPercentage: c.HAAdmissionControlPercentage,
			HAUsableMemoryGB:             haUsableMemory,
			HAUsableCPUCores:             haUsableCPU,
			HAHostFailuresSurvived:       haFailures,
			HAStatus:                     haStatus,
			VMsPerHost:                   vmsPerHost,
			HostMemoryUtilizationPercent: hostMemoryUtil,
			HostCPUUtilizationPercent:    hostCPUUtil,
			N1MemoryGB:                   n1Memory,
			UsableMemoryGB:               usableMemory,
			DiegoCellCount:               c.DiegoCellCount,
			DiegoCellMemoryGB:            c.DiegoCellMemoryGB,
			DiegoCellCPU:                 c.DiegoCellCPU,
			DiegoCellDiskGB:              c.DiegoCellDiskGB,
			TotalVCPUs:                   clusterVCPUs,
			TotalCellMemoryGB:            clusterCellMemory,
			VCPURatio:                    clusterVCPURatio,
		}

		state.TotalMemoryGB += clusterMemory
		state.TotalN1MemoryGB += n1Memory
		state.TotalHAUsableMemoryGB += haUsableMemory
		state.TotalHAUsableCPUCores += haUsableCPU
		state.TotalCellMemoryGB += clusterCellMemory
		state.TotalHostCount += c.HostCount
		state.TotalCellCount += c.DiegoCellCount
		state.TotalCPUCores += clusterCPU
		state.TotalVCPUs += clusterVCPUs
	}

	// Calculate overall vCPU:pCPU ratio
	if state.TotalCPUCores > 0 {
		state.VCPURatio = float64(state.TotalVCPUs) / float64(state.TotalCPUCores)
	}
	state.CPURiskLevel = CPURiskLevel(state.VCPURatio)

	// Calculate aggregate host utilization percentages
	if state.TotalMemoryGB > 0 {
		state.HostMemoryUtilizationPercent = (float64(state.TotalCellMemoryGB) / float64(state.TotalMemoryGB)) * 100.0
	}
	if state.TotalCPUCores > 0 {
		state.HostCPUUtilizationPercent = (float64(state.TotalVCPUs) / float64(state.TotalCPUCores)) * 100.0
	}

	// Calculate aggregate HA status (minimum failures survived across all clusters)
	state.HAMinHostFailuresSurvived = -1 // Use -1 as uninitialized
	state.HAStatus = "ok"
	for _, cluster := range state.Clusters {
		if state.HAMinHostFailuresSurvived == -1 || cluster.HAHostFailuresSurvived < state.HAMinHostFailuresSurvived {
			state.HAMinHostFailuresSurvived = cluster.HAHostFailuresSurvived
		}
		if cluster.HAStatus == "at-risk" {
			state.HAStatus = "at-risk"
		}
	}
	if state.HAMinHostFailuresSurvived == -1 {
		state.HAMinHostFailuresSurvived = 0
	}

	// Calculate average instance memory
	if state.TotalAppInstances > 0 {
		state.AvgInstanceMemoryMB = state.TotalAppMemoryGB * 1024 / state.TotalAppInstances
	}

	return state
}
