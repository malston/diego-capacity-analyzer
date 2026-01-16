// ABOUTME: Root bubbletea model for the TUI application
// ABOUTME: Manages screen state and routes keyboard input to child components

package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/comparison"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/dashboard"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/filepicker"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/menu"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/recentfiles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/samples"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/wizard"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenFilePicker
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

// fileLoadedMsg is sent when a JSON file is loaded
type fileLoadedMsg struct {
	path string
	data []byte
}

// infraPostedMsg is sent when infrastructure is posted to backend
type infraPostedMsg struct {
	err error
}

// App is the root model for the TUI
type App struct {
	client            *client.Client
	screen            Screen
	width             int
	height            int
	err               error
	infra             *client.InfrastructureState
	comparison        *client.ScenarioComparison
	dashboard         *dashboard.Dashboard
	compView          *comparison.Comparison
	dataSource        menu.DataSource
	vsphereConfigured bool
	repoBasePath      string

	// Child models
	menu       *menu.Menu
	filePicker *filepicker.FilePicker

	// Recent files manager
	recentFiles *recentfiles.RecentFiles
}

// New creates a new TUI application
func New(apiClient *client.Client, vsphereConfigured bool, repoBasePath string) *App {
	return &App{
		client:            apiClient,
		screen:            ScreenMenu,
		vsphereConfigured: vsphereConfigured,
		repoBasePath:      repoBasePath,
		recentFiles:       recentfiles.New(recentfiles.DefaultConfigDir()),
		menu:              menu.New(vsphereConfigured),
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.dashboard != nil {
			a.dashboard.SetSize(a.dashboardWidth(), a.height-4)
		}
		// Forward to child models
		if a.menu != nil {
			a.menu.Update(msg)
		}
		if a.filePicker != nil {
			a.filePicker.Update(msg)
		}
		return a, nil

	case tea.KeyMsg:
		// Handle global quit
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Route to current screen
		switch a.screen {
		case ScreenMenu:
			return a.updateMenu(msg)
		case ScreenFilePicker:
			return a.updateFilePicker(msg)
		case ScreenDashboard:
			return a.updateDashboard(msg)
		case ScreenComparison:
			return a.updateComparison(msg)
		}

	case menu.DataSourceSelectedMsg:
		return a.handleDataSourceSelected(msg)

	case menu.CancelledMsg:
		return a, tea.Quit

	case filepicker.FileSelectedMsg:
		return a.handleFileSelected(msg)

	case filepicker.CancelledMsg:
		// Go back to menu
		a.screen = ScreenMenu
		a.filePicker = nil
		return a, nil

	case fileLoadedMsg:
		return a.handleFileLoaded(msg)

	case infraLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.infra = msg.infra
		a.dashboard = dashboard.New(a.infra, a.dashboardWidth(), a.height-4)
		a.screen = ScreenDashboard
		return a, nil

	case infraPostedMsg:
		// Backend post completed (success or failure doesn't block UI)
		// The infrastructure is already loaded locally
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

func (a *App) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.menu == nil {
		return a, nil
	}
	model, cmd := a.menu.Update(msg)
	a.menu = model.(*menu.Menu)
	return a, cmd
}

func (a *App) updateFilePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.filePicker == nil {
		return a, nil
	}
	model, cmd := a.filePicker.Update(msg)
	a.filePicker = model.(*filepicker.FilePicker)
	return a, cmd
}

func (a *App) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "r":
		return a, a.loadInfrastructure()
	case "w":
		if a.infra != nil {
			return a, a.runWizard()
		}
	case "b":
		// Go back to menu
		a.screen = ScreenMenu
		a.dashboard = nil
		a.infra = nil
		a.err = nil
		return a, nil
	}
	return a, nil
}

func (a *App) updateComparison(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "b":
		a.screen = ScreenDashboard
		a.comparison = nil
		a.compView = nil
		return a, nil
	case "w":
		if a.infra != nil {
			return a, a.runWizard()
		}
	}
	return a, nil
}

