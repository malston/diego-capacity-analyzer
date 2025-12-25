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
	DiegoCellDiskGB   int    `json:"diego_cell_disk_gb"`
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
	Name              string  `json:"name"`
	HostCount         int     `json:"host_count"`
	MemoryGB          int     `json:"memory_gb"`
	CPUCores          int     `json:"cpu_cores"`
	N1MemoryGB        int     `json:"n1_memory_gb"`
	UsableMemoryGB    int     `json:"usable_memory_gb"`
	DiegoCellCount    int     `json:"diego_cell_count"`
	DiegoCellMemoryGB int     `json:"diego_cell_memory_gb"`
	DiegoCellCPU      int     `json:"diego_cell_cpu"`
	DiegoCellDiskGB   int     `json:"diego_cell_disk_gb"`
	TotalVCPUs        int     `json:"total_vcpus"`
	VCPURatio         float64 `json:"vcpu_ratio"`
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
	TotalCPUCores     int            `json:"total_cpu_cores"`
	TotalVCPUs        int            `json:"total_vcpus"`
	VCPURatio         float64        `json:"vcpu_ratio"`
	CPURiskLevel      string         `json:"cpu_risk_level"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppDiskGB    int            `json:"total_app_disk_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
	Timestamp         time.Time      `json:"timestamp"`
	Cached            bool           `json:"cached"`
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
		n1Memory := (c.HostCount - 1) * c.MemoryGBPerHost
		usableMemory := int(float64(n1Memory) * 0.9) // 10% overhead

		var clusterVCPURatio float64
		if clusterCPU > 0 {
			clusterVCPURatio = float64(clusterVCPUs) / float64(clusterCPU)
		}

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
			DiegoCellDiskGB:   c.DiegoCellDiskGB,
			TotalVCPUs:        clusterVCPUs,
			VCPURatio:         clusterVCPURatio,
		}

		state.TotalMemoryGB += clusterMemory
		state.TotalN1MemoryGB += n1Memory
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

	return state
}
