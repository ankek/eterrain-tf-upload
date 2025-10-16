package testutil

import (
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestNewTestFixture tests the test fixture creation
func TestNewTestFixture(t *testing.T) {
	fixture := NewTestFixture(t)

	if fixture.TempDir == "" {
		t.Error("Expected non-empty TempDir")
	}

	if fixture.InitConfig == "" {
		t.Error("Expected non-empty InitConfig path")
	}

	if fixture.AuthConfig == "" {
		t.Error("Expected non-empty AuthConfig path")
	}
}

// TestTestFixtureAddOrg tests adding organizations to fixture
func TestTestFixtureAddOrg(t *testing.T) {
	fixture := NewTestFixture(t)

	orgID := uuid.New()
	fixture.AddOrg(orgID, "key1", "key2")

	if len(fixture.Orgs) != 1 {
		t.Fatalf("Expected 1 org, got %d", len(fixture.Orgs))
	}

	if fixture.Orgs[0].OrgID != orgID {
		t.Error("Org ID mismatch")
	}

	if len(fixture.Orgs[0].APIKeys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(fixture.Orgs[0].APIKeys))
	}
}

// TestTestFixtureWriteInitConfig tests writing init config
func TestTestFixtureWriteInitConfig(t *testing.T) {
	fixture := NewTestFixture(t)
	fixture.AddRandomOrg("key1", "key2")

	if err := fixture.WriteInitConfig(); err != nil {
		t.Fatalf("Failed to write init config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(fixture.InitConfig); os.IsNotExist(err) {
		t.Error("Init config file was not created")
	}
}

// TestTestFixtureWriteAuthConfig tests writing auth config
func TestTestFixtureWriteAuthConfig(t *testing.T) {
	fixture := NewTestFixture(t)
	fixture.AddRandomOrg("key1", "key2")

	if err := fixture.WriteAuthConfigDefault(); err != nil {
		t.Fatalf("Failed to write auth config: %v", err)
	}

	// Verify file exists and contains bcrypt hashes
	content, err := os.ReadFile(fixture.AuthConfig)
	if err != nil {
		t.Fatalf("Failed to read auth config: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "$2a$") {
		t.Error("Auth config should contain bcrypt hashes")
	}

	// Should not contain plaintext keys
	if contains(contentStr, "key1") {
		t.Error("Auth config should not contain plaintext keys")
	}
}

// TestWriteInitConfig tests the WriteInitConfig helper
func TestWriteInitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/init-config.cfg"

	orgs := []OrgConfig{
		{OrgID: TestUUID1, APIKeys: []string{"key1", "key2"}},
		{OrgID: TestUUID2, APIKeys: []string{"key3"}},
	}

	if err := WriteInitConfig(configPath, orgs); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, TestUUID1.String()) {
		t.Error("Config should contain first UUID")
	}
	if !contains(contentStr, "key1") {
		t.Error("Config should contain key1")
	}
}

// TestWriteAuthConfig tests the WriteAuthConfig helper
func TestWriteAuthConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/auth.cfg"

	orgs := []OrgConfig{
		{OrgID: TestUUID1, APIKeys: []string{"key1"}},
	}

	if err := WriteAuthConfig(configPath, orgs, DefaultBcryptCost); err != nil {
		t.Fatalf("Failed to write auth config: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read auth config: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "$2a$") {
		t.Error("Auth config should contain bcrypt hash")
	}
	if contains(contentStr, "key1") {
		t.Error("Auth config should not contain plaintext key")
	}
}

// TestCreateSimpleAuthConfig tests simple auth config creation
func TestCreateSimpleAuthConfig(t *testing.T) {
	configPath := CreateSimpleAuthConfig(t, TestUUID1, TestKey1)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}

	content, _ := os.ReadFile(configPath)
	if !contains(string(content), TestUUID1.String()) {
		t.Error("Config should contain test UUID")
	}
}

// TestCreateMultiOrgAuthConfig tests multi-org config creation
func TestCreateMultiOrgAuthConfig(t *testing.T) {
	configPath, orgs := CreateMultiOrgAuthConfig(t, 5, 3)

	if len(orgs) != 5 {
		t.Errorf("Expected 5 orgs, got %d", len(orgs))
	}

	for _, org := range orgs {
		if len(org.APIKeys) != 3 {
			t.Errorf("Expected 3 keys per org, got %d", len(org.APIKeys))
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}
}

// TestHashAPIKey tests API key hashing
func TestHashAPIKey(t *testing.T) {
	hash, err := HashAPIKey("test-key", DefaultBcryptCost)
	if err != nil {
		t.Fatalf("Failed to hash key: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if !contains(hash, "$2a$") {
		t.Error("Hash should be bcrypt format")
	}
}

// TestMustHashAPIKey tests the must-hash helper
func TestMustHashAPIKey(t *testing.T) {
	hash := MustHashAPIKey("test-key", DefaultBcryptCost)

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

// TestGenerateTestOrgs tests organization generation
func TestGenerateTestOrgs(t *testing.T) {
	orgs := GenerateTestOrgs(3, 2)

	if len(orgs) != 3 {
		t.Errorf("Expected 3 orgs, got %d", len(orgs))
	}

	for i, org := range orgs {
		if org.OrgID == uuid.Nil {
			t.Errorf("Org %d has nil UUID", i)
		}
		if len(org.APIKeys) != 2 {
			t.Errorf("Org %d expected 2 keys, got %d", i, len(org.APIKeys))
		}
	}
}

// TestCreateCorruptedAuthConfig tests corrupted config creation
func TestCreateCorruptedAuthConfig(t *testing.T) {
	tests := []struct {
		corruptionType string
		shouldContain  string
	}{
		{"invalid-uuid", "not-a-uuid"},
		{"key-before-org", "test-key"},
		{"malformed-bcrypt", "$2a$invalid$hash"},
	}

	for _, tt := range tests {
		t.Run(tt.corruptionType, func(t *testing.T) {
			configPath := CreateCorruptedAuthConfig(t, tt.corruptionType)

			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read corrupted config: %v", err)
			}

			if !contains(string(content), tt.shouldContain) {
				t.Errorf("Expected corrupted config to contain %q", tt.shouldContain)
			}
		})
	}
}

// TestAssertFileExists tests file existence assertion
func TestAssertFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := tmpDir + "/exists.txt"
	os.WriteFile(existingFile, []byte("test"), 0644)

	// This should not fail
	AssertFileExists(t, existingFile)
}

// TestAssertFileContains tests file content assertion
func TestAssertFileContains(t *testing.T) {
	tmpDir := t.TempDir()
	file := tmpDir + "/test.txt"
	os.WriteFile(file, []byte("Hello World"), 0644)

	// This should not fail
	AssertFileContains(t, file, "Hello")
	AssertFileContains(t, file, "World")
}

// TestCommonTestConstants tests that common constants are available
func TestCommonTestConstants(t *testing.T) {
	if TestUUID1 == uuid.Nil {
		t.Error("TestUUID1 should not be nil")
	}

	if TestKey1 == "" {
		t.Error("TestKey1 should not be empty")
	}

	if ZeroUUID != uuid.Nil {
		t.Error("ZeroUUID should be nil UUID")
	}
}

// BenchmarkHashAPIKey benchmarks the hash function
func BenchmarkHashAPIKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashAPIKey("test-key", DefaultBcryptCost)
	}
}

// BenchmarkGenerateTestOrgs benchmarks org generation
func BenchmarkGenerateTestOrgs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateTestOrgs(10, 5)
	}
}
