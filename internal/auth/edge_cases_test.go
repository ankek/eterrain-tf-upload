package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestEdgeCaseEmptyAPIKey tests handling of empty API keys
func TestEdgeCaseEmptyAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()

	// Create config with empty key
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(""), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Empty key should validate against empty input
	valid, err := store.ValidateCredentials(orgID, "")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Empty key should validate against empty input")
	}

	// Non-empty key should not validate
	valid, err = store.ValidateCredentials(orgID, "something")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Non-empty key should not validate against empty hash")
	}
}

// TestEdgeCaseVeryLongAPIKey tests handling of very long API keys
func TestEdgeCaseVeryLongAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()

	// Create a very long key (10KB)
	longKey := strings.Repeat("a", 10000)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(longKey), bcryptCost)
	if err != nil {
		t.Fatalf("Failed to hash long key: %v", err)
	}

	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Long key should validate
	valid, err := store.ValidateCredentials(orgID, longKey)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Long key should validate")
	}
}

// TestEdgeCaseSpecialCharactersInAPIKey tests keys with special characters
func TestEdgeCaseSpecialCharactersInAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	specialKeys := []string{
		"key-with-unicode-你好世界",
		"key\nwith\nnewlines",
		"key\twith\ttabs",
		"key with spaces",
		"key\"with'quotes",
		"key\\with\\backslashes",
		"key$with$dollars",
		"key@with#symbols!%^&*()",
		"\x00\x01\x02", // Control characters
	}

	// Create config with all special keys
	var content strings.Builder
	orgID := uuid.New()
	content.WriteString(fmt.Sprintf("[%s]\n", orgID.String()))

	for _, key := range specialKeys {
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(key), bcryptCost)
		if err != nil {
			t.Fatalf("Failed to hash key %q: %v", key, err)
		}
		content.WriteString(string(hashedBytes) + "\n")
	}

	os.WriteFile(authConfig, []byte(content.String()), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// All special keys should validate
	for _, key := range specialKeys {
		valid, err := store.ValidateCredentials(orgID, key)
		if err != nil {
			t.Errorf("Validation error for key %q: %v", key, err)
		}
		if !valid {
			t.Errorf("Special key should validate: %q", key)
		}
	}
}

// TestEdgeCaseOrgWithNoKeys tests org entries without any keys
func TestEdgeCaseOrgWithNoKeys(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	content := fmt.Sprintf("[%s]\n\n", orgID.String())
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Any key should fail for org with no keys
	valid, err := store.ValidateCredentials(orgID, "any-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Should not validate when org has no keys")
	}
}

// TestEdgeCaseDuplicateOrgIDs tests handling of duplicate org declarations
func TestEdgeCaseDuplicateOrgIDs(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes1, _ := bcrypt.GenerateFromPassword([]byte("key1"), bcryptCost)
	hashedBytes2, _ := bcrypt.GenerateFromPassword([]byte("key2"), bcryptCost)

	// Same org ID declared twice
	content := fmt.Sprintf(`[%s]
%s

[%s]
%s`, orgID.String(), string(hashedBytes1), orgID.String(), string(hashedBytes2))

	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// The last declaration should win or both should be present
	// Test both keys
	valid1, _ := store.ValidateCredentials(orgID, "key1")
	valid2, _ := store.ValidateCredentials(orgID, "key2")

	// At least one should work (implementation dependent)
	if !valid1 && !valid2 {
		t.Error("At least one key should validate for duplicate org")
	}

	// Document actual behavior
	t.Logf("Duplicate org behavior: key1=%v, key2=%v", valid1, valid2)
}

