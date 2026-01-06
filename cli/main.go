// ABOUTME: Entry point for diego-capacity CLI
// ABOUTME: Command-line tool for capacity monitoring and CI/CD integration

package main

import (
	"fmt"
	"os"

	"github.com/markalston/diego-capacity-analyzer/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
