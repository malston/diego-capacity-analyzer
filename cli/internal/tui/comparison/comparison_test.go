// ABOUTME: Tests for comparison view component
// ABOUTME: Validates current vs proposed scenario display

package comparison

import (
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestComparisonView(t *testing.T) {
	result := &client.ScenarioComparison{
		Current: client.ScenarioResult{
			CellCount:      10,
			CellMemoryGB:   64,
			UtilizationPct: 75.0,
		},
		Proposed: client.ScenarioResult{
			CellCount:      15,
			CellMemoryGB:   64,
			UtilizationPct: 50.0,
		},
		Delta: client.ScenarioDelta{
			CapacityChangeGB:     320,
			UtilizationChangePct: -25.0,
		},
	}

	c := New(result, 80)
	view := c.View()

	if !strings.Contains(view, "Current") {
		t.Error("expected view to contain 'Current'")
	}
	if !strings.Contains(view, "Proposed") {
		t.Error("expected view to contain 'Proposed'")
	}
	if !strings.Contains(view, "320") {
		t.Error("expected view to contain capacity change")
	}
}

func TestComparisonViewNilResult(t *testing.T) {
	c := New(nil, 80)
	view := c.View()

	if !strings.Contains(view, "No comparison data") {
		t.Error("expected view to show 'No comparison data' for nil result")
	}
}

func TestComparisonViewWithWarnings(t *testing.T) {
	result := &client.ScenarioComparison{
		Current: client.ScenarioResult{
			CellCount:      10,
			CellMemoryGB:   64,
			UtilizationPct: 75.0,
		},
		Proposed: client.ScenarioResult{
			CellCount:      5,
			CellMemoryGB:   64,
			UtilizationPct: 95.0,
		},
		Delta: client.ScenarioDelta{
			CapacityChangeGB:     -320,
			UtilizationChangePct: 20.0,
		},
		Warnings: []client.ScenarioWarning{
			{Severity: "warning", Message: "High utilization risk"},
			{Severity: "critical", Message: "Insufficient capacity"},
		},
	}

	c := New(result, 80)
	view := c.View()

	if !strings.Contains(view, "Warnings") {
		t.Error("expected view to contain 'Warnings' section")
	}
	if !strings.Contains(view, "High utilization risk") {
		t.Error("expected view to contain warning message")
	}
	if !strings.Contains(view, "Insufficient capacity") {
		t.Error("expected view to contain critical message")
	}
}

func TestComparisonViewWithVCPURatio(t *testing.T) {
	result := &client.ScenarioComparison{
		Current: client.ScenarioResult{
			CellCount:      10,
			CellMemoryGB:   64,
			UtilizationPct: 75.0,
			VCPURatio:      4.5,
		},
		Proposed: client.ScenarioResult{
			CellCount:      15,
			CellMemoryGB:   64,
			UtilizationPct: 50.0,
			VCPURatio:      6.0,
		},
		Delta: client.ScenarioDelta{
			CapacityChangeGB:     320,
			UtilizationChangePct: -25.0,
		},
	}

	c := New(result, 80)
	view := c.View()

	if !strings.Contains(view, "vCPU Ratio") {
		t.Error("expected view to contain 'vCPU Ratio'")
	}
	if !strings.Contains(view, "4.5") {
		t.Error("expected view to contain current vCPU ratio")
	}
}

func TestComparisonViewNegativeCapacityChange(t *testing.T) {
	result := &client.ScenarioComparison{
		Current: client.ScenarioResult{
			CellCount:      10,
			CellMemoryGB:   64,
			UtilizationPct: 50.0,
		},
		Proposed: client.ScenarioResult{
			CellCount:      5,
			CellMemoryGB:   64,
			UtilizationPct: 90.0,
		},
		Delta: client.ScenarioDelta{
			CapacityChangeGB:     -320,
			UtilizationChangePct: 40.0,
		},
	}

	c := New(result, 80)
	view := c.View()

	// Negative capacity change should appear without double negative
	if !strings.Contains(view, "-320") {
		t.Error("expected view to contain negative capacity change")
	}
}
