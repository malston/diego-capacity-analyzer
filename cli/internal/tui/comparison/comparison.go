// ABOUTME: Comparison view showing current vs proposed scenario results
// ABOUTME: Uses visual panels, delta badges, and styled warnings

package comparison

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/widgets"
)

// Comparison displays scenario comparison results
type Comparison struct {
	result *client.ScenarioComparison
	width  int
}

// New creates a new comparison view
func New(result *client.ScenarioComparison, width int) *Comparison {
	return &Comparison{
		result: result,
		width:  width,
	}
}

// View renders the comparison
func (c *Comparison) View() string {
	if c.result == nil {
		return "No comparison data"
	}

	var sb strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	sb.WriteString(titleStyle.Render(fmt.Sprintf("%s Scenario Comparison", icons.Chart.String())))
	sb.WriteString("\n\n")

	// Side by side panels
	colWidth := (c.width - 6) / 2
	if colWidth < 30 {
		colWidth = 30
	}

	currentPanel := c.renderScenarioPanel("Current", icons.Server, &c.result.Current, colWidth)
	proposedPanel := c.renderScenarioPanel("Proposed", icons.TrendUp, &c.result.Proposed, colWidth)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, currentPanel, "  ", proposedPanel))
	sb.WriteString("\n\n")

	// Impact summary panel
	impactPanel := c.renderImpactPanel()
	sb.WriteString(impactPanel)
	sb.WriteString("\n\n")

	// Warnings panel
	if len(c.result.Warnings) > 0 {
		warningsPanel := c.renderWarningsPanel()
		sb.WriteString(warningsPanel)
	}

	return lipgloss.NewStyle().Width(c.width).Render(sb.String())
}

func (c *Comparison) renderScenarioPanel(title string, icon icons.Icon, s *client.ScenarioResult, width int) string {
	var sb strings.Builder

	// Cell info
	sb.WriteString(fmt.Sprintf("Cells:     %d\n", s.CellCount))
	sb.WriteString(fmt.Sprintf("Memory:    %d GB each\n", s.CellMemoryGB))
	sb.WriteString(fmt.Sprintf("Total:     %d GB\n", s.AppCapacityGB))
	sb.WriteString("\n")

	// Utilization with progress bar
	barConfig := widgets.DefaultProgressBarConfig()
	barConfig.Width = width - 8
	bar := widgets.ProgressBarWithLabel(s.UtilizationPct, barConfig, true)
	sb.WriteString(fmt.Sprintf("Utilization\n%s", bar))

	// vCPU ratio if available
	if s.VCPURatio > 0 {
		sb.WriteString(fmt.Sprintf("\nvCPU:      %.1f:1", s.VCPURatio))
	}

	return c.buildPanel(title, icon, sb.String(), width)
}

func (c *Comparison) renderImpactPanel() string {
	var sb strings.Builder
	delta := c.result.Delta

	// Capacity change
	capacityChange := delta.CapacityChangeGB
	// Calculate percentage change from current capacity
	var capacityPct float64
	if c.result.Current.AppCapacityGB > 0 {
		capacityPct = (float64(capacityChange) / float64(c.result.Current.AppCapacityGB)) * 100
	}

	var capacityColor lipgloss.Color
	var capacityPrefix string
	if capacityChange > 0 {
		capacityColor = styles.DeltaPositive
		capacityPrefix = "+"
	} else if capacityChange < 0 {
		capacityColor = styles.DeltaNegative
		capacityPrefix = ""
	} else {
		capacityColor = styles.DeltaNeutral
		capacityPrefix = ""
	}

	capacityStyle := lipgloss.NewStyle().Foreground(capacityColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Capacity:      %s\n",
		capacityStyle.Render(fmt.Sprintf("%s%d GB (%s%.0f%%)", capacityPrefix, capacityChange, capacityPrefix, capacityPct))))

	// Utilization change (inverted - decrease is good)
	utilChange := delta.UtilizationChangePct
	var utilColor lipgloss.Color
	var utilArrow string
	if utilChange < 0 {
		utilColor = styles.DeltaPositive // Decrease is good
		utilArrow = icons.TrendDown.String()
	} else if utilChange > 0 {
		utilColor = styles.DeltaNegative // Increase is concerning
		utilArrow = icons.TrendUp.String()
	} else {
		utilColor = styles.DeltaNeutral
		utilArrow = "→"
	}

	utilStyle := lipgloss.NewStyle().Foreground(utilColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Utilization:   %.1f%% → %.1f%%  %s %s\n",
		c.result.Current.UtilizationPct,
		c.result.Proposed.UtilizationPct,
		utilStyle.Render(fmt.Sprintf("(%+.1f%%)", utilChange)),
		lipgloss.NewStyle().Foreground(utilColor).Render(utilArrow)))

	// Headroom change
	currentHeadroom := 100 - c.result.Current.UtilizationPct
	proposedHeadroom := 100 - c.result.Proposed.UtilizationPct
	headroomChange := proposedHeadroom - currentHeadroom

	var headroomColor lipgloss.Color
	if headroomChange > 0 {
		headroomColor = styles.DeltaPositive
	} else if headroomChange < 0 {
		headroomColor = styles.DeltaNegative
	} else {
		headroomColor = styles.DeltaNeutral
	}

	headroomStyle := lipgloss.NewStyle().Foreground(headroomColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Headroom:      %s",
		headroomStyle.Render(fmt.Sprintf("%+.1f%% available", headroomChange))))

	return c.buildPanel("Impact Summary", icons.TrendUp, sb.String(), c.width-4)
}

func (c *Comparison) renderWarningsPanel() string {
	var sb strings.Builder

	for i, w := range c.result.Warnings {
		var status widgets.StatusLevel
		if w.Severity == "critical" {
			status = widgets.StatusCritical
		} else {
			status = widgets.StatusWarning
		}

		sb.WriteString(widgets.StatusText(w.Message, status))
		if i < len(c.result.Warnings)-1 {
			sb.WriteString("\n")
		}
	}

	return c.buildPanel("Warnings", icons.Warning, sb.String(), c.width-4)
}

func (c *Comparison) buildPanel(title string, icon icons.Icon, content string, width int) string {
	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary)

	innerWidth := width - 4
	fullTitle := fmt.Sprintf("%s %s", icon.String(), title)
	titleWidth := lipgloss.Width(fullTitle)
	styledTitle := titleStyle.Render(fullTitle)

	// Top border with title - use lipgloss.Width for accurate width
	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", fillWidth) + "┐"

	// Content lines with side borders
	lines := strings.Split(content, "\n")
	var contentLines []string
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
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
