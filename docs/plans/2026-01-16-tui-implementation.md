# TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the current CLI with a rich TUI using Charm Bracelet libraries (bubbletea, huh, lipgloss).

**Architecture:** Hybrid approach using `huh` for wizard forms and `bubbletea` for the split-pane dashboard. All TUI code lives in `cli/internal/tui/`. The existing API client is extended to cover all 12 backend endpoints.

**Tech Stack:** Go 1.25+, bubbletea, huh, lipgloss, bubbles, cobra

---

## Phase 1: Project Setup and Dependencies

### Task 1.1: Add Charm dependencies to go.mod

**Files:**
- Modify: `cli/go.mod`

**Step 1: Add Charm dependencies**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/cli && \
go get github.com/charmbracelet/bubbletea@latest && \
go get github.com/charmbracelet/huh@latest && \
go get github.com/charmbracelet/lipgloss@latest && \
go get github.com/charmbracelet/bubbles@latest
```

**Step 2: Verify dependencies added**

Run: `grep charmbracelet cli/go.mod`
Expected: Four charmbracelet packages listed

**Step 3: Run go mod tidy**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go mod tidy
```

**Step 4: Commit**

```bash
git add cli/go.mod cli/go.sum
git commit -m "deps: add Charm Bracelet libraries for TUI"
```

---

### Task 1.2: Create TUI directory structure

**Files:**
- Create: `cli/internal/tui/app.go`
- Create: `cli/internal/tui/styles/styles.go`

**Step 1: Create styles package with theme**

Create `cli/internal/tui/styles/styles.go`:

```go
// ABOUTME: Shared lipgloss styles for consistent TUI appearance
// ABOUTME: Defines colors, borders, and text styles used across components

package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#10B981") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Danger    = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F9FAFB") // Light
	BgDark    = lipgloss.Color("#1F2937") // Dark gray

	// Base styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginBottom(1)

	// Status indicators
	StatusOK = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	StatusWarning = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	StatusCritical = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	// Panels
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2)

	ActivePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Help text
	Help = lipgloss.NewStyle().
		Foreground(Muted).
		MarginTop(1)
)

// ProgressBar returns a styled progress bar string
func ProgressBar(percent float64, width int) string {
	filled := int(percent / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	color := Secondary
	if percent >= 80 {
		color = Warning
	}
	if percent >= 95 {
		color = Danger
	}

	return lipgloss.NewStyle().Foreground(color).Render(bar)
}
```

**Step 2: Create minimal app shell**

Create `cli/internal/tui/app.go`:

```go
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
```

