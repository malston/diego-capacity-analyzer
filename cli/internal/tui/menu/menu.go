// ABOUTME: Data source selection menu for TUI startup
// ABOUTME: Allows user to choose between vSphere, JSON file, or manual input

package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DataSource represents the selected data source
type DataSource int

const (
	SourceVSphere DataSource = iota
	SourceJSON
	SourceManual
)

// DataSourceSelectedMsg is sent when a data source is selected
type DataSourceSelectedMsg struct {
	Source DataSource
}

// CancelledMsg is sent when the user cancels
type CancelledMsg struct{}

type option struct {
	label   string
	value   DataSource
	enabled bool
}

// Menu represents the data source selection menu
type Menu struct {
	options []option
	cursor  int
	err     string
	width   int
	height  int
}

// Styles
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// New creates a new data source menu
func New(vsphereConfigured bool) *Menu {
	return &Menu{
		options: []option{
			{label: "Live vSphere", value: SourceVSphere, enabled: vsphereConfigured},
			{label: "Load JSON file", value: SourceJSON, enabled: true},
			{label: "Manual input", value: SourceManual, enabled: true},
		},
		cursor: 0,
	}
}

// Init implements tea.Model
func (m *Menu) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Clear error on any key press
		m.err = ""

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			return m.selectOption()
		case "esc", "q":
			return m, func() tea.Msg { return CancelledMsg{} }
		}
	}

	return m, nil
}

func (m *Menu) selectOption() (tea.Model, tea.Cmd) {
	opt := m.options[m.cursor]

	if !opt.enabled {
		m.err = "vSphere is not configured"
		return m, nil
	}

	return m, func() tea.Msg {
		return DataSourceSelectedMsg{Source: opt.value}
	}
}

// View implements tea.Model
func (m *Menu) View() string {
	var b strings.Builder

	// Title and prompt (header frame now has app name)
	b.WriteString(titleStyle.Render("Select Data Source"))
	b.WriteString("\n\n")

	for i, opt := range m.options {
		cursor := "  "
		style := normalStyle

		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		label := opt.label
		if !opt.enabled {
			label = opt.label + " (not configured)"
			if i != m.cursor {
				style = disabledStyle
			}
		}

		b.WriteString(cursor + style.Render(label) + "\n")
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + m.err))
	}

	// Footer frame now has keyboard shortcuts, so we don't need them here

	return b.String()
}

// String returns the string representation of a DataSource
func (ds DataSource) String() string {
	switch ds {
	case SourceVSphere:
		return "vsphere"
	case SourceJSON:
		return "json"
	case SourceManual:
		return "manual"
	default:
		return "unknown"
	}
}
