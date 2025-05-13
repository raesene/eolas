package cmd

import (
	"fmt"
	"os"

	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var listStorageDir string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored Kubernetes cluster configurations",
	Long:  `List all Kubernetes cluster configurations that have been ingested and stored.`,
	Run: func(cmd *cobra.Command, args []string) {
		store, err := storage.NewFileStore(listStorageDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}

		configs, err := store.ListConfigs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing configurations: %v\n", err)
			os.Exit(1)
		}

		if len(configs) == 0 {
			fmt.Println("No stored configurations found.")
			return
		}

		fmt.Println("Stored configurations:")
		for _, config := range configs {
			fmt.Printf("  - %s\n", config)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listStorageDir, "storage-dir", "s", ".eolas", "Directory where configurations are stored")
}