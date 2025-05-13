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