func (a *App) handleDataSourceSelected(msg menu.DataSourceSelectedMsg) (tea.Model, tea.Cmd) {
	a.dataSource = msg.Source

	switch msg.Source {
	case menu.SourceVSphere:
		a.screen = ScreenDashboard
		return a, a.loadInfrastructure()

	case menu.SourceJSON:
		// Initialize file picker with recent files and samples
		recentList, _ := a.recentFiles.Load()
		samplesDir := samples.FindSamplesDir(a.repoBasePath)
		sampleFiles, _ := samples.Discover(samplesDir)
		a.filePicker = filepicker.New(recentList, sampleFiles)
		a.screen = ScreenFilePicker
		return a, nil

	case menu.SourceManual:
		// Manual input goes directly to dashboard (will implement manual input later)
		a.screen = ScreenDashboard
		return a, a.loadInfrastructure()
	}

	return a, nil
}

func (a *App) handleFileSelected(msg filepicker.FileSelectedMsg) (tea.Model, tea.Cmd) {
	// Add to recent files
	a.recentFiles.Add(msg.Path)

	// Return a command that parses the JSON
	return a, func() tea.Msg {
		return fileLoadedMsg{path: msg.Path, data: msg.Data}
	}
}

func (a *App) handleFileLoaded(msg fileLoadedMsg) (tea.Model, tea.Cmd) {
	// Parse the infrastructure JSON
	var infra client.InfrastructureState
	if err := json.Unmarshal(msg.data, &infra); err != nil {
		a.err = err
		if a.filePicker != nil {
			a.filePicker.SetError("Invalid JSON: " + err.Error())
		}
		return a, nil
	}

	// Store infrastructure and transition to dashboard
	a.infra = &infra
	a.dashboard = dashboard.New(a.infra, a.dashboardWidth(), a.height-4)
	a.screen = ScreenDashboard
	a.filePicker = nil

	// POST infrastructure state to backend so scenario comparison works
	return a, a.postInfrastructureState(&infra)
}

// postInfrastructureState sends the loaded infrastructure to the backend
func (a *App) postInfrastructureState(infra *client.InfrastructureState) tea.Cmd {
	return func() tea.Msg {
		_, err := a.client.SetInfrastructureState(context.Background(), infra)
		if err != nil {
			// Don't block UI on backend errors - we already have the data loaded
			// This just enables scenario comparison
			return infraPostedMsg{err: err}
		}
		return infraPostedMsg{}
	}
}

// View implements tea.Model
func (a *App) View() string {
	switch a.screen {
	case ScreenMenu:
		return a.viewMenu()
	case ScreenFilePicker:
		return a.viewFilePicker()
	case ScreenDashboard:
		return a.viewDashboard()
	case ScreenComparison:
		return a.viewComparison()
	default:
		return a.viewMenu()
	}
}

// viewMenu renders the menu screen
func (a *App) viewMenu() string {
	if a.menu != nil {
		return a.menu.View()
	}
	return ""
}

// viewFilePicker renders the file picker screen
func (a *App) viewFilePicker() string {
	if a.filePicker != nil {
		return a.filePicker.View()
	}
	return ""
}

// viewDashboard renders the dashboard with actions pane
func (a *App) viewDashboard() string {
	if a.err != nil {
		return styles.StatusCritical.Render("Error: "+a.err.Error()) + "\n\n" +
			styles.Help.Render("Press 'b' to go back, 'q' to quit")
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
	rightContent += "[b] Back to menu\n"
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
	// Find repository base path for sample files
	repoBasePath := findRepoBasePath()

	app := New(apiClient, vsphereConfigured, repoBasePath)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}

// findRepoBasePath attempts to locate the repository root
func findRepoBasePath() string {
	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		// Check if frontend/public/samples exists
		samplesDir := filepath.Join(cwd, "frontend", "public", "samples")
		if _, err := os.Stat(samplesDir); err == nil {
			return cwd
		}
	}

	// Fall back to empty (will rely on DIEGO_SAMPLES_PATH env var)
	return ""
}
