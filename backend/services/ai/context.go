// ABOUTME: Serializes in-memory capacity data into annotated markdown for the LLM
// ABOUTME: Pure function accepting only model types -- no config, clients, or services

package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const maxApps = 10

// ContextInput bundles all data sources the context builder can serialize.
// Nil pointers indicate absent/unconfigured data sources.
type ContextInput struct {
	Dashboard *models.DashboardResponse
	Infra     *models.InfrastructureState
	Scenario  *models.ScenarioComparison

	// Data source availability flags (distinct from nil data --
	// a source can be configured but have no data yet)
	BOSHConfigured    bool
	VSphereConfigured bool
	LogCacheAvailable bool
}

// BuildContext serializes capacity data into annotated markdown for the LLM.
func BuildContext(input ContextInput) string {
	var b strings.Builder
	writeDataSourceSummary(&b, input)
	writeInfrastructure(&b, input.Infra, input.VSphereConfigured)
	writeDiegoCells(&b, input.Dashboard)
	writeApps(&b, input.Dashboard)
	writeScenario(&b, input.Scenario)
	return b.String()
}

// utilizationFlag returns an inline threshold annotation for utilization percentages.
func utilizationFlag(pct float64) string {
	if pct > 90 {
		return " [CRITICAL]"
	}
	if pct > 80 {
		return " [HIGH]"
	}
	return ""
}

// vcpuRatioFlag returns an inline threshold annotation for vCPU:pCPU ratios.
func vcpuRatioFlag(ratio float64) string {
	if ratio > 8 {
		return " [CRITICAL]"
	}
	if ratio > 4 {
		return " [HIGH]"
	}
	return ""
}

func writeDataSourceSummary(b *strings.Builder, input ContextInput) {
	b.WriteString("## Data Sources\n")

	// CF API
	cfStatus := "available"
	if input.Dashboard == nil || (len(input.Dashboard.Apps) == 0 && len(input.Dashboard.Cells) == 0) {
		cfStatus = "UNAVAILABLE"
	}
	fmt.Fprintf(b, "- CF API: %s\n", cfStatus)

	// BOSH
	if !input.BOSHConfigured {
		b.WriteString("- BOSH: NOT CONFIGURED\n")
	} else if input.Dashboard == nil || !input.Dashboard.Metadata.BOSHAvailable {
		b.WriteString("- BOSH: UNAVAILABLE\n")
	} else {
		b.WriteString("- BOSH: available\n")
	}

	// vSphere
	if !input.VSphereConfigured {
		b.WriteString("- vSphere: NOT CONFIGURED\n")
	} else if input.Infra == nil {
		b.WriteString("- vSphere: UNAVAILABLE\n")
	} else {
		b.WriteString("- vSphere: available\n")
	}

	// Log Cache
	if input.LogCacheAvailable {
		b.WriteString("- Log Cache: available\n")
	} else {
		b.WriteString("- Log Cache: not available\n")
	}

	b.WriteString("\n")
}

func writeInfrastructure(b *strings.Builder, infra *models.InfrastructureState, configured bool) {
	b.WriteString("## Infrastructure\n")

	if !configured {
		b.WriteString("vSphere data: NOT CONFIGURED\n\n")
		return
	}
	if infra == nil {
		b.WriteString("vSphere data: UNAVAILABLE\n\n")
		return
	}

	b.WriteString("Physical hosts and clusters backing Diego cells.\n\n")

	for _, c := range infra.Clusters {
		fmt.Fprintf(b, "**%s**: %d hosts, %d GB memory, %d GB HA-usable, HA: %s",
			c.Name, c.HostCount, c.MemoryGB, c.HAUsableMemoryGB, c.HAStatus)
		if c.HAHostFailuresSurvived > 0 {
			fmt.Fprintf(b, " (survives %d host failure(s))", c.HAHostFailuresSurvived)
		}
		b.WriteString("\n")

		memUtil := c.HostMemoryUtilizationPercent
		fmt.Fprintf(b, "- Host memory utilization: %.1f%%%s\n", memUtil, utilizationFlag(memUtil))

		if c.VCPURatio > 0 {
			fmt.Fprintf(b, "- vCPU:pCPU ratio: %.1f:1%s\n", c.VCPURatio, vcpuRatioFlag(c.VCPURatio))
		}
	}

	b.WriteString("\n")
	fmt.Fprintf(b, "**Totals**: %d hosts, %d GB memory, HA: %s\n\n",
		infra.TotalHostCount, infra.TotalMemoryGB, infra.HAStatus)
}

type segmentSummary struct {
	Name          string
	CellCount     int
	TotalMemoryMB int
	AllocatedMB   int
	UsedMB        int
}

