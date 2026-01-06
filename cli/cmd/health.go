// ABOUTME: Health command for diego-capacity CLI
// ABOUTME: Checks backend connectivity and service status

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

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check backend connectivity",
	Long:  `Check connectivity to the Diego Capacity Analyzer backend and verify service status.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		exitCode := runHealth(ctx, os.Stdout)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

// runHealth executes the health check and returns exit code
func runHealth(ctx context.Context, w io.Writer) int {
	url := GetAPIURL()
	c := client.New(url)

	resp, err := c.Health(ctx)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return 2
	}

	if IsJSONOutput() {
		fmt.Fprintln(w, formatHealthJSON(url, resp))
	} else {
		fmt.Fprintln(w, formatHealthHuman(url, resp))
	}

	return 0
}

// formatHealthHuman formats health response for human readability
func formatHealthHuman(url string, resp *client.HealthResponse) string {
	return fmt.Sprintf(`Backend:      %s
CF API:       %s
BOSH:         %s
Cells Cached: %t
Apps Cached:  %t`, url, resp.CFAPI, resp.BOSHAPI, resp.CacheStatus.CellsCached, resp.CacheStatus.AppsCached)
}

// formatHealthJSON formats health response as JSON
func formatHealthJSON(url string, resp *client.HealthResponse) string {
	output := map[string]interface{}{
		"backend":  url,
		"cf_api":   resp.CFAPI,
		"bosh_api": resp.BOSHAPI,
		"cache_status": map[string]bool{
			"cells_cached": resp.CacheStatus.CellsCached,
			"apps_cached":  resp.CacheStatus.AppsCached,
		},
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}
