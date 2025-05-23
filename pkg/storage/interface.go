package storage

import (
	"github.com/raesene/eolas/pkg/kubernetes"
)

// Store defines the interface for configuration storage backends
type Store interface {
	// Basic storage operations (existing functionality)
	SaveConfig(config *kubernetes.ClusterConfig, name string) error
	LoadConfig(name string) (*kubernetes.ClusterConfig, error)
	ListConfigs() ([]string, error)
	
	// Enhanced operations for comparison and history
	SaveConfigWithMetadata(config *kubernetes.ClusterConfig, metadata ConfigMetadata) error
	LoadConfigByID(id string) (*kubernetes.ClusterConfig, error)
	GetConfigHistory(name string) ([]ConfigMetadata, error)
	GetConfigMetadata(id string) (*ConfigMetadata, error)
	DeleteConfig(id string) error
	
	// Comparison operations
	CompareConfigs(id1, id2 string) (*ConfigComparison, error)
	
	// Security analysis operations
	GetSecurityAnalysisHistory(name string) ([]StoredSecurityAnalysis, error)
	
	// Storage management
	Close() error
}