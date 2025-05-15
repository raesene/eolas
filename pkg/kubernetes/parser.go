package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"
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

// HostPathVolume represents a workload with hostPath volumes
type HostPathVolume struct {
	Name            string
	Namespace       string
	Kind            string
	HostPaths       []string
	ReadOnly        []bool
}

// GetPrivilegedContainers identifies containers running with privileged security context
func GetPrivilegedContainers(config *ClusterConfig) []PrivilegedContainer {
	var results []PrivilegedContainer
	var podResults []PrivilegedContainer
	var controllerResults []PrivilegedContainer
	
	// First pass: collect all controller resources
	for _, item := range config.Items {
		if item.Kind == "Deployment" || item.Kind == "StatefulSet" || 
		   item.Kind == "DaemonSet" || item.Kind == "ReplicaSet" || 
		   item.Kind == "Job" || item.Kind == "CronJob" {
			// Resources that create pods
			processPrivilegedContainersInWorkload(item, &controllerResults)
		}
	}
	
	// Second pass: collect all standalone pods (not managed by controllers)
	for _, item := range config.Items {
		if item.Kind == "Pod" {
			// Check if this pod is managed by a controller we've already processed
			managed := false
			for _, ref := range item.Metadata.OwnerReferences {
				if ref.Kind == "Deployment" || ref.Kind == "StatefulSet" || 
				   ref.Kind == "DaemonSet" || ref.Kind == "ReplicaSet" || 
				   ref.Kind == "Job" || ref.Kind == "CronJob" {
					managed = true
					break
				}
			}
			
			// Only process unmanaged pods
			if !managed {
				processPrivilegedContainersInPod(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &podResults)
			}
		}
	}
	
	// Combine results
	results = append(results, controllerResults...)
	results = append(results, podResults...)
	
	// Deduplicate results (in case the same owner has multiple containers)
	return deduplicatePrivilegedResults(results)
}

// deduplicatePrivilegedResults removes duplicate entries that refer to the same resource
// It prioritizes higher level resources like Deployments over their child resources
func deduplicatePrivilegedResults(results []PrivilegedContainer) []PrivilegedContainer {
	// Map container name to a slice of results for that container
	containerMap := make(map[string][]PrivilegedContainer)
	
	// Group results by container name
	for _, result := range results {
		key := fmt.Sprintf("%s|%s", result.Namespace, result.Name)
		containerMap[key] = append(containerMap[key], result)
	}
	
	// For each container, prioritize higher-level resources
	var deduplicated []PrivilegedContainer
	for _, resources := range containerMap {
		// Find highest priority resource (Deployment > ReplicaSet > Pod)
		highestPriority := findHighestPriorityResource(resources)
		deduplicated = append(deduplicated, highestPriority)
	}
	
	return deduplicated
}

// findHighestPriorityResource selects the highest priority resource from a slice of resources
// Priority: Deployment > StatefulSet > DaemonSet > Job > CronJob > ReplicaSet > Pod
func findHighestPriorityResource(resources []PrivilegedContainer) PrivilegedContainer {
	if len(resources) == 0 {
		// This should never happen, but handle it gracefully
		return PrivilegedContainer{}
	}
	
	if len(resources) == 1 {
		return resources[0]
	}
	
	// Define priority order (higher number = higher priority)
	kindPriority := map[string]int{
		"Pod":         1,
		"ReplicaSet":  2,
		"Job":         3,
		"CronJob":     4,
		"DaemonSet":   5,
		"StatefulSet": 6,
		"Deployment":  7,
	}
	
	highest := resources[0]
	highestPriority := kindPriority[highest.Kind]
	
	for _, res := range resources[1:] {
		priority := kindPriority[res.Kind]
		if priority > highestPriority {
			highest = res
			highestPriority = priority
		}
	}
	
	return highest
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
	var podResults []CapabilityContainer
	var controllerResults []CapabilityContainer
	
	// First pass: collect all controller resources
	for _, item := range config.Items {
		if item.Kind == "Deployment" || item.Kind == "StatefulSet" || 
		   item.Kind == "DaemonSet" || item.Kind == "ReplicaSet" || 
		   item.Kind == "Job" || item.Kind == "CronJob" {
			// Resources that create pods
			processCapabilityContainersInWorkload(item, &controllerResults)
		}
	}
	
	// Second pass: collect all standalone pods (not managed by controllers)
	for _, item := range config.Items {
		if item.Kind == "Pod" {
			// Check if this pod is managed by a controller we've already processed
			managed := false
			for _, ref := range item.Metadata.OwnerReferences {
				if ref.Kind == "Deployment" || ref.Kind == "StatefulSet" || 
				   ref.Kind == "DaemonSet" || ref.Kind == "ReplicaSet" || 
				   ref.Kind == "Job" || ref.Kind == "CronJob" {
					managed = true
					break
				}
			}
			
			// Only process unmanaged pods
			if !managed {
				processCapabilityContainersInPod(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &podResults)
			}
		}
	}
	
	// Combine results
	results = append(results, controllerResults...)
	results = append(results, podResults...)
	
	// Deduplicate results
	return deduplicateCapabilityResults(results)
}

