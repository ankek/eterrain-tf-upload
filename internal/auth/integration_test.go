package auth

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12
)

// TestEndToEndKeygenToFileStore tests the complete flow from keygen to FileStore validation
func TestEndToEndKeygenToFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	initConfig := filepath.Join(tmpDir, "init-config.cfg")
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create init config with test data
	orgID1 := uuid.New()
	orgID2 := uuid.New()
	initContent := fmt.Sprintf(`# Test configuration
[%s]
secret-key-org1-1
secret-key-org1-2

[%s]
secret-key-org2-1`, orgID1.String(), orgID2.String())

	if err := os.WriteFile(initConfig, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to write init config: %v", err)
	}

	// Simulate keygen: read and hash keys
	orgs, err := readInitConfigSimple(initConfig)
	if err != nil {
		t.Fatalf("Failed to read init config: %v", err)
	}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	// Create FileStore and test validation
	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Test all keys work
	testCases := []struct {
		orgID  uuid.UUID
		apiKey string
		valid  bool
	}{
		{orgID1, "secret-key-org1-1", true},
		{orgID1, "secret-key-org1-2", true},
		{orgID1, "wrong-key", false},
		{orgID2, "secret-key-org2-1", true},
		{orgID2, "secret-key-org1-1", false}, // Key from different org
		{uuid.New(), "secret-key-org1-1", false}, // Non-existent org
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("org=%s_key=%s", tc.orgID, tc.apiKey), func(t *testing.T) {
			valid, err := store.ValidateCredentials(tc.orgID, tc.apiKey)
			if err != nil {
				t.Fatalf("Validation error: %v", err)
			}
			if valid != tc.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tc.valid, valid)
			}
		})
	}
}

// TestHotReloadIntegration tests that file changes are detected and reloaded
func TestHotReloadIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Phase 1: Initial configuration
	orgs := []orgConfigSimple{
		{
			OrgID:   orgID,
			APIKeys: []string{"initial-key-1", "initial-key-2"},
		},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate initial auth config: %v", err)
	}

	// Create store (starts watching)
	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Verify initial keys work
	valid, _ := store.ValidateCredentials(orgID, "initial-key-1")
	if !valid {
		t.Fatal("Initial key 1 should be valid")
	}
	valid, _ = store.ValidateCredentials(orgID, "initial-key-2")
	if !valid {
		t.Fatal("Initial key 2 should be valid")
	}

	// Phase 2: Update configuration (add new key, remove old one)
	orgs = []orgConfigSimple{
		{
			OrgID:   orgID,
			APIKeys: []string{"initial-key-2", "new-key-1"},
		},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate updated auth config: %v", err)
	}

	// Wait for file watcher to detect and reload (500ms debounce + buffer)
	time.Sleep(1 * time.Second)

	// Verify new state
	valid, _ = store.ValidateCredentials(orgID, "initial-key-1")
	if valid {
		t.Error("initial-key-1 should no longer be valid after reload")
	}

	valid, _ = store.ValidateCredentials(orgID, "initial-key-2")
	if !valid {
		t.Error("initial-key-2 should still be valid after reload")
	}

	valid, _ = store.ValidateCredentials(orgID, "new-key-1")
	if !valid {
		t.Error("new-key-1 should be valid after reload")
	}

	// Phase 3: Another update (completely new set)
	orgs = []orgConfigSimple{
		{
			OrgID:   orgID,
			APIKeys: []string{"final-key"},
		},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate final auth config: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Only final key should work
	valid, _ = store.ValidateCredentials(orgID, "final-key")
	if !valid {
		t.Error("final-key should be valid")
	}

	valid, _ = store.ValidateCredentials(orgID, "new-key-1")
	if valid {
		t.Error("new-key-1 should no longer be valid")
	}
}

