package kubernetes

import (
	"encoding/json"
	"fmt"
)

// ParseConfig parses Kubernetes configuration JSON data
func ParseConfig(data []byte) (*ClusterConfig, error) {
	var config ClusterConfig
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Kubernetes configuration: %w", err)
	}
	return &config, nil
}

// GetResourceCounts returns counts of different resource types in the configuration
func GetResourceCounts(config *ClusterConfig) map[string]int {
	counts := make(map[string]int)
	
	for _, item := range config.Items {
		counts[item.Kind]++
	}
	
	return counts
}

// PrivilegedContainer represents a container running with privileged security context
type PrivilegedContainer struct {
	Name      string
	Namespace string
	Kind      string
	PodName   string
}

// CapabilityContainer represents a container with added Linux capabilities
type CapabilityContainer struct {
	Name         string
	Namespace    string
	Kind         string
	PodName      string
	Capabilities []string
}

// HostNamespaceWorkload represents a workload using host namespaces
type HostNamespaceWorkload struct {
	Name            string
	Namespace       string
	Kind            string
	HostPID         bool
	HostIPC         bool
	HostNetwork     bool
	HostPorts       []int
	ContainerNames  []string
}

// GetPrivilegedContainers identifies containers running with privileged security context
func GetPrivilegedContainers(config *ClusterConfig) []PrivilegedContainer {
	var results []PrivilegedContainer
	
	// Process all items in the cluster configuration
	for _, item := range config.Items {
		// Look for pod-related resources (Pod, Deployment, StatefulSet, DaemonSet, etc.)
		switch item.Kind {
		case "Pod":
			// Direct Pod resources
			processPrivilegedContainersInPod(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &results)
		case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job", "CronJob":
			// Resources that create pods
			processPrivilegedContainersInWorkload(item, &results)
		}
	}
	
	return results
}

// processPrivilegedContainersInPod checks if pod spec contains privileged containers
func processPrivilegedContainersInPod(item Item, podName, namespace, kind string, results *[]PrivilegedContainer) {
	// Access pod spec
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// Check containers
		if containers, ok := spec["containers"].([]interface{}); ok {
			checkContainersForPrivileged(containers, podName, namespace, kind, results)
		}
		
		// Check init containers if they exist
		if initContainers, ok := spec["initContainers"].([]interface{}); ok {
			checkContainersForPrivileged(initContainers, podName, namespace, kind, results)
		}
	}
}

// processPrivilegedContainersInWorkload extracts pod spec from workload resources
func processPrivilegedContainersInWorkload(item Item, results *[]PrivilegedContainer) {
	workloadName := item.Metadata.Name
	namespace := item.Metadata.Namespace
	kind := item.Kind
	
	// Navigate to pod spec based on resource type
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// For CronJob, need to go through jobTemplate
		if kind == "CronJob" {
			if jobTemplate, ok := spec["jobTemplate"].(map[string]interface{}); ok {
				if jobSpec, ok := jobTemplate["spec"].(map[string]interface{}); ok {
					spec = jobSpec // Update spec to job spec
				}
			}
		}
		
		// Get template for all workload types
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				// Found pod spec, now check containers
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					checkContainersForPrivileged(containers, workloadName, namespace, kind, results)
				}
				
				// Check init containers
				if initContainers, ok := podSpec["initContainers"].([]interface{}); ok {
					checkContainersForPrivileged(initContainers, workloadName, namespace, kind, results)
				}
			}
		}
	}
}

// checkContainersForPrivileged inspects container definitions for privileged security context
func checkContainersForPrivileged(containers []interface{}, ownerName, namespace, kind string, results *[]PrivilegedContainer) {
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		containerName, _ := container["name"].(string)
		
		// Check if security context exists and has privileged: true
		if securityContext, ok := container["securityContext"].(map[string]interface{}); ok {
			if privileged, ok := securityContext["privileged"].(bool); ok && privileged {
				*results = append(*results, PrivilegedContainer{
					Name:      containerName,
					Namespace: namespace,
					Kind:      kind,
					PodName:   ownerName,
				})
			}
		}
	}
}

// GetCapabilityContainers identifies containers with added Linux capabilities
func GetCapabilityContainers(config *ClusterConfig) []CapabilityContainer {
	var results []CapabilityContainer
	
	// Process all items in the cluster configuration
	for _, item := range config.Items {
		// Look for pod-related resources (Pod, Deployment, StatefulSet, DaemonSet, etc.)
		switch item.Kind {
		case "Pod":
			// Direct Pod resources
			processCapabilityContainersInPod(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &results)
		case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job", "CronJob":
			// Resources that create pods
			processCapabilityContainersInWorkload(item, &results)
		}
	}
	
	return results
}

// processCapabilityContainersInPod checks if pod spec contains containers with added capabilities
func processCapabilityContainersInPod(item Item, podName, namespace, kind string, results *[]CapabilityContainer) {
	// Access pod spec
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// Check containers
		if containers, ok := spec["containers"].([]interface{}); ok {
			checkContainersForCapabilities(containers, podName, namespace, kind, results)
		}
		
		// Check init containers if they exist
		if initContainers, ok := spec["initContainers"].([]interface{}); ok {
			checkContainersForCapabilities(initContainers, podName, namespace, kind, results)
		}
	}
}

