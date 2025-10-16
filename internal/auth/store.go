package auth

import (
	"bufio"
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
// $2a$12$hashedAPIKey1...
// $2a$12$hashedAPIKey2...
//
// [22222222-3333-4444-5555-666666666666]
// $2a$12$hashedAPIKey3...
//
// API keys are stored as bcrypt hashes for security.
// The file is monitored for changes and automatically reloaded.
type FileStore struct {
	mu          sync.RWMutex
	credentials map[uuid.UUID][]string // orgID -> list of hashed API keys
	filePath    string
	watcher     *fsnotify.Watcher
	stopChan    chan struct{}
}

// NewFileStore creates a new file-based credential store with automatic file watching
func NewFileStore(filePath string) (*FileStore, error) {
	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    filePath,
		stopChan:    make(chan struct{}),
	}

	// Load initial credentials
	if err := store.LoadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load credentials from file: %w", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	store.watcher = watcher

	// Add file to watcher
	if err := watcher.Add(filePath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch auth config file: %w", err)
	}

	// Start watching for file changes in background
	go store.watchFile()

	log.Printf("File watcher started for %s - credentials will auto-reload on changes", filePath)

	return store, nil
}

// watchFile monitors the auth config file for changes and reloads credentials
func (s *FileStore) watchFile() {
	// Debounce timer to avoid reloading multiple times for rapid changes
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only reload on write or create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDuration, func() {
					log.Printf("Detected change in %s, reloading credentials...", s.filePath)
					if err := s.Reload(); err != nil {
						log.Printf("ERROR: Failed to reload credentials: %v", err)
					} else {
						log.Println("Credentials reloaded successfully")
					}
				})
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)

		case <-s.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// Close stops the file watcher and cleans up resources
func (s *FileStore) Close() error {
	close(s.stopChan)
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
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
// Uses bcrypt comparison for hashed keys (which includes constant-time comparison internally)
func (s *FileStore) ValidateCredentials(orgID uuid.UUID, apiKey string) (bool, error) {
	s.mu.RLock()
	hashedKeys := s.credentials[orgID]
	s.mu.RUnlock()

	if len(hashedKeys) == 0 {
		return false, nil
	}

	// Check if the provided API key matches any of the hashed keys for this org
	for _, hashedKey := range hashedKeys {
		// Check if this is a bcrypt hash (starts with $2a$, $2b$, or $2y$)
		if strings.HasPrefix(hashedKey, "$2a$") || strings.HasPrefix(hashedKey, "$2b$") || strings.HasPrefix(hashedKey, "$2y$") {
			// Use bcrypt comparison for hashed keys
			err := bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(apiKey))
			if err == nil {
				return true, nil
			}
			// If error is not "mismatch", return the error
			if err != bcrypt.ErrMismatchedHashAndPassword {
				return false, fmt.Errorf("bcrypt comparison failed: %w", err)
			}
		} else {
			// Fallback to constant-time comparison for plain-text keys (backward compatibility)
			if subtle.ConstantTimeCompare([]byte(hashedKey), []byte(apiKey)) == 1 {
				return true, nil
			}
		}
	}

	return false, nil
}

// Reload reloads credentials from the file
func (s *FileStore) Reload() error {
	return s.LoadFromFile()
}