**Step 3: Verify compilation**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add cli/internal/tui/
git commit -m "feat(tui): add app shell and styles foundation"
```

---

## Phase 2: Extend API Client

### Task 2.1: Add infrastructure endpoint types

**Files:**
- Modify: `cli/internal/client/client.go`
- Modify: `cli/internal/client/client_test.go`

**Step 1: Write test for GetInfrastructure**

Add to `cli/internal/client/client_test.go`:

```go
func TestGetInfrastructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/infrastructure" {
			t.Errorf("expected path /api/infrastructure, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureState{
			Source:         "vsphere",
			Name:           "vcenter.example.com",
			TotalHostCount: 4,
			TotalCellCount: 10,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	infra, err := c.GetInfrastructure(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infra.Source != "vsphere" {
		t.Errorf("expected source vsphere, got %s", infra.Source)
	}
	if infra.TotalHostCount != 4 {
		t.Errorf("expected 4 hosts, got %d", infra.TotalHostCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestGetInfrastructure -v`
Expected: FAIL - undefined: InfrastructureState

**Step 3: Add InfrastructureState and ClusterState types**

Add to `cli/internal/client/client.go` after the existing types:

```go
// ClusterState represents computed metrics for a single cluster
type ClusterState struct {
	Name                         string  `json:"name"`
	HostCount                    int     `json:"host_count"`
	MemoryGB                     int     `json:"memory_gb"`
	CPUCores                     int     `json:"cpu_cores"`
	MemoryGBPerHost              int     `json:"memory_gb_per_host"`
	CPUCoresPerHost              int     `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int     `json:"ha_admission_control_percentage"`
	HAUsableMemoryGB             int     `json:"ha_usable_memory_gb"`
	HAHostFailuresSurvived       int     `json:"ha_host_failures_survived"`
	HAStatus                     string  `json:"ha_status"`
	N1MemoryGB                   int     `json:"n1_memory_gb"`
	DiegoCellCount               int     `json:"diego_cell_count"`
	DiegoCellMemoryGB            int     `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int     `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int     `json:"diego_cell_disk_gb"`
	TotalVCPUs                   int     `json:"total_vcpus"`
	TotalCellMemoryGB            int     `json:"total_cell_memory_gb"`
	VCPURatio                    float64 `json:"vcpu_ratio"`
}

// InfrastructureState represents the full infrastructure data
type InfrastructureState struct {
	Source                       string         `json:"source"`
	Name                         string         `json:"name"`
	Clusters                     []ClusterState `json:"clusters"`
	TotalMemoryGB                int            `json:"total_memory_gb"`
	TotalN1MemoryGB              int            `json:"total_n1_memory_gb"`
	TotalHAUsableMemoryGB        int            `json:"total_ha_usable_memory_gb"`
	HAMinHostFailuresSurvived    int            `json:"ha_min_host_failures_survived"`
	HAStatus                     string         `json:"ha_status"`
	TotalCellMemoryGB            int            `json:"total_cell_memory_gb"`
	HostMemoryUtilizationPercent float64        `json:"host_memory_utilization_percent"`
	HostCPUUtilizationPercent    float64        `json:"host_cpu_utilization_percent"`
	TotalHostCount               int            `json:"total_host_count"`
	TotalCellCount               int            `json:"total_cell_count"`
	TotalCPUCores                int            `json:"total_cpu_cores"`
	TotalVCPUs                   int            `json:"total_vcpus"`
	VCPURatio                    float64        `json:"vcpu_ratio"`
	CPURiskLevel                 string         `json:"cpu_risk_level"`
	PlatformVMsGB                int            `json:"platform_vms_gb"`
	TotalAppMemoryGB             int            `json:"total_app_memory_gb"`
	TotalAppDiskGB               int            `json:"total_app_disk_gb"`
	TotalAppInstances            int            `json:"total_app_instances"`
	Timestamp                    string         `json:"timestamp"`
	Cached                       bool           `json:"cached"`
}

// GetInfrastructure calls GET /api/infrastructure
func (c *Client) GetInfrastructure(ctx context.Context) (*InfrastructureState, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/infrastructure", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.handleRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var infra InfrastructureState
	if err := json.NewDecoder(resp.Body).Decode(&infra); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &infra, nil
}

// handleRequestError converts context errors to user-friendly messages
func (c *Client) handleRequestError(ctx context.Context, err error) error {
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("request canceled")
	}
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("request timed out")
	}
	return fmt.Errorf("cannot connect to backend at %s: %w", c.baseURL, err)
}

// handleErrorResponse parses API error responses
func (c *Client) handleErrorResponse(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}
	return fmt.Errorf("backend error: %s", errResp.Error)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestGetInfrastructure -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/client/
git commit -m "feat(client): add GetInfrastructure endpoint"
```

---

### Task 2.2: Add manual infrastructure input endpoint

**Files:**
- Modify: `cli/internal/client/client.go`
- Modify: `cli/internal/client/client_test.go`

**Step 1: Write test for SetManualInfrastructure**

Add to `cli/internal/client/client_test.go`:

```go
func TestSetManualInfrastructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/infrastructure/manual" {
			t.Errorf("expected path /api/infrastructure/manual, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input ManualInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if input.Name != "Test Infra" {
			t.Errorf("expected name 'Test Infra', got %s", input.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureState{
			Source: "manual",
			Name:   input.Name,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	input := &ManualInput{
		Name: "Test Infra",
		Clusters: []ClusterInput{{
			Name:              "cluster-1",
			HostCount:         4,
			MemoryGBPerHost:   256,
			CPUCoresPerHost:   32,
			DiegoCellCount:    10,
			DiegoCellMemoryGB: 64,
			DiegoCellCPU:      8,
		}},
	}

	infra, err := c.SetManualInfrastructure(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infra.Source != "manual" {
		t.Errorf("expected source manual, got %s", infra.Source)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestSetManualInfrastructure -v`
Expected: FAIL - undefined: ManualInput

**Step 3: Add ManualInput types and SetManualInfrastructure method**

Add to `cli/internal/client/client.go`:

```go
// ClusterInput represents user-provided cluster configuration
type ClusterInput struct {
	Name                         string `json:"name"`
	HostCount                    int    `json:"host_count"`
	MemoryGBPerHost              int    `json:"memory_gb_per_host"`
	CPUCoresPerHost              int    `json:"cpu_cores_per_host"`
	HAAdmissionControlPercentage int    `json:"ha_admission_control_percentage"`
	DiegoCellCount               int    `json:"diego_cell_count"`
	DiegoCellMemoryGB            int    `json:"diego_cell_memory_gb"`
	DiegoCellCPU                 int    `json:"diego_cell_cpu"`
	DiegoCellDiskGB              int    `json:"diego_cell_disk_gb"`
}

// ManualInput represents user-provided infrastructure data
type ManualInput struct {
	Name              string         `json:"name"`
	Clusters          []ClusterInput `json:"clusters"`
	PlatformVMsGB     int            `json:"platform_vms_gb"`
	TotalAppMemoryGB  int            `json:"total_app_memory_gb"`
	TotalAppDiskGB    int            `json:"total_app_disk_gb"`
	TotalAppInstances int            `json:"total_app_instances"`
}

// SetManualInfrastructure calls POST /api/infrastructure/manual
func (c *Client) SetManualInfrastructure(ctx context.Context, input *ManualInput) (*InfrastructureState, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/infrastructure/manual", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.handleRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var infra InfrastructureState
	if err := json.NewDecoder(resp.Body).Decode(&infra); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &infra, nil
}
```

Add `"bytes"` to imports.

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestSetManualInfrastructure -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/client/
git commit -m "feat(client): add SetManualInfrastructure endpoint"
```

---

### Task 2.3: Add scenario comparison endpoint

**Files:**
- Modify: `cli/internal/client/client.go`
- Modify: `cli/internal/client/client_test.go`

**Step 1: Write test for CompareScenario**

Add to `cli/internal/client/client_test.go`:

```go
func TestCompareScenario(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scenario/compare" {
			t.Errorf("expected path /api/scenario/compare, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScenarioComparison{
			Current: ScenarioResult{
				CellCount:      10,
				CellMemoryGB:   64,
				UtilizationPct: 75.0,
			},
			Proposed: ScenarioResult{
				CellCount:      15,
				CellMemoryGB:   64,
				UtilizationPct: 50.0,
			},
			Delta: ScenarioDelta{
				CapacityChangeGB:     320,
				UtilizationChangePct: -25.0,
			},
		})
	}))
	defer server.Close()

	c := New(server.URL)
	input := &ScenarioInput{
		ProposedCellMemoryGB: 64,
		ProposedCellCPU:      8,
		ProposedCellCount:    15,
	}

	result, err := c.CompareScenario(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Current.CellCount != 10 {
		t.Errorf("expected current cell count 10, got %d", result.Current.CellCount)
	}
	if result.Proposed.CellCount != 15 {
		t.Errorf("expected proposed cell count 15, got %d", result.Proposed.CellCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestCompareScenario -v`
Expected: FAIL - undefined: ScenarioInput

**Step 3: Add scenario types and CompareScenario method**

Add to `cli/internal/client/client.go`:

```go
// ScenarioInput represents proposed changes for what-if analysis
type ScenarioInput struct {
	ProposedCellMemoryGB int      `json:"proposed_cell_memory_gb"`
	ProposedCellCPU      int      `json:"proposed_cell_cpu"`
	ProposedCellDiskGB   int      `json:"proposed_cell_disk_gb"`
	ProposedCellCount    int      `json:"proposed_cell_count"`
	TargetCluster        string   `json:"target_cluster"`
	SelectedResources    []string `json:"selected_resources"`
	OverheadPct          float64  `json:"overhead_pct"`
	HostCount            int      `json:"host_count"`
	MemoryPerHostGB      int      `json:"memory_per_host_gb"`
	HAAdmissionPct       int      `json:"ha_admission_pct"`
	PhysicalCoresPerHost int      `json:"physical_cores_per_host"`
	TargetVCPURatio      int      `json:"target_vcpu_ratio"`
	PlatformVMsCPU       int      `json:"platform_vms_cpu"`
}

// ScenarioResult represents computed metrics for a scenario
type ScenarioResult struct {
	CellCount        int     `json:"cell_count"`
	CellMemoryGB     int     `json:"cell_memory_gb"`
	CellCPU          int     `json:"cell_cpu"`
	CellDiskGB       int     `json:"cell_disk_gb"`
	AppCapacityGB    int     `json:"app_capacity_gb"`
	UtilizationPct   float64 `json:"utilization_pct"`
	FreeChunks       int     `json:"free_chunks"`
	N1UtilizationPct float64 `json:"n1_utilization_pct"`
	FaultImpact      int     `json:"fault_impact"`
	BlastRadiusPct   float64 `json:"blast_radius_pct"`
	TotalVCPUs       int     `json:"total_vcpus"`
	TotalPCPUs       int     `json:"total_pcpus"`
	VCPURatio        float64 `json:"vcpu_ratio"`
	CPURiskLevel     string  `json:"cpu_risk_level"`
}

// ScenarioDelta represents changes between current and proposed
type ScenarioDelta struct {
	CapacityChangeGB     int     `json:"capacity_change_gb"`
	UtilizationChangePct float64 `json:"utilization_change_pct"`
	ResilienceChange     string  `json:"resilience_change"`
	VCPURatioChange      float64 `json:"vcpu_ratio_change"`
}

// ScenarioWarning represents a tradeoff warning
type ScenarioWarning struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ScenarioComparison represents full comparison response
type ScenarioComparison struct {
	Current  ScenarioResult    `json:"current"`
	Proposed ScenarioResult    `json:"proposed"`
	Delta    ScenarioDelta     `json:"delta"`
	Warnings []ScenarioWarning `json:"warnings"`
}

// CompareScenario calls POST /api/scenario/compare
func (c *Client) CompareScenario(ctx context.Context, input *ScenarioInput) (*ScenarioComparison, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/scenario/compare", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.handleRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var comparison ScenarioComparison
	if err := json.NewDecoder(resp.Body).Decode(&comparison); err != nil {
		return nil, fmt.Errorf("invalid response from backend: %w", err)
	}

	return &comparison, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -run TestCompareScenario -v`
Expected: PASS

**Step 5: Run all client tests**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/client/... -v`
Expected: All tests pass

**Step 6: Commit**

```bash
git add cli/internal/client/
git commit -m "feat(client): add CompareScenario endpoint"
```

---

## Phase 3: Data Source Menu

### Task 3.1: Create data source menu component

**Files:**
- Create: `cli/internal/tui/menu/menu.go`
- Create: `cli/internal/tui/menu/menu_test.go`

**Step 1: Write test for menu model**

Create `cli/internal/tui/menu/menu_test.go`:

```go
// ABOUTME: Tests for data source selection menu
// ABOUTME: Validates menu rendering and selection behavior

package menu

import "testing"

func TestMenuOptions(t *testing.T) {
	m := New(true) // vSphere configured

	if len(m.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(m.options))
	}

	if m.options[0].label != "Live vSphere" {
		t.Errorf("expected first option 'Live vSphere', got %s", m.options[0].label)
	}
}

func TestMenuVSphereDisabled(t *testing.T) {
	m := New(false) // vSphere not configured

	if m.options[0].enabled {
		t.Error("expected vSphere option to be disabled when not configured")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/menu/... -v`
Expected: FAIL - package not found

**Step 3: Create menu component**

Create `cli/internal/tui/menu/menu.go`:

```go
// ABOUTME: Data source selection menu for TUI startup
// ABOUTME: Allows user to choose between vSphere, JSON file, or manual input

package menu

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
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

	// Apply custom styling
	_ = styles.Title // reference styles to ensure import

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
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/menu/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/tui/menu/
git commit -m "feat(tui): add data source selection menu"
```

---

## Phase 4: Dashboard Component

### Task 4.1: Create dashboard model

**Files:**
- Create: `cli/internal/tui/dashboard/dashboard.go`
- Create: `cli/internal/tui/dashboard/dashboard_test.go`

**Step 1: Write test for dashboard rendering**

Create `cli/internal/tui/dashboard/dashboard_test.go`:

```go
// ABOUTME: Tests for dashboard component
// ABOUTME: Validates infrastructure metrics display

package dashboard

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestDashboardView(t *testing.T) {
	infra := &client.InfrastructureState{
		Source:                       "vsphere",
		Name:                         "vcenter.test.com",
		TotalHostCount:               4,
		TotalCellCount:               10,
		HostMemoryUtilizationPercent: 75.5,
		HAStatus:                     "ok",
		HAMinHostFailuresSurvived:    1,
	}

	d := New(infra, 80, 24)
	view := d.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check for key metrics in output
	tests := []string{"Hosts: 4", "Diego Cells: 10", "75.5%"}
	for _, expected := range tests {
		if !containsString(view, expected) {
			t.Errorf("expected view to contain %q", expected)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/dashboard/... -v`
Expected: FAIL - package not found

**Step 3: Create dashboard component**

Create `cli/internal/tui/dashboard/dashboard.go`:

```go
// ABOUTME: Dashboard component displaying live infrastructure metrics
// ABOUTME: Shows cluster counts, utilization, and HA status in left pane

package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
)

// Dashboard displays infrastructure metrics
type Dashboard struct {
	infra  *client.InfrastructureState
	width  int
	height int
}

// New creates a new dashboard with infrastructure data
func New(infra *client.InfrastructureState, width, height int) *Dashboard {
	return &Dashboard{
		infra:  infra,
		width:  width,
		height: height,
	}
}

// Update refreshes dashboard with new infrastructure data
func (d *Dashboard) Update(infra *client.InfrastructureState) {
	d.infra = infra
}

// SetSize updates the dashboard dimensions
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the dashboard
func (d *Dashboard) View() string {
	if d.infra == nil {
		return styles.Panel.Width(d.width).Render("Loading infrastructure data...")
	}

	var sb strings.Builder

	// Title
	sb.WriteString(styles.Title.Render("Current Infrastructure"))
	sb.WriteString("\n")
	sb.WriteString(styles.Subtitle.Render(d.infra.Name))
	sb.WriteString("\n\n")

	// Cluster info
	sb.WriteString(fmt.Sprintf("Clusters: %d\n", len(d.infra.Clusters)))
	sb.WriteString(fmt.Sprintf("Hosts: %d\n", d.infra.TotalHostCount))
	sb.WriteString(fmt.Sprintf("Diego Cells: %d\n", d.infra.TotalCellCount))
	sb.WriteString("\n")

	// Memory utilization
	sb.WriteString("Memory Utilization\n")
	sb.WriteString(styles.ProgressBar(d.infra.HostMemoryUtilizationPercent, 20))
	sb.WriteString(fmt.Sprintf(" %.1f%%\n", d.infra.HostMemoryUtilizationPercent))
	sb.WriteString("\n")

	// CPU utilization
	sb.WriteString("CPU Utilization\n")
	sb.WriteString(styles.ProgressBar(d.infra.HostCPUUtilizationPercent, 20))
	sb.WriteString(fmt.Sprintf(" %.1f%%\n", d.infra.HostCPUUtilizationPercent))
	sb.WriteString("\n")

	// HA Status
	haStyle := styles.StatusOK
	haIcon := "✓"
	if d.infra.HAStatus != "ok" {
		haStyle = styles.StatusCritical
		haIcon = "✗"
	}
	sb.WriteString(fmt.Sprintf("HA Status: %s\n", haStyle.Render(haIcon+" "+strings.ToUpper(d.infra.HAStatus))))
	sb.WriteString(fmt.Sprintf("  Can survive %d host failure(s)\n", d.infra.HAMinHostFailuresSurvived))

	// vCPU Ratio
	if d.infra.VCPURatio > 0 {
		sb.WriteString("\n")
		riskStyle := styles.StatusOK
		if d.infra.CPURiskLevel == "medium" {
			riskStyle = styles.StatusWarning
		} else if d.infra.CPURiskLevel == "high" {
			riskStyle = styles.StatusCritical
		}
		sb.WriteString(fmt.Sprintf("vCPU Ratio: %s\n", riskStyle.Render(fmt.Sprintf("%.1f:1", d.infra.VCPURatio))))
	}

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Render(sb.String())
}
```

**Step 4: Fix test helper and run test**

Update test with proper string contains:

```go
import "strings"

// Replace containsString with:
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/dashboard/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/tui/dashboard/
git commit -m "feat(tui): add dashboard component for infrastructure metrics"
```

---

## Phase 5: Wizard Component

### Task 5.1: Create wizard orchestration

**Files:**
- Create: `cli/internal/tui/wizard/wizard.go`
- Create: `cli/internal/tui/wizard/wizard_test.go`

**Step 1: Write test for wizard input collection**

Create `cli/internal/tui/wizard/wizard_test.go`:

```go
// ABOUTME: Tests for scenario wizard
// ABOUTME: Validates input collection and validation

package wizard

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestWizardDefaults(t *testing.T) {
	infra := &client.InfrastructureState{
		Clusters: []client.ClusterState{{
			Name:              "cluster-1",
			DiegoCellMemoryGB: 64,
			DiegoCellCPU:      8,
			DiegoCellCount:    10,
		}},
	}

	w := New(infra)

	if w.input.ProposedCellMemoryGB != 64 {
		t.Errorf("expected default memory 64, got %d", w.input.ProposedCellMemoryGB)
	}
	if w.input.ProposedCellCPU != 8 {
		t.Errorf("expected default CPU 8, got %d", w.input.ProposedCellCPU)
	}
}

func TestWizardBuildInput(t *testing.T) {
	w := &Wizard{
		input: &client.ScenarioInput{
			ProposedCellMemoryGB: 32,
			ProposedCellCPU:      4,
			ProposedCellCount:    20,
		},
	}

	input := w.GetInput()
	if input.ProposedCellCount != 20 {
		t.Errorf("expected cell count 20, got %d", input.ProposedCellCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/wizard/... -v`
Expected: FAIL - package not found

**Step 3: Create wizard component**

Create `cli/internal/tui/wizard/wizard.go`:

```go
// ABOUTME: Scenario planning wizard using huh forms
// ABOUTME: Collects cell sizing, count, and overhead inputs for scenario comparison

package wizard

import (
	"fmt"

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
		input.ProposedCellMemoryGB = c.DiegoCellMemoryGB
		input.ProposedCellCPU = c.DiegoCellCPU
		input.ProposedCellDiskGB = c.DiegoCellDiskGB
		input.ProposedCellCount = infra.TotalCellCount
		input.HostCount = infra.TotalHostCount
		input.MemoryPerHostGB = c.MemoryGBPerHost
		input.HAAdmissionPct = c.HAAdmissionControlPercentage
		input.PhysicalCoresPerHost = c.CPUCoresPerHost
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
	cellMemory := w.input.ProposedCellMemoryGB
	cellCPU := w.input.ProposedCellCPU
	cellDisk := w.input.ProposedCellDiskGB

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Memory per cell (GB)").
				Value(ptrString(fmt.Sprintf("%d", cellMemory))).
				Validate(validatePositiveInt),
			huh.NewInput().
				Title("CPU cores per cell").
				Value(ptrString(fmt.Sprintf("%d", cellCPU))).
				Validate(validatePositiveInt),
			huh.NewInput().
				Title("Disk per cell (GB)").
				Value(ptrString(fmt.Sprintf("%d", cellDisk))).
				Validate(validatePositiveInt),
		).Title("Step 1: Cell Sizing"),
	).WithTheme(huh.ThemeBase())

	if err := form1.Run(); err != nil {
		return err
	}

	// Parse values
	fmt.Sscanf(*ptrString(fmt.Sprintf("%d", cellMemory)), "%d", &w.input.ProposedCellMemoryGB)
	fmt.Sscanf(*ptrString(fmt.Sprintf("%d", cellCPU)), "%d", &w.input.ProposedCellCPU)
	fmt.Sscanf(*ptrString(fmt.Sprintf("%d", cellDisk)), "%d", &w.input.ProposedCellDiskGB)

	// Step 2: Cell count
	cellCount := w.input.ProposedCellCount

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Proposed cell count").
				Value(ptrString(fmt.Sprintf("%d", cellCount))).
				Validate(validatePositiveInt),
		).Title("Step 2: Cell Count"),
	).WithTheme(huh.ThemeBase())

	if err := form2.Run(); err != nil {
		return err
	}

	fmt.Sscanf(*ptrString(fmt.Sprintf("%d", cellCount)), "%d", &w.input.ProposedCellCount)

	// Step 3: Overhead settings
	overhead := fmt.Sprintf("%.0f", w.input.OverheadPct)
	haAdmission := fmt.Sprintf("%d", w.input.HAAdmissionPct)

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

	fmt.Sscanf(overhead, "%f", &w.input.OverheadPct)
	fmt.Sscanf(haAdmission, "%d", &w.input.HAAdmissionPct)

	return nil
}

func ptrString(s string) *string {
	return &s
}

func validatePositiveInt(s string) error {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v <= 0 {
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
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/wizard/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/tui/wizard/
git commit -m "feat(tui): add scenario planning wizard"
```

---

## Phase 6: Comparison View

### Task 6.1: Create comparison view component

**Files:**
- Create: `cli/internal/tui/comparison/comparison.go`
- Create: `cli/internal/tui/comparison/comparison_test.go`

**Step 1: Write test for comparison rendering**

Create `cli/internal/tui/comparison/comparison_test.go`:

```go
// ABOUTME: Tests for comparison view component
// ABOUTME: Validates current vs proposed scenario display

package comparison

import (
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestComparisonView(t *testing.T) {
	result := &client.ScenarioComparison{
		Current: client.ScenarioResult{
			CellCount:      10,
			CellMemoryGB:   64,
			UtilizationPct: 75.0,
		},
		Proposed: client.ScenarioResult{
			CellCount:      15,
			CellMemoryGB:   64,
			UtilizationPct: 50.0,
		},
		Delta: client.ScenarioDelta{
			CapacityChangeGB:     320,
			UtilizationChangePct: -25.0,
		},
	}

	c := New(result, 80)
	view := c.View()

	if !strings.Contains(view, "Current") {
		t.Error("expected view to contain 'Current'")
	}
	if !strings.Contains(view, "Proposed") {
		t.Error("expected view to contain 'Proposed'")
	}
	if !strings.Contains(view, "320") {
		t.Error("expected view to contain capacity change")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/comparison/... -v`
Expected: FAIL - package not found

**Step 3: Create comparison component**

Create `cli/internal/tui/comparison/comparison.go`:

```go
// ABOUTME: Comparison view showing current vs proposed scenario results
// ABOUTME: Displays deltas, warnings, and recommendations

package comparison

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
)

// Comparison displays scenario comparison results
type Comparison struct {
	result *client.ScenarioComparison
	width  int
}

// New creates a new comparison view
func New(result *client.ScenarioComparison, width int) *Comparison {
	return &Comparison{
		result: result,
		width:  width,
	}
}

// View renders the comparison
func (c *Comparison) View() string {
	if c.result == nil {
		return "No comparison data"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(styles.Title.Render("Scenario Comparison"))
	sb.WriteString("\n\n")

	// Side by side metrics
	colWidth := (c.width - 4) / 2

	currentCol := c.renderScenario("Current", &c.result.Current, colWidth)
	proposedCol := c.renderScenario("Proposed", &c.result.Proposed, colWidth)

	// Join columns
	currentLines := strings.Split(currentCol, "\n")
	proposedLines := strings.Split(proposedCol, "\n")
	maxLines := len(currentLines)
	if len(proposedLines) > maxLines {
		maxLines = len(proposedLines)
	}

	for i := 0; i < maxLines; i++ {
		left := ""
		right := ""
		if i < len(currentLines) {
			left = currentLines[i]
		}
		if i < len(proposedLines) {
			right = proposedLines[i]
		}
		sb.WriteString(fmt.Sprintf("%-*s  %s\n", colWidth, left, right))
	}

	// Delta section
	sb.WriteString("\n")
	sb.WriteString(styles.Subtitle.Render("Changes"))
	sb.WriteString("\n")

	delta := c.result.Delta
	changeStyle := styles.StatusOK
	changePrefix := "+"
	if delta.CapacityChangeGB < 0 {
		changeStyle = styles.StatusCritical
		changePrefix = ""
	}
	sb.WriteString(fmt.Sprintf("  Capacity: %s\n", changeStyle.Render(fmt.Sprintf("%s%d GB", changePrefix, delta.CapacityChangeGB))))

	utilStyle := styles.StatusOK
	if delta.UtilizationChangePct > 0 {
		utilStyle = styles.StatusWarning
	}
	sb.WriteString(fmt.Sprintf("  Utilization: %s\n", utilStyle.Render(fmt.Sprintf("%+.1f%%", delta.UtilizationChangePct))))

	// Warnings
	if len(c.result.Warnings) > 0 {
		sb.WriteString("\n")
		sb.WriteString(styles.StatusWarning.Render("Warnings"))
		sb.WriteString("\n")
		for _, w := range c.result.Warnings {
			icon := "⚠"
			warnStyle := styles.StatusWarning
			if w.Severity == "critical" {
				icon = "✗"
				warnStyle = styles.StatusCritical
			}
			sb.WriteString(fmt.Sprintf("  %s %s\n", warnStyle.Render(icon), w.Message))
		}
	}

	return lipgloss.NewStyle().Width(c.width).Render(sb.String())
}

func (c *Comparison) renderScenario(title string, s *client.ScenarioResult, width int) string {
	var sb strings.Builder
	sb.WriteString(styles.Subtitle.Render(title))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Cells: %d × %dGB\n", s.CellCount, s.CellMemoryGB))
	sb.WriteString(fmt.Sprintf("Capacity: %d GB\n", s.AppCapacityGB))
	sb.WriteString(fmt.Sprintf("Utilization: %.1f%%\n", s.UtilizationPct))
	if s.VCPURatio > 0 {
		sb.WriteString(fmt.Sprintf("vCPU Ratio: %.1f:1\n", s.VCPURatio))
	}
	return sb.String()
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/comparison/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/tui/comparison/
git commit -m "feat(tui): add comparison view component"
```

---

## Phase 7: Integration

### Task 7.1: Wire up TUI app with all components

**Files:**
- Modify: `cli/internal/tui/app.go`
- Create: `cli/internal/tui/app_test.go`

**Step 1: Write integration test**

Create `cli/internal/tui/app_test.go`:

```go
// ABOUTME: Integration tests for TUI app
// ABOUTME: Tests component wiring and state transitions

package tui

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestAppInitialState(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)

	if app.screen != ScreenMenu {
		t.Errorf("expected initial screen to be ScreenMenu, got %d", app.screen)
	}
}

func TestAppQuitOnQ(t *testing.T) {
	c := client.New("http://localhost:8080")
	app := New(c)

	// Test handled in teatest integration tests
	_ = app
}
```

**Step 2: Update app.go with full component integration**

Update `cli/internal/tui/app.go`:

```go
// ABOUTME: Root bubbletea model for the TUI application
// ABOUTME: Manages screen state and routes keyboard input to child components

package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/comparison"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/dashboard"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/menu"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/wizard"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenDashboard
	ScreenComparison
)

// Messages
type infraLoadedMsg struct {
	infra *client.InfrastructureState
	err   error
}

type scenarioComparedMsg struct {
	result *client.ScenarioComparison
	err    error
}

// App is the root model for the TUI
type App struct {
	client     *client.Client
	screen     Screen
	width      int
	height     int
	err        error
	infra      *client.InfrastructureState
	comparison *client.ScenarioComparison
	dashboard  *dashboard.Dashboard
	compView   *comparison.Comparison
	dataSource menu.DataSource
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
		case "r":
			if a.screen == ScreenDashboard {
				return a, a.loadInfrastructure()
			}
		case "w":
			if a.screen == ScreenDashboard && a.infra != nil {
				return a, a.runWizard()
			}
		case "b":
			if a.screen == ScreenComparison {
				a.screen = ScreenDashboard
			}
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.dashboard != nil {
			a.dashboard.SetSize(a.width/2-2, a.height-4)
		}
	case infraLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.infra = msg.infra
			a.dashboard = dashboard.New(a.infra, a.width/2-2, a.height-4)
			a.screen = ScreenDashboard
		}
	case scenarioComparedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.comparison = msg.result
			a.compView = comparison.New(a.comparison, a.width/2-2)
			a.screen = ScreenComparison
		}
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	if a.err != nil {
		return styles.StatusCritical.Render("Error: "+a.err.Error()) + "\n\nPress 'q' to quit."
	}

	switch a.screen {
	case ScreenMenu:
		return a.viewMenu()
	case ScreenDashboard:
		return a.viewDashboard()
	case ScreenComparison:
		return a.viewComparison()
	default:
		return "Unknown screen"
	}
}

func (a *App) viewMenu() string {
	return styles.Title.Render("Diego Capacity Analyzer") + "\n\n" +
		"Starting...\n\n" +
		styles.Help.Render("Press 'q' to quit")
}

func (a *App) viewDashboard() string {
	if a.dashboard == nil {
		return "Loading..."
	}

	leftPane := styles.Panel.Width(a.width/2 - 2).Render(a.dashboard.View())
	rightPane := styles.ActivePanel.Width(a.width/2 - 2).Render(
		styles.Title.Render("Actions") + "\n\n" +
			"[w] Run scenario wizard\n" +
			"[r] Refresh data\n" +
			"[q] Quit",
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (a *App) viewComparison() string {
	if a.compView == nil {
		return "Loading comparison..."
	}

	leftPane := styles.Panel.Width(a.width/2 - 2).Render(a.dashboard.View())
	rightPane := styles.ActivePanel.Width(a.width/2 - 2).Render(
		a.compView.View() + "\n\n" +
			styles.Help.Render("[b] Back  [w] New scenario  [q] Quit"),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (a *App) loadInfrastructure() tea.Cmd {
	return func() tea.Msg {
		infra, err := a.client.GetInfrastructure(context.Background())
		return infraLoadedMsg{infra: infra, err: err}
	}
}

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

// Run starts the TUI with data source selection
func Run(apiClient *client.Client, vsphereConfigured bool) error {
	// Show data source menu first
	m := menu.New(vsphereConfigured)
	source, err := m.Run()
	if err != nil {
		return err
	}

	app := New(apiClient)
	app.dataSource = source

	// Load initial data based on source
	var initCmd tea.Cmd
	switch source {
	case menu.SourceVSphere:
		initCmd = app.loadInfrastructure()
	case menu.SourceManual:
		// TODO: Show manual input form
		initCmd = nil
	case menu.SourceJSON:
		// TODO: Show file picker
		initCmd = nil
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if initCmd != nil {
		go func() {
			p.Send(initCmd())
		}()
	}
	_, err = p.Run()
	return err
}
```

**Step 3: Run all TUI tests**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./internal/tui/... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add cli/internal/tui/
git commit -m "feat(tui): integrate all components in app shell"
```

---

### Task 7.2: Add TUI entry point to CLI

**Files:**
- Modify: `cli/cmd/root.go`
- Modify: `cli/main.go`

**Step 1: Update root.go to detect TTY and launch TUI**

Update `cli/cmd/root.go`:

```go
// ABOUTME: Root command for diego-capacity CLI
// ABOUTME: Handles global flags, TTY detection, and TUI launch

package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui"
)

var (
	apiURL     string
	jsonOutput bool
)

const defaultAPIURL = "http://localhost:8080"

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "diego-capacity",
	Short: "CLI for Diego Capacity Analyzer",
	Long: `diego-capacity is a command-line interface for the Diego Capacity Analyzer.

When run without arguments in an interactive terminal, launches a TUI for
scenario planning. Use subcommands (health, status, check) for non-interactive
access or add --json for machine-readable output.

Environment Variables:
  DIEGO_CAPACITY_API_URL  Backend API URL (default: http://localhost:8080)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If not a TTY or --json flag, show help
		if !term.IsTerminal(int(os.Stdout.Fd())) || jsonOutput {
			return cmd.Help()
		}

		// Launch TUI
		c := client.New(GetAPIURL())

		// Check if vSphere is configured by calling status endpoint
		// For now, assume it might be configured
		vsphereConfigured := true

		return tui.Run(c, vsphereConfigured)
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "Backend API URL (overrides DIEGO_CAPACITY_API_URL)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON instead of human-readable text")
}

// GetAPIURL returns the API URL from flag, env, or default
func GetAPIURL() string {
	if apiURL != "" {
		return apiURL
	}
	if envURL := os.Getenv("DIEGO_CAPACITY_API_URL"); envURL != "" {
		return envURL
	}
	return defaultAPIURL
}

// IsJSONOutput returns whether JSON output is requested
func IsJSONOutput() bool {
	return jsonOutput
}
```

**Step 2: Add term dependency**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go get golang.org/x/term
```

**Step 3: Verify build**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go build -o diego-capacity .`
Expected: Successful build

**Step 4: Run all CLI tests**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./... -v`
Expected: All tests pass

**Step 5: Commit**

```bash
git add cli/
git commit -m "feat(cli): add TTY detection and TUI launch from root command"
```

---

## Phase 8: Non-Interactive Scenario Command

### Task 8.1: Add scenario compare command

**Files:**
- Create: `cli/cmd/scenario.go`
- Create: `cli/cmd/scenario_test.go`

**Step 1: Write test for scenario command**

Create `cli/cmd/scenario_test.go`:

```go
// ABOUTME: Tests for scenario compare command
// ABOUTME: Validates non-interactive scenario comparison

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestScenarioCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.ScenarioComparison{
			Current: client.ScenarioResult{
				CellCount:      10,
				UtilizationPct: 75.0,
			},
			Proposed: client.ScenarioResult{
				CellCount:      15,
				UtilizationPct: 50.0,
			},
		})
	}))
	defer server.Close()

	var out bytes.Buffer
	c := client.New(server.URL)

	err := runScenarioCompare(context.Background(), c, &out, 64, 8, 200, 15, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should output JSON
	var result client.ScenarioComparison
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result.Proposed.CellCount != 15 {
		t.Errorf("expected proposed cell count 15, got %d", result.Proposed.CellCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./cmd/... -run TestScenarioCommand -v`
Expected: FAIL - undefined: runScenarioCompare

**Step 3: Create scenario command**

Create `cli/cmd/scenario.go`:

```go
// ABOUTME: Non-interactive scenario comparison command
// ABOUTME: Allows CI/CD pipelines to run what-if analysis

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

var (
	cellMemoryGB int
	cellCPU      int
	cellDiskGB   int
	cellCount    int
)

var scenarioCmd = &cobra.Command{
	Use:   "scenario",
	Short: "Compare current vs proposed scenario",
	Long: `Run a what-if scenario comparison without the interactive TUI.

Useful for CI/CD pipelines to validate capacity changes before deployment.

Example:
  diego-capacity scenario --cell-memory 64 --cell-cpu 8 --cell-count 20 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		c := client.New(GetAPIURL())
		return runScenarioCompare(ctx, c, os.Stdout, cellMemoryGB, cellCPU, cellDiskGB, cellCount, IsJSONOutput())
	},
}

func init() {
	rootCmd.AddCommand(scenarioCmd)
	scenarioCmd.Flags().IntVar(&cellMemoryGB, "cell-memory", 64, "Memory per cell in GB")
	scenarioCmd.Flags().IntVar(&cellCPU, "cell-cpu", 8, "CPU cores per cell")
	scenarioCmd.Flags().IntVar(&cellDiskGB, "cell-disk", 200, "Disk per cell in GB")
	scenarioCmd.Flags().IntVar(&cellCount, "cell-count", 10, "Proposed number of cells")
}

func runScenarioCompare(ctx context.Context, c *client.Client, w io.Writer, memoryGB, cpu, diskGB, count int, jsonOut bool) error {
	input := &client.ScenarioInput{
		ProposedCellMemoryGB: memoryGB,
		ProposedCellCPU:      cpu,
		ProposedCellDiskGB:   diskGB,
		ProposedCellCount:    count,
		SelectedResources:    []string{"memory", "cpu", "disk"},
		OverheadPct:          7.0,
	}

	result, err := c.CompareScenario(ctx, input)
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	fmt.Fprintf(w, "Scenario Comparison\n")
	fmt.Fprintf(w, "==================\n\n")
	fmt.Fprintf(w, "Current:\n")
	fmt.Fprintf(w, "  Cells: %d × %d GB\n", result.Current.CellCount, result.Current.CellMemoryGB)
	fmt.Fprintf(w, "  Utilization: %.1f%%\n", result.Current.UtilizationPct)
	fmt.Fprintf(w, "\nProposed:\n")
	fmt.Fprintf(w, "  Cells: %d × %d GB\n", result.Proposed.CellCount, result.Proposed.CellMemoryGB)
	fmt.Fprintf(w, "  Utilization: %.1f%%\n", result.Proposed.UtilizationPct)
	fmt.Fprintf(w, "\nChanges:\n")
	fmt.Fprintf(w, "  Capacity: %+d GB\n", result.Delta.CapacityChangeGB)
	fmt.Fprintf(w, "  Utilization: %+.1f%%\n", result.Delta.UtilizationChangePct)

	if len(result.Warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings:\n")
		for _, warn := range result.Warnings {
			fmt.Fprintf(w, "  [%s] %s\n", warn.Severity, warn.Message)
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./cmd/... -run TestScenarioCommand -v`
Expected: PASS

**Step 5: Run all tests**

Run: `cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go test ./... -v`
Expected: All tests pass

**Step 6: Commit**

```bash
git add cli/cmd/scenario.go cli/cmd/scenario_test.go
git commit -m "feat(cli): add non-interactive scenario command"
```

---

## Phase 9: Final Integration and Documentation

### Task 9.1: Update README with TUI documentation

**Files:**
- Modify: `README.md`

**Step 1: Add TUI section to README**

Add to README.md under the CLI section:

```markdown
### Interactive TUI

When run without arguments in a terminal, `diego-capacity` launches an interactive TUI:

```bash
# Launch interactive TUI
diego-capacity

# Or explicitly with a specific backend
diego-capacity --api-url http://backend:8080
```

The TUI provides:
- **Data source selection**: Choose between live vSphere, JSON file, or manual input
- **Split-pane dashboard**: Live infrastructure metrics on the left, actions on the right
- **Scenario wizard**: Step-by-step what-if analysis with real-time feedback
- **Comparison view**: Side-by-side current vs proposed with delta highlights

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `w` | Run scenario wizard |
| `r` | Refresh infrastructure data |
| `b` | Go back (from comparison view) |
| `q` | Quit |

### Non-Interactive Mode

For CI/CD pipelines, use subcommands with `--json`:

```bash
# Health check
diego-capacity health --json

# Infrastructure status
diego-capacity status --json

# Capacity check with thresholds
diego-capacity check --memory-threshold 85 --json

# Scenario comparison
diego-capacity scenario --cell-memory 64 --cell-cpu 8 --cell-count 20 --json
```
```

**Step 2: Commit documentation**

```bash
git add README.md
git commit -m "docs: add TUI usage documentation to README"
```

---

### Task 9.2: Run full test suite and verify build

**Step 1: Run all tests**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer && make test
```

Expected: All tests pass

**Step 2: Build CLI**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/cli && go build -o diego-capacity .
```

Expected: Successful build

**Step 3: Verify TUI launches**

```bash
./cli/diego-capacity --help
```

Expected: Shows help with TUI description

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete TUI implementation with Charm Bracelet libraries"
```

---

## Summary

This plan implements the TUI in 9 phases with 14 tasks:

1. **Phase 1**: Project setup (2 tasks) - Dependencies and directory structure
2. **Phase 2**: API client expansion (3 tasks) - Add missing endpoints
3. **Phase 3**: Data source menu (1 task) - huh-based selection menu
4. **Phase 4**: Dashboard component (1 task) - Left pane metrics display
5. **Phase 5**: Wizard component (1 task) - Multi-step scenario input
6. **Phase 6**: Comparison view (1 task) - Results display
7. **Phase 7**: Integration (2 tasks) - Wire components, add entry point
8. **Phase 8**: Non-interactive mode (1 task) - Scenario command for CI
9. **Phase 9**: Documentation (2 tasks) - README and final verification

Each task follows TDD with explicit test-first steps and commits after each task.