// deduplicateCapabilityResults removes duplicate entries that refer to the same resource
// It prioritizes higher level resources like Deployments over their child resources
func deduplicateCapabilityResults(results []CapabilityContainer) []CapabilityContainer {
	// Map container name to a slice of results for that container
	containerMap := make(map[string][]CapabilityContainer)
	
	// Group results by container name
	for _, result := range results {
		key := fmt.Sprintf("%s|%s", result.Namespace, result.Name)
		containerMap[key] = append(containerMap[key], result)
	}
	
	// For each container, prioritize higher-level resources
	var deduplicated []CapabilityContainer
	for _, resources := range containerMap {
		// Find highest priority resource (Deployment > ReplicaSet > Pod)
		highestPriority := findHighestPriorityCapabilityResource(resources)
		deduplicated = append(deduplicated, highestPriority)
	}
	
	return deduplicated
}

// findHighestPriorityCapabilityResource selects the highest priority resource from a slice of resources
// Priority: Deployment > StatefulSet > DaemonSet > Job > CronJob > ReplicaSet > Pod
func findHighestPriorityCapabilityResource(resources []CapabilityContainer) CapabilityContainer {
	if len(resources) == 0 {
		// This should never happen, but handle it gracefully
		return CapabilityContainer{}
	}
	
	if len(resources) == 1 {
		return resources[0]
	}
	
	// Define priority order (higher number = higher priority)
	kindPriority := map[string]int{
		"Pod":         1,
		"ReplicaSet":  2,
		"Job":         3,
		"CronJob":     4,
		"DaemonSet":   5,
		"StatefulSet": 6,
		"Deployment":  7,
	}
	
	highest := resources[0]
	highestPriority := kindPriority[highest.Kind]
	
	for _, res := range resources[1:] {
		priority := kindPriority[res.Kind]
		if priority > highestPriority {
			highest = res
			highestPriority = priority
		}
	}
	
	return highest
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
	var podResults []HostNamespaceWorkload
	var controllerResults []HostNamespaceWorkload
	
	// Create a map to track the pods we've already associated with controllers
	processedPods := make(map[string]bool)
	
	// First pass: collect all controller resources
	for _, item := range config.Items {
		if item.Kind == "Deployment" || item.Kind == "StatefulSet" || 
		   item.Kind == "DaemonSet" || item.Kind == "ReplicaSet" || 
		   item.Kind == "Job" || item.Kind == "CronJob" {
			// Resources that create pods
			checkWorkloadForHostNamespaces(item, &controllerResults)
			
			// Mark pods managed by this controller as processed
			key := fmt.Sprintf("%s/%s", item.Kind, item.Metadata.Name)
			markManagedPods(config, key, processedPods)
		}
	}
	
	// Second pass: collect all pods not already processed
	for _, item := range config.Items {
		if item.Kind == "Pod" {
			// Only process if we haven't seen this pod associated with a controller
			podKey := fmt.Sprintf("%s/%s", item.Metadata.Namespace, item.Metadata.Name)
			if !processedPods[podKey] {
				checkPodForHostNamespaces(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &podResults)
			}
		}
	}
	
	// Combine results
	results = append(results, controllerResults...)
	results = append(results, podResults...)
	
	// Deduplicate results
	return deduplicateHostNamespaceWorkloads(results)
}

// markManagedPods marks all pods managed by a specific controller
func markManagedPods(config *ClusterConfig, controllerKey string, processedPods map[string]bool) {
	for _, item := range config.Items {
		if item.Kind == "Pod" {
			for _, ref := range item.Metadata.OwnerReferences {
				refKey := fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
				if refKey == controllerKey {
					podKey := fmt.Sprintf("%s/%s", item.Metadata.Namespace, item.Metadata.Name)
					processedPods[podKey] = true
					break
				}
			}
		}
	}
}

