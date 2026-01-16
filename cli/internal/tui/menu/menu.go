// ABOUTME: Data source selection menu for TUI startup
// ABOUTME: Allows user to choose between vSphere, JSON file, or manual input

package menu

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// DataSource represents the selected data source
type DataSource int

const (
	SourceVSphere DataSource = iota
	SourceJSON
	SourceManual
)

type option struct {
	label   string
	value   DataSource
	enabled bool
}

// Menu represents the data source selection menu
type Menu struct {
	options  []option
	selected DataSource
}

// New creates a new data source menu
func New(vsphereConfigured bool) *Menu {
	return &Menu{
		options: []option{
			{label: "Live vSphere", value: SourceVSphere, enabled: vsphereConfigured},
			{label: "Load JSON file", value: SourceJSON, enabled: true},
			{label: "Manual input", value: SourceManual, enabled: true},
		},
		selected: SourceVSphere,
	}
}

// Run displays the menu and returns the selected data source
func (m *Menu) Run() (DataSource, error) {
	var options []huh.Option[DataSource]
	for _, opt := range m.options {
		label := opt.label
		if !opt.enabled {
			label = fmt.Sprintf("%s (not configured)", label)
		}
		options = append(options, huh.NewOption(label, opt.value))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[DataSource]().
				Title("Select data source").
				Options(options...).
				Value(&m.selected),
		),
	).WithTheme(huh.ThemeBase())

	if err := form.Run(); err != nil {
		return 0, err
	}

	// Check if selected option is enabled
	for _, opt := range m.options {
		if opt.value == m.selected && !opt.enabled {
			return 0, fmt.Errorf("vSphere is not configured")
		}
	}

	return m.selected, nil
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
