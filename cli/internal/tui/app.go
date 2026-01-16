// ABOUTME: Root bubbletea model for the TUI application
// ABOUTME: Manages screen state and routes keyboard input to child components

package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/comparison"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/dashboard"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/menu"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/wizard"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenDashboard
	ScreenComparison
)

// Layout constants
const (
	minTerminalWidth = 80 // Minimum width before using single-column layout
	panelPadding     = 4  // Total horizontal padding from panel borders (2 each side)
)

// infraLoadedMsg is sent when infrastructure data is loaded
type infraLoadedMsg struct {
	infra *client.InfrastructureState
	err   error
}

// scenarioComparedMsg is sent when scenario comparison completes
type scenarioComparedMsg struct {
	result *client.ScenarioComparison
	err    error
}

// App is the root model for the TUI
type App struct {
	client     *client.Client
	screen     Screen
	width      int
	height     int
	err        error
	infra      *client.InfrastructureState
	comparison *client.ScenarioComparison
	dashboard  *dashboard.Dashboard
	compView   *comparison.Comparison
	dataSource menu.DataSource
}

// New creates a new TUI application
func New(apiClient *client.Client) *App {
	return &App{
		client: apiClient,
		screen: ScreenMenu,
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "r":
			if a.screen == ScreenDashboard {
				return a, a.loadInfrastructure()
			}
		case "w":
			if a.screen == ScreenDashboard && a.infra != nil {
				return a, a.runWizard()
			}
		case "b":
			if a.screen == ScreenComparison {
				a.screen = ScreenDashboard
				a.comparison = nil
				a.compView = nil
				return a, nil
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.dashboard != nil {
			a.dashboard.SetSize(a.dashboardWidth(), a.height-4)
		}

	case infraLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.infra = msg.infra
		a.dashboard = dashboard.New(a.infra, a.dashboardWidth(), a.height-4)
		a.screen = ScreenDashboard
		return a, nil

	case scenarioComparedMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.comparison = msg.result
		a.compView = comparison.New(a.comparison, a.comparisonWidth())
		a.screen = ScreenComparison
		return a, nil
	}

	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	switch a.screen {
	case ScreenMenu:
		return a.viewMenu()
	case ScreenDashboard:
		return a.viewDashboard()
	case ScreenComparison:
		return a.viewComparison()
	default:
		return a.viewMenu()
	}
}

// viewMenu renders the initial menu screen
func (a *App) viewMenu() string {
	var s string
	s += styles.Title.Render("Diego Capacity Analyzer")
	s += "\n\n"
	s += "Loading data source selection..."
	s += "\n\n"
	s += styles.Help.Render("Press 'q' to quit")
	return s
}

// viewDashboard renders the dashboard with actions pane
func (a *App) viewDashboard() string {
	if a.err != nil {
		return styles.StatusCritical.Render("Error: "+a.err.Error()) + "\n\n" +
			styles.Help.Render("Press 'q' to quit, 'r' to retry")
	}

	leftPane := ""
	if a.dashboard != nil {
		leftPane = styles.ActivePanel.Width(a.dashboardWidth()).Render(a.dashboard.View())
	} else {
		leftPane = styles.Panel.Width(a.dashboardWidth()).Render("Loading...")
	}

	// Actions pane on the right
	rightContent := styles.Title.Render("Actions") + "\n\n"
	rightContent += "[r] Refresh data\n"
	rightContent += "[w] Run scenario wizard\n"
	rightContent += "[q] Quit\n"
	rightPane := styles.Panel.Width(a.actionsWidth()).Render(rightContent)

	// Join panes side by side
	view := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return view
}

// viewComparison renders the dashboard with comparison results
func (a *App) viewComparison() string {
	if a.err != nil {
		return styles.StatusCritical.Render("Error: "+a.err.Error()) + "\n\n" +
			styles.Help.Render("Press 'b' to go back, 'q' to quit")
	}

	leftPane := ""
	if a.dashboard != nil {
		leftPane = styles.Panel.Width(a.dashboardWidth()).Render(a.dashboard.View())
	}

	rightPane := ""
	if a.compView != nil {
		rightPane = styles.ActivePanel.Width(a.comparisonWidth()).Render(a.compView.View())
	}

	view := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	view += "\n" + styles.Help.Render("[b] Back to dashboard  [w] New scenario  [q] Quit")

	return view
}

// dashboardWidth calculates the width for the dashboard pane
func (a *App) dashboardWidth() int {
	if a.width < minTerminalWidth {
		return a.width - panelPadding
	}
	return (a.width - panelPadding) / 2
}

// actionsWidth calculates the width for the actions pane
func (a *App) actionsWidth() int {
	return a.width - a.dashboardWidth() - 4
}

// comparisonWidth calculates the width for the comparison pane
func (a *App) comparisonWidth() int {
	return a.width - a.dashboardWidth() - 4
}

// loadInfrastructure creates a command to fetch infrastructure data
func (a *App) loadInfrastructure() tea.Cmd {
	return func() tea.Msg {
		infra, err := a.client.GetInfrastructure(context.Background())
		return infraLoadedMsg{infra: infra, err: err}
	}
}

// runWizard creates a command to run the scenario wizard
func (a *App) runWizard() tea.Cmd {
	return func() tea.Msg {
		w := wizard.New(a.infra)
		if err := w.Run(); err != nil {
			return scenarioComparedMsg{err: err}
		}

		result, err := a.client.CompareScenario(context.Background(), w.GetInput())
		return scenarioComparedMsg{result: result, err: err}
	}
}

// Run starts the TUI
func Run(apiClient *client.Client, vsphereConfigured bool) error {
	// Show menu first to select data source
	m := menu.New(vsphereConfigured)
	source, err := m.Run()
	if err != nil {
		return err
	}

	// Create app and set initial command based on data source
	app := New(apiClient)
	app.dataSource = source

	var initCmd tea.Cmd
	switch source {
	case menu.SourceVSphere, menu.SourceJSON:
		initCmd = app.loadInfrastructure()
	case menu.SourceManual:
		// Manual input will be handled by a separate wizard in future
		initCmd = app.loadInfrastructure()
	}

	// Override Init to use our initial command
	p := tea.NewProgram(
		&appWithInit{App: app, initCmd: initCmd},
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}

// appWithInit wraps App to provide a custom Init command
type appWithInit struct {
	*App
	initCmd tea.Cmd
}

// Init returns the initial command for the app
func (a *appWithInit) Init() tea.Cmd {
	return a.initCmd
}

// Update delegates to the wrapped App
func (a *appWithInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return a.App.Update(msg)
}

// View delegates to the wrapped App
func (a *appWithInit) View() string {
	return a.App.View()
}