// deduplicateHostNamespaceWorkloads removes duplicate entries that refer to the same workload
func deduplicateHostNamespaceWorkloads(results []HostNamespaceWorkload) []HostNamespaceWorkload {
	// Map to track unique workloads based on namespace + name
	seen := make(map[string]bool)
	var uniqueResults []HostNamespaceWorkload
	
	// First, add all controller resources (non-Pod) to the result list
	for _, result := range results {
		// For controller resources (Deployment, DaemonSet, etc.)
		if result.Kind != "Pod" {
			key := fmt.Sprintf("%s/%s/%s", result.Namespace, result.Kind, result.Name)
			if !seen[key] {
				seen[key] = true
				uniqueResults = append(uniqueResults, result)
			}
		}
	}
	
	// Then, add pods that are not controlled by any resources we've already added
	for _, result := range results {
		if result.Kind == "Pod" {
			// We always want to include these control-plane pods
			isControlPlanePod := false
			controlPlanePods := []string{
				"etcd-", "kube-apiserver-", "kube-controller-manager-", "kube-scheduler-",
			}
			
			for _, prefix := range controlPlanePods {
				if strings.HasPrefix(result.Name, prefix) {
					isControlPlanePod = true
					break
				}
			}
			
			key := fmt.Sprintf("%s/%s/%s", result.Namespace, result.Kind, result.Name)
			if isControlPlanePod && !seen[key] {
				seen[key] = true
				uniqueResults = append(uniqueResults, result)
			}
		}
	}
	
	return uniqueResults
}

// deduplicateHostNamespaceResults removes duplicate entries that refer to the same resource
// It prioritizes higher level resources like Deployments over their child resources
func deduplicateHostNamespaceResults(results []HostNamespaceWorkload) []HostNamespaceWorkload {
	// Map workload name to a slice of results for that workload
	workloadMap := make(map[string][]HostNamespaceWorkload)
	
	// Group results by workload name
	for _, result := range results {
		key := fmt.Sprintf("%s|%s", result.Namespace, result.Name)
		workloadMap[key] = append(workloadMap[key], result)
	}
	
	// For each workload, prioritize higher-level resources
	var deduplicated []HostNamespaceWorkload
	for _, resources := range workloadMap {
		// Find highest priority resource (Deployment > ReplicaSet > Pod)
		highestPriority := findHighestPriorityHostNamespaceResource(resources)
		deduplicated = append(deduplicated, highestPriority)
	}
	
	return deduplicated
}