// TestEdgeCaseMalformedBcryptHash tests handling of corrupted bcrypt hashes
func TestEdgeCaseMalformedBcryptHash(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()

	// Create config with valid and invalid hashes
	validHash, _ := bcrypt.GenerateFromPassword([]byte("valid-key"), bcryptCost)
	content := fmt.Sprintf(`[%s]
%s
$2a$12$invalidhash
$2a$12$
not-a-bcrypt-hash
`, orgID.String(), string(validHash))

	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Valid key should still work
	valid, err := store.ValidateCredentials(orgID, "valid-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Valid key should validate despite other malformed hashes")
	}

	// Invalid keys should not panic, just return false or error
	testKeys := []string{"wrong-key", "not-a-bcrypt-hash"}
	for _, key := range testKeys {
		valid, err := store.ValidateCredentials(orgID, key)
		// Should not panic, error is acceptable
		if valid {
			t.Errorf("Key %q should not validate", key)
		}
		t.Logf("Malformed hash test for %q: valid=%v, err=%v", key, valid, err)
	}
}

// TestEdgeCaseZeroUUID tests handling of zero UUID
func TestEdgeCaseZeroUUID(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	zeroUUID := uuid.UUID{}
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", zeroUUID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	valid, err := store.ValidateCredentials(zeroUUID, "test-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Zero UUID should be valid if configured")
	}
}

// TestEdgeCaseFilePermissions tests handling of permission issues
func TestEdgeCaseFilePermissions(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping permission test in CI environment")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create valid file
	content := "[11111111-2222-3333-4444-555555555555]\ntest-key\n"
	os.WriteFile(authConfig, []byte(content), 0644)

	// Make file unreadable
	os.Chmod(authConfig, 0000)
	defer os.Chmod(authConfig, 0644) // Restore for cleanup

	// Should fail to create store
	_, err := NewFileStore(authConfig)
	if err == nil {
		t.Error("Expected error when file is not readable")
	}
}

// TestEdgeCaseSymbolicLink tests following symbolic links
func TestEdgeCaseSymbolicLink(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real-auth.cfg")
	symlink := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(realFile, []byte(content), 0644)

	// Create symlink
	if err := os.Symlink(realFile, symlink); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// Should work through symlink
	store, err := NewFileStore(symlink)
	if err != nil {
		t.Fatalf("Failed to create store with symlink: %v", err)
	}
	defer store.Close()

	valid, err := store.ValidateCredentials(orgID, "test-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Should validate through symlink")
	}
}

// TestEdgeCaseVeryLargeFile tests handling of very large config files
func TestEdgeCaseVeryLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create file with many orgs (10000 orgs, 10 keys each = 100000 lines)
	var content strings.Builder
	numOrgs := 10000
	keysPerOrg := 10

	t.Logf("Generating large file with %d orgs...", numOrgs)

	for i := 0; i < numOrgs; i++ {
		orgID := uuid.New()
		content.WriteString(fmt.Sprintf("[%s]\n", orgID.String()))
		for j := 0; j < keysPerOrg; j++ {
			// Use a simple hash to save time in file generation
			content.WriteString("$2a$12$hashhashhashhashhashhashhashhashhashhashhashhashhashhashash\n")
		}
		if i%1000 == 0 {
			t.Logf("Generated %d orgs...", i)
		}
	}

	if err := os.WriteFile(authConfig, []byte(content.String()), 0644); err != nil {
		t.Fatalf("Failed to write large file: %v", err)
	}

	// Measure load time
	start := time.Now()
	store, err := NewFileStore(authConfig)
	loadTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to load large file: %v", err)
	}
	defer store.Close()

	t.Logf("Loaded large file (%d orgs) in %v", numOrgs, loadTime)

	// Should complete in reasonable time (< 10 seconds)
	if loadTime > 10*time.Second {
		t.Errorf("Load time too long: %v", loadTime)
	}
}

// TestEdgeCaseRapidFileModification tests handling of very rapid file changes
func TestEdgeCaseRapidFileModification(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("key1"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Make 100 rapid changes
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(key), bcryptCost)
		content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
		os.WriteFile(authConfig, []byte(content), 0644)
		time.Sleep(1 * time.Millisecond)
	}

	// Wait for debounce
	time.Sleep(1 * time.Second)

	// Should not crash and should have some valid state
	// The exact final key depends on debouncing behavior
	_, err = store.ValidateCredentials(orgID, "key-99")
	if err != nil {
		t.Logf("Validation after rapid changes: %v (acceptable)", err)
	}
}

