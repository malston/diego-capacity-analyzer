// ABOUTME: Check command for diego-capacity CLI
// ABOUTME: Validates capacity thresholds for CI/CD pipelines

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
	n1Threshold     int
	memoryThreshold int
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check capacity thresholds",
	Long: `Check capacity thresholds and exit non-zero if any are exceeded.

Exit codes:
  0 - All checks passed
  1 - One or more thresholds exceeded
  2 - Error (connectivity, no data, invalid input)`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		exitCode := runCheck(ctx, os.Stdout)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().IntVar(&n1Threshold, "n1-threshold", 85, "N-1 capacity threshold percentage")
	checkCmd.Flags().IntVar(&memoryThreshold, "memory-threshold", 90, "Memory utilization threshold percentage")
}

// checkResult represents the result of a single threshold check
type checkResult struct {
	name      string
	value     float64
	threshold float64
	unit      string
	passed    bool
}

// runCheck executes the threshold checks and returns exit code
func runCheck(ctx context.Context, w io.Writer) int {
	if err := validateThresholds(n1Threshold, memoryThreshold); err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return 2
	}

	url := GetAPIURL()
	c := client.New(url)

	resp, err := c.InfrastructureStatus(ctx)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return 2
	}

	if !resp.HasData {
		fmt.Fprintln(w, "Error: no infrastructure data. Load data via UI or API first.")
		return 2
	}

	results := performChecks(resp)

	if IsJSONOutput() {
		fmt.Fprintln(w, formatCheckJSON(results))
	} else {
		fmt.Fprintln(w, formatCheckHuman(results))
	}

	_, failed := countResults(results)
	if failed > 0 {
		return 1
	}
	return 0
}

// validateThresholds ensures threshold values are valid
func validateThresholds(n1, memory int) error {
	if n1 < 0 || n1 > 100 {
		return fmt.Errorf("--n1-threshold must be between 0 and 100")
	}
	if memory < 0 || memory > 100 {
		return fmt.Errorf("--memory-threshold must be between 0 and 100")
	}
	return nil
}

// performChecks runs all threshold checks against the infrastructure status
func performChecks(resp *client.InfrastructureStatus) []checkResult {
	var results []checkResult

	// N-1 capacity check
	n1Check := checkResult{
		name:      "N-1 capacity",
		value:     resp.N1CapacityPercent,
		threshold: float64(n1Threshold),
		unit:      "%",
		passed:    resp.N1CapacityPercent <= float64(n1Threshold),
	}
	results = append(results, n1Check)

	// Memory utilization check
	memCheck := checkResult{
		name:      "Memory utilization",
		value:     resp.MemoryUtilization,
		threshold: float64(memoryThreshold),
		unit:      "%",
		passed:    resp.MemoryUtilization <= float64(memoryThreshold),
	}
	results = append(results, memCheck)

	return results
}

// countResults returns the count of passed and failed checks
func countResults(results []checkResult) (passed, failed int) {
	for _, r := range results {
		if r.passed {
			passed++
		} else {
			failed++
		}
	}
	return
}

// formatCheckHuman formats check results for human readability
func formatCheckHuman(results []checkResult) string {
	var output string

	for _, r := range results {
		symbol := "✓"
		if !r.passed {
			symbol = "✗"
		}
		output += fmt.Sprintf("%s %s: %.0f%s (threshold: %.0f%s)\n",
			symbol, r.name, r.value, r.unit, r.threshold, r.unit)
	}

	passed, failed := countResults(results)
	if failed > 0 {
		output += fmt.Sprintf("\nFAILED: %d check(s) exceeded threshold", failed)
	} else {
		output += fmt.Sprintf("\nPASSED: All %d check(s) within thresholds", passed)
	}

	return output
}

// formatCheckJSON formats check results as JSON
func formatCheckJSON(results []checkResult) string {
	_, failed := countResults(results)

	checks := make([]map[string]interface{}, len(results))
	for i, r := range results {
		checks[i] = map[string]interface{}{
			"name":      r.name,
			"value":     r.value,
			"threshold": r.threshold,
			"unit":      r.unit,
			"passed":    r.passed,
		}
	}

	status := "passed"
	if failed > 0 {
		status = "failed"
	}

	output := map[string]interface{}{
		"status": status,
		"checks": checks,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}
