// ABOUTME: Root bubbletea model for the TUI application
// ABOUTME: Manages screen state and routes keyboard input to child components

package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/comparison"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/dashboard"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/filepicker"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
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
	ScreenWizard
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
	lastUpdate        time.Time
	infraName         string // Name of the infrastructure source for header

	// Child models
	menu         *menu.Menu
	filePicker   *filepicker.FilePicker
	wizardScreen *wizard.Wizard

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
			a.dashboard.SetSize(a.dashboardWidth(), a.contentHeight())
		}
		// Forward to child models
		if a.menu != nil {
			a.menu.Update(msg)
		}
		if a.filePicker != nil {
			a.filePicker.Update(msg)
		}
		if a.wizardScreen != nil {
			return a.updateWizard(msg)
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
		case ScreenWizard:
			return a.updateWizard(msg)
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

	case wizard.WizardCompleteMsg:
		// Wizard finished, call backend to compare scenario
		a.wizardScreen = nil
		return a, a.compareScenario(msg.Input)

	case wizard.WizardCancelledMsg:
		// Go back to dashboard
		a.screen = ScreenDashboard
		a.wizardScreen = nil
		return a, nil

	case fileLoadedMsg:
		return a.handleFileLoaded(msg)

	case infraLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.infra = msg.infra
		a.lastUpdate = time.Now()
		a.infraName = a.deriveInfraName()
		a.dashboard = dashboard.New(a.infra, a.dashboardWidth(), a.contentHeight())
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

	default:
		// Forward unknown messages to wizard when active (needed for huh form internals)
		if a.screen == ScreenWizard && a.wizardScreen != nil {
			return a.updateWizard(msg)
		}
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

func (a *App) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.wizardScreen == nil {
		return a, nil
	}
	model, cmd := a.wizardScreen.Update(msg)
	a.wizardScreen = model.(*wizard.Wizard)
	return a, cmd
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
	// Try to detect the JSON format - ManualInput vs InfrastructureState
	// ManualInput has clusters[].memory_gb_per_host, InfrastructureState has clusters[].memory_gb
	if isManualInputFormat(msg.data) {
		// Parse as ManualInput and send to backend for computation
		var input client.ManualInput
		if err := json.Unmarshal(msg.data, &input); err != nil {
			a.err = err
			if a.filePicker != nil {
				a.filePicker.SetError("Invalid JSON: " + err.Error())
			}
			return a, nil
		}

		// Transition to dashboard with loading state
		a.screen = ScreenDashboard
		a.filePicker = nil

		// Call backend to compute infrastructure state
		return a, a.computeManualInfrastructure(&input)
	}

	// Parse as InfrastructureState (pre-computed format)
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
	a.lastUpdate = time.Now()
	a.infraName = a.deriveInfraName()
	a.dashboard = dashboard.New(a.infra, a.dashboardWidth(), a.contentHeight())
	a.screen = ScreenDashboard
	a.filePicker = nil

	// POST infrastructure state to backend so scenario comparison works
	return a, a.postInfrastructureState(&infra)
}

// isManualInputFormat detects if JSON is ManualInput format (has memory_gb_per_host)
func isManualInputFormat(data []byte) bool {
	// Quick check: ManualInput has "memory_gb_per_host", InfrastructureState has "memory_gb"
	// but NOT "memory_gb_per_host"
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}

	clusters, ok := raw["clusters"].([]interface{})
	if !ok || len(clusters) == 0 {
		return false
	}

	firstCluster, ok := clusters[0].(map[string]interface{})
	if !ok {
		return false
	}

	// ManualInput format has memory_gb_per_host
	_, hasPerHost := firstCluster["memory_gb_per_host"]
	return hasPerHost
}

