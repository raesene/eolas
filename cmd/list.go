package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	listStorageDir     string
	listUseHomeDir     bool
	listStorageBackend string
	listShowHistory    bool
	listConfigName     string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored Kubernetes cluster configurations",
	Long:  `List all Kubernetes cluster configurations that have been ingested and stored.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate storage backend
		if err := storage.ValidateBackend(listStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Determine storage directory
		var storeDir string
		if listStorageDir != "" {
			// Use explicitly provided storage directory
			storeDir = listStorageDir
		} else if listUseHomeDir {
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

		// Create storage backend
		storageConfig := storage.StorageConfig{
			Backend:    storage.Backend(listStorageBackend),
			StorageDir: storeDir,
			UseHomeDir: listUseHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		// Handle history listing for specific configuration
		if listShowHistory {
			if listConfigName == "" {
				fmt.Fprintf(os.Stderr, "Error: --name is required when using --history\n")
				os.Exit(1)
			}

			history, err := store.GetConfigHistory(listConfigName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting configuration history: %v\n", err)
				os.Exit(1)
			}

			if len(history) == 0 {
				fmt.Printf("No configurations found for name '%s'.\n", listConfigName)
				return
			}

			fmt.Printf("Configuration history for '%s' (%s backend):\n", listConfigName, listStorageBackend)
			fmt.Printf("%-36s %-20s %-15s %s\n", "ID", "TIMESTAMP", "RESOURCES", "DESCRIPTION")
			fmt.Printf("%-36s %-20s %-15s %s\n", "--", "---------", "---------", "-----------")

			for _, config := range history {
				totalResources := 0
				for _, count := range config.ResourceCounts {
					totalResources += count
				}

				description := config.Description
				if description == "" {
					description = "-"
				}

				fmt.Printf("%-36s %-20s %-15d %s\n", 
					config.ID, 
					config.Timestamp.Format("2006-01-02 15:04:05"),
					totalResources,
					description,
				)
			}
			return
		}

		// Standard listing
		configs, err := store.ListConfigs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing configurations: %v\n", err)
			os.Exit(1)
		}

		if len(configs) == 0 {
			fmt.Println("No stored configurations found.")
			return
		}

		// For SQLite backend, show more detailed information
		if listStorageBackend == "sqlite" {
			fmt.Printf("Stored configurations (%s backend) in %s:\n", listStorageBackend, storeDir)
			fmt.Printf("Use --history --name <config-name> to see version history for a specific configuration.\n\n")
			
			for _, configName := range configs {
				// Get the latest configuration for each name to show summary info
				history, err := store.GetConfigHistory(configName)
				if err != nil {
					fmt.Printf("  - %s (error getting details: %v)\n", configName, err)
					continue
				}

				if len(history) > 0 {
					latest := history[0] // History is ordered by timestamp DESC
					totalResources := 0
					for _, count := range latest.ResourceCounts {
						totalResources += count
					}

					fmt.Printf("  - %s\n", configName)
					fmt.Printf("    Latest: %s (%d resources, %d versions)\n", 
						latest.Timestamp.Format("2006-01-02 15:04:05"), 
						totalResources,
						len(history),
					)
				} else {
					fmt.Printf("  - %s (no versions found)\n", configName)
				}
			}
		} else {
			// Simple listing for file backend
			fmt.Printf("Stored configurations (%s backend) in %s:\n", listStorageBackend, storeDir)
			for _, config := range configs {
				fmt.Printf("  - %s\n", config)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	listCmd.Flags().BoolVarP(&listUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	listCmd.Flags().StringVar(&listStorageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
	listCmd.Flags().BoolVar(&listShowHistory, "history", false, "Show configuration history for a specific configuration (requires --name)")
	listCmd.Flags().StringVarP(&listConfigName, "name", "n", "", "Configuration name to show history for (used with --history)")
}