// processCapabilityContainersInWorkload extracts pod spec from workload resources
func processCapabilityContainersInWorkload(item Item, results *[]CapabilityContainer) {
	workloadName := item.Metadata.Name
	namespace := item.Metadata.Namespace
	kind := item.Kind
	
	// Navigate to pod spec based on resource type
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// For CronJob, need to go through jobTemplate
		if kind == "CronJob" {
			if jobTemplate, ok := spec["jobTemplate"].(map[string]interface{}); ok {
				if jobSpec, ok := jobTemplate["spec"].(map[string]interface{}); ok {
					spec = jobSpec // Update spec to job spec
				}
			}
		}
		
		// Get template for all workload types
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				// Found pod spec, now check containers
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					checkContainersForCapabilities(containers, workloadName, namespace, kind, results)
				}
				
				// Check init containers
				if initContainers, ok := podSpec["initContainers"].([]interface{}); ok {
					checkContainersForCapabilities(initContainers, workloadName, namespace, kind, results)
				}
			}
		}
	}
}

// checkContainersForCapabilities inspects container definitions for added Linux capabilities
func checkContainersForCapabilities(containers []interface{}, ownerName, namespace, kind string, results *[]CapabilityContainer) {
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		containerName, _ := container["name"].(string)
		
		// Check if security context exists and has added capabilities
		if securityContext, ok := container["securityContext"].(map[string]interface{}); ok {
			if capabilities, ok := securityContext["capabilities"].(map[string]interface{}); ok {
				var addedCaps []string
				
				// Check for added capabilities
				if add, ok := capabilities["add"].([]interface{}); ok && len(add) > 0 {
					for _, cap := range add {
						if capStr, ok := cap.(string); ok {
							addedCaps = append(addedCaps, capStr)
						}
					}
					
					// Only append to results if there are added capabilities
					if len(addedCaps) > 0 {
						*results = append(*results, CapabilityContainer{
							Name:         containerName,
							Namespace:    namespace,
							Kind:         kind,
							PodName:      ownerName,
							Capabilities: addedCaps,
						})
					}
				}
			}
		}
	}
}

// GetHostNamespaceWorkloads identifies workloads using host namespaces
func GetHostNamespaceWorkloads(config *ClusterConfig) []HostNamespaceWorkload {
	var results []HostNamespaceWorkload
	
	// Process all items in the cluster configuration
	for _, item := range config.Items {
		// Look for pod-related resources
		switch item.Kind {
		case "Pod":
			// Direct Pod resources
			checkPodForHostNamespaces(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &results)
		case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job", "CronJob":
			// Resources that create pods
			checkWorkloadForHostNamespaces(item, &results)
		}
	}
	
	return results
}

// checkPodForHostNamespaces examines a pod for host namespace usage
func checkPodForHostNamespaces(item Item, name, namespace, kind string, results *[]HostNamespaceWorkload) {
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		var hostNamespaceUsed bool
		workload := HostNamespaceWorkload{
			Name:      name,
			Namespace: namespace,
			Kind:      kind,
		}
		
		// Check for host PID namespace
		if hostPID, ok := spec["hostPID"].(bool); ok && hostPID {
			workload.HostPID = true
			hostNamespaceUsed = true
		}
		
		// Check for host IPC namespace
		if hostIPC, ok := spec["hostIPC"].(bool); ok && hostIPC {
			workload.HostIPC = true
			hostNamespaceUsed = true
		}
		
		// Check for host network namespace
		if hostNetwork, ok := spec["hostNetwork"].(bool); ok && hostNetwork {
			workload.HostNetwork = true
			hostNamespaceUsed = true
		}
		
		// Collect container names and check for host ports
		if containers, ok := spec["containers"].([]interface{}); ok {
			checkContainersForHostPorts(containers, &workload)
		}
		
		// Check init containers if they exist
		if initContainers, ok := spec["initContainers"].([]interface{}); ok {
			checkContainersForHostPorts(initContainers, &workload)
		}
		
		// Only append if any host namespace is used or host ports are used
		if hostNamespaceUsed || len(workload.HostPorts) > 0 {
			*results = append(*results, workload)
		}
	}
}

// checkWorkloadForHostNamespaces examines workload resources for host namespace usage
func checkWorkloadForHostNamespaces(item Item, results *[]HostNamespaceWorkload) {
	workloadName := item.Metadata.Name
	namespace := item.Metadata.Namespace
	kind := item.Kind
	
	// Navigate to pod spec based on resource type
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// For CronJob, need to go through jobTemplate
		if kind == "CronJob" {
			if jobTemplate, ok := spec["jobTemplate"].(map[string]interface{}); ok {
				if jobSpec, ok := jobTemplate["spec"].(map[string]interface{}); ok {
					spec = jobSpec // Update spec to job spec
				}
			}
		}
		
		// Get template for all workload types
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				// Create a mock Pod item to reuse the pod checking logic
				mockPod := Item{
					Kind: kind,
					Metadata: Metadata{
						Name:      workloadName,
						Namespace: namespace,
					},
					Spec: podSpec,
				}
				
				checkPodForHostNamespaces(mockPod, workloadName, namespace, kind, results)
			}
		}
	}
}

// checkContainersForHostPorts examines containers for host port mappings
func checkContainersForHostPorts(containers []interface{}, workload *HostNamespaceWorkload) {
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		containerName, _ := container["name"].(string)
		
		// Add container name to the list if not already present
		if !containsString(workload.ContainerNames, containerName) && containerName != "" {
			workload.ContainerNames = append(workload.ContainerNames, containerName)
		}
		
		// Check for host ports in container port mappings
		if ports, ok := container["ports"].([]interface{}); ok {
			for _, p := range ports {
				port, ok := p.(map[string]interface{})
				if !ok {
					continue
				}
				
				// Check for hostPort setting
				if hostPort, ok := port["hostPort"].(float64); ok && hostPort > 0 {
					hostPortInt := int(hostPort)
					if !containsInt(workload.HostPorts, hostPortInt) {
						workload.HostPorts = append(workload.HostPorts, hostPortInt)
					}
				}
			}
		}
	}
}

// containsString checks if a string is in a slice
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// containsInt checks if an int is in a slice
func containsInt(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}