// computeManualInfrastructure calls the backend to compute infrastructure from manual input
func (a *App) computeManualInfrastructure(input *client.ManualInput) tea.Cmd {
	return func() tea.Msg {
		infra, err := a.client.SetManualInfrastructure(context.Background(), input)
		if err != nil {
			return infraLoadedMsg{err: err}
		}
		return infraLoadedMsg{infra: infra}
	}
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
	var content string

	switch a.screen {
	case ScreenMenu:
		content = a.viewMenu()
	case ScreenFilePicker:
		content = a.viewFilePicker()
	case ScreenDashboard:
		content = a.viewDashboard()
	case ScreenComparison:
		content = a.viewComparison()
	case ScreenWizard:
		content = a.viewWizard()
	default:
		content = a.viewMenu()
	}

	return a.wrapWithFrame(content)
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
		return styles.StatusCritical.Render("Error: " + a.err.Error())
	}

	leftPane := ""
	if a.dashboard != nil {
		leftPane = styles.ActivePanel.Width(a.dashboardWidth()).Render(a.dashboard.View())
	} else {
		leftPane = styles.Panel.Width(a.dashboardWidth()).Render("Loading...")
	}

	// Actions pane on the right - shows available actions
	rightContent := styles.Title.Render(icons.Settings.String()+" Actions") + "\n\n"
	rightContent += icons.Refresh.String() + " Refresh data\n"
	rightContent += icons.Wizard.String() + " Run scenario wizard\n"
	rightContent += icons.Back.String() + " Back to menu\n"
	rightContent += icons.Quit.String() + " Quit application\n"
	rightPane := styles.Panel.Width(a.actionsWidth()).Render(rightContent)

	// Join panes side by side
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

// viewWizard renders the wizard screen
func (a *App) viewWizard() string {
	if a.wizardScreen != nil {
		return a.wizardScreen.View()
	}
	return ""
}

