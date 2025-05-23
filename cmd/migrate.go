package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	migrateFrom       string
	migrateTo         string
	migrateStorageDir string
	migrateUseHomeDir bool
	migrateDryRun     bool
	migrateForce      bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate configuration data between storage backends",
	Long: `Migrate configuration data from one storage backend to another.

This command allows you to:
- Migrate from file storage to SQLite for advanced features
- Migrate from SQLite to file storage for simplicity
- Preserve all configuration data and metadata where possible
- Validate data integrity during migration

Examples:
  # Migrate from file to SQLite (most common)
  eolas migrate --from file --to sqlite

  # Dry run to see what would be migrated
  eolas migrate --from file --to sqlite --dry-run

  # Force overwrite existing configurations
  eolas migrate --from file --to sqlite --force`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate backends
		if err := storage.ValidateBackend(migrateFrom); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid source backend - %v\n", err)
			os.Exit(1)
		}

		if err := storage.ValidateBackend(migrateTo); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid destination backend - %v\n", err)
			os.Exit(1)
		}

		if migrateFrom == migrateTo {
			fmt.Fprintf(os.Stderr, "Error: Source and destination backends cannot be the same\n")
			os.Exit(1)
		}

		// Determine storage directory
		var storeDir string
		if migrateStorageDir != "" {
			storeDir = migrateStorageDir
		} else if migrateUseHomeDir {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
				os.Exit(1)
			}
			storeDir = filepath.Join(homeDir, ".eolas")
		} else {
			storeDir = ".eolas"
		}

		// Perform migration
		if err := performMigration(migrateFrom, migrateTo, storeDir); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// performMigration handles the actual migration process
func performMigration(from, to, storeDir string) error {
	fmt.Printf("Starting migration from %s to %s...\n", from, to)
	fmt.Printf("Storage directory: %s\n", storeDir)

	if migrateDryRun {
		fmt.Printf("DRY RUN MODE - No changes will be made\n")
	}
	fmt.Println()

	// Create source storage
	sourceConfig := storage.StorageConfig{
		Backend:    storage.Backend(from),
		StorageDir: storeDir,
		UseHomeDir: migrateUseHomeDir,
	}

	sourceStore, err := storage.NewStore(sourceConfig)
	if err != nil {
		return fmt.Errorf("failed to create source storage: %w", err)
	}
	defer sourceStore.Close()

	// Create destination storage
	destConfig := storage.StorageConfig{
		Backend:    storage.Backend(to),
		StorageDir: storeDir,
		UseHomeDir: migrateUseHomeDir,
	}

	destStore, err := storage.NewStore(destConfig)
	if err != nil {
		return fmt.Errorf("failed to create destination storage: %w", err)
	}
	defer destStore.Close()

	// Get list of configurations to migrate
	configs, err := sourceStore.ListConfigs()
	if err != nil {
		return fmt.Errorf("failed to list source configurations: %w", err)
	}

	if len(configs) == 0 {
		fmt.Printf("No configurations found in %s backend to migrate.\n", from)
		return nil
	}

	fmt.Printf("Found %d configurations to migrate:\n", len(configs))
	for _, config := range configs {
		fmt.Printf("  - %s\n", config)
	}
	fmt.Println()

	// Check for existing configurations in destination
	if !migrateDryRun && !migrateForce {
		destConfigs, err := destStore.ListConfigs()
		if err != nil {
			return fmt.Errorf("failed to list destination configurations: %w", err)
		}

		if len(destConfigs) > 0 {
			fmt.Printf("Destination %s backend already contains %d configurations:\n", to, len(destConfigs))
			for _, config := range destConfigs {
				fmt.Printf("  - %s\n", config)
			}
			fmt.Printf("\nUse --force to overwrite existing configurations, or --dry-run to preview migration.\n")
			return fmt.Errorf("destination backend not empty")
		}
	}

	// Migrate each configuration
	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, configName := range configs {
		fmt.Printf("Migrating %s...", configName)

		if migrateDryRun {
			fmt.Printf(" [DRY RUN]\n")
			migratedCount++
			continue
		}

		err := migrateConfiguration(sourceStore, destStore, configName, from, to)
		if err != nil {
			fmt.Printf(" ERROR: %v\n", err)
			errorCount++
			continue
		}

		fmt.Printf(" ✓\n")
		migratedCount++
	}

	// Print summary
	fmt.Printf("\nMigration Summary:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Successfully migrated: %d\n", migratedCount)
	if skippedCount > 0 {
		fmt.Printf("Skipped: %d\n", skippedCount)
	}
	if errorCount > 0 {
		fmt.Printf("Errors: %d\n", errorCount)
	}

	if migrateDryRun {
		fmt.Printf("\nThis was a dry run. Use the same command without --dry-run to perform the migration.\n")
	} else if errorCount == 0 {
		fmt.Printf("\nMigration completed successfully!\n")
		fmt.Printf("All configurations are now available in the %s backend.\n", to)
		
		if to == "sqlite" {
			fmt.Printf("\nYou can now use advanced features:\n")
			fmt.Printf("  - Configuration versioning and history\n")
			fmt.Printf("  - Timeline reports: eolas timeline --name <config>\n")
			fmt.Printf("  - Enhanced comparisons with pre-computed analysis\n")
		}
	} else {
		fmt.Printf("\nMigration completed with %d errors. Check the error messages above.\n", errorCount)
	}

	return nil
}

// migrateConfiguration migrates a single configuration between backends
func migrateConfiguration(sourceStore, destStore storage.Store, configName, from, to string) error {
	// Load configuration from source
	config, err := sourceStore.LoadConfig(configName)
	if err != nil {
		return fmt.Errorf("failed to load from source: %w", err)
	}

	// For file-to-SQLite migration, we need to create proper metadata
	if from == "file" && to == "sqlite" {
		// Get file metadata for timestamp
		metadata, err := sourceStore.GetConfigMetadata(configName)
		if err != nil {
			return fmt.Errorf("failed to get source metadata: %w", err)
		}

		// Create enhanced metadata for SQLite
		newMetadata := storage.ConfigMetadata{
			Name:           configName,
			Timestamp:      metadata.Timestamp,
			CreatedAt:      time.Now(),
			ResourceCounts: kubernetes.GetResourceCounts(config),
			Description:    fmt.Sprintf("Migrated from file storage on %s", time.Now().Format("2006-01-02")),
		}

		// Use enhanced save method for SQLite
		err = destStore.SaveConfigWithMetadata(config, newMetadata)
		if err != nil {
			return fmt.Errorf("failed to save to destination: %w", err)
		}
	} else {
		// For other migrations, use basic save
		err = destStore.SaveConfig(config, configName)
		if err != nil {
			return fmt.Errorf("failed to save to destination: %w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "", "Source storage backend (file, sqlite) (required)")
	migrateCmd.Flags().StringVar(&migrateTo, "to", "", "Destination storage backend (file, sqlite) (required)")
	migrateCmd.Flags().StringVarP(&migrateStorageDir, "storage-dir", "s", "", "Directory containing storage data (defaults to .eolas in home directory)")
	migrateCmd.Flags().BoolVarP(&migrateUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be migrated without making changes")
	migrateCmd.Flags().BoolVar(&migrateForce, "force", false, "Overwrite existing configurations in destination backend")
	migrateCmd.MarkFlagRequired("from")
	migrateCmd.MarkFlagRequired("to")
}