// findHighestPriorityHostNamespaceResource selects the highest priority resource from a slice of resources
// Priority: Deployment > StatefulSet > DaemonSet > Job > CronJob > ReplicaSet > Pod
func findHighestPriorityHostNamespaceResource(resources []HostNamespaceWorkload) HostNamespaceWorkload {
	if len(resources) == 0 {
		// This should never happen, but handle it gracefully
		return HostNamespaceWorkload{}
	}
	
	if len(resources) == 1 {
		return resources[0]
	}
	
	// Define priority order (higher number = higher priority)
	kindPriority := map[string]int{
		"Pod":         1,
		"ReplicaSet":  2,
		"Job":         3,
		"CronJob":     4,
		"DaemonSet":   5,
		"StatefulSet": 6,
		"Deployment":  7,
	}
	
	highest := resources[0]
	highestPriority := kindPriority[highest.Kind]
	
	for _, res := range resources[1:] {
		priority := kindPriority[res.Kind]
		if priority > highestPriority {
			highest = res
			highestPriority = priority
		}
	}
	
	return highest
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

// GetHostPathVolumes identifies workloads with hostPath volumes
func GetHostPathVolumes(config *ClusterConfig) []HostPathVolume {
	var results []HostPathVolume
	var podResults []HostPathVolume
	var controllerResults []HostPathVolume
	
	// Create a map to track the pods we've already associated with controllers
	processedPods := make(map[string]bool)
	
	// First pass: collect all controller resources
	for _, item := range config.Items {
		if item.Kind == "Deployment" || item.Kind == "StatefulSet" || 
		   item.Kind == "DaemonSet" || item.Kind == "ReplicaSet" || 
		   item.Kind == "Job" || item.Kind == "CronJob" {
			// Resources that create pods
			checkWorkloadForHostPathVolumes(item, &controllerResults)
			
			// Mark pods managed by this controller as processed
			key := fmt.Sprintf("%s/%s", item.Kind, item.Metadata.Name)
			markManagedPods(config, key, processedPods)
		}
	}
	
	// Second pass: collect all pods not already processed
	for _, item := range config.Items {
		if item.Kind == "Pod" {
			// Only process if we haven't seen this pod associated with a controller
			podKey := fmt.Sprintf("%s/%s", item.Metadata.Namespace, item.Metadata.Name)
			if !processedPods[podKey] {
				checkPodForHostPathVolumes(item, item.Metadata.Name, item.Metadata.Namespace, item.Kind, &podResults)
			}
		}
	}
	
	// Combine results
	results = append(results, controllerResults...)
	results = append(results, podResults...)
	
	// Deduplicate results
	return deduplicateHostPathWorkloads(results)
}

// deduplicateHostPathWorkloads removes duplicate entries that refer to the same workload
func deduplicateHostPathWorkloads(results []HostPathVolume) []HostPathVolume {
	// Map to track unique workloads based on namespace + name
	seen := make(map[string]bool)
	var uniqueResults []HostPathVolume
	
	// First, add all controller resources (non-Pod) to the result list
	for _, result := range results {
		// For controller resources (Deployment, DaemonSet, etc.)
		if result.Kind != "Pod" {
			key := fmt.Sprintf("%s/%s/%s", result.Namespace, result.Kind, result.Name)
			if !seen[key] {
				seen[key] = true
				uniqueResults = append(uniqueResults, result)
			}
		}
	}
	
	// Then, add pods that are not controlled by any resources we've already added
	for _, result := range results {
		if result.Kind == "Pod" {
			// We always want to include these control-plane pods
			isControlPlanePod := false
			controlPlanePods := []string{
				"etcd-", "kube-apiserver-", "kube-controller-manager-", "kube-scheduler-",
			}
			
			for _, prefix := range controlPlanePods {
				if strings.HasPrefix(result.Name, prefix) {
					isControlPlanePod = true
					break
				}
			}
			
			key := fmt.Sprintf("%s/%s/%s", result.Namespace, result.Kind, result.Name)
			if isControlPlanePod && !seen[key] {
				seen[key] = true
				uniqueResults = append(uniqueResults, result)
			}
		}
	}
	
	return uniqueResults
}

// checkPodForHostPathVolumes examines a pod for hostPath volume usage
func checkPodForHostPathVolumes(item Item, name, namespace, kind string, results *[]HostPathVolume) {
	if spec, ok := item.Spec.(map[string]interface{}); ok {
		// Check if volumes are defined
		volumes, ok := spec["volumes"].([]interface{})
		if !ok || len(volumes) == 0 {
			return // No volumes defined
		}
		
		var hostPaths []string
		var readOnly []bool
		
		// Check each volume for hostPath type
		for _, v := range volumes {
			volume, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			
			// Look for hostPath volume type
			hostPath, ok := volume["hostPath"].(map[string]interface{})
			if !ok {
				continue // Not a hostPath volume
			}
			
			// Get the path from the hostPath
			path, ok := hostPath["path"].(string)
			if !ok || path == "" {
				continue // No path defined
			}
			
			// Get the volume name to check if it's mounted read-only
			volumeName, _ := volume["name"].(string)
			
			// Check if this volume is mounted read-only in any container
			isReadOnly := false
			
			// Check regular containers
			if containers, ok := spec["containers"].([]interface{}); ok {
				isReadOnly = checkContainersForReadOnlyMount(containers, volumeName)
			}
			
			// Check init containers
			if initContainers, ok := spec["initContainers"].([]interface{}); ok {
				if readOnly := checkContainersForReadOnlyMount(initContainers, volumeName); readOnly {
					isReadOnly = true
				}
			}
			
			hostPaths = append(hostPaths, path)
			readOnly = append(readOnly, isReadOnly)
		}
		
		// Only add to results if hostPath volumes were found
		if len(hostPaths) > 0 {
			*results = append(*results, HostPathVolume{
				Name:      name,
				Namespace: namespace,
				Kind:      kind,
				HostPaths: hostPaths,
				ReadOnly:  readOnly,
			})
		}
	}
}

// checkWorkloadForHostPathVolumes examines workload resources for hostPath volumes
func checkWorkloadForHostPathVolumes(item Item, results *[]HostPathVolume) {
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
				
				checkPodForHostPathVolumes(mockPod, workloadName, namespace, kind, results)
			}
		}
	}
}

// checkContainersForReadOnlyMount checks if a volume is mounted read-only in any container
func checkContainersForReadOnlyMount(containers []interface{}, volumeName string) bool {
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		volumeMounts, ok := container["volumeMounts"].([]interface{})
		if !ok {
			continue
		}
		
		for _, vm := range volumeMounts {
			mount, ok := vm.(map[string]interface{})
			if !ok {
				continue
			}
			
			name, _ := mount["name"].(string)
			if name == volumeName {
				readOnly, ok := mount["readOnly"].(bool)
				if ok && readOnly {
					return true
				}
			}
		}
	}
	
	return false
}