// ABOUTME: Dashboard component displaying live infrastructure metrics
// ABOUTME: Shows cluster counts, utilization, and HA status in left pane

package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
)

// Dashboard displays infrastructure metrics
type Dashboard struct {
	infra  *client.InfrastructureState
	width  int
	height int
}

// New creates a new dashboard with infrastructure data
func New(infra *client.InfrastructureState, width, height int) *Dashboard {
	return &Dashboard{
		infra:  infra,
		width:  width,
		height: height,
	}
}

// Update refreshes dashboard with new infrastructure data
func (d *Dashboard) Update(infra *client.InfrastructureState) {
	d.infra = infra
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

	// Title
	sb.WriteString(styles.Title.Render("Current Infrastructure"))
	sb.WriteString("\n")
	sb.WriteString(styles.Subtitle.Render(d.infra.Name))
	sb.WriteString("\n\n")

	// Cluster info
	sb.WriteString(fmt.Sprintf("Clusters: %d\n", len(d.infra.Clusters)))
	sb.WriteString(fmt.Sprintf("Hosts: %d\n", d.infra.TotalHostCount))
	sb.WriteString(fmt.Sprintf("Diego Cells: %d\n", d.infra.TotalCellCount))
	sb.WriteString("\n")

	// Memory utilization
	sb.WriteString("Memory Utilization\n")
	sb.WriteString(styles.ProgressBar(d.infra.HostMemoryUtilizationPercent, 20))
	sb.WriteString(fmt.Sprintf(" %.1f%%\n", d.infra.HostMemoryUtilizationPercent))
	sb.WriteString("\n")

	// CPU utilization
	sb.WriteString("CPU Utilization\n")
	sb.WriteString(styles.ProgressBar(d.infra.HostCPUUtilizationPercent, 20))
	sb.WriteString(fmt.Sprintf(" %.1f%%\n", d.infra.HostCPUUtilizationPercent))
	sb.WriteString("\n")

	// HA Status
	haStyle := styles.StatusOK
	haIcon := "+"
	if d.infra.HAStatus != "ok" {
		haStyle = styles.StatusCritical
		haIcon = "x"
	}
	sb.WriteString(fmt.Sprintf("HA Status: %s\n", haStyle.Render(haIcon+" "+strings.ToUpper(d.infra.HAStatus))))
	sb.WriteString(fmt.Sprintf("  Can survive %d host failure(s)\n", d.infra.HAMinHostFailuresSurvived))

	// vCPU Ratio
	if d.infra.VCPURatio > 0 {
		sb.WriteString("\n")
		riskStyle := styles.StatusOK
		if d.infra.CPURiskLevel == "moderate" {
			riskStyle = styles.StatusWarning
		} else if d.infra.CPURiskLevel == "aggressive" {
			riskStyle = styles.StatusCritical
		}
		sb.WriteString(fmt.Sprintf("vCPU Ratio: %s\n", riskStyle.Render(fmt.Sprintf("%.1f:1", d.infra.VCPURatio))))
	}

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Render(sb.String())
}
