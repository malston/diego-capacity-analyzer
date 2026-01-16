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

// Run executes the wizard and collects input
func (w *Wizard) Run() error {
	// Step 1: Cell sizing
	cellMemory := strconv.Itoa(w.input.ProposedCellMemoryGB)
	cellCPU := strconv.Itoa(w.input.ProposedCellCPU)
	cellDisk := strconv.Itoa(w.input.ProposedCellDiskGB)

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Memory per cell (GB)").
				Value(&cellMemory).
				Validate(validatePositiveInt),
			huh.NewInput().
				Title("CPU cores per cell").
				Value(&cellCPU).
				Validate(validatePositiveInt),
			huh.NewInput().
				Title("Disk per cell (GB)").
				Value(&cellDisk).
				Validate(validatePositiveInt),
		).Title("Step 1: Cell Sizing"),
	).WithTheme(huh.ThemeBase())

	if err := form1.Run(); err != nil {
		return err
	}

	// Parse values from step 1
	w.input.ProposedCellMemoryGB, _ = strconv.Atoi(cellMemory)
	w.input.ProposedCellCPU, _ = strconv.Atoi(cellCPU)
	w.input.ProposedCellDiskGB, _ = strconv.Atoi(cellDisk)

	// Step 2: Cell count
	cellCount := strconv.Itoa(w.input.ProposedCellCount)

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Proposed cell count").
				Value(&cellCount).
				Validate(validatePositiveInt),
		).Title("Step 2: Cell Count"),
	).WithTheme(huh.ThemeBase())

	if err := form2.Run(); err != nil {
		return err
	}

	w.input.ProposedCellCount, _ = strconv.Atoi(cellCount)

	// Step 3: Overhead settings
	overhead := fmt.Sprintf("%.0f", w.input.OverheadPct)
	haAdmission := strconv.Itoa(w.input.HAAdmissionPct)

	form3 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Memory overhead %").
				Value(&overhead).
				Validate(validatePercentage),
			huh.NewInput().
				Title("HA admission control %").
				Value(&haAdmission).
				Validate(validatePercentage),
		).Title("Step 3: Overhead & HA"),
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
