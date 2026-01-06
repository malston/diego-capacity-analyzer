// ABOUTME: Root command for diego-capacity CLI
// ABOUTME: Handles global flags and configuration

package cmd

import (
	"os"

	"github.com/spf13/cobra"
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

It enables CI/CD pipelines to monitor TAS capacity and alert when thresholds are exceeded.

Environment Variables:
  DIEGO_CAPACITY_API_URL  Backend API URL (default: http://localhost:8080)`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "Backend API URL (overrides DIEGO_CAPACITY_API_URL)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON instead of human-readable text")
}

// GetAPIURL returns the API URL from flag, env, or default (in priority order)
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
