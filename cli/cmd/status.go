// ABOUTME: Status command for diego-capacity CLI
// ABOUTME: Shows current infrastructure status and capacity metrics

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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current infrastructure status",
	Long:  `Display the current infrastructure status including clusters, hosts, Diego cells, and capacity metrics.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		exitCode := runStatus(ctx, os.Stdout)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// runStatus executes the status check and returns exit code
func runStatus(ctx context.Context, w io.Writer) int {
	url := GetAPIURL()
	c := client.New(url)

	resp, err := c.InfrastructureStatus(ctx)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return 2
	}

	if !resp.HasData {
		if IsJSONOutput() {
			fmt.Fprintln(w, formatStatusJSON(resp))
		} else {
			fmt.Fprintln(w, formatStatusHuman(resp))
		}
		return 2
	}

	if IsJSONOutput() {
		fmt.Fprintln(w, formatStatusJSON(resp))
	} else {
		fmt.Fprintln(w, formatStatusHuman(resp))
	}

	return 0
}

// formatStatusHuman formats status response for human readability
func formatStatusHuman(resp *client.InfrastructureStatus) string {
	if !resp.HasData {
		msg := "No infrastructure data loaded.\n"
		if !resp.VSphereConfigured {
			msg += "vSphere is not configured.\n"
		}
		msg += "Load data via UI or API first."
		return msg
	}

	n1Status := resp.N1Status
	if n1Status == "" {
		n1Status = capacityStatus(resp.N1CapacityPercent, 85, 95)
	}
	memStatus := capacityStatus(resp.MemoryUtilization, 80, 90)

	return fmt.Sprintf(`Infrastructure: %s (%s)
Clusters:       %d
Hosts:          %d
Diego Cells:    %d

N-1 Capacity:   %.0f%% [%s]
Memory:         %.0f%% [%s]
Constraining:   %s`,
		resp.Name, resp.Source,
		resp.ClusterCount,
		resp.HostCount,
		resp.CellCount,
		resp.N1CapacityPercent, n1Status,
		resp.MemoryUtilization, memStatus,
		resp.ConstrainingResource)
}

// formatStatusJSON formats status response as JSON
func formatStatusJSON(resp *client.InfrastructureStatus) string {
	data, _ := json.MarshalIndent(resp, "", "  ")
	return string(data)
}

// capacityStatus returns ok/warning/critical based on thresholds
func capacityStatus(percent, warningThreshold, criticalThreshold float64) string {
	if percent >= criticalThreshold {
		return "critical"
	}
	if percent >= warningThreshold {
		return "warning"
	}
	return "ok"
}
