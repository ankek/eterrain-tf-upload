package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CSVStorage implements CSV file-based storage for terraform data uploads
type CSVStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// DataUpload represents a single data upload from Terraform provider
type DataUpload struct {
	Timestamp time.Time              `json:"timestamp"`
	OrgID     uuid.UUID              `json:"org_id"`
	Data      map[string]interface{} `json:"data"`
}

// NewCSVStorage creates a new CSV storage backend
func NewCSVStorage(dataDir string) (*CSVStorage, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Get absolute path for security validation
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for data directory: %w", err)
	}

	return &CSVStorage{
		dataDir: absDataDir,
	}, nil
}

// sanitizeFilePath validates and returns a safe file path for the given org ID
// This provides defense-in-depth against path traversal attacks
func (s *CSVStorage) sanitizeFilePath(orgID uuid.UUID) (string, error) {
	// Validate UUID string format (should be safe, but defense-in-depth)
	orgIDStr := orgID.String()

	// Check for path traversal characters
	if strings.ContainsAny(orgIDStr, "/\\..") {
		return "", fmt.Errorf("invalid org ID: contains path traversal characters")
	}

	// Build the filename
	filename := orgIDStr + ".csv"

	// Join with data directory
	filePath := filepath.Join(s.dataDir, filename)

	// Ensure the resulting path is within dataDir (canonical path check)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify the path is within the data directory
	if !strings.HasPrefix(absPath, s.dataDir+string(filepath.Separator)) && absPath != s.dataDir {
		return "", fmt.Errorf("invalid path: attempted directory traversal")
	}

	return filePath, nil
}

// AppendData appends data to the organization's CSV file
func (s *CSVStorage) AppendData(orgID uuid.UUID, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate and sanitize file path
	filePath, err := s.sanitizeFilePath(orgID)
	if err != nil {
		return fmt.Errorf("invalid org ID for file path: %w", err)
	}

	// Check if file exists to determine if we need to write headers
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	timestamp := time.Now().UTC()

	// Convert data to JSON string for storage
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write header if file is new
	if !fileExists {
		header := []string{"timestamp", "org_id", "data"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
	}

	// Write data row
	row := []string{
		timestamp.Format(time.RFC3339),
		orgID.String(),
		string(dataJSON),
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	return nil
}

// GetOrgData retrieves all data for an organization
func (s *CSVStorage) GetOrgData(orgID uuid.UUID) ([]DataUpload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate and sanitize file path
	filePath, err := s.sanitizeFilePath(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org ID for file path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []DataUpload{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	// Skip header and parse records
	uploads := make([]DataUpload, 0, len(records)-1)
	for i, record := range records {
		if i == 0 {
			// Skip header row
			continue
		}

		if len(record) < 3 {
			continue
		}

		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			continue
		}

		parsedOrgID, err := uuid.Parse(record[1])
		if err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(record[2]), &data); err != nil {
			continue
		}

		uploads = append(uploads, DataUpload{
			Timestamp: timestamp,
			OrgID:     parsedOrgID,
			Data:      data,
		})
	}

	return uploads, nil
}
