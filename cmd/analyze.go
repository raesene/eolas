package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/output"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	analyzeClusterName        string
	analyzeStorageDir         string
	analyzeUseHomeDir         bool
	analyzeStorageBackend     string
	securityAnalysisFlag      bool
	privilegedAnalysisFlag    bool
	capabilityAnalysisFlag    bool
	hostNamespaceAnalysisFlag bool
	hostPathAnalysisFlag      bool
	htmlOutputFlag            bool
	outputFileFlag            string
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

		// Validate storage backend
		if err := storage.ValidateBackend(analyzeStorageBackend); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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

		// Create storage backend
		storageConfig := storage.StorageConfig{
			Backend:    storage.Backend(analyzeStorageBackend),
			StorageDir: storeDir,
			UseHomeDir: analyzeUseHomeDir,
		}

		store, err := storage.NewStore(storageConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		// Load configuration
		config, err := store.LoadConfig(analyzeClusterName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", analyzeClusterName, err)
			os.Exit(1)
		}

		// Get resource counts for all analysis types
		resourceCounts := kubernetes.GetResourceCounts(config)
		
		// Collect security analysis data if needed for any output format
		var privilegedContainers []kubernetes.PrivilegedContainer
		var capabilityContainers []kubernetes.CapabilityContainer
		var hostNamespaceWorkloads []kubernetes.HostNamespaceWorkload
		var hostPathVolumes []kubernetes.HostPathVolume
		
		if securityAnalysisFlag || privilegedAnalysisFlag || htmlOutputFlag {
			privilegedContainers = kubernetes.GetPrivilegedContainers(config)
		}
		
		if securityAnalysisFlag || capabilityAnalysisFlag || htmlOutputFlag {
			capabilityContainers = kubernetes.GetCapabilityContainers(config)
		}
		
		if securityAnalysisFlag || hostNamespaceAnalysisFlag || htmlOutputFlag {
			hostNamespaceWorkloads = kubernetes.GetHostNamespaceWorkloads(config)
		}
		
		if securityAnalysisFlag || hostPathAnalysisFlag || htmlOutputFlag {
			hostPathVolumes = kubernetes.GetHostPathVolumes(config)
		}
		
		// Handle HTML output if requested
		if htmlOutputFlag {
			htmlFormatter, err := output.NewHTMLFormatter()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating HTML formatter: %v\n", err)
				os.Exit(1)
			}
			
			htmlContent, err := htmlFormatter.GenerateHTML(
				analyzeClusterName,
				resourceCounts,
				privilegedContainers,
				capabilityContainers,
				hostNamespaceWorkloads,
				hostPathVolumes,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
				os.Exit(1)
			}
			
			// Write to file if output file specified, otherwise stdout
			if outputFileFlag != "" {
				// Automatically add .html extension if not present
				if !strings.HasSuffix(strings.ToLower(outputFileFlag), ".html") {
					outputFileFlag += ".html"
				}
				
				if err := htmlFormatter.WriteHTMLToFile(htmlContent, outputFileFlag); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing HTML to file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("HTML report saved to: %s\n", outputFileFlag)
			} else {
				// Write to stdout
				fmt.Println(string(htmlContent))
			}
			
			return
		}
		
		// Standard text output (original functionality)
		fmt.Printf("Analyzing cluster configuration: %s\n\n", analyzeClusterName)

		// Standard resource analysis
		if !securityAnalysisFlag && !privilegedAnalysisFlag && !capabilityAnalysisFlag && !hostNamespaceAnalysisFlag {
			showResourceAnalysis(config)
		}

		// Security analysis
		if securityAnalysisFlag || privilegedAnalysisFlag || capabilityAnalysisFlag || hostNamespaceAnalysisFlag || hostPathAnalysisFlag {
			// If any security flag is enabled, show security analysis header
			fmt.Println("Security Analysis:")
			fmt.Println("=================")

			// Privileged container analysis
			if securityAnalysisFlag || privilegedAnalysisFlag {
				showPrivilegedContainersText(privilegedContainers)
			}
			
			// Capability analysis
			if securityAnalysisFlag || capabilityAnalysisFlag {
				showCapabilityContainersText(capabilityContainers)
			}
			
			// Host namespace analysis
			if securityAnalysisFlag || hostNamespaceAnalysisFlag {
				showHostNamespaceWorkloadsText(hostNamespaceWorkloads)
			}
			
			// Host path volume analysis
			if securityAnalysisFlag || hostPathAnalysisFlag {
				showHostPathVolumesText(hostPathVolumes)
			}
		}
	},
}

// showResourceAnalysis displays standard resource counts
func showResourceAnalysis(config *kubernetes.ClusterConfig) {
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
	fmt.Println()
}

