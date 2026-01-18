// ABOUTME: Test to verify dashboard screen renders with visible header/footer
// ABOUTME: Ensures content doesn't push header/footer off screen

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestDashboardRendersWithHeader(t *testing.T) {
	// Create an app with nil client - that's fine for rendering
	app := New(nil, false, "")

	// Simulate window size
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(*App)

	// Create mock infrastructure data
	infra := &client.InfrastructureState{
		Name:                           "Test Infrastructure",
		TotalMemoryGB:                  2048,
		TotalCellCount:                 30,
		TotalHostCount:                 6,
		HostMemoryUtilizationPercent:   65.5,
		VCPURatio:                      3.2,
		CPURiskLevel:                   "conservative",
		HAStatus:                       "ok",
		HAMinHostFailuresSurvived:      1,
		Clusters: []client.ClusterState{
			{
				Name:              "Cluster-1",
				HostCount:         6,
				DiegoCellMemoryGB: 64,
				DiegoCellCPU:      8,
				DiegoCellDiskGB:   200,
				MemoryGBPerHost:   512,
				CPUCoresPerHost:   32,
			},
		},
	}

	// Simulate infrastructure loaded message
	model, _ = app.Update(infraLoadedMsg{infra: infra})
	app = model.(*App)

	// Verify we're on dashboard screen
	if app.screen != ScreenDashboard {
		t.Fatalf("Expected ScreenDashboard, got %v", app.screen)
	}

	// Get the view
	view := app.View()

	// Analyze the output
	lines := strings.Split(view, "\n")
	t.Logf("Total lines: %d", len(lines))

	// Check header/footer - only first ╭ is header, only last ╰ is footer
	headerLineIdx := -1
	footerLineIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "╭") && headerLineIdx == -1 {
			headerLineIdx = i
		}
		if strings.Contains(line, "╰") {
			footerLineIdx = i // keep updating, last one is the footer
		}
	}

	t.Logf("Header found at line %d, footer at line %d", headerLineIdx, footerLineIdx)

	// Print all lines for debugging
	t.Logf("\n=== All %d lines ===", len(lines))
	for i, line := range lines {
		w := lipgloss.Width(line)
		t.Logf("%2d [w=%3d]: %s", i, w, line)
	}

	if headerLineIdx != 0 {
		t.Errorf("Header should be at line 0, found at %d", headerLineIdx)
	}
	if footerLineIdx != len(lines)-1 && footerLineIdx != len(lines)-2 {
		t.Errorf("Footer should be at last line, found at %d of %d", footerLineIdx, len(lines))
	}
	if headerLineIdx == -1 {
		t.Error("Header not found in dashboard output")
	}
	if footerLineIdx == -1 {
		t.Error("Footer not found in dashboard output")
	}
}