// viewComparison renders the dashboard with comparison results
func (a *App) viewComparison() string {
	if a.err != nil {
		return styles.StatusCritical.Render("Error: " + a.err.Error())
	}

	leftPane := ""
	if a.dashboard != nil {
		leftPane = styles.Panel.Width(a.dashboardWidth()).Render(a.dashboard.View())
	}

	rightPane := ""
	if a.compView != nil {
		rightPane = styles.ActivePanel.Width(a.comparisonWidth()).Render(a.compView.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
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

// contentHeight calculates the height available for dashboard content
func (a *App) contentHeight() int {
	// Total overhead:
	// - Header: 1 line
	// - Newline after header: 1 line
	// - ActivePanel border+padding: 4 lines (top border, top padding, bottom padding, bottom border)
	// - Newline before footer: 1 line
	// - Footer: 1 line
	// Total: 8 lines overhead
	return a.height - 8
}

// deriveInfraName extracts a display name for the infrastructure source
func (a *App) deriveInfraName() string {
	switch a.dataSource {
	case menu.SourceVSphere:
		return "vSphere"
	case menu.SourceJSON:
		if a.infra != nil && len(a.infra.Clusters) > 0 {
			return a.infra.Clusters[0].Name
		}
		return "JSON File"
	case menu.SourceManual:
		return "Manual Input"
	default:
		if a.infra != nil && len(a.infra.Clusters) > 0 {
			return a.infra.Clusters[0].Name
		}
		return "Infrastructure"
	}
}

// renderHeader creates the header bar with app branding and context
func (a *App) renderHeader() string {
	// Guard against zero/small width before WindowSizeMsg is received
	width := a.width
	if width < minTerminalWidth {
		width = minTerminalWidth
	}

	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	contextStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

	icon := icons.App.String()
	title := "Diego Capacity Analyzer"

	// Build left content
	leftText := fmt.Sprintf(" %s %s", icon, titleStyle.Render(title))

	// Build right content (only on certain screens)
	rightText := ""
	if a.infraName != "" && a.screen != ScreenMenu && a.screen != ScreenFilePicker {
		rightText = contextStyle.Render(a.infraName) + " "
	}

	// Use lipgloss to handle the width properly
	// Create a style that fills the width with the border character
	leftStyle := lipgloss.NewStyle()
	rightStyle := lipgloss.NewStyle().Align(lipgloss.Right)

	leftRendered := leftStyle.Render(leftText)
	rightRendered := rightStyle.Render(rightText)

	// Calculate fill needed
	leftWidth := lipgloss.Width(leftRendered)
	rightWidth := lipgloss.Width(rightRendered)
	fillWidth := width - 4 - leftWidth - rightWidth // -4 for ╭─ and ─╮
	if fillWidth < 0 {
		fillWidth = 0
	}

	fill := strings.Repeat("─", fillWidth)

	header := "╭─" + leftRendered + fill + rightRendered + "─╮"

	return borderStyle.Render(header)
}

// renderFooter creates the footer with keyboard shortcuts and status
func (a *App) renderFooter() string {
	// Guard against zero/small width before WindowSizeMsg is received
	width := a.width
	if width < minTerminalWidth {
		width = minTerminalWidth
	}

	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(styles.Primary)
	labelStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	statusStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

	// Build keyboard shortcuts based on current screen
	var shortcuts []string
	switch a.screen {
	case ScreenMenu:
		shortcuts = []string{"↑↓ Navigate", "Enter Select", "q Quit"}
	case ScreenFilePicker:
		shortcuts = []string{"↑↓ Navigate", "Enter Select", "b Back", "q Quit"}
	case ScreenDashboard:
		shortcuts = []string{"r Refresh", "w Wizard", "b Back", "q Quit"}
	case ScreenComparison:
		shortcuts = []string{"w New scenario", "b Back", "q Quit"}
	case ScreenWizard:
		shortcuts = []string{"↑↓ Select", "Enter Confirm", "Esc Cancel"}
	}

	// Build styled shortcuts
	var styledShortcuts []string
	for _, s := range shortcuts {
		parts := strings.SplitN(s, " ", 2)
		if len(parts) == 2 {
			styledShortcuts = append(styledShortcuts, keyStyle.Render(parts[0])+" "+labelStyle.Render(parts[1]))
		} else {
			styledShortcuts = append(styledShortcuts, s)
		}
	}

	leftText := " " + strings.Join(styledShortcuts, "  ")
	leftPlainText := " " + strings.Join(shortcuts, "  ")

	// Right side status (last update time)
	rightText := ""
	rightPlainText := ""
	if !a.lastUpdate.IsZero() && a.screen != ScreenMenu && a.screen != ScreenFilePicker && a.screen != ScreenWizard {
		elapsed := a.formatTimeSince(a.lastUpdate)
		rightText = statusStyle.Render("Updated "+elapsed) + " "
		rightPlainText = "Updated " + elapsed + " "
	}

	// Calculate widths
	leftWidth := lipgloss.Width(leftPlainText)
	rightWidth := lipgloss.Width(rightPlainText)
	fillWidth := width - 4 - leftWidth - rightWidth // -4 for ╰─ and ─╯
	if fillWidth < 0 {
		fillWidth = 0
	}

	fill := strings.Repeat("─", fillWidth)

	footer := "╰─" + leftText + fill + rightText + "─╯"

	return borderStyle.Render(footer)
}

// formatTimeSince formats a duration since the given time in human-readable form
func (a *App) formatTimeSince(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		secs := int(d.Seconds())
		if secs < 5 {
			return "just now"
		}
		return fmt.Sprintf("%ds ago", secs)
	}

	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	}

	hours := int(d.Hours())
	if hours == 1 {
		return "1h ago"
	}
	return fmt.Sprintf("%dh ago", hours)
}

// wrapWithFrame wraps content with header and footer
func (a *App) wrapWithFrame(content string) string {
	var sb strings.Builder

	sb.WriteString(a.renderHeader())
	sb.WriteString("\n")
	sb.WriteString(content)
	sb.WriteString("\n")
	sb.WriteString(a.renderFooter())

	return sb.String()
}

// loadInfrastructure creates a command to fetch infrastructure data
func (a *App) loadInfrastructure() tea.Cmd {
	return func() tea.Msg {
		infra, err := a.client.GetInfrastructure(context.Background())
		return infraLoadedMsg{infra: infra, err: err}
	}
}

// runWizard transitions to the wizard screen
func (a *App) runWizard() tea.Cmd {
	a.wizardScreen = wizard.New(a.infra)
	a.screen = ScreenWizard
	return a.wizardScreen.Init()
}

// compareScenario calls the backend to compare the scenario
func (a *App) compareScenario(input *client.ScenarioInput) tea.Cmd {
	return func() tea.Msg {
		result, err := a.client.CompareScenario(context.Background(), input)
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
