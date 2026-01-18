// ABOUTME: Root command for diego-capacity CLI
// ABOUTME: Handles global flags, TTY detection, and TUI launch

package cmd

import (
	"context"
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
		status, err := c.InfrastructureStatus(context.Background())
		vsphereConfigured := err == nil && status.VSphereConfigured

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
