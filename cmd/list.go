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

		configs, err := store.ListConfigs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing configurations: %v\n", err)
			os.Exit(1)
		}

		if len(configs) == 0 {
			fmt.Println("No stored configurations found.")
			return
		}

		fmt.Printf("Stored configurations (%s backend) in %s:\n", listStorageBackend, storeDir)
		for _, config := range configs {
			fmt.Printf("  - %s\n", config)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	listCmd.Flags().BoolVarP(&listUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	listCmd.Flags().StringVar(&listStorageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
}