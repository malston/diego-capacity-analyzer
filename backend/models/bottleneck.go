// ABOUTME: Multi-resource bottleneck analysis for capacity planning
// ABOUTME: Ranks resources by utilization and identifies constraining resource

package models

import (
	"fmt"
	"sort"
)

// ResourceUtilization represents the utilization of a single resource type
type ResourceUtilization struct {
	Name           string  `json:"name"`
	UsedPercent    float64 `json:"used_percent"`
	TotalCapacity  int     `json:"total_capacity"`
	UsedCapacity   int     `json:"used_capacity"`
	Unit           string  `json:"unit"`
	IsConstraining bool    `json:"is_constraining"`
}

// BottleneckAnalysis represents the complete bottleneck analysis result
type BottleneckAnalysis struct {
	Resources            []ResourceUtilization `json:"resources"`
	ConstrainingResource string                `json:"constraining_resource"`
	Summary              string                `json:"summary"`
}

// RankResourcesByUtilization sorts resources by utilization percentage in descending order
// and marks the highest utilization resource as constraining.
func RankResourcesByUtilization(resources []ResourceUtilization) []ResourceUtilization {
	if len(resources) == 0 {
		return resources
	}

	// Make a copy to avoid modifying the original slice
	ranked := make([]ResourceUtilization, len(resources))
	copy(ranked, resources)

	// Stable sort by utilization descending (preserves original order for equal values)
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].UsedPercent > ranked[j].UsedPercent
	})

	// Mark the first (highest utilization) as constraining
	for i := range ranked {
		ranked[i].IsConstraining = (i == 0)
	}

	return ranked
}

// GetConstrainingResource returns the resource with the highest utilization
func GetConstrainingResource(resources []ResourceUtilization) *ResourceUtilization {
	if len(resources) == 0 {
		return nil
	}

	ranked := RankResourcesByUtilization(resources)
	return &ranked[0]
}

// AnalyzeBottleneck performs multi-resource bottleneck analysis on infrastructure state
func AnalyzeBottleneck(state InfrastructureState) BottleneckAnalysis {
	resources := buildResourceList(state)
	ranked := RankResourcesByUtilization(resources)

	analysis := BottleneckAnalysis{
		Resources: ranked,
	}

	if len(ranked) > 0 {
		analysis.ConstrainingResource = ranked[0].Name
		analysis.Summary = buildSummary(ranked)
	}

	return analysis
}

// buildResourceList extracts resource utilization data from infrastructure state
func buildResourceList(state InfrastructureState) []ResourceUtilization {
	var resources []ResourceUtilization

	// Memory utilization (app memory used / total cell memory capacity)
	if state.TotalCellMemoryGB > 0 {
		memoryPercent := (float64(state.TotalAppMemoryGB) / float64(state.TotalCellMemoryGB)) * 100.0
		resources = append(resources, ResourceUtilization{
			Name:          "Memory",
			UsedPercent:   memoryPercent,
			TotalCapacity: state.TotalCellMemoryGB,
			UsedCapacity:  state.TotalAppMemoryGB,
			Unit:          "GB",
		})
	}

	// CPU utilization (based on vCPU:pCPU ratio)
	// Uses host CPU utilization percent directly since it represents vCPU overcommit
	if state.TotalCPUCores > 0 {
		resources = append(resources, ResourceUtilization{
			Name:          "CPU",
			UsedPercent:   state.HostCPUUtilizationPercent,
			TotalCapacity: state.TotalCPUCores,
			UsedCapacity:  state.TotalVCPUs,
			Unit:          "cores",
		})
	}

	// Disk utilization (app disk used / total cell disk capacity)
	totalCellDiskGB := calculateTotalCellDisk(state)
	if totalCellDiskGB > 0 {
		diskPercent := (float64(state.TotalAppDiskGB) / float64(totalCellDiskGB)) * 100.0
		resources = append(resources, ResourceUtilization{
			Name:          "Disk",
			UsedPercent:   diskPercent,
			TotalCapacity: totalCellDiskGB,
			UsedCapacity:  state.TotalAppDiskGB,
			Unit:          "GB",
		})
	}

	return resources
}

// calculateTotalCellDisk sums disk capacity across all clusters
func calculateTotalCellDisk(state InfrastructureState) int {
	total := 0
	for _, cluster := range state.Clusters {
		total += cluster.DiegoCellCount * cluster.DiegoCellDiskGB
	}
	return total
}

// buildSummary generates a human-readable summary of the bottleneck analysis
func buildSummary(ranked []ResourceUtilization) string {
	if len(ranked) == 0 {
		return "No resources to analyze."
	}

	constraining := ranked[0]
	return fmt.Sprintf("%s is your constraint at %.1f%% utilization. Address %s capacity before other resources.",
		constraining.Name, constraining.UsedPercent, constraining.Name)
}
