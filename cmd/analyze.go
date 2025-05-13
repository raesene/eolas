package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	analyzeClusterName string
	analyzeStorageDir  string
	analyzeUseHomeDir  bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze a stored Kubernetes cluster configuration",
	Long:  `Analyze a stored Kubernetes cluster configuration to extract useful information.`,
	Run: func(cmd *cobra.Command, args []string) {
		if analyzeClusterName == "" {
			fmt.Println("Error: cluster name is required")
			cmd.Help()
			return
		}

		// Determine storage directory
		var storeDir string
		if analyzeStorageDir != "" {
			// Use explicitly provided storage directory
			storeDir = analyzeStorageDir
		} else if analyzeUseHomeDir {
			// Use .eolas in home directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
				os.Exit(1)
			}
			storeDir = filepath.Join(homeDir, ".eolas")
		} else {
			// Use default .eolas in current directory
			storeDir = ".eolas"
		}

		// Create storage handler
		store, err := storage.NewFileStore(storeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}

		// Load configuration
		config, err := store.LoadConfig(analyzeClusterName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", analyzeClusterName, err)
			os.Exit(1)
		}

		fmt.Printf("Analyzing cluster configuration: %s\n\n", analyzeClusterName)

		// Get resource counts
		resourceCounts := kubernetes.GetResourceCounts(config)

		// Sort resource types by name for consistent output
		var resourceTypes []string
		for resourceType := range resourceCounts {
			resourceTypes = append(resourceTypes, resourceType)
		}
		sort.Strings(resourceTypes)

		// Display resource type counts
		fmt.Println("Resource Types:")
		fmt.Println("==============")
		totalResources := 0
		for _, resourceType := range resourceTypes {
			count := resourceCounts[resourceType]
			totalResources += count
			fmt.Printf("%-25s %d\n", resourceType+":", count)
		}
		fmt.Println("-------------------------")
		fmt.Printf("%-25s %d\n", "Total:", totalResources)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&analyzeClusterName, "name", "n", "", "Name of the cluster configuration to analyze (required)")
	analyzeCmd.Flags().StringVarP(&analyzeStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	analyzeCmd.Flags().BoolVarP(&analyzeUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	analyzeCmd.MarkFlagRequired("name")
}