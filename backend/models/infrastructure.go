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
	Name              string `json:"name"`
	HostCount         int    `json:"host_count"`
	MemoryGB          int    `json:"memory_gb"`
	CPUCores          int    `json:"cpu_cores"`
	N1MemoryGB        int    `json:"n1_memory_gb"`
	UsableMemoryGB    int    `json:"usable_memory_gb"`
	DiegoCellCount    int    `json:"diego_cell_count"`
	DiegoCellMemoryGB int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU      int    `json:"diego_cell_cpu"`
	DiegoCellDiskGB   int    `json:"diego_cell_disk_gb"`
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
	TotalAppDiskGB    int            `json:"total_app_disk_gb"`
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
		TotalAppDiskGB:    mi.TotalAppDiskGB,
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
			DiegoCellDiskGB:   c.DiegoCellDiskGB,
		}

		state.TotalMemoryGB += clusterMemory
		state.TotalN1MemoryGB += n1Memory
		state.TotalHostCount += c.HostCount
		state.TotalCellCount += c.DiegoCellCount
	}

	return state
}
