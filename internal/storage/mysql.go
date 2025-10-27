package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// MySQLStorage implements MySQL database-based storage for terraform data uploads
type MySQLStorage struct {
	db         *sql.DB
	dbName     string
	mu         sync.RWMutex
	tableMutex sync.Mutex // Protects table creation
}

// NewMySQLStorage creates a new MySQL storage backend with retry logic
func NewMySQLStorage(dsn string, dbName string) (*MySQLStorage, error) {
	// Connect to MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Test connection with retry logic (for Docker startup delays)
	maxRetries := 30
	retryDelay := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			log.Printf("MySQL not ready yet (attempt %d/%d), retrying in %v...", i+1, maxRetries, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL after %d attempts: %w", maxRetries, err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &MySQLStorage{
		db:     db,
		dbName: dbName,
	}, nil
}

// sanitizeTableName ensures the table name is safe to use
// Tables are named after organization UUIDs
func (s *MySQLStorage) sanitizeTableName(orgID uuid.UUID) string {
	// Replace hyphens with underscores for MySQL table name
	// MySQL table names cannot contain hyphens
	tableName := strings.ReplaceAll(orgID.String(), "-", "_")
	return fmt.Sprintf("org_%s", tableName)
}

// ensureTableExists creates the organization's table if it doesn't exist
func (s *MySQLStorage) ensureTableExists(orgID uuid.UUID) error {
	s.tableMutex.Lock()
	defer s.tableMutex.Unlock()

	tableName := s.sanitizeTableName(orgID)

	// Create table if not exists
	// Structure: timestamp, org_id, data (same as CSV)
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			timestamp DATETIME(6) NOT NULL,
			org_id VARCHAR(36) NOT NULL,
			data JSON NOT NULL,
			INDEX idx_timestamp (timestamp),
			INDEX idx_org_id (org_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)

	_, err := s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	return nil
}

// AppendData appends data to the organization's MySQL table
func (s *MySQLStorage) AppendData(orgID uuid.UUID, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure table exists
	if err := s.ensureTableExists(orgID); err != nil {
		return err
	}

	tableName := s.sanitizeTableName(orgID)
	timestamp := time.Now().UTC()

	// Convert data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Insert data
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (timestamp, org_id, data)
		VALUES (?, ?, ?)
	`, tableName)

	_, err = s.db.Exec(insertSQL, timestamp, orgID.String(), dataJSON)
	if err != nil {
		return fmt.Errorf("failed to insert data into %s: %w", tableName, err)
	}

	return nil
}

// GetOrgData retrieves all data for an organization
func (s *MySQLStorage) GetOrgData(orgID uuid.UUID) ([]DataUpload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tableName := s.sanitizeTableName(orgID)

	// Check if table exists
	checkTableSQL := `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ?
		AND table_name = ?
	`
	var tableCount int
	err := s.db.QueryRow(checkTableSQL, s.dbName, tableName).Scan(&tableCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check if table exists: %w", err)
	}

	if tableCount == 0 {
		// Table doesn't exist, return empty array
		return []DataUpload{}, nil
	}

	// Query all data
	querySQL := fmt.Sprintf(`
		SELECT timestamp, org_id, data
		FROM %s
		ORDER BY timestamp ASC
	`, tableName)

	rows, err := s.db.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query data from %s: %w", tableName, err)
	}
	defer rows.Close()

	uploads := make([]DataUpload, 0)
	for rows.Next() {
		var timestamp time.Time
		var orgIDStr string
		var dataJSON []byte

		if err := rows.Scan(&timestamp, &orgIDStr, &dataJSON); err != nil {
			continue
		}

		parsedOrgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			continue
		}

		// Extract report_name if present
		reportName := ""
		if name, ok := data["report_name"].(string); ok {
			reportName = name
		}

		uploads = append(uploads, DataUpload{
			Timestamp:  timestamp,
			OrgID:      parsedOrgID,
			ReportName: reportName,
			Data:       data,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return uploads, nil
}

// Close closes the database connection
func (s *MySQLStorage) Close() error {
	return s.db.Close()
}
