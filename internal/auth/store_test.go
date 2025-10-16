package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestInMemoryStore tests the basic in-memory store functionality
func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()
	orgID := uuid.New()
	apiKey := "test-api-key"

	// Test adding credentials
	store.AddCredentials(orgID, apiKey)

	// Test validation with correct credentials
	valid, err := store.ValidateCredentials(orgID, apiKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Expected credentials to be valid")
	}

	// Test validation with wrong API key
	valid, err = store.ValidateCredentials(orgID, "wrong-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected credentials to be invalid")
	}

	// Test validation with non-existent org
	valid, err = store.ValidateCredentials(uuid.New(), apiKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected credentials to be invalid for non-existent org")
	}

	// Test removing credentials
	store.RemoveCredentials(orgID)
	valid, err = store.ValidateCredentials(orgID, apiKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected credentials to be invalid after removal")
	}
}

func TestInMemoryStoreConcurrency(t *testing.T) {
	store := NewInMemoryStore()
	orgID := uuid.New()
	apiKey := "test-key"

	store.AddCredentials(orgID, apiKey)

	// Test concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			valid, err := store.ValidateCredentials(orgID, apiKey)
			if err != nil {
				errors <- err
				return
			}
			if !valid {
				errors <- fmt.Errorf("validation failed")
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent validation error: %v", err)
	}
}

func TestFileStoreLoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, store *FileStore)
	}{
		{
			name: "valid config with bcrypt hashes",
			content: `[11111111-2222-3333-4444-555555555555]
$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYqRTbXNZ9i
$2a$12$EyQmfAqPNvRH3T5nP7yVweHj5LLaP8m3P7yqHW5TY6Z8P7yqHW5TY`,
			wantErr: false,
			validate: func(t *testing.T, store *FileStore) {
				orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
				store.mu.RLock()
				keys := store.credentials[orgID]
				store.mu.RUnlock()

				if len(keys) != 2 {
					t.Errorf("Expected 2 keys, got %d", len(keys))
				}
			},
		},
		{
			name: "multiple orgs",
			content: `[11111111-2222-3333-4444-555555555555]
$2a$12$hash1

[22222222-3333-4444-5555-666666666666]
$2a$12$hash2`,
			wantErr: false,
			validate: func(t *testing.T, store *FileStore) {
				store.mu.RLock()
				defer store.mu.RUnlock()

				if len(store.credentials) != 2 {
					t.Errorf("Expected 2 orgs, got %d", len(store.credentials))
				}
			},
		},
		{
			name: "comments and empty lines",
			content: `# Comment
[11111111-2222-3333-4444-555555555555]
# Another comment
$2a$12$hash1

`,
			wantErr: false,
		},
		{
			name: "invalid UUID",
			content: `[invalid-uuid]
$2a$12$hash1`,
			wantErr:     true,
			errContains: "invalid UUID",
		},
		{
			name: "key before org declaration",
			content: `$2a$12$hash1
[11111111-2222-3333-4444-555555555555]`,
			wantErr:     true,
			errContains: "appears before any org ID",
		},
		{
			name:    "empty file",
			content: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "auth.cfg")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			store := &FileStore{
				credentials: make(map[uuid.UUID][]string),
				filePath:    tmpFile,
			}

			err := store.LoadFromFile()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, store)
			}
		})
	}
}

func TestFileStoreLoadFromFileNotFound(t *testing.T) {
	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    "/nonexistent/file.cfg",
	}

	err := store.LoadFromFile()
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFileStoreValidateCredentialsBcrypt(t *testing.T) {
	// Create a test file with bcrypt hashes
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	apiKey := "my-secret-key"

	// Hash the API key
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	if err != nil {
		t.Fatalf("Failed to hash API key: %v", err)
	}

	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create store
	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    tmpFile,
	}

	if err := store.LoadFromFile(); err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Test valid credentials
	valid, err := store.ValidateCredentials(orgID, apiKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Expected credentials to be valid")
	}

	// Test invalid credentials
	valid, err = store.ValidateCredentials(orgID, "wrong-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected credentials to be invalid")
	}

	// Test non-existent org
	valid, err = store.ValidateCredentials(uuid.New(), apiKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected credentials to be invalid for non-existent org")
	}
}

func TestFileStoreValidateCredentialsMultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	keys := []string{"key1", "key2", "key3"}

	// Create content with multiple hashed keys
	var content strings.Builder
	content.WriteString(fmt.Sprintf("[%s]\n", orgID.String()))
	for _, key := range keys {
		hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(key), bcryptCost)
		content.WriteString(string(hashedBytes) + "\n")
	}

	if err := os.WriteFile(tmpFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    tmpFile,
	}

	if err := store.LoadFromFile(); err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// All keys should be valid
	for _, key := range keys {
		valid, err := store.ValidateCredentials(orgID, key)
		if err != nil {
			t.Fatalf("Validation error for key %s: %v", key, err)
		}
		if !valid {
			t.Errorf("Expected key %s to be valid", key)
		}
	}

	// Wrong key should be invalid
	valid, err := store.ValidateCredentials(orgID, "wrong-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Expected wrong key to be invalid")
	}
}

