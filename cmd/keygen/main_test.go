package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestReadInitConfig(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantOrgs    int
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with single org",
			content: `[11111111-2222-3333-4444-555555555555]
demo-api-key-1
demo-api-key-2`,
			wantOrgs: 1,
			wantErr:  false,
		},
		{
			name: "valid config with multiple orgs",
			content: `[11111111-2222-3333-4444-555555555555]
demo-api-key-1

[22222222-3333-4444-5555-666666666666]
demo-api-key-2
demo-api-key-3`,
			wantOrgs: 2,
			wantErr:  false,
		},
		{
			name: "config with comments and empty lines",
			content: `# This is a comment
[11111111-2222-3333-4444-555555555555]
demo-api-key-1
# Another comment

demo-api-key-2`,
			wantOrgs: 1,
			wantErr:  false,
		},
		{
			name: "invalid UUID format",
			content: `[invalid-uuid]
demo-api-key-1`,
			wantOrgs:    0,
			wantErr:     true,
			errContains: "invalid UUID",
		},
		{
			name: "API key before org declaration",
			content: `demo-api-key-1
[11111111-2222-3333-4444-555555555555]`,
			wantOrgs:    0,
			wantErr:     true,
			errContains: "appears before any org ID",
		},
		{
			name:     "empty file",
			content:  "",
			wantOrgs: 0,
			wantErr:  false,
		},
		{
			name: "org with no API keys",
			content: `[11111111-2222-3333-4444-555555555555]

[22222222-3333-4444-5555-666666666666]
demo-api-key-1`,
			wantOrgs: 2,
			wantErr:  false,
		},
		{
			name: "whitespace handling",
			content: `  [11111111-2222-3333-4444-555555555555]
  demo-api-key-1
	demo-api-key-2	`,
			wantOrgs: 1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(t.TempDir(), "init-config.cfg")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test readInitConfig
			orgs, err := readInitConfig(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(orgs) != tt.wantOrgs {
				t.Errorf("Expected %d orgs, got %d", tt.wantOrgs, len(orgs))
			}
		})
	}
}

func TestReadInitConfigValidatesData(t *testing.T) {
	content := `[11111111-2222-3333-4444-555555555555]
api-key-1
api-key-2

[22222222-3333-4444-5555-666666666666]
api-key-3`

	tmpFile := filepath.Join(t.TempDir(), "init-config.cfg")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	orgs, err := readInitConfig(tmpFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(orgs) != 2 {
		t.Fatalf("Expected 2 orgs, got %d", len(orgs))
	}

	// Validate first org
	expectedOrgID1 := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	if orgs[0].OrgID != expectedOrgID1 {
		t.Errorf("Expected org ID %v, got %v", expectedOrgID1, orgs[0].OrgID)
	}
	if len(orgs[0].APIKeys) != 2 {
		t.Errorf("Expected 2 API keys for org 1, got %d", len(orgs[0].APIKeys))
	}
	if orgs[0].APIKeys[0] != "api-key-1" {
		t.Errorf("Expected first key to be 'api-key-1', got %q", orgs[0].APIKeys[0])
	}

	// Validate second org
	expectedOrgID2 := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	if orgs[1].OrgID != expectedOrgID2 {
		t.Errorf("Expected org ID %v, got %v", expectedOrgID2, orgs[1].OrgID)
	}
	if len(orgs[1].APIKeys) != 1 {
		t.Errorf("Expected 1 API key for org 2, got %d", len(orgs[1].APIKeys))
	}
}

func TestReadInitConfigFileNotFound(t *testing.T) {
	_, err := readInitConfig("/nonexistent/file.cfg")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Expected 'failed to open file' error, got: %v", err)
	}
}

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "simple key",
			apiKey: "test-key-123",
		},
		{
			name:   "complex key",
			apiKey: "AbCd123!@#$%^&*()",
		},
		{
			name:   "long key",
			apiKey: strings.Repeat("a", 100),
		},
		{
			name:   "empty key",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := hashAPIKey(tt.apiKey)
			if err != nil {
				t.Fatalf("Failed to hash API key: %v", err)
			}

			// Verify it's a valid bcrypt hash
			if !strings.HasPrefix(hashed, "$2a$") && !strings.HasPrefix(hashed, "$2b$") {
				t.Errorf("Hash doesn't look like a bcrypt hash: %s", hashed)
			}

			// Verify we can compare it
			err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(tt.apiKey))
			if err != nil {
				t.Errorf("Hash comparison failed: %v", err)
			}

			// Verify wrong key doesn't match
			err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(tt.apiKey+"wrong"))
			if err != bcrypt.ErrMismatchedHashAndPassword {
				t.Errorf("Expected mismatch error, got: %v", err)
			}
		})
	}
}

func TestHashAPIKeyDeterminism(t *testing.T) {
	// bcrypt should produce different hashes for the same input (due to salt)
	apiKey := "test-key"
	hash1, err1 := hashAPIKey(apiKey)
	hash2, err2 := hashAPIKey(apiKey)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash: %v, %v", err1, err2)
	}

	if hash1 == hash2 {
		t.Error("bcrypt should produce different hashes with different salts")
	}

	// But both should validate correctly
	if err := bcrypt.CompareHashAndPassword([]byte(hash1), []byte(apiKey)); err != nil {
		t.Error("First hash failed validation")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash2), []byte(apiKey)); err != nil {
		t.Error("Second hash failed validation")
	}
}

func TestGenerateAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		orgs     []OrgConfig
		wantErr  bool
		validate func(t *testing.T, content string)
	}{
		{
			name: "single org with multiple keys",
			orgs: []OrgConfig{
				{
					OrgID:   uuid.MustParse("11111111-2222-3333-4444-555555555555"),
					APIKeys: []string{"key1", "key2"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "[11111111-2222-3333-4444-555555555555]") {
					t.Error("Missing org ID in output")
				}
				// Should have 2 bcrypt hashes
				bcryptCount := strings.Count(content, "$2a$")
				if bcryptCount < 2 {
					t.Errorf("Expected at least 2 bcrypt hashes, found %d", bcryptCount)
				}
			},
		},
		{
			name: "multiple orgs",
			orgs: []OrgConfig{
				{
					OrgID:   uuid.MustParse("11111111-2222-3333-4444-555555555555"),
					APIKeys: []string{"key1"},
				},
				{
					OrgID:   uuid.MustParse("22222222-3333-4444-5555-666666666666"),
					APIKeys: []string{"key2"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "[11111111-2222-3333-4444-555555555555]") {
					t.Error("Missing first org ID")
				}
				if !strings.Contains(content, "[22222222-3333-4444-5555-666666666666]") {
					t.Error("Missing second org ID")
				}
			},
		},
		{
			name:    "empty org list",
			orgs:    []OrgConfig{},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "# Authentication configuration file") {
					t.Error("Missing header comment")
				}
			},
		},
		{
			name: "org with no API keys",
			orgs: []OrgConfig{
				{
					OrgID:   uuid.MustParse("11111111-2222-3333-4444-555555555555"),
					APIKeys: []string{},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "[11111111-2222-3333-4444-555555555555]") {
					t.Error("Missing org ID")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "auth.cfg")

			err := generateAuthConfig(tt.orgs, tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Read and validate output
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, string(content))
			}
		})
	}
}

func TestGenerateAuthConfigInvalidPath(t *testing.T) {
	orgs := []OrgConfig{
		{
			OrgID:   uuid.MustParse("11111111-2222-3333-4444-555555555555"),
			APIKeys: []string{"key1"},
		},
	}

	err := generateAuthConfig(orgs, "/invalid/path/auth.cfg")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
	if !strings.Contains(err.Error(), "failed to create output file") {
		t.Errorf("Expected 'failed to create output file' error, got: %v", err)
	}
}

func TestGenerateRandomAPIKey(t *testing.T) {
	// Generate multiple keys
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := generateRandomAPIKey()
		if err != nil {
			t.Fatalf("Failed to generate API key: %v", err)
		}

		// Should be non-empty
		if key == "" {
			t.Error("Generated empty API key")
		}

		// Should be unique
		if keys[key] {
			t.Errorf("Generated duplicate key: %s", key)
		}
		keys[key] = true

		// Should be base64 URL encoded
		if strings.ContainsAny(key, "+/") {
			t.Errorf("Key contains non-URL-safe base64 characters: %s", key)
		}
	}
}

func TestEndToEndKeygenFlow(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "init-config.cfg")
	outputFile := filepath.Join(tmpDir, "auth.cfg")

	// Create input config
	inputContent := `[11111111-2222-3333-4444-555555555555]
my-secret-key-1
my-secret-key-2

[22222222-3333-4444-5555-666666666666]
another-secret-key`

	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	// Read config
	orgs, err := readInitConfig(inputFile)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Generate auth config
	if err := generateAuthConfig(orgs, outputFile); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	// Verify output file exists and is readable
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(content)

	// Verify structure
	if !strings.Contains(outputStr, "[11111111-2222-3333-4444-555555555555]") {
		t.Error("Missing first org in output")
	}
	if !strings.Contains(outputStr, "[22222222-3333-4444-5555-666666666666]") {
		t.Error("Missing second org in output")
	}

	// Count bcrypt hashes (should have 3 total)
	bcryptCount := strings.Count(outputStr, "$2a$")
	if bcryptCount < 3 {
		t.Errorf("Expected at least 3 bcrypt hashes, found %d", bcryptCount)
	}

	// Verify we can't find plaintext keys in output
	if strings.Contains(outputStr, "my-secret-key-1") {
		t.Error("Found plaintext key in output - keys should be hashed!")
	}
	if strings.Contains(outputStr, "another-secret-key") {
		t.Error("Found plaintext key in output - keys should be hashed!")
	}
}

func BenchmarkHashAPIKey(b *testing.B) {
	apiKey := "test-api-key-for-benchmarking"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hashAPIKey(apiKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadInitConfig(b *testing.B) {
	// Create test file
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench-config.cfg")
	content := `[11111111-2222-3333-4444-555555555555]
key1
key2
key3

[22222222-3333-4444-5555-666666666666]
key4
key5`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readInitConfig(tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}
