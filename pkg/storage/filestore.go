package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/raesene/eolas/pkg/kubernetes"
)

// FileStore handles saving and loading Kubernetes configurations
type FileStore struct {
	StorageDir string
}

// NewFileStore creates a new file storage handler
func NewFileStore(storageDir string) (*FileStore, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &FileStore{
		StorageDir: storageDir,
	}, nil
}

// SaveConfig saves a Kubernetes configuration to the file store
func (fs *FileStore) SaveConfig(config *kubernetes.ClusterConfig, name string) error {
	if name == "" {
		name = fmt.Sprintf("cluster_%s", time.Now().Format("20060102_150405"))
	}
	
	filePath := filepath.Join(fs.StorageDir, fmt.Sprintf("%s.json", name))
	
	// Convert config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// LoadConfig loads a Kubernetes configuration from the file store
func (fs *FileStore) LoadConfig(name string) (*kubernetes.ClusterConfig, error) {
	filePath := filepath.Join(fs.StorageDir, fmt.Sprintf("%s.json", name))
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse JSON
	var config kubernetes.ClusterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// ListConfigs returns a list of saved configurations
func (fs *FileStore) ListConfigs() ([]string, error) {
	var configs []string
	
	entries, err := os.ReadDir(fs.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			configs = append(configs, filepath.Base(entry.Name()[:len(entry.Name())-5]))
		}
	}
	
	return configs, nil
}

// SaveConfigWithMetadata saves a configuration with full metadata (file-based implementation)
// Note: File storage has limited metadata support compared to SQLite
func (fs *FileStore) SaveConfigWithMetadata(config *kubernetes.ClusterConfig, metadata ConfigMetadata) error {
	// Use the name from metadata, or generate one if empty
	name := metadata.Name
	if name == "" {
		name = fmt.Sprintf("cluster_%s", time.Now().Format("20060102_150405"))
	}
	
	// File storage uses name as filename, so we'll save the basic config
	return fs.SaveConfig(config, name)
}

// LoadConfigByID loads a configuration by ID (file storage treats ID as filename)
func (fs *FileStore) LoadConfigByID(id string) (*kubernetes.ClusterConfig, error) {
	// In file storage, we'll treat the ID as the configuration name
	return fs.LoadConfig(id)
}

// GetConfigHistory returns configuration history (limited for file storage)
func (fs *FileStore) GetConfigHistory(name string) ([]ConfigMetadata, error) {
	// File storage doesn't maintain history, so return single entry if exists
	config, err := fs.LoadConfig(name)
	if err != nil {
		return nil, err
	}
	
	// Get file info for timestamp
	filePath := filepath.Join(fs.StorageDir, fmt.Sprintf("%s.json", name))
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	metadata := ConfigMetadata{
		ID:             name, // Use name as ID for file storage
		Name:           name,
		Timestamp:      fileInfo.ModTime(),
		CreatedAt:      fileInfo.ModTime(),
		ResourceCounts: kubernetes.GetResourceCounts(config),
	}
	
	return []ConfigMetadata{metadata}, nil
}

// GetConfigMetadata retrieves metadata for a configuration
func (fs *FileStore) GetConfigMetadata(id string) (*ConfigMetadata, error) {
	history, err := fs.GetConfigHistory(id)
	if err != nil {
		return nil, err
	}
	
	if len(history) == 0 {
		return nil, fmt.Errorf("configuration '%s' not found", id)
	}
	
	return &history[0], nil
}

// DeleteConfig removes a configuration file
func (fs *FileStore) DeleteConfig(id string) error {
	filePath := filepath.Join(fs.StorageDir, fmt.Sprintf("%s.json", id))
	
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configuration '%s' not found", id)
		}
		return fmt.Errorf("failed to delete configuration: %w", err)
	}
	
	return nil
}

// CompareConfigs compares two configurations (basic implementation for file storage)
func (fs *FileStore) CompareConfigs(id1, id2 string) (*ConfigComparison, error) {
	// Load both configurations
	config1, err := fs.LoadConfig(id1)
	if err != nil {
		return nil, fmt.Errorf("failed to load config %s: %w", id1, err)
	}
	
	config2, err := fs.LoadConfig(id2)
	if err != nil {
		return nil, fmt.Errorf("failed to load config %s: %w", id2, err)
	}
	
	// Get metadata for both
	metadata1, err := fs.GetConfigMetadata(id1)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for %s: %w", id1, err)
	}
	
	metadata2, err := fs.GetConfigMetadata(id2)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for %s: %w", id2, err)
	}
	
	// Compare resource counts
	resourceDiff := make(map[string]ResourceDifference)
	
	// Get all unique resource types
	allTypes := make(map[string]bool)
	for resourceType := range metadata1.ResourceCounts {
		allTypes[resourceType] = true
	}
	for resourceType := range metadata2.ResourceCounts {
		allTypes[resourceType] = true
	}
	
	// Calculate differences
	for resourceType := range allTypes {
		before := metadata1.ResourceCounts[resourceType]
		after := metadata2.ResourceCounts[resourceType]
		
		if before != after {
			resourceDiff[resourceType] = ResourceDifference{
				Before: before,
				After:  after,
				Change: after - before,
			}
		}
	}
	
	// Basic security comparison (real-time analysis)
	privileged1 := kubernetes.GetPrivilegedContainers(config1)
	privileged2 := kubernetes.GetPrivilegedContainers(config2)
	
	capability1 := kubernetes.GetCapabilityContainers(config1)
	capability2 := kubernetes.GetCapabilityContainers(config2)
	
	hostNamespace1 := kubernetes.GetHostNamespaceWorkloads(config1)
	hostNamespace2 := kubernetes.GetHostNamespaceWorkloads(config2)
	
	hostPath1 := kubernetes.GetHostPathVolumes(config1)
	hostPath2 := kubernetes.GetHostPathVolumes(config2)
	
	securityDiff := SecurityDifference{
		PrivilegedContainers: SecurityFindingDiff{
			Before: len(privileged1),
			After:  len(privileged2),
			Change: len(privileged2) - len(privileged1),
		},
		CapabilityContainers: SecurityFindingDiff{
			Before: len(capability1),
			After:  len(capability2),
			Change: len(capability2) - len(capability1),
		},
		HostNamespaceUsage: SecurityFindingDiff{
			Before: len(hostNamespace1),
			After:  len(hostNamespace2),
			Change: len(hostNamespace2) - len(hostNamespace1),
		},
		HostPathVolumes: SecurityFindingDiff{
			Before: len(hostPath1),
			After:  len(hostPath2),
			Change: len(hostPath2) - len(hostPath1),
		},
	}
	
	return &ConfigComparison{
		Config1:      *metadata1,
		Config2:      *metadata2,
		ResourceDiff: resourceDiff,
		SecurityDiff: securityDiff,
	}, nil
}

// Close is a no-op for file storage
func (fs *FileStore) Close() error {
	return nil
}