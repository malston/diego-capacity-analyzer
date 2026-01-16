// ABOUTME: Root bubbletea model for the TUI application
// ABOUTME: Manages screen state and routes keyboard input to child components

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenDashboard
)

// App is the root model for the TUI
type App struct {
	client *client.Client
	screen Screen
	width  int
	height int
	err    error
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
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	return "Diego Capacity Analyzer\n\nPress 'q' to quit.\n"
}

// Run starts the TUI
func Run(apiClient *client.Client) error {
	p := tea.NewProgram(New(apiClient), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
