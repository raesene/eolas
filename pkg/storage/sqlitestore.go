package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/google/uuid"
	"github.com/raesene/eolas/pkg/kubernetes"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements the Store interface using SQLite database
type SQLiteStore struct {
	db       *sql.DB
	dbPath   string
}

// NewSQLiteStore creates a new SQLite storage handler
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	store := &SQLiteStore{
		db:     db,
		dbPath: dbPath,
	}
	
	// Initialize database schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}
	
	return store, nil
}

// initSchema creates the required database tables
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS configs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		raw_data TEXT NOT NULL,
		resource_counts TEXT,
		tags TEXT,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_name_timestamp ON configs(name, timestamp);
	CREATE INDEX IF NOT EXISTS idx_created_at ON configs(created_at);
	
	CREATE TABLE IF NOT EXISTS security_analysis (
		config_id TEXT PRIMARY KEY,
		privileged_containers TEXT,
		capability_containers TEXT,
		host_namespace_workloads TEXT,
		host_path_volumes TEXT,
		FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE
	);
	`
	
	_, err := s.db.Exec(schema)
	return err
}

// SaveConfig saves a configuration with a generated name (legacy interface)
func (s *SQLiteStore) SaveConfig(config *kubernetes.ClusterConfig, name string) error {
	metadata := ConfigMetadata{
		ID:             uuid.New().String(),
		Name:           name,
		Timestamp:      time.Now(),
		CreatedAt:      time.Now(),
		ResourceCounts: kubernetes.GetResourceCounts(config),
	}
	
	return s.SaveConfigWithMetadata(config, metadata)
}

// SaveConfigWithMetadata saves a configuration with full metadata
func (s *SQLiteStore) SaveConfigWithMetadata(config *kubernetes.ClusterConfig, metadata ConfigMetadata) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Generate ID if not provided
	if metadata.ID == "" {
		metadata.ID = uuid.New().String()
	}
	
	// Set timestamps if not provided
	if metadata.Timestamp.IsZero() {
		metadata.Timestamp = time.Now()
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}
	
	// Set resource counts if not provided
	if metadata.ResourceCounts == nil {
		metadata.ResourceCounts = kubernetes.GetResourceCounts(config)
	}
	
	// Serialize config data
	rawData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Serialize metadata fields
	resourceCountsJSON, err := json.Marshal(metadata.ResourceCounts)
	if err != nil {
		return fmt.Errorf("failed to marshal resource counts: %w", err)
	}
	
	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	
	// Insert config record
	_, err = tx.Exec(`
		INSERT INTO configs (id, name, timestamp, raw_data, resource_counts, tags, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, metadata.ID, metadata.Name, metadata.Timestamp, string(rawData), 
		string(resourceCountsJSON), string(tagsJSON), metadata.Description, metadata.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to insert config: %w", err)
	}
	
	// Pre-compute and store security analysis
	if err := s.saveSecurityAnalysis(tx, metadata.ID, config); err != nil {
		return fmt.Errorf("failed to save security analysis: %w", err)
	}
	
	return tx.Commit()
}

// saveSecurityAnalysis pre-computes and stores security analysis results
func (s *SQLiteStore) saveSecurityAnalysis(tx *sql.Tx, configID string, config *kubernetes.ClusterConfig) error {
	analysis := StoredSecurityAnalysis{
		ConfigID:               configID,
		PrivilegedContainers:   kubernetes.GetPrivilegedContainers(config),
		CapabilityContainers:   kubernetes.GetCapabilityContainers(config),
		HostNamespaceWorkloads: kubernetes.GetHostNamespaceWorkloads(config),
		HostPathVolumes:        kubernetes.GetHostPathVolumes(config),
	}
	
	// Serialize analysis results
	privilegedJSON, _ := json.Marshal(analysis.PrivilegedContainers)
	capabilityJSON, _ := json.Marshal(analysis.CapabilityContainers)
	hostNamespaceJSON, _ := json.Marshal(analysis.HostNamespaceWorkloads)
	hostPathJSON, _ := json.Marshal(analysis.HostPathVolumes)
	
	_, err := tx.Exec(`
		INSERT INTO security_analysis (config_id, privileged_containers, capability_containers, 
			host_namespace_workloads, host_path_volumes)
		VALUES (?, ?, ?, ?, ?)
	`, configID, string(privilegedJSON), string(capabilityJSON), 
		string(hostNamespaceJSON), string(hostPathJSON))
	
	return err
}

