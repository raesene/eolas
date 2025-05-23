package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	exportConfigName    string
	exportStorageDir    string
	exportUseHomeDir    bool
	exportStorageBackend string
	exportFormat        string
	exportOutputFile    string
	exportType          string
)

// ExportData represents the complete export data structure
type ExportData struct {
	ConfigName           string                               `json:"config_name"`
	ConfigID             string                               `json:"config_id,omitempty"`
	Timestamp            time.Time                            `json:"timestamp"`
	ExportedAt           time.Time                            `json:"exported_at"`
	ResourceCounts       map[string]int                       `json:"resource_counts"`
	TotalResources       int                                  `json:"total_resources"`
	PrivilegedContainers []kubernetes.PrivilegedContainer    `json:"privileged_containers"`
	CapabilityContainers []kubernetes.CapabilityContainer    `json:"capability_containers"`
	HostNamespaceWorkloads []kubernetes.HostNamespaceWorkload `json:"host_namespace_workloads"`
	HostPathVolumes      []kubernetes.HostPathVolume         `json:"host_path_volumes"`
	SecuritySummary      SecuritySummary                      `json:"security_summary"`
}

// SecuritySummary provides a summary of security findings
type SecuritySummary struct {
	TotalFindings        int `json:"total_findings"`
	PrivilegedCount      int `json:"privileged_count"`
	CapabilityCount      int `json:"capability_count"`
	HostNamespaceCount   int `json:"host_namespace_count"`
	HostPathCount        int `json:"host_path_count"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration analysis data in various formats",
	Long: `Export configuration analysis data in JSON or CSV format for external processing.

This command allows you to:
- Export complete analysis results for a configuration
- Output in JSON format for programmatic processing
- Output in CSV format for spreadsheet analysis
- Export security findings only or complete analysis
- Support for both file and SQLite backends

Examples:
  # Export complete analysis as JSON
  eolas export --name prod-cluster --format json

  # Export only security findings as CSV
  eolas export --name prod-cluster --format csv --type security

  # Export to specific file
  eolas export --name prod-cluster --format json -o analysis.json`,
	Run: func(cmd *cobra.Command, args []string) {
		if exportConfigName == "" {
			fmt.Println("Error: configuration name is required")
			fmt.Println("Usage: eolas export --name <config-name> --format <json|csv>")
			cmd.Help()
			return
		}

		// Validate format
		if exportFormat != "json" && exportFormat != "csv" {
			fmt.Fprintf(os.Stderr, "Error: Invalid format '%s'. Valid formats are: json, csv\n", exportFormat)
			os.Exit(1)
		}

		// Validate type
		if exportType != "all" && exportType != "security" && exportType != "resources" {
			fmt.Fprintf(os.Stderr, "Error: Invalid type '%s'. Valid types are: all, security, resources\n", exportType)
			os.Exit(1)
		}

		// Validate storage backend
		if err := storage.ValidateBackend(exportStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Determine storage directory
		var storeDir string
		if exportStorageDir != "" {
			storeDir = exportStorageDir
		} else if exportUseHomeDir {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
				os.Exit(1)
			}
			storeDir = filepath.Join(homeDir, ".eolas")
		} else {
			storeDir = ".eolas"
		}

		// Create storage backend
		storageConfig := storage.StorageConfig{
			Backend:    storage.Backend(exportStorageBackend),
			StorageDir: storeDir,
			UseHomeDir: exportUseHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		// Load configuration
		config, err := store.LoadConfig(exportConfigName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", exportConfigName, err)
			os.Exit(1)
		}

		// Get metadata - for SQLite backend, we need to get the latest version
		var metadata *storage.ConfigMetadata
		if exportStorageBackend == "sqlite" {
			// Get history and use the latest version
			history, err := store.GetConfigHistory(exportConfigName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting configuration history: %v\n", err)
				os.Exit(1)
			}
			if len(history) == 0 {
				fmt.Fprintf(os.Stderr, "No configurations found for name '%s'\n", exportConfigName)
				os.Exit(1)
			}
			// Use the latest version (first in list since it's ordered DESC)
			metadata = &history[0]
		} else {
			// For file backend, get metadata directly
			var err error
			metadata, err = store.GetConfigMetadata(exportConfigName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting configuration metadata: %v\n", err)
				os.Exit(1)
			}
		}

		// Perform analysis
		resourceCounts := kubernetes.GetResourceCounts(config)
		totalResources := 0
		for _, count := range resourceCounts {
			totalResources += count
		}

		var privilegedContainers []kubernetes.PrivilegedContainer
		var capabilityContainers []kubernetes.CapabilityContainer
		var hostNamespaceWorkloads []kubernetes.HostNamespaceWorkload
		var hostPathVolumes []kubernetes.HostPathVolume

		if exportType == "all" || exportType == "security" {
			privilegedContainers = kubernetes.GetPrivilegedContainers(config)
			capabilityContainers = kubernetes.GetCapabilityContainers(config)
			hostNamespaceWorkloads = kubernetes.GetHostNamespaceWorkloads(config)
			hostPathVolumes = kubernetes.GetHostPathVolumes(config)
		}

		// Create export data structure
		exportData := ExportData{
			ConfigName:             exportConfigName,
			ConfigID:               metadata.ID,
			Timestamp:              metadata.Timestamp,
			ExportedAt:             time.Now(),
			ResourceCounts:         resourceCounts,
			TotalResources:         totalResources,
			PrivilegedContainers:   privilegedContainers,
			CapabilityContainers:   capabilityContainers,
			HostNamespaceWorkloads: hostNamespaceWorkloads,
			HostPathVolumes:        hostPathVolumes,
			SecuritySummary: SecuritySummary{
				TotalFindings:      len(privilegedContainers) + len(capabilityContainers) + len(hostNamespaceWorkloads) + len(hostPathVolumes),
				PrivilegedCount:    len(privilegedContainers),
				CapabilityCount:    len(capabilityContainers),
				HostNamespaceCount: len(hostNamespaceWorkloads),
				HostPathCount:      len(hostPathVolumes),
			},
		}

		// Export based on format
		var outputData []byte
		var defaultExt string

		switch exportFormat {
		case "json":
			outputData, err = exportAsJSON(exportData, exportType)
			defaultExt = "json"
		case "csv":
			outputData, err = exportAsCSV(exportData, exportType)
			defaultExt = "csv"
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating export data: %v\n", err)
			os.Exit(1)
		}

		// Determine output file
		outputFile := exportOutputFile
		if outputFile == "" {
			outputFile = fmt.Sprintf("%s-%s-export.%s", exportConfigName, exportType, defaultExt)
		}

		// Automatically add extension if not present
		if !strings.HasSuffix(strings.ToLower(outputFile), "."+defaultExt) {
			outputFile += "." + defaultExt
		}

		// Write output
		if outputFile == "-" {
			// Write to stdout
			fmt.Print(string(outputData))
		} else {
			if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing export data to file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Export completed successfully!\n")
			fmt.Printf("Configuration: %s\n", exportConfigName)
			fmt.Printf("Format: %s\n", exportFormat)
			fmt.Printf("Type: %s\n", exportType)
			fmt.Printf("Output file: %s\n", outputFile)
			fmt.Printf("Total resources: %d\n", totalResources)
			fmt.Printf("Security findings: %d\n", exportData.SecuritySummary.TotalFindings)
		}
	},
}

// exportAsJSON exports data in JSON format
func exportAsJSON(data ExportData, exportType string) ([]byte, error) {
	switch exportType {
	case "security":
		// Export only security-related data
		securityData := map[string]interface{}{
			"config_name":             data.ConfigName,
			"config_id":               data.ConfigID,
			"timestamp":               data.Timestamp,
			"exported_at":             data.ExportedAt,
			"security_summary":        data.SecuritySummary,
			"privileged_containers":   data.PrivilegedContainers,
			"capability_containers":   data.CapabilityContainers,
			"host_namespace_workloads": data.HostNamespaceWorkloads,
			"host_path_volumes":       data.HostPathVolumes,
		}
		return json.MarshalIndent(securityData, "", "  ")
	case "resources":
		// Export only resource data
		resourceData := map[string]interface{}{
			"config_name":     data.ConfigName,
			"config_id":       data.ConfigID,
			"timestamp":       data.Timestamp,
			"exported_at":     data.ExportedAt,
			"resource_counts": data.ResourceCounts,
			"total_resources": data.TotalResources,
		}
		return json.MarshalIndent(resourceData, "", "  ")
	default:
		// Export all data
		return json.MarshalIndent(data, "", "  ")
	}
}

// exportAsCSV exports data in CSV format
func exportAsCSV(data ExportData, exportType string) ([]byte, error) {
	var records [][]string

	switch exportType {
	case "security":
		// CSV header for security findings
		records = append(records, []string{
			"Finding Type", "Namespace", "Resource Type", "Resource Name", 
			"Container Name", "Details", "Timestamp",
		})

		// Add privileged containers
		for _, pc := range data.PrivilegedContainers {
			records = append(records, []string{
				"Privileged Container", pc.Namespace, pc.Kind, pc.PodName,
				pc.Name, "Privileged: true", data.Timestamp.Format(time.RFC3339),
			})
		}

		// Add capability containers
		for _, cc := range data.CapabilityContainers {
			caps := strings.Join(cc.Capabilities, ", ")
			records = append(records, []string{
				"Container Capabilities", cc.Namespace, cc.Kind, cc.PodName,
				cc.Name, "Capabilities: " + caps, data.Timestamp.Format(time.RFC3339),
			})
		}

		// Add host namespace workloads
		for _, hn := range data.HostNamespaceWorkloads {
			details := fmt.Sprintf("HostPID: %t, HostIPC: %t, HostNetwork: %t", 
				hn.HostPID, hn.HostIPC, hn.HostNetwork)
			records = append(records, []string{
				"Host Namespace", hn.Namespace, hn.Kind, hn.Name,
				strings.Join(hn.ContainerNames, ", "), details, data.Timestamp.Format(time.RFC3339),
			})
		}

		// Add host path volumes
		for _, hp := range data.HostPathVolumes {
			paths := strings.Join(hp.HostPaths, ", ")
			records = append(records, []string{
				"Host Path Volume", hp.Namespace, hp.Kind, hp.Name,
				"-", "Paths: " + paths, data.Timestamp.Format(time.RFC3339),
			})
		}

	case "resources":
		// CSV header for resource counts
		records = append(records, []string{
			"Resource Type", "Count", "Config Name", "Timestamp",
		})

		for resourceType, count := range data.ResourceCounts {
			records = append(records, []string{
				resourceType, fmt.Sprintf("%d", count), data.ConfigName, 
				data.Timestamp.Format(time.RFC3339),
			})
		}

	default:
		// CSV with summary data
		records = append(records, []string{
			"Metric", "Value", "Config Name", "Timestamp",
		})

		records = append(records, []string{
			"Total Resources", fmt.Sprintf("%d", data.TotalResources), 
			data.ConfigName, data.Timestamp.Format(time.RFC3339),
		})
		records = append(records, []string{
			"Total Security Findings", fmt.Sprintf("%d", data.SecuritySummary.TotalFindings), 
			data.ConfigName, data.Timestamp.Format(time.RFC3339),
		})
		records = append(records, []string{
			"Privileged Containers", fmt.Sprintf("%d", data.SecuritySummary.PrivilegedCount), 
			data.ConfigName, data.Timestamp.Format(time.RFC3339),
		})
		records = append(records, []string{
			"Capability Containers", fmt.Sprintf("%d", data.SecuritySummary.CapabilityCount), 
			data.ConfigName, data.Timestamp.Format(time.RFC3339),
		})
	}

	// Convert records to CSV
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)
	
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("error writing CSV record: %w", err)
		}
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error writing CSV: %w", err)
	}

	return []byte(csvData.String()), nil
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportConfigName, "name", "n", "", "Configuration name to export (required)")
	exportCmd.Flags().StringVarP(&exportStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	exportCmd.Flags().BoolVarP(&exportUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	exportCmd.Flags().StringVar(&exportStorageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json, csv)")
	exportCmd.Flags().StringVarP(&exportOutputFile, "output", "o", "", "Output file (default: <config>-<type>-export.<format>, use '-' for stdout)")
	exportCmd.Flags().StringVarP(&exportType, "type", "t", "all", "Export type (all, security, resources)")
	exportCmd.MarkFlagRequired("name")
}