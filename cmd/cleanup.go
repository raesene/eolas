package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	cleanupStorageDir      string
	cleanupUseHomeDir      bool
	cleanupStorageBackend  string
	cleanupDryRun          bool
	cleanupOlderThan       string
	cleanupKeepVersions    int
	cleanupConfigName      string
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old configurations and perform maintenance operations",
	Long: `Clean up old configurations and perform maintenance operations on storage backends.

This command helps maintain your configuration storage by:
- Removing old configuration versions (SQLite backend)
- Cleaning up unused files
- Optimizing database storage
- Providing storage usage statistics

Examples:
  # Remove configurations older than 30 days
  eolas cleanup --older-than 30d

  # Keep only the latest 5 versions of each configuration
  eolas cleanup --keep-versions 5

  # Clean up specific configuration, keeping latest 3 versions
  eolas cleanup --name prod-cluster --keep-versions 3

  # Dry run to see what would be cleaned
  eolas cleanup --older-than 7d --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate storage backend
		if err := storage.ValidateBackend(cleanupStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Validate parameters
		if cleanupOlderThan != "" && cleanupKeepVersions > 0 {
			fmt.Fprintf(os.Stderr, "Error: Cannot specify both --older-than and --keep-versions\n")
			os.Exit(1)
		}

		if cleanupOlderThan == "" && cleanupKeepVersions == 0 {
			fmt.Fprintf(os.Stderr, "Error: Must specify either --older-than or --keep-versions\n")
			os.Exit(1)
		}

		// Parse time duration if specified
		var cutoffTime time.Time
		if cleanupOlderThan != "" {
			duration, err := parseDuration(cleanupOlderThan)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing time duration '%s': %v\n", cleanupOlderThan, err)
				os.Exit(1)
			}
			cutoffTime = time.Now().Add(-duration)
		}

		// Determine storage directory
		var storeDir string
		if cleanupStorageDir != "" {
			storeDir = cleanupStorageDir
		} else if cleanupUseHomeDir {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
				os.Exit(1)
			}
			storeDir = filepath.Join(homeDir, ".eolas")
		} else {
			storeDir = ".eolas"
		}

		fmt.Printf("Starting cleanup operation...\n")
		fmt.Printf("Storage backend: %s\n", cleanupStorageBackend)
		fmt.Printf("Storage directory: %s\n", storeDir)

		if cleanupDryRun {
			fmt.Printf("DRY RUN MODE - No changes will be made\n")
		}
		fmt.Println()

		// Perform cleanup based on backend
		if cleanupStorageBackend == "sqlite" {
			err := performSQLiteCleanup(storeDir, cutoffTime, cleanupKeepVersions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			err := performFileCleanup(storeDir, cutoffTime)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// performSQLiteCleanup handles cleanup for SQLite backend
func performSQLiteCleanup(storeDir string, cutoffTime time.Time, keepVersions int) error {
	// Create storage backend
	storageConfig := storage.StorageConfig{
		Backend:    storage.Backend("sqlite"),
		StorageDir: storeDir,
		UseHomeDir: cleanupUseHomeDir,
	}

	store, err := storage.NewStore(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer store.Close()

	// Get list of configurations
	configs, err := store.ListConfigs()
	if err != nil {
		return fmt.Errorf("failed to list configurations: %w", err)
	}

	if len(configs) == 0 {
		fmt.Println("No configurations found.")
		return nil
	}

	totalDeleted := 0

	// Process each configuration
	for _, configName := range configs {
		if cleanupConfigName != "" && configName != cleanupConfigName {
			continue // Skip if specific config name specified
		}

		history, err := store.GetConfigHistory(configName)
		if err != nil {
			fmt.Printf("Warning: Failed to get history for %s: %v\n", configName, err)
			continue
		}

		if len(history) <= 1 {
			fmt.Printf("Configuration %s: Only 1 version, skipping\n", configName)
			continue
		}

		// Sort by timestamp (newest first)
		sort.Slice(history, func(i, j int) bool {
			return history[i].Timestamp.After(history[j].Timestamp)
		})

		var toDelete []string

		if keepVersions > 0 {
			// Keep only the specified number of latest versions
			if len(history) > keepVersions {
				for i := keepVersions; i < len(history); i++ {
					toDelete = append(toDelete, history[i].ID)
				}
			}
		} else {
			// Delete versions older than cutoff time
			for _, version := range history[1:] { // Keep at least the latest
				if version.Timestamp.Before(cutoffTime) {
					toDelete = append(toDelete, version.ID)
				}
			}
		}

		if len(toDelete) == 0 {
			fmt.Printf("Configuration %s: No versions to clean up\n", configName)
			continue
		}

		fmt.Printf("Configuration %s: Found %d versions to delete\n", configName, len(toDelete))

		if !cleanupDryRun {
			for _, id := range toDelete {
				if err := store.DeleteConfig(id); err != nil {
					fmt.Printf("  Error deleting version %s: %v\n", id, err)
				} else {
					fmt.Printf("  Deleted version %s\n", id)
					totalDeleted++
				}
			}
		} else {
			for _, id := range toDelete {
				fmt.Printf("  Would delete version %s\n", id)
				totalDeleted++
			}
		}
	}

	// Show summary
	fmt.Printf("\nCleanup Summary:\n")
	fmt.Printf("================\n")
	if cleanupDryRun {
		fmt.Printf("Would delete %d configuration versions\n", totalDeleted)
	} else {
		fmt.Printf("Deleted %d configuration versions\n", totalDeleted)
		fmt.Printf("Storage space potentially freed\n")
	}

	return nil
}

// performFileCleanup handles cleanup for file backend
func performFileCleanup(storeDir string, cutoffTime time.Time) error {
	// Create storage backend
	storageConfig := storage.StorageConfig{
		Backend:    storage.Backend("file"),
		StorageDir: storeDir,
		UseHomeDir: cleanupUseHomeDir,
	}

	store, err := storage.NewStore(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer store.Close()

	// Get list of configurations
	configs, err := store.ListConfigs()
	if err != nil {
		return fmt.Errorf("failed to list configurations: %w", err)
	}

	if len(configs) == 0 {
		fmt.Println("No configurations found.")
		return nil
	}

	totalDeleted := 0
	var totalSize int64

	// Process each configuration file
	for _, configName := range configs {
		if cleanupConfigName != "" && configName != cleanupConfigName {
			continue
		}

		filePath := filepath.Join(storeDir, fmt.Sprintf("%s.json", configName))
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			fmt.Printf("Warning: Cannot stat file %s: %v\n", filePath, err)
			continue
		}

		// Check if file is older than cutoff
		if fileInfo.ModTime().Before(cutoffTime) {
			fmt.Printf("Configuration %s: Modified %s (older than cutoff)\n", 
				configName, fileInfo.ModTime().Format("2006-01-02 15:04:05"))

			if !cleanupDryRun {
				if err := os.Remove(filePath); err != nil {
					fmt.Printf("  Error deleting %s: %v\n", filePath, err)
				} else {
					fmt.Printf("  Deleted %s\n", configName)
					totalDeleted++
					totalSize += fileInfo.Size()
				}
			} else {
				fmt.Printf("  Would delete %s (size: %d bytes)\n", configName, fileInfo.Size())
				totalDeleted++
				totalSize += fileInfo.Size()
			}
		} else {
			fmt.Printf("Configuration %s: Modified %s (keeping)\n", 
				configName, fileInfo.ModTime().Format("2006-01-02 15:04:05"))
		}
	}

	// Show summary
	fmt.Printf("\nCleanup Summary:\n")
	fmt.Printf("================\n")
	if cleanupDryRun {
		fmt.Printf("Would delete %d configuration files\n", totalDeleted)
		fmt.Printf("Would free %d bytes (%.2f KB)\n", totalSize, float64(totalSize)/1024)
	} else {
		fmt.Printf("Deleted %d configuration files\n", totalDeleted)
		fmt.Printf("Freed %d bytes (%.2f KB)\n", totalSize, float64(totalSize)/1024)
	}

	return nil
}

// parseDuration parses duration strings like "30d", "7d", "1h", etc.
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1:]
	valueStr := s[:len(s)-1]

	var value time.Duration
	switch unit {
	case "d":
		if d, err := time.ParseDuration(valueStr + "h"); err == nil {
			value = d * 24 // Convert to days
		} else {
			return 0, fmt.Errorf("invalid duration value")
		}
	case "h", "m", "s":
		var err error
		value, err = time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %w", err)
		}
	default:
		return 0, fmt.Errorf("unsupported time unit '%s'. Use d, h, m, or s", unit)
	}

	return value, nil
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVarP(&cleanupStorageDir, "storage-dir", "s", "", "Directory containing storage data (defaults to .eolas in home directory)")
	cleanupCmd.Flags().BoolVarP(&cleanupUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	cleanupCmd.Flags().StringVar(&cleanupStorageBackend, "backend", "file", "Storage backend to clean (file, sqlite)")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanupCmd.Flags().StringVar(&cleanupOlderThan, "older-than", "", "Remove configurations older than specified duration (e.g., 30d, 7d, 24h)")
	cleanupCmd.Flags().IntVar(&cleanupKeepVersions, "keep-versions", 0, "Keep only the specified number of latest versions (SQLite only)")
	cleanupCmd.Flags().StringVarP(&cleanupConfigName, "name", "n", "", "Clean up only the specified configuration")
}