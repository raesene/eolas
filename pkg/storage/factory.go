package storage

import (
	"fmt"
	"path/filepath"
)

// Backend represents the available storage backend types
type Backend string

const (
	// FileBackend represents file-based storage
	FileBackend Backend = "file"
	// SQLiteBackend represents SQLite database storage
	SQLiteBackend Backend = "sqlite"
)

// StorageConfig contains configuration for creating a storage backend
type StorageConfig struct {
	Backend    Backend
	StorageDir string
	UseHomeDir bool
}

// NewStore creates a new storage backend based on the provided configuration
func NewStore(config StorageConfig) (Store, error) {
	switch config.Backend {
	case FileBackend:
		return NewFileStore(config.StorageDir)
	case SQLiteBackend:
		// For SQLite, use the storage directory to determine database location
		dbPath := filepath.Join(config.StorageDir, "eolas.db")
		return NewSQLiteStore(dbPath)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", config.Backend)
	}
}

// ValidateBackend checks if the provided backend string is valid
func ValidateBackend(backend string) error {
	switch Backend(backend) {
	case FileBackend, SQLiteBackend:
		return nil
	default:
		return fmt.Errorf("invalid backend '%s'. Valid backends are: file, sqlite", backend)
	}
}