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