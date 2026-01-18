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

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/spf13/cobra"
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
	fmt.Fprintf(w, "  Cells: %d x %d GB\n", result.Current.CellCount, result.Current.CellMemoryGB)
	fmt.Fprintf(w, "  Utilization: %.1f%%\n", result.Current.UtilizationPct)
	fmt.Fprintf(w, "\nProposed:\n")
	fmt.Fprintf(w, "  Cells: %d x %d GB\n", result.Proposed.CellCount, result.Proposed.CellMemoryGB)
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
