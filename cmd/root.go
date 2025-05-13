package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set by build flags
var (
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "eolas",
	Short: "Eolas is a command line utility for analyzing Kubernetes clusters",
	Long: `Eolas is a command line utility for analyzing Kubernetes cluster configurations.
It ingests JSON files containing Kubernetes resources and provides analysis capabilities.

For more information visit: https://github.com/raesene/eolas`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Eolas - Kubernetes Cluster Analyzer")
		fmt.Println("Version:", version)
		fmt.Println("\nAvailable Commands:")
		fmt.Println("  eolas ingest    - Ingest a Kubernetes cluster configuration")
		fmt.Println("  eolas list      - List stored configurations")
		fmt.Println("  eolas analyze   - Analyze a stored cluster configuration")
		fmt.Println("  eolas version   - Display version information")
		fmt.Println("  eolas help      - Display help for any command")
		fmt.Println("\nExamples:")
		fmt.Println("  # Ingest a cluster configuration and name it (stored in ~/.eolas)")
		fmt.Println("  eolas ingest -f sample_data/sample-kind.json -n my-kind-cluster")
		fmt.Println()
		fmt.Println("  # List all stored configurations from home directory")
		fmt.Println("  eolas list")
		fmt.Println()
		fmt.Println("  # Analyze a stored cluster configuration")
		fmt.Println("  eolas analyze -n my-kind-cluster")
		fmt.Println()
		fmt.Println("  # Use a custom storage directory instead of home directory")
		fmt.Println("  eolas ingest -f sample_data/sample-kind.json -s /path/to/store --use-home=false")
		fmt.Println("\nFor more details on a command, use:")
		fmt.Println("  eolas [command] --help")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}