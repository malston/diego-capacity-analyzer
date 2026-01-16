// ABOUTME: Integration tests for TUI app
// ABOUTME: Tests component wiring and state transitions

package tui

import (
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestAppInitialState(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)

	if app.screen != ScreenMenu {
		t.Errorf("expected initial screen to be ScreenMenu, got %d", app.screen)
	}
}

func TestScreenConstants(t *testing.T) {
	// Verify screen constants are defined correctly
	if ScreenMenu != 0 {
		t.Errorf("expected ScreenMenu to be 0, got %d", ScreenMenu)
	}
	if ScreenDashboard != 1 {
		t.Errorf("expected ScreenDashboard to be 1, got %d", ScreenDashboard)
	}
	if ScreenComparison != 2 {
		t.Errorf("expected ScreenComparison to be 2, got %d", ScreenComparison)
	}
}

func TestAppInfraLoadedMsg(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)
	app.width = 100
	app.height = 40

	// Simulate receiving infrastructure data
	infra := &client.InfrastructureState{
		Name:           "test-infra",
		TotalHostCount: 4,
		TotalCellCount: 12,
	}

	msg := infraLoadedMsg{infra: infra, err: nil}
	updatedApp, _ := app.Update(msg)

	result := updatedApp.(*App)
	if result.screen != ScreenDashboard {
		t.Errorf("expected screen to be ScreenDashboard after infra loaded, got %d", result.screen)
	}
	if result.infra != infra {
		t.Error("expected infra to be set")
	}
	if result.dashboard == nil {
		t.Error("expected dashboard to be created")
	}
}

func TestAppScenarioComparedMsg(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)
	app.width = 100
	app.height = 40
	app.screen = ScreenDashboard

	// Simulate receiving comparison result
	comparison := &client.ScenarioComparison{
		Current:  client.ScenarioResult{CellCount: 10},
		Proposed: client.ScenarioResult{CellCount: 12},
	}

	msg := scenarioComparedMsg{result: comparison, err: nil}
	updatedApp, _ := app.Update(msg)

	result := updatedApp.(*App)
	if result.screen != ScreenComparison {
		t.Errorf("expected screen to be ScreenComparison after scenario compared, got %d", result.screen)
	}
	if result.comparison != comparison {
		t.Error("expected comparison to be set")
	}
	if result.compView == nil {
		t.Error("expected comparison view to be created")
	}
}

func TestAppViewReturnsContent(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)
	app.width = 100
	app.height = 40

	// Menu view should contain the title
	view := app.View()
	if !strings.Contains(view, "Diego Capacity Analyzer") {
		t.Error("expected menu view to contain 'Diego Capacity Analyzer'")
	}

	// Dashboard view should contain actions pane with keybindings
	app.screen = ScreenDashboard
	view = app.View()
	if !strings.Contains(view, "Actions") {
		t.Error("expected dashboard view to contain 'Actions'")
	}
	if !strings.Contains(view, "[w]") {
		t.Error("expected dashboard view to contain '[w]' keybinding")
	}

	// Comparison view should contain back navigation help
	app.screen = ScreenComparison
	view = app.View()
	if !strings.Contains(view, "[b]") {
		t.Error("expected comparison view to contain '[b]' keybinding")
	}
	if !strings.Contains(view, "Back") {
		t.Error("expected comparison view to contain 'Back'")
	}
}
