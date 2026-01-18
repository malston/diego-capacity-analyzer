// ABOUTME: Dashboard component displaying live infrastructure metrics
// ABOUTME: Uses compact metric blocks with sparklines and visual indicators

package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/widgets"
)

// Dashboard displays infrastructure metrics
type Dashboard struct {
	infra         *client.InfrastructureState
	width         int
	height        int
	historyMemory []float64 // Historical memory values for sparkline
	historyCPU    []float64 // Historical CPU ratio values for sparkline
}

// New creates a new dashboard with infrastructure data
func New(infra *client.InfrastructureState, width, height int) *Dashboard {
	d := &Dashboard{
		infra:         infra,
		width:         width,
		height:        height,
		historyMemory: make([]float64, 0, 8),
		historyCPU:    make([]float64, 0, 8),
	}
	if infra != nil {
		d.recordHistory(infra)
	}
	return d
}

// Update refreshes dashboard with new infrastructure data
func (d *Dashboard) Update(infra *client.InfrastructureState) {
	d.infra = infra
	if infra != nil {
		d.recordHistory(infra)
	}
}

// recordHistory adds current values to history for sparklines
func (d *Dashboard) recordHistory(infra *client.InfrastructureState) {
	d.historyMemory = append(d.historyMemory, infra.HostMemoryUtilizationPercent)
	if len(d.historyMemory) > 8 {
		d.historyMemory = d.historyMemory[1:]
	}

	d.historyCPU = append(d.historyCPU, infra.VCPURatio)
	if len(d.historyCPU) > 8 {
		d.historyCPU = d.historyCPU[1:]
	}
}

// SetSize updates the dashboard dimensions
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the dashboard
func (d *Dashboard) View() string {
	if d.infra == nil {
		return styles.Panel.Width(d.width).Render("Loading infrastructure data...")
	}

	var sb strings.Builder

	// Title with infrastructure name
	titleIcon := icons.App.String()
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.Muted)

	sb.WriteString(titleStyle.Render(fmt.Sprintf("%s Current Infrastructure", titleIcon)))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render(d.infra.Name))
	sb.WriteString("\n\n")

	// Row 1: Key metrics in compact blocks
	row1 := d.renderMetricsRow()
	sb.WriteString(row1)
	sb.WriteString("\n\n")

	// Row 2: Capacity and HA panels
	row2 := d.renderCapacityRow()
	sb.WriteString(row2)

	// Only constrain width - let height flow naturally so header/footer aren't pushed off
	return lipgloss.NewStyle().
		Width(d.width).
		Render(sb.String())
}

// renderMetricsRow renders the top row of metric blocks
func (d *Dashboard) renderMetricsRow() string {
	// Calculate block width based on available space
	// Available content width = d.width - some margin for internal spacing
	// For 2 blocks per row with 2-char gap: blockWidth = (contentWidth - 2) / 2
	contentWidth := d.width - 4 // margin for internal content
	blockWidth := (contentWidth - 2) / 2
	if blockWidth < 18 {
		blockWidth = 18 // minimum readable width
	}
	if blockWidth > 24 {
		blockWidth = 24 // maximum for aesthetic
	}

	config := widgets.DefaultMetricBlockConfig()
	config.Width = blockWidth

	// Memory block with bar
	usedMemoryGB := float64(d.infra.TotalMemoryGB) * (d.infra.HostMemoryUtilizationPercent / 100)
	memoryBlock := widgets.MetricBlockWithBar(
		icons.Memory,
		"Memory",
		d.infra.HostMemoryUtilizationPercent,
		fmt.Sprintf("%.0f/%d GB", usedMemoryGB, d.infra.TotalMemoryGB),
		config,
	)

	// CPU block with ratio and risk (use backend-provided risk level)
	riskLabel := d.infra.CPURiskLevel
	if riskLabel == "" {
		riskLabel = "unknown"
	}
	// Title-case the risk label for display
	cpuSubtitle := strings.Title(riskLabel)
	cpuValue := fmt.Sprintf("%.1f:1", d.infra.VCPURatio)
	cpuConfig := config
	if riskLabel == "moderate" {
		cpuConfig.ValueColor = styles.Warning
	} else if riskLabel == "aggressive" {
		cpuConfig.ValueColor = styles.Danger
	}

	cpuBlock := widgets.MetricBlockWithSparkline(
		icons.CPU,
		"CPU Ratio",
		cpuValue,
		d.historyCPU,
		cpuSubtitle,
		cpuConfig,
	)

	// Cluster count block
	clusterBlock := widgets.CountBlock(
		icons.Cluster,
		"Clusters",
		len(d.infra.Clusters),
		"clusters",
		config,
	)

	// Host count block
	hostBlock := widgets.CountBlock(
		icons.Host,
		"Hosts",
		d.infra.TotalHostCount,
		"hosts",
		config,
	)

	// Arrange in 2 rows of 2 blocks each
	row1 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		memoryBlock,
		"  ",
		cpuBlock,
	)
	row2 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		clusterBlock,
		"  ",
		hostBlock,
	)

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// renderCapacityRow renders the bottom row with capacity and HA panels
func (d *Dashboard) renderCapacityRow() string {
	// Account for the fact that the outer ActivePanel style adds borders and padding
	// Available content width is roughly d.width - 6 (2 border + 4 padding)
	contentWidth := d.width - 6
	if contentWidth < 40 {
		contentWidth = 40 // minimum for any reasonable layout
	}

	// Use full content width for each panel, stacked vertically
	panelWidth := contentWidth - 2 // leave margin

	// N-1 Capacity panel
	capacityPanel := d.renderCapacityPanel(panelWidth)

	// HA Status panel
	haPanel := d.renderHAPanel(panelWidth)

	// Stack vertically for better fit
	return lipgloss.JoinVertical(lipgloss.Left, capacityPanel, haPanel)
}

