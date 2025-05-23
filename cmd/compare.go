package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	compareConfig1        string
	compareConfig2        string
	compareStorageDir     string
	compareUseHomeDir     bool
	compareStorageBackend string
	compareHtmlOutput     bool
	compareOutputFile     string
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare two stored Kubernetes cluster configurations",
	Long: `Compare two stored Kubernetes cluster configurations to identify differences in resources and security findings.
	
Configurations can be specified by name (for file backend) or by ID (for SQLite backend).
For SQLite backend, use 'eolas list --backend sqlite' to see available configuration IDs.`,
	Run: func(cmd *cobra.Command, args []string) {
		if compareConfig1 == "" || compareConfig2 == "" {
			fmt.Println("Error: both configuration identifiers are required")
			fmt.Println("Usage: eolas compare --config1 <name/id> --config2 <name/id>")
			cmd.Help()
			return
		}

		// Validate storage backend
		if err := storage.ValidateBackend(compareStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Determine storage directory
		var storeDir string
		if compareStorageDir != "" {
			// Use explicitly provided storage directory
			storeDir = compareStorageDir
		} else if compareUseHomeDir {
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
			Backend:    storage.Backend(compareStorageBackend),
			StorageDir: storeDir,
			UseHomeDir: compareUseHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		// Perform comparison
		comparison, err := store.CompareConfigs(compareConfig1, compareConfig2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing configurations: %v\n", err)
			os.Exit(1)
		}

		// Handle HTML output if requested
		if compareHtmlOutput {
			htmlContent, err := generateComparisonHTML(comparison)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating HTML comparison: %v\n", err)
				os.Exit(1)
			}

			// Write to file if output file specified, otherwise stdout
			if compareOutputFile != "" {
				// Automatically add .html extension if not present
				if !strings.HasSuffix(strings.ToLower(compareOutputFile), ".html") {
					compareOutputFile += ".html"
				}

				if err := os.WriteFile(compareOutputFile, htmlContent, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing HTML to file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("HTML comparison report saved to: %s\n", compareOutputFile)
			} else {
				// Write to stdout
				fmt.Println(string(htmlContent))
			}

			return
		}

		// Standard text output
		displayComparisonText(comparison)
	},
}

// displayComparisonText shows the comparison results in text format
func displayComparisonText(comparison *storage.ConfigComparison) {
	fmt.Printf("Configuration Comparison\n")
	fmt.Printf("========================\n\n")

	// Configuration details
	fmt.Printf("Configuration 1: %s\n", comparison.Config1.Name)
	fmt.Printf("  ID: %s\n", comparison.Config1.ID)
	fmt.Printf("  Timestamp: %s\n", comparison.Config1.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Printf("\nConfiguration 2: %s\n", comparison.Config2.Name)
	fmt.Printf("  ID: %s\n", comparison.Config2.ID)
	fmt.Printf("  Timestamp: %s\n", comparison.Config2.Timestamp.Format("2006-01-02 15:04:05"))

	// Resource differences
	if len(comparison.ResourceDiff) > 0 {
		fmt.Printf("\nResource Differences:\n")
		fmt.Printf("====================\n")
		fmt.Printf("%-25s %-10s %-10s %-10s\n", "RESOURCE TYPE", "BEFORE", "AFTER", "CHANGE")
		fmt.Printf("%-25s %-10s %-10s %-10s\n", "-------------", "------", "-----", "------")

		// Sort resource types for consistent output
		var resourceTypes []string
		for resourceType := range comparison.ResourceDiff {
			resourceTypes = append(resourceTypes, resourceType)
		}
		sort.Strings(resourceTypes)

		for _, resourceType := range resourceTypes {
			diff := comparison.ResourceDiff[resourceType]
			changeStr := fmt.Sprintf("%+d", diff.Change)
			if diff.Change > 0 {
				changeStr = fmt.Sprintf("+%d", diff.Change)
			}
			fmt.Printf("%-25s %-10d %-10d %-10s\n", resourceType, diff.Before, diff.After, changeStr)
		}
	} else {
		fmt.Printf("\nResource Differences: None\n")
	}

	// Security differences
	fmt.Printf("\nSecurity Analysis Differences:\n")
	fmt.Printf("==============================\n")

	secDiff := comparison.SecurityDiff
	fmt.Printf("%-30s %-10s %-10s %-10s\n", "SECURITY FINDING", "BEFORE", "AFTER", "CHANGE")
	fmt.Printf("%-30s %-10s %-10s %-10s\n", "----------------", "------", "-----", "------")

	findings := []struct {
		name string
		diff storage.SecurityFindingDiff
	}{
		{"Privileged Containers", secDiff.PrivilegedContainers},
		{"Containers w/ Capabilities", secDiff.CapabilityContainers},
		{"Host Namespace Usage", secDiff.HostNamespaceUsage},
		{"Host Path Volumes", secDiff.HostPathVolumes},
	}

	for _, finding := range findings {
		changeStr := fmt.Sprintf("%+d", finding.diff.Change)
		if finding.diff.Change > 0 {
			changeStr = fmt.Sprintf("+%d", finding.diff.Change)
		}
		fmt.Printf("%-30s %-10d %-10d %-10s\n", finding.name, finding.diff.Before, finding.diff.After, changeStr)
	}

	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("========\n")

	totalResourceChanges := 0
	for _, diff := range comparison.ResourceDiff {
		if diff.Change != 0 {
			totalResourceChanges++
		}
	}

	totalSecurityChanges := 0
	if secDiff.PrivilegedContainers.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.CapabilityContainers.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.HostNamespaceUsage.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.HostPathVolumes.Change != 0 {
		totalSecurityChanges++
	}

	fmt.Printf("- %d resource types changed\n", totalResourceChanges)
	fmt.Printf("- %d security finding types changed\n", totalSecurityChanges)

	if totalResourceChanges == 0 && totalSecurityChanges == 0 {
		fmt.Printf("- No significant differences detected\n")
	}
}

// generateComparisonHTML creates HTML output for comparison results
func generateComparisonHTML(comparison *storage.ConfigComparison) ([]byte, error) {
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Configuration Comparison - Eolas</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1, h2 { color: #333; margin-bottom: 20px; }
        h1 { text-align: center; border-bottom: 2px solid #007acc; padding-bottom: 10px; }
        .config-info { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; }
        .config-card { background: #f8f9fa; padding: 15px; border-radius: 5px; border-left: 4px solid #007acc; }
        .config-card h3 { margin-top: 0; color: #007acc; }
        table { width: 100%; border-collapse: collapse; margin-bottom: 30px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: 600; }
        tr:hover { background-color: #f5f5f5; }
        .positive { color: #28a745; font-weight: bold; }
        .negative { color: #dc3545; font-weight: bold; }
        .neutral { color: #6c757d; }
        .summary { background: #e9ecef; padding: 20px; border-radius: 5px; margin-top: 20px; }
        .no-changes { color: #28a745; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Configuration Comparison</h1>
        
        <div class="config-info">
            <div class="config-card">
                <h3>Configuration 1</h3>
                <p><strong>Name:</strong> ` + comparison.Config1.Name + `</p>
                <p><strong>ID:</strong> ` + comparison.Config1.ID + `</p>
                <p><strong>Timestamp:</strong> ` + comparison.Config1.Timestamp.Format("2006-01-02 15:04:05") + `</p>
            </div>
            <div class="config-card">
                <h3>Configuration 2</h3>
                <p><strong>Name:</strong> ` + comparison.Config2.Name + `</p>
                <p><strong>ID:</strong> ` + comparison.Config2.ID + `</p>
                <p><strong>Timestamp:</strong> ` + comparison.Config2.Timestamp.Format("2006-01-02 15:04:05") + `</p>
            </div>
        </div>`

	// Resource differences table
	if len(comparison.ResourceDiff) > 0 {
		htmlContent += `
        <h2>Resource Differences</h2>
        <table>
            <thead>
                <tr>
                    <th>Resource Type</th>
                    <th>Before</th>
                    <th>After</th>
                    <th>Change</th>
                </tr>
            </thead>
            <tbody>`

		// Sort resource types for consistent output
		var resourceTypes []string
		for resourceType := range comparison.ResourceDiff {
			resourceTypes = append(resourceTypes, resourceType)
		}
		sort.Strings(resourceTypes)

		for _, resourceType := range resourceTypes {
			diff := comparison.ResourceDiff[resourceType]
			changeClass := "neutral"
			changeText := fmt.Sprintf("%+d", diff.Change)
			
			if diff.Change > 0 {
				changeClass = "positive"
				changeText = "+" + fmt.Sprintf("%d", diff.Change)
			} else if diff.Change < 0 {
				changeClass = "negative"
			}

			htmlContent += fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%d</td>
                    <td>%d</td>
                    <td class="%s">%s</td>
                </tr>`, resourceType, diff.Before, diff.After, changeClass, changeText)
		}

		htmlContent += `
            </tbody>
        </table>`
	} else {
		htmlContent += `<h2>Resource Differences</h2><p class="no-changes">No resource differences detected.</p>`
	}

	// Security differences table
	htmlContent += `
        <h2>Security Analysis Differences</h2>
        <table>
            <thead>
                <tr>
                    <th>Security Finding</th>
                    <th>Before</th>
                    <th>After</th>
                    <th>Change</th>
                </tr>
            </thead>
            <tbody>`

	secDiff := comparison.SecurityDiff
	findings := []struct {
		name string
		diff storage.SecurityFindingDiff
	}{
		{"Privileged Containers", secDiff.PrivilegedContainers},
		{"Containers w/ Capabilities", secDiff.CapabilityContainers},
		{"Host Namespace Usage", secDiff.HostNamespaceUsage},
		{"Host Path Volumes", secDiff.HostPathVolumes},
	}

	for _, finding := range findings {
		changeClass := "neutral"
		changeText := fmt.Sprintf("%+d", finding.diff.Change)
		
		if finding.diff.Change > 0 {
			changeClass = "negative" // More security issues is bad
			changeText = "+" + fmt.Sprintf("%d", finding.diff.Change)
		} else if finding.diff.Change < 0 {
			changeClass = "positive" // Fewer security issues is good
		}

		htmlContent += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%d</td>
                <td>%d</td>
                <td class="%s">%s</td>
            </tr>`, finding.name, finding.diff.Before, finding.diff.After, changeClass, changeText)
	}

	htmlContent += `
            </tbody>
        </table>`

	// Summary section
	totalResourceChanges := 0
	for _, diff := range comparison.ResourceDiff {
		if diff.Change != 0 {
			totalResourceChanges++
		}
	}

	totalSecurityChanges := 0
	if secDiff.PrivilegedContainers.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.CapabilityContainers.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.HostNamespaceUsage.Change != 0 {
		totalSecurityChanges++
	}
	if secDiff.HostPathVolumes.Change != 0 {
		totalSecurityChanges++
	}

	htmlContent += fmt.Sprintf(`
        <div class="summary">
            <h2>Summary</h2>
            <ul>
                <li><strong>%d</strong> resource types changed</li>
                <li><strong>%d</strong> security finding types changed</li>`, totalResourceChanges, totalSecurityChanges)

	if totalResourceChanges == 0 && totalSecurityChanges == 0 {
		htmlContent += `<li class="no-changes">No significant differences detected</li>`
	}

	htmlContent += `
            </ul>
        </div>
    </div>
</body>
</html>`

	return []byte(htmlContent), nil
}

func init() {
	rootCmd.AddCommand(compareCmd)
	compareCmd.Flags().StringVar(&compareConfig1, "config1", "", "First configuration to compare (name for file backend, ID for SQLite) (required)")
	compareCmd.Flags().StringVar(&compareConfig2, "config2", "", "Second configuration to compare (name for file backend, ID for SQLite) (required)")
	compareCmd.Flags().StringVarP(&compareStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	compareCmd.Flags().BoolVarP(&compareUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	compareCmd.Flags().StringVar(&compareStorageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
	compareCmd.Flags().BoolVar(&compareHtmlOutput, "html", false, "Generate HTML output")
	compareCmd.Flags().StringVarP(&compareOutputFile, "output", "o", "", "File to write output to (default is stdout)")
	compareCmd.MarkFlagRequired("config1")
	compareCmd.MarkFlagRequired("config2")
}