// showPrivilegedContainersText displays privileged containers in the cluster (text output)
func showPrivilegedContainersText(privilegedContainers []kubernetes.PrivilegedContainer) {
	fmt.Println("Privileged Containers:")
	fmt.Println("=====================")
	
	if len(privilegedContainers) == 0 {
		fmt.Println("No privileged containers found in the cluster.")
		fmt.Println()
		return
	}
	
	fmt.Printf("Found %d privileged containers\n\n", len(privilegedContainers))
	fmt.Printf("%-20s %-20s %-20s %-30s\n", "NAMESPACE", "RESOURCE TYPE", "RESOURCE NAME", "CONTAINER NAME")
	fmt.Printf("%-20s %-20s %-20s %-30s\n", "---------", "------------", "------------", "--------------")
	
	for _, pc := range privilegedContainers {
		namespace := pc.Namespace
		if namespace == "" {
			namespace = "default"
		}
		fmt.Printf("%-20s %-20s %-20s %-30s\n", namespace, pc.Kind, pc.PodName, pc.Name)
	}
	
	fmt.Println()
	fmt.Println("Note: Privileged containers have full access to the host's kernel capabilities and")
	fmt.Println("device nodes, similar to root access on the host. These should be reviewed carefully")
	fmt.Println("for security implications.")
	fmt.Println()
}

// showCapabilityContainersText displays containers with added Linux capabilities (text output)
func showCapabilityContainersText(capContainers []kubernetes.CapabilityContainer) {
	
	fmt.Println("Containers with Added Linux Capabilities:")
	fmt.Println("=======================================")
	
	if len(capContainers) == 0 {
		fmt.Println("No containers with added Linux capabilities found in the cluster.")
		fmt.Println()
		return
	}
	
	fmt.Printf("Found %d containers with added Linux capabilities\n\n", len(capContainers))
	fmt.Printf("%-20s %-15s %-20s %-15s %-30s\n", "NAMESPACE", "RESOURCE TYPE", "RESOURCE NAME", "CONTAINER", "CAPABILITIES")
	fmt.Printf("%-20s %-15s %-20s %-15s %-30s\n", "---------", "------------", "------------", "---------", "------------")
	
	for _, cc := range capContainers {
		namespace := cc.Namespace
		if namespace == "" {
			namespace = "default"
		}
		
		// Join capabilities for display, limit length if too many
		caps := cc.Capabilities
		capsStr := ""
		if len(caps) <= 3 {
			capsStr = joinStrings(caps, ", ")
		} else {
			capsStr = joinStrings(caps[:3], ", ") + ", +" + fmt.Sprintf("%d", len(caps)-3) + " more"
		}
		
		fmt.Printf("%-20s %-15s %-20s %-15s %-30s\n", namespace, cc.Kind, cc.PodName, cc.Name, capsStr)
	}
	
	fmt.Println()
	fmt.Println("Note: Added Linux capabilities provide containers with elevated privileges.")
	fmt.Println("Particularly dangerous capabilities include: CAP_SYS_ADMIN, CAP_NET_ADMIN,")
	fmt.Println("CAP_SYS_PTRACE, and CAP_NET_RAW. These should be reviewed for necessity.")
	fmt.Println()
}

// joinStrings joins string slice with separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	
	return result
}

// showHostNamespaceWorkloadsText displays workloads using host namespaces (text output)
func showHostNamespaceWorkloadsText(workloads []kubernetes.HostNamespaceWorkload) {
	
	fmt.Println("Workloads Using Host Namespaces:")
	fmt.Println("===============================")
	
	if len(workloads) == 0 {
		fmt.Println("No workloads using host namespaces found in the cluster.")
		fmt.Println()
		return
	}
	
	fmt.Printf("Found %d workloads using host namespaces\n\n", len(workloads))
	
	// Print table header
	fmt.Printf("%-20s %-15s %-20s %-15s %-15s %-15s %s\n", 
		"NAMESPACE", "RESOURCE TYPE", "NAME", "HOST PID", "HOST IPC", "HOST NETWORK", "HOST PORTS")
	fmt.Printf("%-20s %-15s %-20s %-15s %-15s %-15s %s\n", 
		"---------", "------------", "----", "--------", "--------", "------------", "----------")
	
	// Print each workload
	for _, w := range workloads {
		namespace := w.Namespace
		if namespace == "" {
			namespace = "default"
		}
		
		// Format ports array for display
		portsStr := ""
		if len(w.HostPorts) > 0 {
			if len(w.HostPorts) <= 3 {
				for i, port := range w.HostPorts {
					if i > 0 {
						portsStr += ", "
					}
					portsStr += fmt.Sprintf("%d", port)
				}
			} else {
				// If more than 3 ports, show first 3 and count of remaining
				for i := 0; i < 3; i++ {
					if i > 0 {
						portsStr += ", "
					}
					portsStr += fmt.Sprintf("%d", w.HostPorts[i])
				}
				portsStr += fmt.Sprintf(", +%d more", len(w.HostPorts)-3)
			}
		} else {
			portsStr = "None"
		}
		
		fmt.Printf("%-20s %-15s %-20s %-15t %-15t %-15t %s\n", 
			namespace, w.Kind, w.Name, w.HostPID, w.HostIPC, w.HostNetwork, portsStr)
	}
	
	fmt.Println()
	fmt.Println("Note: Host namespaces provide containers with access to the host's resources.")
	fmt.Println("These pose significant security risks because they reduce isolation between")
	fmt.Println("containers and the host system. Each namespace type has specific security implications:")
	fmt.Println("- hostPID: Allows visibility of all processes on the host system")
	fmt.Println("- hostIPC: Enables shared memory access with the host and all containers")
	fmt.Println("- hostNetwork: Provides direct access to the host's network interfaces")
	fmt.Println("- hostPorts: Exposes ports directly on the host's network interfaces")
	fmt.Println()
}

