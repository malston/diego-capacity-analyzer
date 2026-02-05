// ABOUTME: Scenario planning wizard as a bubbletea model
// ABOUTME: Uses huh forms with visual progress indicator for step navigation

package wizard

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
)

// WizardCompleteMsg is sent when the wizard finishes successfully
type WizardCompleteMsg struct {
	Input *client.ScenarioInput
}

// WizardCancelledMsg is sent when the wizard is cancelled
type WizardCancelledMsg struct{}

// Wizard manages the scenario planning wizard flow as a bubbletea model
type Wizard struct {
	infra *client.InfrastructureState
	input *client.ScenarioInput
	form  *huh.Form
	step  int
	width int

	// Form field values (strings for huh)
	cellMemory  string
	cellCPU     string
	cellDisk    string
	cellCount   string
	overhead    string
	haAdmission string
}

// Step names for progress indicator
var stepNames = []string{"Cell Sizing", "Cell Count", "Overhead & HA"}

// createTheme returns a custom huh theme matching the frontend React colors
func createTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Colors matching frontend React theme
	cyan := lipgloss.Color("#06B6D4")    // Cyan-500 - primary
	cyanLight := lipgloss.Color("#22D3EE") // Cyan-400 - accents
	blue := lipgloss.Color("#3B82F6")    // Blue-500 - info
	gray := lipgloss.Color("#9CA3AF")    // Gray-400 - muted
	grayLight := lipgloss.Color("#E5E7EB") // Gray-200 - text
	red := lipgloss.Color("#F87171")     // Red-400 - errors
	slate := lipgloss.Color("#334155")   // Slate-700 - borders

	// Group styles (section headers)
	t.Group.Title = lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		MarginBottom(1)
	t.Group.Description = lipgloss.NewStyle().
		Foreground(gray).
		MarginBottom(1)

	// Focused field styles
	t.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(cyan)
	t.Focused.Title = lipgloss.NewStyle().
		Foreground(cyanLight).
		Bold(true)
	t.Focused.Description = lipgloss.NewStyle().
		Foreground(gray)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().
		Foreground(red).
		SetString(" *")
	t.Focused.ErrorMessage = lipgloss.NewStyle().
		Foreground(red)

	// Select field styles
	t.Focused.SelectSelector = lipgloss.NewStyle().
		Foreground(cyan).
		SetString("> ")
	t.Focused.Option = lipgloss.NewStyle().
		Foreground(grayLight)
	t.Focused.SelectedOption = lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true)
	t.Focused.NextIndicator = lipgloss.NewStyle().
		Foreground(cyan).
		MarginLeft(1).
		SetString("→")
	t.Focused.PrevIndicator = lipgloss.NewStyle().
		Foreground(cyan).
		MarginRight(1).
		SetString("←")

	// Text input styles
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().
		Foreground(cyan)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(gray)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().
		Foreground(cyan)
	t.Focused.TextInput.Text = lipgloss.NewStyle().
		Foreground(grayLight)

	// Button styles
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(blue).
		Padding(0, 2).
		MarginRight(1)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(gray).
		Background(slate).
		Padding(0, 2).
		MarginRight(1)

	// Blurred field styles (inherit from focused with muted colors)
	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.HiddenBorder()).
		BorderLeft(true)
	t.Blurred.Title = lipgloss.NewStyle().
		Foreground(gray)
	t.Blurred.SelectSelector = lipgloss.NewStyle().
		Foreground(gray).
		SetString("  ")
	t.Blurred.Option = lipgloss.NewStyle().
		Foreground(gray)

	return t
}

// Common cell memory sizes
var memoryOptions = []huh.Option[string]{
	huh.NewOption("16 GB", "16"),
	huh.NewOption("32 GB", "32"),
	huh.NewOption("64 GB", "64"),
	huh.NewOption("128 GB", "128"),
	huh.NewOption("256 GB", "256"),
}

// Common CPU core counts
var cpuOptions = []huh.Option[string]{
	huh.NewOption("4 cores", "4"),
	huh.NewOption("8 cores", "8"),
	huh.NewOption("16 cores", "16"),
	huh.NewOption("32 cores", "32"),
}

