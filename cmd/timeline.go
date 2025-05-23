package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raesene/eolas/pkg/output"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	timelineConfigName    string
	timelineStorageDir    string
	timelineUseHomeDir    bool
	timelineStorageBackend string
	timelineOutputFile    string
)

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Generate a timeline report for a configuration's evolution",
	Long: `Generate a timeline report showing how a configuration has evolved over time.
	
This command creates an interactive HTML report showing:
- Configuration evolution timeline
- Resource count trends over time  
- Security posture changes
- Current vs previous snapshot comparison

Note: Timeline reports are only available for SQLite backend.`,
	Run: func(cmd *cobra.Command, args []string) {
		if timelineConfigName == "" {
			fmt.Println("Error: configuration name is required")
			fmt.Println("Usage: eolas timeline --name <config-name>")
			cmd.Help()
			return
		}

		// Validate storage backend
		if err := storage.ValidateBackend(timelineStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Timeline reports are only available for SQLite backend
		if timelineStorageBackend != "sqlite" {
			fmt.Fprintf(os.Stderr, "Error: Timeline reports are only available with SQLite backend\n")
			fmt.Fprintf(os.Stderr, "Use: eolas timeline --backend sqlite --name %s\n", timelineConfigName)
			os.Exit(1)
		}

		// Determine storage directory
		var storeDir string
		if timelineStorageDir != "" {
			// Use explicitly provided storage directory
			storeDir = timelineStorageDir
		} else if timelineUseHomeDir {
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
			Backend:    storage.Backend(timelineStorageBackend),
			StorageDir: storeDir,
			UseHomeDir: timelineUseHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		// Get configuration history
		history, err := store.GetConfigHistory(timelineConfigName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting configuration history: %v\n", err)
			os.Exit(1)
		}

		if len(history) == 0 {
			fmt.Printf("No configurations found for name '%s'.\n", timelineConfigName)
			fmt.Printf("Use 'eolas list --backend sqlite' to see available configurations.\n")
			return
		}

		if len(history) < 2 {
			fmt.Printf("Timeline reports require at least 2 versions of a configuration.\n")
			fmt.Printf("Configuration '%s' only has %d version(s).\n", timelineConfigName, len(history))
			return
		}

		// Get security analysis history
		securityHistory, err := store.GetSecurityAnalysisHistory(timelineConfigName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting security analysis history: %v\n", err)
			os.Exit(1)
		}

		// Create timeline formatter
		formatter, err := output.NewTimelineFormatter()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating timeline formatter: %v\n", err)
			os.Exit(1)
		}

		// Generate timeline HTML
		htmlContent, err := formatter.GenerateTimelineHTML(timelineConfigName, history, securityHistory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating timeline report: %v\n", err)
			os.Exit(1)
		}

		// Determine output file
		outputFile := timelineOutputFile
		if outputFile == "" {
			// Generate default filename
			outputFile = fmt.Sprintf("timeline-%s.html", timelineConfigName)
		}

		// Automatically add .html extension if not present
		if !strings.HasSuffix(strings.ToLower(outputFile), ".html") {
			outputFile += ".html"
		}

		// Write HTML to file
		if err := os.WriteFile(outputFile, htmlContent, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing timeline report to file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Timeline report generated successfully!\n")
		fmt.Printf("Report saved to: %s\n", outputFile)
		fmt.Printf("Open the file in your web browser to view the interactive timeline.\n")
		fmt.Printf("\nReport includes:\n")
		fmt.Printf("- %d configuration versions from %s to %s\n", 
			len(history), 
			history[0].Timestamp.Format("Jan 2, 2006"),
			history[len(history)-1].Timestamp.Format("Jan 2, 2006"))
		fmt.Printf("- Resource evolution trends\n")
		fmt.Printf("- Security posture changes\n")
		fmt.Printf("- Interactive timeline with detailed changes\n")
	},
}

func init() {
	rootCmd.AddCommand(timelineCmd)
	timelineCmd.Flags().StringVarP(&timelineConfigName, "name", "n", "", "Configuration name to generate timeline for (required)")
	timelineCmd.Flags().StringVarP(&timelineStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	timelineCmd.Flags().BoolVarP(&timelineUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	timelineCmd.Flags().StringVar(&timelineStorageBackend, "backend", "sqlite", "Storage backend to use (must be sqlite for timeline reports)")
	timelineCmd.Flags().StringVarP(&timelineOutputFile, "output", "o", "", "Output file for timeline report (default: timeline-<config-name>.html)")
	timelineCmd.MarkFlagRequired("name")
}