// showHostPathVolumesText displays workloads using hostPath volumes (text output)
func showHostPathVolumesText(volumes []kubernetes.HostPathVolume) {
	
	fmt.Println("Workloads Using Host Path Volumes:")
	fmt.Println("=================================")
	
	if len(volumes) == 0 {
		fmt.Println("No workloads using hostPath volumes found in the cluster.")
		fmt.Println()
		return
	}
	
	fmt.Printf("Found %d workloads using hostPath volumes\n\n", len(volumes))
	
	// Print table header
	fmt.Printf("%-20s %-15s %-20s %-15s %s\n", 
		"NAMESPACE", "RESOURCE TYPE", "NAME", "READ-ONLY", "HOST PATH")
	fmt.Printf("%-20s %-15s %-20s %-15s %s\n", 
		"---------", "------------", "----", "---------", "---------")
	
	// Print each workload with their host paths
	for _, v := range volumes {
		namespace := v.Namespace
		if namespace == "" {
			namespace = "default"
		}
		
		// Print a row for each hostPath in the workload
		for i, path := range v.HostPaths {
			readOnly := "No"
			if i < len(v.ReadOnly) && v.ReadOnly[i] {
				readOnly = "Yes"
			}
			
			// For the first path, include the workload details
			if i == 0 {
				fmt.Printf("%-20s %-15s %-20s %-15s %s\n", 
					namespace, v.Kind, v.Name, readOnly, path)
			} else {
				// For subsequent paths, just include the path and read-only status
				fmt.Printf("%-20s %-15s %-20s %-15s %s\n", 
					"", "", "", readOnly, path)
			}
		}
	}
	
	fmt.Println()
	fmt.Println("Note: hostPath volumes allow pods to mount files or directories from the host node's")
	fmt.Println("filesystem directly into the pod. This poses significant security risks as it enables")
	fmt.Println("containers to access and potentially modify sensitive areas of the host filesystem.")
	fmt.Println("Risks include:")
	fmt.Println("- Read access to sensitive host files")
	fmt.Println("- Potential modification of host system files (when not read-only)")
	fmt.Println("- Persistence across pod restarts, potentially allowing data exfiltration")
	fmt.Println("- Potential for privilege escalation through the host filesystem")
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&analyzeClusterName, "name", "n", "", "Name of the cluster configuration to analyze (required)")
	analyzeCmd.Flags().StringVarP(&analyzeStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	analyzeCmd.Flags().BoolVarP(&analyzeUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	analyzeCmd.Flags().StringVar(&analyzeStorageBackend, "backend", "file", "Storage backend to use (file, sqlite)")
	analyzeCmd.Flags().BoolVar(&securityAnalysisFlag, "security", false, "Run security-focused analysis on the cluster configuration")
	analyzeCmd.Flags().BoolVar(&privilegedAnalysisFlag, "privileged", false, "Check for privileged containers in the cluster configuration")
	analyzeCmd.Flags().BoolVar(&capabilityAnalysisFlag, "capabilities", false, "Check for containers with added Linux capabilities")
	analyzeCmd.Flags().BoolVar(&hostNamespaceAnalysisFlag, "host-namespaces", false, "Check for workloads using host namespaces")
	analyzeCmd.Flags().BoolVar(&hostPathAnalysisFlag, "host-path", false, "Check for workloads using hostPath volumes")
	analyzeCmd.Flags().BoolVar(&htmlOutputFlag, "html", false, "Generate HTML output")
	analyzeCmd.Flags().StringVarP(&outputFileFlag, "output", "o", "", "File to write output to (default is stdout)")
	analyzeCmd.MarkFlagRequired("name")
}