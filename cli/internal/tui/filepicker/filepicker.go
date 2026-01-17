// ABOUTME: File picker TUI component for selecting JSON files
// ABOUTME: Shows recent files, path input, and sample files

package filepicker

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/samples"
)

// State represents the current UI state
type state int

const (
	stateList state = iota
	stateInput
	stateSamples
)

// FileSelectedMsg is sent when a file is selected
type FileSelectedMsg struct {
	Path string
	Data []byte
}

// CancelledMsg is sent when the user cancels
type CancelledMsg struct{}

// FilePicker is the file selection component
type FilePicker struct {
	recentFiles []string
	samples     []samples.SampleFile
	hasSamples  bool
	cursor      int
	state       state
	textInput   textinput.Model
	err         string
	width       int
	height      int
}

// Styles
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// New creates a new FilePicker
func New(recentFiles []string, sampleFiles []samples.SampleFile) *FilePicker {
	ti := textinput.New()
	ti.Placeholder = "/path/to/infrastructure.json"
	ti.CharLimit = 256
	ti.Width = 60

	return &FilePicker{
		recentFiles: recentFiles,
		samples:     sampleFiles,
		hasSamples:  len(sampleFiles) > 0,
		cursor:      0,
		state:       stateList,
		textInput:   ti,
	}
}

// Init implements tea.Model
func (fp *FilePicker) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (fp *FilePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fp.width = msg.Width
		fp.height = msg.Height
		return fp, nil

	case tea.KeyMsg:
		// Clear error on any key press
		fp.err = ""

		switch fp.state {
		case stateList:
			return fp.updateList(msg)
		case stateInput:
			return fp.updateInput(msg)
		case stateSamples:
			return fp.updateSamples(msg)
		}
	}

	return fp, nil
}

func (fp *FilePicker) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxItems := fp.listItemCount()

	switch msg.String() {
	case "up", "k":
		if fp.cursor > 0 {
			fp.cursor--
		}
	case "down", "j":
		if fp.cursor < maxItems-1 {
			fp.cursor++
		}
	case "enter":
		return fp.selectListItem()
	case "esc", "b":
		return fp, func() tea.Msg { return CancelledMsg{} }
	}

	return fp, nil
}

func (fp *FilePicker) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		fp.state = stateList
		fp.textInput.SetValue("")
		return fp, nil
	case "enter":
		path := fp.textInput.Value()
		if path == "" {
			fp.err = "Please enter a file path"
			return fp, nil
		}
		return fp.loadFile(path)
	}

	var cmd tea.Cmd
	fp.textInput, cmd = fp.textInput.Update(msg)
	return fp, cmd
}

func (fp *FilePicker) updateSamples(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxItems := len(fp.samples) + 1 // +1 for [back]

	switch msg.String() {
	case "up", "k":
		if fp.cursor > 0 {
			fp.cursor--
		}
	case "down", "j":
		if fp.cursor < maxItems-1 {
			fp.cursor++
		}
	case "enter":
		if fp.cursor == len(fp.samples) {
			// [back] selected
			fp.state = stateList
			fp.cursor = 0
			return fp, nil
		}
		// Sample selected
		sample := fp.samples[fp.cursor]
		return fp.loadFile(sample.Path)
	case "esc", "b":
		fp.state = stateList
		fp.cursor = 0
		return fp, nil
	}

	return fp, nil
}

func (fp *FilePicker) listItemCount() int {
	count := len(fp.recentFiles) + 1 // +1 for "Enter path..."
	if fp.hasSamples {
		count++ // +1 for "Load sample file..."
	}
	return count
}

func (fp *FilePicker) selectListItem() (tea.Model, tea.Cmd) {
	recentCount := len(fp.recentFiles)

	if fp.cursor < recentCount {
		// Recent file selected
		path := fp.recentFiles[fp.cursor]
		return fp.loadFile(path)
	}

	if fp.cursor == recentCount {
		// "Enter path..." selected
		fp.state = stateInput
		fp.textInput.Focus()
		return fp, textinput.Blink
	}

	if fp.hasSamples && fp.cursor == recentCount+1 {
		// "Load sample file..." selected
		fp.state = stateSamples
		fp.cursor = 0
		return fp, nil
	}

	return fp, nil
}

