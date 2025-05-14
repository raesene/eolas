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