// LoadConfig loads a configuration by name (loads most recent if multiple exist)
func (s *SQLiteStore) LoadConfig(name string) (*kubernetes.ClusterConfig, error) {
	var rawData string
	err := s.db.QueryRow(`
		SELECT raw_data FROM configs 
		WHERE name = ? 
		ORDER BY timestamp DESC 
		LIMIT 1
	`, name).Scan(&rawData)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query config: %w", err)
	}
	
	var config kubernetes.ClusterConfig
	if err := json.Unmarshal([]byte(rawData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// LoadConfigByID loads a configuration by its unique ID
func (s *SQLiteStore) LoadConfigByID(id string) (*kubernetes.ClusterConfig, error) {
	var rawData string
	err := s.db.QueryRow(`
		SELECT raw_data FROM configs WHERE id = ?
	`, id).Scan(&rawData)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration with ID '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to query config: %w", err)
	}
	
	var config kubernetes.ClusterConfig
	if err := json.Unmarshal([]byte(rawData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// GetConfigMetadata retrieves metadata for a configuration by ID
func (s *SQLiteStore) GetConfigMetadata(id string) (*ConfigMetadata, error) {
	var metadata ConfigMetadata
	var resourceCountsJSON, tagsJSON string
	
	err := s.db.QueryRow(`
		SELECT id, name, timestamp, resource_counts, tags, description, created_at
		FROM configs WHERE id = ?
	`, id).Scan(&metadata.ID, &metadata.Name, &metadata.Timestamp, 
		&resourceCountsJSON, &tagsJSON, &metadata.Description, &metadata.CreatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration with ID '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to query config metadata: %w", err)
	}
	
	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(resourceCountsJSON), &metadata.ResourceCounts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource counts: %w", err)
	}
	
	if tagsJSON != "null" && tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &metadata.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}
	
	return &metadata, nil
}

// ListConfigs returns a list of configuration names (legacy interface)
func (s *SQLiteStore) ListConfigs() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT name FROM configs ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()
	
	var configs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan config name: %w", err)
		}
		configs = append(configs, name)
	}
	
	return configs, rows.Err()
}