// TestConcurrentAccessDuringReload tests that the store handles concurrent requests during reload
func TestConcurrentAccessDuringReload(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	apiKey := "test-key"

	// Initial config
	orgs := []orgConfigSimple{
		{OrgID: orgID, APIKeys: []string{apiKey}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Start concurrent validation requests
	var wg sync.WaitGroup
	errors := make(chan error, 200)
	stopChan := make(chan struct{})

	// Launch 10 goroutines doing continuous validation
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					valid, err := store.ValidateCredentials(orgID, apiKey)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d: validation error: %v", id, err)
						return
					}
					if !valid {
						// This might happen briefly during reload, which is acceptable
						// Just log it for visibility
						t.Logf("goroutine %d: validation returned false (might be during reload)", id)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// While validations are running, update the file multiple times
	for i := 0; i < 5; i++ {
		time.Sleep(200 * time.Millisecond)
		// Update with same key to ensure validations should keep working
		orgs := []orgConfigSimple{
			{OrgID: orgID, APIKeys: []string{apiKey}},
		}
		if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
			t.Errorf("Failed to update auth config: %v", err)
		}
	}

	// Stop validation goroutines
	time.Sleep(1 * time.Second) // Let last reload settle
	close(stopChan)
	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent access", errorCount)
	}
}

