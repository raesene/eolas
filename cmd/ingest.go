package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	inputFile      string
	clusterName    string
	storageDir     string
	useHomeDir     bool
	storageBackend string
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a Kubernetes cluster configuration JSON file",
	Long:  `Ingest a JSON file containing Kubernetes cluster configuration for analysis.`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputFile == "" {
			fmt.Println("Error: input file is required")
			cmd.Help()
			return
		}

		// Validate storage backend
		if err := storage.ValidateBackend(storageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Check if file exists
		absPath, err := filepath.Abs(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving file path: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Ingesting file: %s\n", absPath)
		_, err = os.Stat(absPath)
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file %s does not exist\n", absPath)
			os.Exit(1)
		}

		// Read and parse JSON
		data, err := readJSON(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading JSON file: %v\n", err)
			os.Exit(1)
		}

		// Parse Kubernetes configuration
		config, err := kubernetes.ParseConfig(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing Kubernetes configuration: %v\n", err)
			os.Exit(1)
		}

		// Display resource counts
		resourceCounts := kubernetes.GetResourceCounts(config)
		fmt.Println("Successfully ingested Kubernetes configuration")
		fmt.Printf("File size: %d bytes\n", len(data))
		fmt.Println("Resource counts:")
		for kind, count := range resourceCounts {
			fmt.Printf("  %s: %d\n", kind, count)
		}

		// Determine storage directory
		var storeDir string
		if storageDir != "" {
			// Use explicitly provided storage directory
			storeDir = storageDir
		} else if useHomeDir {
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
			Backend:    storage.Backend(storageBackend),
			StorageDir: storeDir,
			UseHomeDir: useHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating storage backend: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		if err := store.SaveConfig(config, clusterName); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration saved as '%s' using %s backend in %s\n", clusterName, storageBackend, storeDir)
	},
}

// readJSON reads and validates the JSON file
func readJSON(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Validate JSON format
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	return data, nil
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.Flags().StringVarP(&inputFile, "file", "f", "", "JSON file containing Kubernetes cluster configuration (required)")
	ingestCmd.Flags().StringVarP(&clusterName, "name", "n", "", "Name to identify the cluster configuration (defaults to timestamp)")
	ingestCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Directory to store parsed configurations (defaults to .eolas in home directory)")
	ingestCmd.Flags().BoolVarP(&useHomeDir, "use-home", "", true, "Store configurations in .eolas directory in user's home directory")
	ingestCmd.Flags().StringVar(&storageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
	ingestCmd.MarkFlagRequired("file")
}