// GetConfigHistory returns all configurations for a given name, ordered by timestamp
func (s *SQLiteStore) GetConfigHistory(name string) ([]ConfigMetadata, error) {
	rows, err := s.db.Query(`
		SELECT id, name, timestamp, resource_counts, tags, description, created_at
		FROM configs 
		WHERE name = ? 
		ORDER BY timestamp DESC
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to query config history: %w", err)
	}
	defer rows.Close()
	
	var history []ConfigMetadata
	for rows.Next() {
		var metadata ConfigMetadata
		var resourceCountsJSON, tagsJSON string
		
		err := rows.Scan(&metadata.ID, &metadata.Name, &metadata.Timestamp,
			&resourceCountsJSON, &tagsJSON, &metadata.Description, &metadata.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config metadata: %w", err)
		}
		
		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(resourceCountsJSON), &metadata.ResourceCounts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource counts: %w", err)
		}
		
		if tagsJSON != "null" && tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &metadata.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}
		
		history = append(history, metadata)
	}
	
	return history, rows.Err()
}

// CompareConfigs compares two configurations and returns the differences
func (s *SQLiteStore) CompareConfigs(id1, id2 string) (*ConfigComparison, error) {
	// Get metadata for both configurations
	metadata1, err := s.GetConfigMetadata(id1)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for config %s: %w", id1, err)
	}
	
	metadata2, err := s.GetConfigMetadata(id2)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for config %s: %w", id2, err)
	}
	
	// Compare resource counts
	resourceDiff := make(map[string]ResourceDifference)
	
	// Get all unique resource types from both configs
	allTypes := make(map[string]bool)
	for resourceType := range metadata1.ResourceCounts {
		allTypes[resourceType] = true
	}
	for resourceType := range metadata2.ResourceCounts {
		allTypes[resourceType] = true
	}
	
	// Calculate differences for each resource type
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
	
	// Compare security analysis results
	securityDiff, err := s.compareSecurityAnalysis(id1, id2)
	if err != nil {
		return nil, fmt.Errorf("failed to compare security analysis: %w", err)
	}
	
	return &ConfigComparison{
		Config1:      *metadata1,
		Config2:      *metadata2,
		ResourceDiff: resourceDiff,
		SecurityDiff: *securityDiff,
	}, nil
}

// compareSecurityAnalysis compares security findings between two configurations
func (s *SQLiteStore) compareSecurityAnalysis(id1, id2 string) (*SecurityDifference, error) {
	// Get security analysis for both configs
	analysis1, err := s.getSecurityAnalysis(id1)
	if err != nil {
		return nil, fmt.Errorf("failed to get security analysis for config %s: %w", id1, err)
	}
	
	analysis2, err := s.getSecurityAnalysis(id2)
	if err != nil {
		return nil, fmt.Errorf("failed to get security analysis for config %s: %w", id2, err)
	}
	
	return &SecurityDifference{
		PrivilegedContainers: SecurityFindingDiff{
			Before: len(analysis1.PrivilegedContainers),
			After:  len(analysis2.PrivilegedContainers),
			Change: len(analysis2.PrivilegedContainers) - len(analysis1.PrivilegedContainers),
		},
		CapabilityContainers: SecurityFindingDiff{
			Before: len(analysis1.CapabilityContainers),
			After:  len(analysis2.CapabilityContainers),
			Change: len(analysis2.CapabilityContainers) - len(analysis1.CapabilityContainers),
		},
		HostNamespaceUsage: SecurityFindingDiff{
			Before: len(analysis1.HostNamespaceWorkloads),
			After:  len(analysis2.HostNamespaceWorkloads),
			Change: len(analysis2.HostNamespaceWorkloads) - len(analysis1.HostNamespaceWorkloads),
		},
		HostPathVolumes: SecurityFindingDiff{
			Before: len(analysis1.HostPathVolumes),
			After:  len(analysis2.HostPathVolumes),
			Change: len(analysis2.HostPathVolumes) - len(analysis1.HostPathVolumes),
		},
	}, nil
}

// getSecurityAnalysis retrieves stored security analysis for a configuration
func (s *SQLiteStore) getSecurityAnalysis(configID string) (*StoredSecurityAnalysis, error) {
	var privilegedJSON, capabilityJSON, hostNamespaceJSON, hostPathJSON string
	
	err := s.db.QueryRow(`
		SELECT privileged_containers, capability_containers, 
			host_namespace_workloads, host_path_volumes
		FROM security_analysis WHERE config_id = ?
	`, configID).Scan(&privilegedJSON, &capabilityJSON, &hostNamespaceJSON, &hostPathJSON)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("security analysis for config '%s' not found", configID)
		}
		return nil, fmt.Errorf("failed to query security analysis: %w", err)
	}
	
	analysis := &StoredSecurityAnalysis{ConfigID: configID}
	
	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(privilegedJSON), &analysis.PrivilegedContainers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal privileged containers: %w", err)
	}
	
	if err := json.Unmarshal([]byte(capabilityJSON), &analysis.CapabilityContainers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capability containers: %w", err)
	}
	
	if err := json.Unmarshal([]byte(hostNamespaceJSON), &analysis.HostNamespaceWorkloads); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host namespace workloads: %w", err)
	}
	
	if err := json.Unmarshal([]byte(hostPathJSON), &analysis.HostPathVolumes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host path volumes: %w", err)
	}
	
	return analysis, nil
}

// DeleteConfig removes a configuration and its associated security analysis
func (s *SQLiteStore) DeleteConfig(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete security analysis first (foreign key constraint)
	_, err = tx.Exec("DELETE FROM security_analysis WHERE config_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete security analysis: %w", err)
	}
	
	// Delete configuration
	result, err := tx.Exec("DELETE FROM configs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("configuration with ID '%s' not found", id)
	}
	
	return tx.Commit()
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}