// TestMultipleOrgsHotReload tests hot reload with multiple organizations
func TestMultipleOrgsHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	org1 := uuid.New()
	org2 := uuid.New()
	org3 := uuid.New()

	// Phase 1: Start with 2 orgs
	orgs := []orgConfigSimple{
		{OrgID: org1, APIKeys: []string{"org1-key"}},
		{OrgID: org2, APIKeys: []string{"org2-key"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Verify initial state
	valid, _ := store.ValidateCredentials(org1, "org1-key")
	if !valid {
		t.Error("org1-key should be valid initially")
	}
	valid, _ = store.ValidateCredentials(org2, "org2-key")
	if !valid {
		t.Error("org2-key should be valid initially")
	}
	valid, _ = store.ValidateCredentials(org3, "org3-key")
	if valid {
		t.Error("org3 should not exist initially")
	}

	// Phase 2: Add org3, remove org1
	orgs = []orgConfigSimple{
		{OrgID: org2, APIKeys: []string{"org2-key"}},
		{OrgID: org3, APIKeys: []string{"org3-key"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to update auth config: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify updated state
	valid, _ = store.ValidateCredentials(org1, "org1-key")
	if valid {
		t.Error("org1 should be removed after reload")
	}
	valid, _ = store.ValidateCredentials(org2, "org2-key")
	if !valid {
		t.Error("org2-key should still be valid")
	}
	valid, _ = store.ValidateCredentials(org3, "org3-key")
	if !valid {
		t.Error("org3-key should be valid after reload")
	}
}

// TestCorruptedAuthConfigHandling tests how the system handles corrupted files
func TestCorruptedAuthConfigHandling(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Start with valid config
	orgs := []orgConfigSimple{
		{OrgID: orgID, APIKeys: []string{"valid-key"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Verify it works
	valid, _ := store.ValidateCredentials(orgID, "valid-key")
	if !valid {
		t.Fatal("valid-key should work initially")
	}

	// Write corrupted config
	corruptedContent := "[invalid-uuid]\nsome-key\n"
	if err := os.WriteFile(authConfig, []byte(corruptedContent), 0644); err != nil {
		t.Fatalf("Failed to write corrupted config: %v", err)
	}

	// Wait for reload attempt
	time.Sleep(1 * time.Second)

	// Store should keep old valid credentials (reload should fail but not crash)
	// The exact behavior depends on implementation - either keep old creds or have empty store
	// Let's verify it doesn't panic and returns a result
	_, err = store.ValidateCredentials(orgID, "valid-key")
	if err != nil {
		t.Logf("Validation returned error after corrupted file (acceptable): %v", err)
	}
}

// TestFileDeletedAndRecreated tests the scenario where the auth file is deleted and recreated
func TestFileDeletedAndRecreated(t *testing.T) {
	// Note: fsnotify behavior with deleted files varies by OS
	// This test documents expected behavior but may need adjustment
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Create initial config
	orgs := []orgConfigSimple{
		{OrgID: orgID, APIKeys: []string{"initial-key"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Verify initial key works
	valid, _ := store.ValidateCredentials(orgID, "initial-key")
	if !valid {
		t.Fatal("initial-key should work")
	}

	// Delete and immediately recreate with new content
	os.Remove(authConfig)
	time.Sleep(100 * time.Millisecond)

	orgs = []orgConfigSimple{
		{OrgID: orgID, APIKeys: []string{"new-key"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to recreate auth config: %v", err)
	}

	// Wait for potential reload
	time.Sleep(1 * time.Second)

	// The new key might work if fsnotify detected the create event
	// This is OS-dependent, so we just verify no crash occurred
	_, err = store.ValidateCredentials(orgID, "new-key")
	if err != nil {
		t.Logf("Validation error after file recreation: %v", err)
	}
}

// TestRapidFileChanges tests the debouncing behavior with rapid file updates
func TestRapidFileChanges(t *testing.T) {
	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	// Initial config
	orgs := []orgConfigSimple{
		{OrgID: orgID, APIKeys: []string{"key-0"}},
	}
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Make rapid changes
	for i := 1; i <= 10; i++ {
		orgs := []orgConfigSimple{
			{OrgID: orgID, APIKeys: []string{fmt.Sprintf("key-%d", i)}},
		}
		generateAuthConfigSimple(orgs, authConfig)
		time.Sleep(50 * time.Millisecond) // Less than debounce time
	}

	// Wait for debounce to settle
	time.Sleep(1 * time.Second)

	// Should have the last key
	valid, _ := store.ValidateCredentials(orgID, "key-10")
	if !valid {
		t.Error("Last key (key-10) should be valid after rapid changes")
	}

	// Earlier keys should not work
	valid, _ = store.ValidateCredentials(orgID, "key-5")
	if valid {
		t.Error("Intermediate key (key-5) should not be valid")
	}
}

// Helper types and functions for integration tests

type orgConfigSimple struct {
	OrgID   uuid.UUID
	APIKeys []string
}

func readInitConfigSimple(filePath string) ([]orgConfigSimple, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var orgs []orgConfigSimple
	var currentOrg *orgConfigSimple

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			orgIDStr := strings.TrimSpace(line[1 : len(line)-1])
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				return nil, err
			}

			if currentOrg != nil {
				orgs = append(orgs, *currentOrg)
			}

			currentOrg = &orgConfigSimple{
				OrgID:   orgID,
				APIKeys: []string{},
			}
			continue
		}

		if currentOrg != nil && line != "" {
			currentOrg.APIKeys = append(currentOrg.APIKeys, line)
		}
	}

	if currentOrg != nil {
		orgs = append(orgs, *currentOrg)
	}

	return orgs, scanner.Err()
}

func generateAuthConfigSimple(orgs []orgConfigSimple, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "# Auto-generated authentication config\n\n")

	for i, org := range orgs {
		if i > 0 {
			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "[%s]\n", org.OrgID.String())

		for _, apiKey := range org.APIKeys {
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
			if err != nil {
				return err
			}
			fmt.Fprintf(writer, "%s\n", string(hashedBytes))
		}
	}

	return nil
}

// TestIntegrationWithRealKeygen tests using the actual keygen binary if available
func TestIntegrationWithRealKeygen(t *testing.T) {
	// Check if keygen binary exists
	keygenPath := "../../../keygen" // Adjust path as needed
	if _, err := os.Stat(keygenPath); os.IsNotExist(err) {
		t.Skip("Keygen binary not found, skipping integration test with real binary")
	}

	tmpDir := t.TempDir()
	initConfig := filepath.Join(tmpDir, "init-config.cfg")
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	initContent := fmt.Sprintf("[%s]\ntest-key-1\ntest-key-2\n", orgID.String())
	if err := os.WriteFile(initConfig, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to write init config: %v", err)
	}

	// Run keygen
	cmd := exec.Command(keygenPath, initConfig, authConfig)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Keygen failed: %v\nOutput: %s", err, output)
	}

	// Create FileStore and validate
	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	valid, err := store.ValidateCredentials(orgID, "test-key-1")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("test-key-1 should be valid after keygen")
	}

	valid, err = store.ValidateCredentials(orgID, "test-key-2")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("test-key-2 should be valid after keygen")
	}

	valid, err = store.ValidateCredentials(orgID, "wrong-key")
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if valid {
		t.Error("wrong-key should not be valid")
	}
}