// Common disk sizes
var diskOptions = []huh.Option[string]{
	huh.NewOption("100 GB", "100"),
	huh.NewOption("200 GB", "200"),
	huh.NewOption("500 GB", "500"),
	huh.NewOption("1000 GB", "1000"),
}

// New creates a new wizard with defaults from current infrastructure
func New(infra *client.InfrastructureState) *Wizard {
	input := &client.ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellDiskGB:   200,
		ProposedCellCount:    10,
		OverheadPct:          7.0,
		SelectedResources:    []string{"memory", "cpu", "disk"},
	}

	// Use current values as defaults if available
	if infra != nil && len(infra.Clusters) > 0 {
		c := infra.Clusters[0]
		if c.DiegoCellMemoryGB > 0 {
			input.ProposedCellMemoryGB = c.DiegoCellMemoryGB
		}
		if c.DiegoCellCPU > 0 {
			input.ProposedCellCPU = c.DiegoCellCPU
		}
		if c.DiegoCellDiskGB > 0 {
			input.ProposedCellDiskGB = c.DiegoCellDiskGB
		}
		if infra.TotalCellCount > 0 {
			input.ProposedCellCount = infra.TotalCellCount
		}
		if infra.TotalHostCount > 0 {
			input.HostCount = infra.TotalHostCount
		}
		if c.MemoryGBPerHost > 0 {
			input.MemoryPerHostGB = c.MemoryGBPerHost
		}
		if c.HAAdmissionControlPercentage > 0 {
			input.HAAdmissionPct = c.HAAdmissionControlPercentage
		}
		if c.CPUThreadsPerHost > 0 {
			input.PhysicalCoresPerHost = c.CPUThreadsPerHost
		}
	}

	w := &Wizard{
		infra:       infra,
		input:       input,
		step:        1,
		cellMemory:  strconv.Itoa(input.ProposedCellMemoryGB),
		cellCPU:     strconv.Itoa(input.ProposedCellCPU),
		cellDisk:    strconv.Itoa(input.ProposedCellDiskGB),
		cellCount:   strconv.Itoa(input.ProposedCellCount),
		overhead:    fmt.Sprintf("%.0f", input.OverheadPct),
		haAdmission: strconv.Itoa(input.HAAdmissionPct),
	}

	w.form = w.createStep1Form()
	return w
}

func (w *Wizard) createStep1Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory per cell").
				Description("Use ↑/↓ to select, Enter to confirm").
				Options(memoryOptions...).
				Value(&w.cellMemory),
			huh.NewSelect[string]().
				Title("CPU cores per cell").
				Description("Use ↑/↓ to select, Enter to confirm").
				Options(cpuOptions...).
				Value(&w.cellCPU),
			huh.NewSelect[string]().
				Title("Disk per cell").
				Description("Use ↑/↓ to select, Enter to confirm").
				Options(diskOptions...).
				Value(&w.cellDisk),
		).Title("Step 1: Cell Sizing").
			Description("Configure the size of each Diego cell VM"),
	).WithTheme(createTheme())
}

func (w *Wizard) createStep2Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Proposed cell count").
				Description("Type a number and press Enter to continue").
				Placeholder("e.g., 20").
				CharLimit(5).
				Value(&w.cellCount).
				Validate(validatePositiveInt),
		).Title("Step 2: Cell Count").
			Description("How many Diego cells do you want in your scenario?"),
	).WithTheme(createTheme())
}

func (w *Wizard) createStep3Form() *huh.Form {
	overheadOptions := []huh.Option[string]{
		huh.NewOption("5%", "5"),
		huh.NewOption("7% (recommended)", "7"),
		huh.NewOption("10%", "10"),
		huh.NewOption("15%", "15"),
	}

	haOptions := []huh.Option[string]{
		huh.NewOption("0% (no HA reserve)", "0"),
		huh.NewOption("10%", "10"),
		huh.NewOption("15%", "15"),
		huh.NewOption("20%", "20"),
		huh.NewOption("25% (N-1 for 4 hosts)", "25"),
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory overhead").
				Description("Diego cell memory overhead percentage").
				Options(overheadOptions...).
				Value(&w.overhead),
			huh.NewSelect[string]().
				Title("HA admission control").
				Description("vSphere HA cluster reservation percentage").
				Options(haOptions...).
				Value(&w.haAdmission),
		).Title("Step 3: Overhead & HA").
			Description("Configure overhead and high availability settings"),
	).WithTheme(createTheme())
}

