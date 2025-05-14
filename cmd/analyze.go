package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/raesene/eolas/pkg/kubernetes"
	"github.com/raesene/eolas/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	analyzeClusterName    string
	analyzeStorageDir     string
	analyzeUseHomeDir     bool
	securityAnalysisFlag  bool
	privilegedAnalysisFlag bool
	capabilityAnalysisFlag bool
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

		// Create storage handler
		store, err := storage.NewFileStore(storeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
			os.Exit(1)
		}

		// Load configuration
		config, err := store.LoadConfig(analyzeClusterName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", analyzeClusterName, err)
			os.Exit(1)
		}

		fmt.Printf("Analyzing cluster configuration: %s\n\n", analyzeClusterName)

		// Standard resource analysis
		if !securityAnalysisFlag && !privilegedAnalysisFlag {
			showResourceAnalysis(config)
		}

		// Security analysis
		if securityAnalysisFlag || privilegedAnalysisFlag || capabilityAnalysisFlag {
			// If any security flag is enabled, show security analysis header
			fmt.Println("Security Analysis:")
			fmt.Println("=================")

			// Privileged container analysis
			if securityAnalysisFlag || privilegedAnalysisFlag {
				showPrivilegedContainers(config)
			}
			
			// Capability analysis
			if securityAnalysisFlag || capabilityAnalysisFlag {
				showCapabilityContainers(config)
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

// showPrivilegedContainers displays privileged containers in the cluster
func showPrivilegedContainers(config *kubernetes.ClusterConfig) {
	privilegedContainers := kubernetes.GetPrivilegedContainers(config)
	
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

// showCapabilityContainers displays containers with added Linux capabilities
func showCapabilityContainers(config *kubernetes.ClusterConfig) {
	capContainers := kubernetes.GetCapabilityContainers(config)
	
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

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&analyzeClusterName, "name", "n", "", "Name of the cluster configuration to analyze (required)")
	analyzeCmd.Flags().StringVarP(&analyzeStorageDir, "storage-dir", "s", "", "Directory where configurations are stored (defaults to .eolas in home directory)")
	analyzeCmd.Flags().BoolVarP(&analyzeUseHomeDir, "use-home", "", true, "Use .eolas directory in user's home directory")
	analyzeCmd.Flags().BoolVar(&securityAnalysisFlag, "security", false, "Run security-focused analysis on the cluster configuration")
	analyzeCmd.Flags().BoolVar(&privilegedAnalysisFlag, "privileged", false, "Check for privileged containers in the cluster configuration")
	analyzeCmd.Flags().BoolVar(&capabilityAnalysisFlag, "capabilities", false, "Check for containers with added Linux capabilities")
	analyzeCmd.MarkFlagRequired("name")
}