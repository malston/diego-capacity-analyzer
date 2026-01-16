// ABOUTME: Comparison view showing current vs proposed scenario results
// ABOUTME: Displays deltas, warnings, and recommendations

package comparison

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
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
	sb.WriteString(styles.Title.Render("Scenario Comparison"))
	sb.WriteString("\n\n")

	// Side by side metrics
	colWidth := (c.width - 4) / 2

	currentCol := c.renderScenario("Current", &c.result.Current, colWidth)
	proposedCol := c.renderScenario("Proposed", &c.result.Proposed, colWidth)

	// Join columns
	currentLines := strings.Split(currentCol, "\n")
	proposedLines := strings.Split(proposedCol, "\n")
	maxLines := len(currentLines)
	if len(proposedLines) > maxLines {
		maxLines = len(proposedLines)
	}

	for i := 0; i < maxLines; i++ {
		left := ""
		right := ""
		if i < len(currentLines) {
			left = currentLines[i]
		}
		if i < len(proposedLines) {
			right = proposedLines[i]
		}
		sb.WriteString(fmt.Sprintf("%-*s  %s\n", colWidth, left, right))
	}

	// Delta section
	sb.WriteString("\n")
	sb.WriteString(styles.Subtitle.Render("Changes"))
	sb.WriteString("\n")

	delta := c.result.Delta
	changeStyle := styles.StatusOK
	changePrefix := "+"
	if delta.CapacityChangeGB < 0 {
		changeStyle = styles.StatusCritical
		changePrefix = ""
	}
	sb.WriteString(fmt.Sprintf("  Capacity: %s\n", changeStyle.Render(fmt.Sprintf("%s%d GB", changePrefix, delta.CapacityChangeGB))))

	utilStyle := styles.StatusOK
	if delta.UtilizationChangePct > 0 {
		utilStyle = styles.StatusWarning
	}
	sb.WriteString(fmt.Sprintf("  Utilization: %s\n", utilStyle.Render(fmt.Sprintf("%+.1f%%", delta.UtilizationChangePct))))

	// Warnings
	if len(c.result.Warnings) > 0 {
		sb.WriteString("\n")
		sb.WriteString(styles.StatusWarning.Render("Warnings"))
		sb.WriteString("\n")
		for _, w := range c.result.Warnings {
			icon := "!"
			warnStyle := styles.StatusWarning
			if w.Severity == "critical" {
				icon = "X"
				warnStyle = styles.StatusCritical
			}
			sb.WriteString(fmt.Sprintf("  %s %s\n", warnStyle.Render(icon), w.Message))
		}
	}

	return lipgloss.NewStyle().Width(c.width).Render(sb.String())
}

func (c *Comparison) renderScenario(title string, s *client.ScenarioResult, width int) string {
	var sb strings.Builder
	sb.WriteString(styles.Subtitle.Render(title))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Cells: %d x %dGB\n", s.CellCount, s.CellMemoryGB))
	sb.WriteString(fmt.Sprintf("Capacity: %d GB\n", s.AppCapacityGB))
	sb.WriteString(fmt.Sprintf("Utilization: %.1f%%\n", s.UtilizationPct))
	if s.VCPURatio > 0 {
		sb.WriteString(fmt.Sprintf("vCPU Ratio: %.1f:1\n", s.VCPURatio))
	}
	return sb.String()
}
