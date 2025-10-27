package storage

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// DualStorage implements storage that writes to both CSV and MySQL
type DualStorage struct {
	csv   *CSVStorage
	mysql *MySQLStorage
}

// NewDualStorage creates a new dual storage backend (CSV + MySQL)
func NewDualStorage(csv *CSVStorage, mysql *MySQLStorage) *DualStorage {
	return &DualStorage{
		csv:   csv,
		mysql: mysql,
	}
}

// AppendData appends data to both CSV and MySQL storage
// If one storage fails, it logs the error but continues with the other
func (s *DualStorage) AppendData(orgID uuid.UUID, data map[string]interface{}) error {
	var csvErr, mysqlErr error

	// Write to CSV
	csvErr = s.csv.AppendData(orgID, data)
	if csvErr != nil {
		log.Printf("ERROR: Failed to write to CSV storage for org %s: %v", orgID, csvErr)
	}

	// Write to MySQL
	mysqlErr = s.mysql.AppendData(orgID, data)
	if mysqlErr != nil {
		log.Printf("ERROR: Failed to write to MySQL storage for org %s: %v", orgID, mysqlErr)
	}

	// Return error if both failed
	if csvErr != nil && mysqlErr != nil {
		return fmt.Errorf("both CSV and MySQL storage failed: CSV error: %v, MySQL error: %v", csvErr, mysqlErr)
	}

	// Return error if only one failed (for visibility, but data was still saved)
	if csvErr != nil {
		return fmt.Errorf("CSV storage failed (data saved to MySQL): %w", csvErr)
	}
	if mysqlErr != nil {
		return fmt.Errorf("MySQL storage failed (data saved to CSV): %w", mysqlErr)
	}

	return nil
}

// GetOrgData retrieves data from CSV storage (primary source)
// Falls back to MySQL if CSV fails
func (s *DualStorage) GetOrgData(orgID uuid.UUID) ([]DataUpload, error) {
	// Try CSV first
	data, err := s.csv.GetOrgData(orgID)
	if err == nil {
		return data, nil
	}

	log.Printf("WARNING: Failed to read from CSV storage for org %s: %v, falling back to MySQL", orgID, err)

	// Fall back to MySQL
	return s.mysql.GetOrgData(orgID)
}

// Close closes both storage backends
func (s *DualStorage) Close() error {
	// MySQL needs to be closed, CSV doesn't have a Close method
	return s.mysql.Close()
}
