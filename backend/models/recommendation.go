// ABOUTME: Upgrade path recommendations for capacity planning
// ABOUTME: Generates actionable recommendations based on resource constraints

package models

import (
	"fmt"
	"sort"
)

// RecommendationType defines the type of upgrade recommendation
type RecommendationType string

const (
	RecommendationAddCells    RecommendationType = "add_cells"
	RecommendationResizeCells RecommendationType = "resize_cells"
	RecommendationAddHosts    RecommendationType = "add_hosts"
)

// Recommendation represents an actionable upgrade recommendation
type Recommendation struct {
	Type            RecommendationType `json:"type"`
	Priority        int                `json:"priority"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	Impact          string             `json:"impact"`
	ImpactLevel     string             `json:"impact_level"`
	Resource        string             `json:"resource"`
	CellsToAdd      int                `json:"cells_to_add,omitempty"`
	HostsToAdd      int                `json:"hosts_to_add,omitempty"`
	NewCellMemoryGB int                `json:"new_cell_memory_gb,omitempty"`
	NewCellCPU      int                `json:"new_cell_cpu,omitempty"`
}

// RecommendationsResponse wraps the list of recommendations with context
type RecommendationsResponse struct {
	Recommendations      []Recommendation `json:"recommendations"`
	ConstrainingResource string           `json:"constraining_resource"`
}

// GenerateAddCellsRecommendation creates a recommendation to add more Diego cells
func GenerateAddCellsRecommendation(state InfrastructureState, constrainingResource string) *Recommendation {
	if len(state.Clusters) == 0 {
		return nil
	}

	cluster := state.Clusters[0]

	// Calculate how many cells to add to reduce utilization to 70%
	var cellsToAdd int
	var impact string

	switch constrainingResource {
	case "Memory":
		if state.TotalCellMemoryGB == 0 || cluster.DiegoCellMemoryGB == 0 {
			return nil
		}
		currentUtil := float64(state.TotalAppMemoryGB) / float64(state.TotalCellMemoryGB)
		targetUtil := 0.70
		if currentUtil <= targetUtil {
			cellsToAdd = 2 // Minimum recommendation
		} else {
			// Calculate cells needed: (appMem / targetUtil) - currentCellMem = neededExtraCapacity
			// neededExtraCapacity / cellMem = cellsToAdd
			targetCapacity := float64(state.TotalAppMemoryGB) / targetUtil
			neededExtra := targetCapacity - float64(state.TotalCellMemoryGB)
			cellsToAdd = int(neededExtra/float64(cluster.DiegoCellMemoryGB)) + 1
		}
		memoryGain := cellsToAdd * cluster.DiegoCellMemoryGB
		impact = fmt.Sprintf("Increases memory capacity by %d GB", memoryGain)

	case "CPU":
		if state.TotalCPUCores == 0 || cluster.DiegoCellCPU == 0 {
			return nil
		}
		// For CPU, we want to reduce vCPU:pCPU ratio
		// Adding cells actually increases vCPUs, so this may not be the best recommendation
		// But we provide it for completeness
		cellsToAdd = 2 // Minimum recommendation
		cpuGain := cellsToAdd * cluster.DiegoCellCPU
		impact = fmt.Sprintf("Adds %d vCPUs (note: may increase overcommit ratio)", cpuGain)

	default:
		cellsToAdd = 2
		impact = "Increases overall capacity"
	}

	if cellsToAdd < 1 {
		cellsToAdd = 1
	}

	return &Recommendation{
		Type:        RecommendationAddCells,
		Priority:    1,
		Title:       "Add Diego Cells",
		Description: fmt.Sprintf("Add %d more Diego cells to increase capacity", cellsToAdd),
		Impact:      impact,
		ImpactLevel: "high",
		Resource:    constrainingResource,
		CellsToAdd:  cellsToAdd,
	}
}

// GenerateResizeCellsRecommendation creates a recommendation to resize Diego cells
func GenerateResizeCellsRecommendation(state InfrastructureState, constrainingResource string) *Recommendation {
	if len(state.Clusters) == 0 {
		return nil
	}

	cluster := state.Clusters[0]

	var newMemory, newCPU int
	var description, impact string

	switch constrainingResource {
	case "Memory":
		// Suggest doubling memory per cell
		newMemory = cluster.DiegoCellMemoryGB * 2
		newCPU = cluster.DiegoCellCPU
		description = fmt.Sprintf("Increase cell memory from %dGB to %dGB",
			cluster.DiegoCellMemoryGB, newMemory)
		impact = fmt.Sprintf("Doubles memory capacity per cell (total: %d GB → %d GB)",
			state.TotalCellMemoryGB, state.TotalCellCount*newMemory)

	case "CPU":
		// Suggest increasing CPU per cell
		newMemory = cluster.DiegoCellMemoryGB
		newCPU = cluster.DiegoCellCPU + 2 // Add 2 vCPUs per cell
		if newCPU < cluster.DiegoCellCPU*2 {
			newCPU = cluster.DiegoCellCPU + 2
		}
		description = fmt.Sprintf("Increase cell vCPU from %d to %d",
			cluster.DiegoCellCPU, newCPU)
		impact = "Increases vCPU per cell for better parallelism"

	default:
		newMemory = cluster.DiegoCellMemoryGB * 2
		newCPU = cluster.DiegoCellCPU * 2
		description = "Resize cells to increase overall capacity"
		impact = "Doubles capacity per cell"
	}

	return &Recommendation{
		Type:            RecommendationResizeCells,
		Priority:        2,
		Title:           "Resize Diego Cells",
		Description:     description,
		Impact:          impact,
		ImpactLevel:     "medium",
		Resource:        constrainingResource,
		NewCellMemoryGB: newMemory,
		NewCellCPU:      newCPU,
	}
}

// GenerateAddHostsRecommendation creates a recommendation to add physical hosts
func GenerateAddHostsRecommendation(state InfrastructureState, constrainingResource string) *Recommendation {
	if len(state.Clusters) == 0 {
		return nil
	}

	cluster := state.Clusters[0]

	// Calculate hosts to add to reduce utilization to ~70%
	var hostsToAdd int
	var impact string

	switch constrainingResource {
	case "Memory":
		if cluster.MemoryGBPerHost == 0 {
			return nil
		}
		currentHostUtil := state.HostMemoryUtilizationPercent
		if currentHostUtil <= 70 {
			hostsToAdd = 1 // Minimum for HA improvement
		} else {
			// Calculate hosts needed to get to 70% utilization
			totalCellMem := state.TotalCellMemoryGB
			targetHostMem := float64(totalCellMem) / 0.70
			neededHosts := int(targetHostMem/float64(cluster.MemoryGBPerHost)) + 1
			hostsToAdd = neededHosts - state.TotalHostCount
			if hostsToAdd < 1 {
				hostsToAdd = 1
			}
		}
		memoryGain := hostsToAdd * cluster.MemoryGBPerHost
		impact = fmt.Sprintf("Adds %d GB of physical memory capacity and improves HA", memoryGain)

	case "CPU":
		if cluster.CPUThreadsPerHost == 0 {
			return nil
		}
		currentCPURatio := state.VCPURatio
		if currentCPURatio <= 4 {
			hostsToAdd = 1 // Minimum for HA
		} else {
			// Calculate hosts needed to get vCPU:pCPU to 4:1
			targetCores := float64(state.TotalVCPUs) / 4.0
			neededHosts := int(targetCores/float64(cluster.CPUThreadsPerHost)) + 1
			hostsToAdd = neededHosts - state.TotalHostCount
			if hostsToAdd < 1 {
				hostsToAdd = 1
			}
		}
		cpuGain := hostsToAdd * cluster.CPUThreadsPerHost
		impact = fmt.Sprintf("Adds %d physical CPU cores, reducing vCPU overcommit", cpuGain)

	default:
		hostsToAdd = 1
		impact = "Adds physical capacity and improves HA resilience"
	}

	return &Recommendation{
		Type:        RecommendationAddHosts,
		Priority:    3,
		Title:       "Add Physical Host",
		Description: fmt.Sprintf("Add %d physical host(s) to your cluster", hostsToAdd),
		Impact:      impact,
		ImpactLevel: "low",
		Resource:    constrainingResource,
		HostsToAdd:  hostsToAdd,
	}
}

// GenerateRecommendations creates a prioritized list of recommendations
func GenerateRecommendations(state InfrastructureState) []Recommendation {
	// First, analyze bottleneck to identify constraining resource
	analysis := AnalyzeBottleneck(state)
	constrainingResource := analysis.ConstrainingResource

	if constrainingResource == "" && len(analysis.Resources) > 0 {
		constrainingResource = analysis.Resources[0].Name
	}

	var recs []Recommendation

	// Generate recommendations for the constraining resource first
	if rec := GenerateAddCellsRecommendation(state, constrainingResource); rec != nil {
		recs = append(recs, *rec)
	}
	if rec := GenerateResizeCellsRecommendation(state, constrainingResource); rec != nil {
		recs = append(recs, *rec)
	}
	if rec := GenerateAddHostsRecommendation(state, constrainingResource); rec != nil {
		recs = append(recs, *rec)
	}

	// Sort by priority
	sort.SliceStable(recs, func(i, j int) bool {
		return recs[i].Priority < recs[j].Priority
	})

	return recs
}
