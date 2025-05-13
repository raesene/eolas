package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "eolas",
	Short: "Eolas is a command line utility for analyzing Kubernetes clusters",
	Long: `Eolas is a command line utility for analyzing Kubernetes cluster configurations.
It ingests JSON files containing Kubernetes resources and provides analysis capabilities.

For more information visit: https://github.com/raesene/eolas`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Eolas - Kubernetes Cluster Analyzer")
		fmt.Println("\nUse the following commands:")
		fmt.Println("  eolas ingest -f <json-file>  - Ingest a Kubernetes cluster configuration")
		fmt.Println("  eolas list                   - List stored configurations")
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