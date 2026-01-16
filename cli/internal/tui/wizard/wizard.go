// ABOUTME: Scenario planning wizard using huh forms
// ABOUTME: Collects cell sizing, count, and overhead inputs for scenario comparison

package wizard

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

// Wizard manages the scenario planning wizard flow
type Wizard struct {
	infra *client.InfrastructureState
	input *client.ScenarioInput
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

	return &Wizard{
		infra: infra,
		input: input,
	}
}

// GetInput returns the collected scenario input
func (w *Wizard) GetInput() *client.ScenarioInput {
	return w.input
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

// findOptionIndex returns the index of a value in options, or 0 if not found
func findOptionIndex(options []huh.Option[string], value string) int {
	for i, opt := range options {
		if opt.Value == value {
			return i
		}
	}
	return 0
}

// Run executes the wizard and collects input
func (w *Wizard) Run() error {
	// Step 1: Cell sizing using Select for common values
	cellMemory := strconv.Itoa(w.input.ProposedCellMemoryGB)
	cellCPU := strconv.Itoa(w.input.ProposedCellCPU)
	cellDisk := strconv.Itoa(w.input.ProposedCellDiskGB)

	// Pre-select current values if they match options
	memOpts := make([]huh.Option[string], len(memoryOptions))
	copy(memOpts, memoryOptions)

	cpuOpts := make([]huh.Option[string], len(cpuOptions))
	copy(cpuOpts, cpuOptions)

	diskOpts := make([]huh.Option[string], len(diskOptions))
	copy(diskOpts, diskOptions)

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(memOpts...).
				Value(&cellMemory),
			huh.NewSelect[string]().
				Title("CPU cores per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(cpuOpts...).
				Value(&cellCPU),
			huh.NewSelect[string]().
				Title("Disk per cell").
				Description("Use arrow keys to select, Enter to confirm").
				Options(diskOpts...).
				Value(&cellDisk),
		).Title("Step 1: Cell Sizing").
			Description("Configure the size of each Diego cell VM"),
	).WithTheme(huh.ThemeBase())

	if err := form1.Run(); err != nil {
		return err
	}

	// Parse values from step 1
	w.input.ProposedCellMemoryGB, _ = strconv.Atoi(cellMemory)
	w.input.ProposedCellCPU, _ = strconv.Atoi(cellCPU)
	w.input.ProposedCellDiskGB, _ = strconv.Atoi(cellDisk)

	// Step 2: Cell count - use Input with better guidance
	cellCount := strconv.Itoa(w.input.ProposedCellCount)

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Proposed cell count").
				Description("Type a number and press Enter to continue").
				Placeholder("e.g., 20").
				CharLimit(5).
				Value(&cellCount).
				Validate(validatePositiveInt),
		).Title("Step 2: Cell Count").
			Description("How many Diego cells do you want in your scenario?"),
	).WithTheme(huh.ThemeBase())

	if err := form2.Run(); err != nil {
		return err
	}

	w.input.ProposedCellCount, _ = strconv.Atoi(cellCount)

	// Step 3: Overhead settings with common presets
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

	overhead := fmt.Sprintf("%.0f", w.input.OverheadPct)
	haAdmission := strconv.Itoa(w.input.HAAdmissionPct)

	form3 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Memory overhead").
				Description("Diego cell memory overhead percentage").
				Options(overheadOptions...).
				Value(&overhead),
			huh.NewSelect[string]().
				Title("HA admission control").
				Description("vSphere HA cluster reservation percentage").
				Options(haOptions...).
				Value(&haAdmission),
		).Title("Step 3: Overhead & HA").
			Description("Configure overhead and high availability settings"),
	).WithTheme(huh.ThemeBase())

	if err := form3.Run(); err != nil {
		return err
	}

	var overheadFloat float64
	fmt.Sscanf(overhead, "%f", &overheadFloat)
	w.input.OverheadPct = overheadFloat
	w.input.HAAdmissionPct, _ = strconv.Atoi(haAdmission)

	return nil
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