// Init implements tea.Model
func (w *Wizard) Init() tea.Cmd {
	return w.form.Init()
}

// Update implements tea.Model
func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		// Forward to form
		form, cmd := w.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			w.form = f
		}
		return w, cmd

	case tea.KeyMsg:
		if msg.String() == "esc" {
			return w, func() tea.Msg { return WizardCancelledMsg{} }
		}
	}

	// Update the current form
	form, cmd := w.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		w.form = f
	}

	// Check if form is complete
	if w.form.State == huh.StateCompleted {
		return w.advanceStep()
	}

	return w, cmd
}

func (w *Wizard) advanceStep() (tea.Model, tea.Cmd) {
	switch w.step {
	case 1:
		// Parse step 1 values and move to step 2
		w.input.ProposedCellMemoryGB, _ = strconv.Atoi(w.cellMemory)
		w.input.ProposedCellCPU, _ = strconv.Atoi(w.cellCPU)
		w.input.ProposedCellDiskGB, _ = strconv.Atoi(w.cellDisk)
		w.step = 2
		w.form = w.createStep2Form()
		return w, w.form.Init()

	case 2:
		// Parse step 2 values and move to step 3
		w.input.ProposedCellCount, _ = strconv.Atoi(w.cellCount)
		w.step = 3
		w.form = w.createStep3Form()
		return w, w.form.Init()

	case 3:
		// Parse step 3 values and complete
		var overheadFloat float64
		fmt.Sscanf(w.overhead, "%f", &overheadFloat)
		w.input.OverheadPct = overheadFloat
		w.input.HAAdmissionPct, _ = strconv.Atoi(w.haAdmission)

		return w, func() tea.Msg {
			return WizardCompleteMsg{Input: w.input}
		}
	}

	return w, nil
}

// SetWidth sets the wizard width for proper rendering
func (w *Wizard) SetWidth(width int) {
	w.width = width
}

// View implements tea.Model
func (w *Wizard) View() string {
	var sb strings.Builder

	// Progress indicator
	sb.WriteString(w.renderProgress())
	sb.WriteString("\n\n")

	// Form content
	sb.WriteString(w.form.View())

	return sb.String()
}