// renderCapacityPanel renders the N-1 capacity information
func (d *Dashboard) renderCapacityPanel(width int) string {
	var sb strings.Builder

	// Utilization with status
	util := d.infra.HostMemoryUtilizationPercent
	status := widgets.StatusFromPercent(util, 80, 95)
	statusIcon := widgets.StatusIcon(status)

	sb.WriteString(fmt.Sprintf("Utilization: %.1f%% %s\n", util, statusIcon))

	// Progress bar with zones
	barConfig := widgets.DefaultProgressBarConfig()
	barConfig.Width = width - 6
	barConfig.ShowZones = true
	bar := widgets.ProgressBarWithLabel(util, barConfig, false)
	sb.WriteString(bar)
	sb.WriteString("\n")

	// Headroom calculation
	headroom := 100 - util
	usedGB := float64(d.infra.TotalMemoryGB) * (util / 100)
	availableGB := float64(d.infra.TotalMemoryGB) - usedGB
	headroomStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	sb.WriteString(headroomStyle.Render(fmt.Sprintf("Headroom: %.0f%% (%.0f GB available)", headroom, availableGB)))

	// Build panel with border
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary)
	title := fmt.Sprintf("%s N-1 Capacity", icons.Gauge.String())

	innerContent := sb.String()
	innerWidth := width - 4

	panel := d.buildPanel(titleStyle.Render(title), innerContent, innerWidth)
	return panel
}

// renderHAPanel renders the HA status information
func (d *Dashboard) renderHAPanel(width int) string {
	var sb strings.Builder

	// HA Status
	var haStatus widgets.StatusLevel
	var haMessage string

	if d.infra.HAStatus == "ok" {
		haStatus = widgets.StatusOK
		haMessage = fmt.Sprintf("Can survive %d host failure(s)", d.infra.HAMinHostFailuresSurvived)
	} else {
		haStatus = widgets.StatusCritical
		haMessage = "Cannot survive host failure"
	}

	sb.WriteString(widgets.StatusText(haMessage, haStatus))
	sb.WriteString("\n")

	// Host info
	infoStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Current: %d hosts", d.infra.TotalHostCount)))
	sb.WriteString("\n")

	// Cells per host
	cellsPerHost := 0
	if d.infra.TotalHostCount > 0 {
		cellsPerHost = d.infra.TotalCellCount / d.infra.TotalHostCount
	}
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Diego cells: %d (%d per host avg)", d.infra.TotalCellCount, cellsPerHost)))

	// Build panel with border
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary)
	title := fmt.Sprintf("%s HA Status", icons.Shield.String())

	innerContent := sb.String()
	innerWidth := width - 4

	panel := d.buildPanel(titleStyle.Render(title), innerContent, innerWidth)
	return panel
}

// buildPanel creates a bordered panel with title
func (d *Dashboard) buildPanel(title, content string, innerWidth int) string {
	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)

	// Top border with title - use lipgloss.Width for styled title
	titleWidth := lipgloss.Width(title)
	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + title + " " + strings.Repeat("─", fillWidth) + "┐"

	// Content lines with side borders
	lines := strings.Split(content, "\n")
	var contentLines []string
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerWidth {
			// Truncate line to fit - use lipgloss to handle ANSI codes
			line = lipgloss.NewStyle().MaxWidth(innerWidth).Render(line)
			lineWidth = lipgloss.Width(line)
		}
		padding := max(0, innerWidth-lineWidth)
		contentLines = append(contentLines, "│ "+line+strings.Repeat(" ", padding)+" │")
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", innerWidth+2) + "┘"

	allLines := []string{topBorder}
	allLines = append(allLines, contentLines...)
	allLines = append(allLines, bottomBorder)

	return borderStyle.Render(strings.Join(allLines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
