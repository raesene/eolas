package storage

import (
	"time"
	
	"github.com/raesene/eolas/pkg/kubernetes"
)

// ConfigMetadata contains metadata about a stored configuration
type ConfigMetadata struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Timestamp      time.Time         `json:"timestamp"`
	CreatedAt      time.Time         `json:"created_at"`
	ResourceCounts map[string]int    `json:"resource_counts"`
	Tags           map[string]string `json:"tags,omitempty"`
	Description    string            `json:"description,omitempty"`
}

// ConfigComparison represents the differences between two configurations
type ConfigComparison struct {
	Config1      ConfigMetadata                    `json:"config1"`
	Config2      ConfigMetadata                    `json:"config2"`
	ResourceDiff map[string]ResourceDifference     `json:"resource_diff"`
	SecurityDiff SecurityDifference                `json:"security_diff"`
}

// ResourceDifference represents changes in resource counts
type ResourceDifference struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Change int `json:"change"`
}

// SecurityDifference represents changes in security analysis results
type SecurityDifference struct {
	PrivilegedContainers SecurityFindingDiff `json:"privileged_containers"`
	CapabilityContainers SecurityFindingDiff `json:"capability_containers"`
	HostNamespaceUsage   SecurityFindingDiff `json:"host_namespace_usage"`
	HostPathVolumes      SecurityFindingDiff `json:"host_path_volumes"`
}

// SecurityFindingDiff represents changes in security findings
type SecurityFindingDiff struct {
	Before    int      `json:"before"`
	After     int      `json:"after"`
	Change    int      `json:"change"`
	Added     []string `json:"added,omitempty"`
	Removed   []string `json:"removed,omitempty"`
	Modified  []string `json:"modified,omitempty"`
}

// StoredSecurityAnalysis represents pre-computed security analysis results
type StoredSecurityAnalysis struct {
	ConfigID                string                               `json:"config_id"`
	PrivilegedContainers    []kubernetes.PrivilegedContainer    `json:"privileged_containers"`
	CapabilityContainers    []kubernetes.CapabilityContainer    `json:"capability_containers"`
	HostNamespaceWorkloads  []kubernetes.HostNamespaceWorkload  `json:"host_namespace_workloads"`
	HostPathVolumes         []kubernetes.HostPathVolume         `json:"host_path_volumes"`
}