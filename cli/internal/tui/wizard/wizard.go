// ABOUTME: Scenario planning wizard as a bubbletea model
// ABOUTME: Uses huh forms embedded in bubbletea for responsive input handling

package wizard

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
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

	// Form field values (strings for huh)
	cellMemory  string
	cellCPU     string
	cellDisk    string
	cellCount   string
	overhead    string
	haAdmission string
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
		if c.CPUCoresPerHost > 0 {
			input.PhysicalCoresPerHost = c.CPUCoresPerHost
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
	).WithTheme(huh.ThemeBase())
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
	).WithTheme(huh.ThemeBase())
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
	).WithTheme(huh.ThemeBase())
}

// Init implements tea.Model
func (w *Wizard) Init() tea.Cmd {
	return w.form.Init()
}

// Update implements tea.Model
func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle escape to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
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

// View implements tea.Model
func (w *Wizard) View() string {
	return w.form.View()
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
	).WithTheme(huh.ThemeBase())

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
	).WithTheme(huh.ThemeBase())

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
	).WithTheme(huh.ThemeBase())

	if err := form3.Run(); err != nil {
		return err
	}

	var overheadFloat float64
	fmt.Sscanf(w.overhead, "%f", &overheadFloat)
	w.input.OverheadPct = overheadFloat
	w.input.HAAdmissionPct, _ = strconv.Atoi(w.haAdmission)

	return nil
}