func writeDiegoCells(b *strings.Builder, dashboard *models.DashboardResponse) {
	b.WriteString("## Diego Cells\n")

	if dashboard == nil {
		b.WriteString("Cell data: UNAVAILABLE\n\n")
		return
	}

	if len(dashboard.Cells) == 0 {
		b.WriteString("No Diego cells reported.\n\n")
		return
	}

	b.WriteString("Diego cell capacity grouped by isolation segment.\n\n")

	segments := make(map[string]*segmentSummary)
	var segOrder []string

	for _, cell := range dashboard.Cells {
		seg := cell.IsolationSegment
		if seg == "" {
			seg = "shared"
		}
		s, ok := segments[seg]
		if !ok {
			s = &segmentSummary{Name: seg}
			segments[seg] = s
			segOrder = append(segOrder, seg)
		}
		s.CellCount++
		s.TotalMemoryMB += cell.MemoryMB
		s.AllocatedMB += cell.AllocatedMB
		s.UsedMB += cell.UsedMB
	}

	// Sort: "shared" first, then alphabetical
	sort.SliceStable(segOrder, func(i, j int) bool {
		if segOrder[i] == "shared" {
			return true
		}
		if segOrder[j] == "shared" {
			return false
		}
		return segOrder[i] < segOrder[j]
	})

	var totalCells, totalMemMB, totalAllocMB, totalUsedMB int
	for _, name := range segOrder {
		s := segments[name]
		totalCells += s.CellCount
		totalMemMB += s.TotalMemoryMB
		totalAllocMB += s.AllocatedMB
		totalUsedMB += s.UsedMB

		var utilPct float64
		if s.TotalMemoryMB > 0 {
			utilPct = float64(s.AllocatedMB) / float64(s.TotalMemoryMB) * 100
		}

		fmt.Fprintf(b, "**%s**: %d cells, %d MB total, %d MB allocated (%.1f%%%s)",
			s.Name, s.CellCount, s.TotalMemoryMB, s.AllocatedMB, utilPct, utilizationFlag(utilPct))

		if s.UsedMB > 0 {
			fmt.Fprintf(b, ", %d MB used", s.UsedMB)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	var overallUtil float64
	if totalMemMB > 0 {
		overallUtil = float64(totalAllocMB) / float64(totalMemMB) * 100
	}
	fmt.Fprintf(b, "**Totals**: %d cells, %d MB memory, %.1f%% utilization%s\n\n",
		totalCells, totalMemMB, overallUtil, utilizationFlag(overallUtil))
}

func writeApps(b *strings.Builder, dashboard *models.DashboardResponse) {
	b.WriteString("## Apps\n")

	if dashboard == nil || len(dashboard.Apps) == 0 {
		b.WriteString("App data: UNAVAILABLE\n\n")
		return
	}

	b.WriteString("Top applications by memory allocation.\n\n")

	// Sort apps by RequestedMB descending
	sorted := make([]models.App, len(dashboard.Apps))
	copy(sorted, dashboard.Apps)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].RequestedMB > sorted[j].RequestedMB
	})

	total := len(sorted)
	show := min(maxApps, total)
	top := sorted[:show]

	for _, app := range top {
		line := fmt.Sprintf("- %s: %d instances, %d MB requested", app.Name, app.Instances, app.RequestedMB)
		if app.ActualMB > 0 {
			line += fmt.Sprintf(", %d MB actual", app.ActualMB)
		}
		if app.IsolationSegment != "" {
			line += fmt.Sprintf(" [%s]", app.IsolationSegment)
		}
		b.WriteString(line + "\n")
	}

	if total > show {
		fmt.Fprintf(b, "\nShowing %d of %d applications.\n", show, total)
	}

	// Check for apps missing actual memory data
	var withActual, withoutActual int
	for _, app := range dashboard.Apps {
		if app.ActualMB > 0 {
			withActual++
		} else {
			withoutActual++
		}
	}
	if withActual > 0 && withoutActual > 0 {
		fmt.Fprintf(b, "\nNote: Memory usage unavailable for %d of %d apps.\n", withoutActual, total)
	}

	b.WriteString("\n")
}

func writeScenario(b *strings.Builder, scenario *models.ScenarioComparison) {
	b.WriteString("## Scenario Comparison\n")

	if scenario == nil {
		b.WriteString("No scenario comparison has been run.\n\n")
		return
	}

	b.WriteString("Current vs proposed capacity changes.\n\n")

	cur := scenario.Current
	pro := scenario.Proposed
	delta := scenario.Delta

	b.WriteString("| Metric | Current | Proposed | Delta |\n")
	b.WriteString("|--------|---------|----------|-------|\n")
	fmt.Fprintf(b, "| Cells | %d | %d | %+d |\n",
		cur.CellCount, pro.CellCount, pro.CellCount-cur.CellCount)
	fmt.Fprintf(b, "| Cell Size | %s | %s | - |\n",
		cur.CellSize(), pro.CellSize())
	fmt.Fprintf(b, "| App Capacity | %d GB | %d GB | %+d GB |\n",
		cur.AppCapacityGB, pro.AppCapacityGB, delta.CapacityChangeGB)
	fmt.Fprintf(b, "| Utilization | %.1f%% | %.1f%% | %+.1f%% |\n",
		cur.UtilizationPct, pro.UtilizationPct, delta.UtilizationChangePct)

	if cur.DiskCapacityGB > 0 || pro.DiskCapacityGB > 0 {
		fmt.Fprintf(b, "| Disk Capacity | %d GB | %d GB | %+d GB |\n",
			cur.DiskCapacityGB, pro.DiskCapacityGB, delta.DiskCapacityChangeGB)
	}

	if len(scenario.Warnings) > 0 {
		b.WriteString("\n")
		for _, w := range scenario.Warnings {
			fmt.Fprintf(b, "- %s: %s\n", w.Severity, w.Message)
		}
	}

	b.WriteString("\n")
}