// renderProgress renders the step progress indicator
func (w *Wizard) renderProgress() string {
	// Use width - 1 to ensure progress box fits within the frame
	// (w.width is already a.width - 1, so this gives a.width - 2 total)
	width := w.width - 1
	if width < 60 {
		width = 60
	}

	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary)

	// Build step indicators
	var steps []string
	for i, name := range stepNames {
		stepNum := i + 1
		var indicator string
		var nameStyle lipgloss.Style

		if stepNum < w.step {
			// Completed step
			indicator = lipgloss.NewStyle().Foreground(styles.Secondary).Render(icons.CheckOK.String())
			nameStyle = lipgloss.NewStyle().Foreground(styles.Muted)
		} else if stepNum == w.step {
			// Current step
			indicator = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("●")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		} else {
			// Future step
			indicator = lipgloss.NewStyle().Foreground(styles.Muted).Render("○")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Muted)
		}

		steps = append(steps, fmt.Sprintf("%s %s", indicator, nameStyle.Render(name)))
	}

	stepsLine := strings.Join(steps, "    ")

	// Progress bar line format: "│  " + bar + " │" = 5 chars overhead
	barWidth := width - 5
	totalSteps := len(stepNames)
	filledWidth := (w.step * barWidth) / totalSteps
	emptyWidth := barWidth - filledWidth

	filledBar := lipgloss.NewStyle().Foreground(styles.Primary).Render(strings.Repeat("━", filledWidth))
	emptyBar := lipgloss.NewStyle().Foreground(styles.Surface).Render(strings.Repeat("─", emptyWidth))
	progressBar := filledBar + emptyBar

	// Build panel with consistent width
	styledTitle := titleStyle.Render("Progress")
	titleWidth := lipgloss.Width("Progress")

	// Top border: "┌─ " + title + " " + fill + "┐"
	// Total = 3 + titleWidth + 1 + fillWidth + 1 = width
	topFillWidth := max(0, width-5-titleWidth)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", topFillWidth) + "┐"

	// Steps line: "│ " + content + padding + " │" = 4 chars overhead
	stepsLineWidth := lipgloss.Width(stepsLine)
	stepsPadding := max(0, width-4-stepsLineWidth)
	stepsLinePadded := "│ " + stepsLine + strings.Repeat(" ", stepsPadding) + " │"

	// Progress line: "│  " + bar + " │" (extra indent for visual alignment)
	progressLinePadded := "│  " + progressBar + " │"

	// Bottom border: "└" + fill + "┘"
	bottomFillWidth := width - 2
	bottomBorder := "└" + strings.Repeat("─", bottomFillWidth) + "┘"

	return borderStyle.Render(strings.Join([]string{
		topBorder,
		stepsLinePadded,
		progressLinePadded,
		bottomBorder,
	}, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetInput returns the collected scenario input
func (w *Wizard) GetInput() *client.ScenarioInput {
	return w.input
}

func validatePositiveInt(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fmt.Errorf("must be a positive number")
	}
	return nil
}

func validatePercentage(s string) error {
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil || v < 0 || v > 100 {
		return fmt.Errorf("must be between 0 and 100")
	}
	return nil
}

// Run is kept for backward compatibility but should not be used with bubbletea
// Deprecated: Use the wizard as a tea.Model instead
func (w *Wizard) Run() error {
	// This is the old blocking implementation
	// Step 1: Cell sizing
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(memoryOptions...).
				Value(&w.cellMemory),
			huh.NewSelect[string]().
				Title("CPU cores per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(cpuOptions...).
				Value(&w.cellCPU),
			huh.NewSelect[string]().
				Title("Disk per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(diskOptions...).
				Value(&w.cellDisk),
		).Title("Step 1: Cell Sizing").
			Description("Configure the size of each Diego cell VM"),
	).WithTheme(createTheme())

	if err := form1.Run(); err != nil {
		return err
	}

	w.input.ProposedCellMemoryGB, _ = strconv.Atoi(w.cellMemory)
	w.input.ProposedCellCPU, _ = strconv.Atoi(w.cellCPU)
	w.input.ProposedCellDiskGB, _ = strconv.Atoi(w.cellDisk)

	// Step 2: Cell count
	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Proposed cell count").
				Description("Type a number and press Enter to continue").
				Placeholder("e.g., 20").
				CharLimit(5).
				Value(&w.cellCount).
				Validate(validatePositiveInt),
		).Title("Step 2: Cell Count").
			Description("How many Diego cells do you want in your scenario?"),
	).WithTheme(createTheme())

	if err := form2.Run(); err != nil {
		return err
	}

	w.input.ProposedCellCount, _ = strconv.Atoi(w.cellCount)

	// Step 3: Overhead settings
	overheadOptions := []huh.Option[string]{
		huh.NewOption("5%", "5"),
		huh.NewOption("7% (recommended)", "7"),
		huh.NewOption("10%", "10"),
		huh.NewOption("15%", "15"),
	}

	haOptions := []huh.Option[string]{
		huh.NewOption("0% (no HA reserve)", "0"),
		huh.NewOption("10%", "10"),
		huh.NewOption("15%", "15"),
		huh.NewOption("20%", "20"),
		huh.NewOption("25% (N-1 for 4 hosts)", "25"),
	}

	form3 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory overhead").
				Description("Diego cell memory overhead percentage").
				Options(overheadOptions...).
				Value(&w.overhead),
			huh.NewSelect[string]().
				Title("HA admission control").
				Description("vSphere HA cluster reservation percentage").
				Options(haOptions...).
				Value(&w.haAdmission),
		).Title("Step 3: Overhead & HA").
			Description("Configure overhead and high availability settings"),
	).WithTheme(createTheme())

	if err := form3.Run(); err != nil {
		return err
	}

	var overheadFloat float64
	fmt.Sscanf(w.overhead, "%f", &overheadFloat)
	w.input.OverheadPct = overheadFloat
	w.input.HAAdmissionPct, _ = strconv.Atoi(w.haAdmission)

	return nil
}