func (fp *FilePicker) loadFile(path string) (tea.Model, tea.Cmd) {
	// Expand ~ to home directory
	expandedPath := expandPath(path)

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			fp.err = "File not found: " + path
		} else if os.IsPermission(err) {
			fp.err = "Cannot read file: permission denied"
		} else {
			fp.err = "Error reading file: " + err.Error()
		}
		return fp, nil
	}

	return fp, func() tea.Msg {
		return FileSelectedMsg{Path: expandedPath, Data: data}
	}
}

// expandPath expands ~ to home directory and resolves relative paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + path[1:]
		}
	} else if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	return path
}

// SetError sets an error message to display
func (fp *FilePicker) SetError(msg string) {
	fp.err = msg
}

// View implements tea.Model
func (fp *FilePicker) View() string {
	switch fp.state {
	case stateInput:
		return fp.viewInput()
	case stateSamples:
		return fp.viewSamples()
	default:
		return fp.viewList()
	}
}

func (fp *FilePicker) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select JSON file"))
	b.WriteString("\n\n")

	// Recent files section
	if len(fp.recentFiles) > 0 {
		b.WriteString(helpStyle.Render("Recent files:"))
		b.WriteString("\n")
		for i, path := range fp.recentFiles {
			cursor := "  "
			style := normalStyle
			if i == fp.cursor {
				cursor = "> "
				style = selectedStyle
			}
			// Truncate long paths
			display := path
			if len(display) > fp.width-10 && fp.width > 20 {
				display = "..." + display[len(display)-(fp.width-13):]
			}
			b.WriteString(cursor + style.Render(display) + "\n")
		}
		b.WriteString("\n")
	}

	// Divider
	if len(fp.recentFiles) > 0 {
		dividerWidth := min(40, fp.width-4)
		if dividerWidth < 1 {
			dividerWidth = 40 // Default width if terminal size unknown
		}
		divider := strings.Repeat("─", dividerWidth)
		b.WriteString(dividerStyle.Render(divider))
		b.WriteString("\n")
	}

	// Enter path option
	idx := len(fp.recentFiles)
	cursor := "  "
	style := normalStyle
	if fp.cursor == idx {
		cursor = "> "
		style = selectedStyle
	}
	b.WriteString(cursor + style.Render("Enter path...") + "\n")

	// Load sample option
	if fp.hasSamples {
		idx++
		cursor = "  "
		style = normalStyle
		if fp.cursor == idx {
			cursor = "> "
			style = selectedStyle
		}
		b.WriteString(cursor + style.Render("Load sample file...") + "\n")
	}

	// Error message
	if fp.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + fp.err))
	}

	// Footer frame now has keyboard shortcuts

	return b.String()
}

func (fp *FilePicker) viewInput() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Enter file path"))
	b.WriteString("\n\n")
	b.WriteString(fp.textInput.View())

	if fp.err != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("Error: " + fp.err))
	}

	// Footer frame now has keyboard shortcuts

	return b.String()
}

func (fp *FilePicker) viewSamples() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select sample file"))
	b.WriteString("\n\n")

	for i, sample := range fp.samples {
		cursor := "  "
		style := normalStyle
		if i == fp.cursor {
			cursor = "> "
			style = selectedStyle
		}
		b.WriteString(cursor + style.Render(sample.Name) + "\n")
	}

	// [back] option
	cursor := "  "
	style := normalStyle
	if fp.cursor == len(fp.samples) {
		cursor = "> "
		style = selectedStyle
	}
	b.WriteString(cursor + style.Render("[back]") + "\n")

	if fp.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + fp.err))
	}

	// Footer frame now has keyboard shortcuts

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
