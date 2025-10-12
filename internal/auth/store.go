package auth

import (
	"bufio"
	"crypto/subtle"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore provides an in-memory implementation of CredentialStore
type InMemoryStore struct {
	mu          sync.RWMutex
	credentials map[uuid.UUID]string // orgID -> apiKey
}

// NewInMemoryStore creates a new in-memory credential store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		credentials: make(map[uuid.UUID]string),
	}
}

// AddCredentials adds or updates credentials for an organization
func (s *InMemoryStore) AddCredentials(orgID uuid.UUID, apiKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[orgID] = apiKey
}

// ValidateCredentials checks if the provided credentials are valid
// Uses constant-time comparison to prevent timing attacks
func (s *InMemoryStore) ValidateCredentials(orgID uuid.UUID, apiKey string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storedKey, exists := s.credentials[orgID]
	if !exists {
		return false, nil
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(storedKey), []byte(apiKey)) == 1 {
		return true, nil
	}

	return false, nil
}

// RemoveCredentials removes credentials for an organization
func (s *InMemoryStore) RemoveCredentials(orgID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, orgID)
}

// FileStore provides a file-based implementation of CredentialStore
// It reads credentials from a configuration file with the following format:
//
// [11111111-2222-3333-4444-555555555555]
// demo-api-key-12345
// demo-api-key-12347
// demo-api-key-12349
//
// [22222222-3333-4444-5555-666666666666]
// another-api-key-67890
type FileStore struct {
	mu          sync.RWMutex
	credentials map[uuid.UUID][]string // orgID -> list of valid API keys
	filePath    string
}

// NewFileStore creates a new file-based credential store
func NewFileStore(filePath string) (*FileStore, error) {
	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    filePath,
	}

	if err := store.LoadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load credentials from file: %w", err)
	}

	return store, nil
}

// LoadFromFile reads credentials from the configuration file
func (s *FileStore) LoadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing credentials
	s.credentials = make(map[uuid.UUID][]string)

	file, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to open auth config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentOrgID uuid.UUID
	var hasCurrentOrg bool

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line is an org ID header [UUID]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			orgIDStr := strings.TrimSpace(line[1 : len(line)-1])
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				return fmt.Errorf("invalid UUID on line %d: %s", lineNum, orgIDStr)
			}
			currentOrgID = orgID
			hasCurrentOrg = true
			// Initialize the key list for this org if it doesn't exist
			if _, exists := s.credentials[currentOrgID]; !exists {
				s.credentials[currentOrgID] = []string{}
			}
			continue
		}

		// If we have a current org, this line is an API key
		if hasCurrentOrg {
			apiKey := line
			if apiKey != "" {
				s.credentials[currentOrgID] = append(s.credentials[currentOrgID], apiKey)
			}
		} else {
			return fmt.Errorf("API key on line %d appears before any org ID declaration", lineNum)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading auth config file: %w", err)
	}

	return nil
}

// ValidateCredentials checks if the provided credentials are valid
// Uses constant-time comparison to prevent timing attacks
func (s *FileStore) ValidateCredentials(orgID uuid.UUID, apiKey string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKeys, exists := s.credentials[orgID]
	if !exists {
		return false, nil
	}

	// Check if the provided API key matches any of the valid keys for this org
	for _, validKey := range apiKeys {
		if subtle.ConstantTimeCompare([]byte(validKey), []byte(apiKey)) == 1 {
			return true, nil
		}
	}

	return false, nil
}

// Reload reloads credentials from the file
func (s *FileStore) Reload() error {
	return s.LoadFromFile()
}