func TestFileStoreValidateCredentialsPlaintext(t *testing.T) {
	// Test backward compatibility with plaintext keys
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	plainKey := "plaintext-key"

	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), plainKey)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    tmpFile,
	}

	if err := store.LoadFromFile(); err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Plaintext key should match with constant-time comparison
	valid, err := store.ValidateCredentials(orgID, plainKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Expected plaintext key to be valid")
	}
}

func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create new file store (should start watching)
	store, err := NewFileStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Verify credentials were loaded
	valid, err := store.ValidateCredentials(orgID, "test-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Expected credentials to be loaded")
	}

	// Verify watcher is running
	if store.watcher == nil {
		t.Error("Expected watcher to be initialized")
	}
}

func TestNewFileStoreFileNotFound(t *testing.T) {
	_, err := NewFileStore("/nonexistent/file.cfg")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFileStoreReload(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Initial content
	hashedBytes1, _ := bcrypt.GenerateFromPassword([]byte("key1"), bcryptCost)
	content1 := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes1))
	if err := os.WriteFile(tmpFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    tmpFile,
	}

	if err := store.LoadFromFile(); err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Verify initial key works
	valid, _ := store.ValidateCredentials(orgID, "key1")
	if !valid {
		t.Error("Initial key should be valid")
	}

	// Update file
	hashedBytes2, _ := bcrypt.GenerateFromPassword([]byte("key2"), bcryptCost)
	content2 := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes2))
	if err := os.WriteFile(tmpFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write updated file: %v", err)
	}

	// Reload
	if err := store.Reload(); err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Old key should no longer work
	valid, _ = store.ValidateCredentials(orgID, "key1")
	if valid {
		t.Error("Old key should be invalid after reload")
	}

	// New key should work
	valid, _ = store.ValidateCredentials(orgID, "key2")
	if !valid {
		t.Error("New key should be valid after reload")
	}
}

func TestFileStoreClose(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	content := "[11111111-2222-3333-4444-555555555555]\ntest-key\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store, err := NewFileStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Close should not error
	if err := store.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Multiple closes should be safe
	if err := store.Close(); err != nil {
		t.Errorf("Second close returned error: %v", err)
	}
}

func TestFileStoreWatchFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Initial content
	hashedBytes1, _ := bcrypt.GenerateFromPassword([]byte("key1"), bcryptCost)
	content1 := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes1))
	if err := os.WriteFile(tmpFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create store (starts watching)
	store, err := NewFileStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify initial key
	valid, _ := store.ValidateCredentials(orgID, "key1")
	if !valid {
		t.Fatal("Initial key should be valid")
	}

	// Update file to trigger watch event
	hashedBytes2, _ := bcrypt.GenerateFromPassword([]byte("key2"), bcryptCost)
	content2 := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes2))
	if err := os.WriteFile(tmpFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write updated file: %v", err)
	}

	// Wait for debounce and reload (500ms debounce + some buffer)
	time.Sleep(1 * time.Second)

	// New key should now work
	valid, _ = store.ValidateCredentials(orgID, "key2")
	if !valid {
		t.Error("New key should be valid after file watch reload")
	}

	// Old key should not work
	valid, _ = store.ValidateCredentials(orgID, "key1")
	if valid {
		t.Error("Old key should be invalid after file watch reload")
	}
}

func TestFileStoreWatchFileDebounce(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("key1"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store, err := NewFileStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Make multiple rapid changes
	for i := 0; i < 5; i++ {
		hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("key%d", i)), bcryptCost)
		content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
		os.WriteFile(tmpFile, []byte(content), 0644)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce to settle
	time.Sleep(1 * time.Second)

	// Should have the last key only
	valid, _ := store.ValidateCredentials(orgID, "key4")
	if !valid {
		t.Error("Final key should be valid after debounced reload")
	}
}

func TestFileStoreConcurrentValidation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	apiKey := "test-key"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store, err := NewFileStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Test concurrent validations
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			valid, err := store.ValidateCredentials(orgID, apiKey)
			if err != nil {
				errors <- err
				return
			}
			if !valid {
				errors <- fmt.Errorf("validation failed")
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent validation error: %v", err)
	}
}

func BenchmarkFileStoreValidateCredentialsBcrypt(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	apiKey := "benchmark-key"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(tmpFile, []byte(content), 0644)

	store, _ := NewFileStore(tmpFile)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.ValidateCredentials(orgID, apiKey)
	}
}

func BenchmarkFileStoreValidateCredentialsParallel(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	apiKey := "benchmark-key"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(tmpFile, []byte(content), 0644)

	store, _ := NewFileStore(tmpFile)
	defer store.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			store.ValidateCredentials(orgID, apiKey)
		}
	})
}

func BenchmarkFileStoreLoadFromFile(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.cfg")

	// Create file with multiple orgs and keys
	var content strings.Builder
	for i := 0; i < 10; i++ {
		orgID := uuid.New()
		content.WriteString(fmt.Sprintf("[%s]\n", orgID.String()))
		for j := 0; j < 5; j++ {
			hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("key-%d-%d", i, j)), bcryptCost)
			content.WriteString(string(hashedBytes) + "\n")
		}
	}
	os.WriteFile(tmpFile, []byte(content.String()), 0644)

	store := &FileStore{
		credentials: make(map[uuid.UUID][]string),
		filePath:    tmpFile,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.LoadFromFile()
	}
}