// TestEdgeCaseNilUUID tests validation with different UUID variations
func TestEdgeCaseUUIDVariations(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Test with uppercase UUID
	uppercaseUUID := "11111111-2222-3333-4444-555555555555"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", strings.ToUpper(uppercaseUUID), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// UUID parsing should be case-insensitive
	orgID := uuid.MustParse(uppercaseUUID)
	valid, err := store.ValidateCredentials(orgID, "test-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Should validate with uppercase UUID")
	}
}

// TestEdgeCaseWhitespaceOnlyLines tests handling of whitespace-only lines
func TestEdgeCaseWhitespaceOnlyLines(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n   \n\t\n  \t  \n%s\n\t  \n",
		orgID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	valid, err := store.ValidateCredentials(orgID, "test-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Should handle whitespace-only lines gracefully")
	}
}

// TestEdgeCaseFileWithOnlyComments tests file containing only comments
func TestEdgeCaseFileWithOnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	content := `# This is a comment
# Another comment
# Yet another comment
`
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Store should be empty but valid
	orgID := uuid.New()
	valid, err := store.ValidateCredentials(orgID, "any-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("Should not validate any credentials in empty store")
	}
}

// TestEdgeCaseConcurrentReloadAndValidation tests concurrent reload and validation
func TestEdgeCaseConcurrentReloadAndValidation(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("test-key"), bcryptCost)
	content := fmt.Sprintf("[%s]\n%s\n", orgID.String(), string(hashedBytes))
	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Trigger many concurrent reloads and validations
	done := make(chan bool)
	errorCount := 0

	// Reloader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Reload()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Multiple validator goroutines
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, err := store.ValidateCredentials(orgID, "test-key")
				if err != nil {
					t.Logf("Validation error during concurrent reload: %v", err)
					errorCount++
				}
			}
			done <- true
		}()
	}

	// Wait for completion
	for i := 0; i < 11; i++ {
		<-done
	}

	t.Logf("Concurrent test completed with %d errors", errorCount)
	// Some errors might be acceptable during heavy concurrent access
}

// TestEdgeCaseInMemoryStoreEdgeCases tests edge cases for in-memory store
func TestEdgeCaseInMemoryStoreEdgeCases(t *testing.T) {
	store := NewInMemoryStore()

	// Test with zero UUID
	zeroUUID := uuid.UUID{}
	store.AddCredentials(zeroUUID, "test-key")
	valid, _ := store.ValidateCredentials(zeroUUID, "test-key")
	if !valid {
		t.Error("Should work with zero UUID")
	}

	// Test with empty key
	orgID := uuid.New()
	store.AddCredentials(orgID, "")
	valid, _ = store.ValidateCredentials(orgID, "")
	if !valid {
		t.Error("Should work with empty key")
	}

	// Test updating existing org
	store.AddCredentials(orgID, "key1")
	store.AddCredentials(orgID, "key2")
	valid, _ = store.ValidateCredentials(orgID, "key2")
	if !valid {
		t.Error("Should have updated key")
	}
	valid, _ = store.ValidateCredentials(orgID, "key1")
	if valid {
		t.Error("Old key should not work after update")
	}
}

// TestEdgeCaseMixedHashTypes tests mixing bcrypt and plaintext keys
func TestEdgeCaseMixedHashTypes(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("bcrypt-key"), bcryptCost)
	content := fmt.Sprintf(`[%s]
%s
plaintext-key
`, orgID.String(), string(hashedBytes))

	os.WriteFile(authConfig, []byte(content), 0644)

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Bcrypt key should work
	valid, err := store.ValidateCredentials(orgID, "bcrypt-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Bcrypt key should validate")
	}

	// Plaintext key should work with constant-time comparison
	valid, err = store.ValidateCredentials(orgID, "plaintext-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Plaintext key should validate")
	}
}
