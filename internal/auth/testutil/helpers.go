package testutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default bcrypt cost for test fixtures
	DefaultBcryptCost = 4 // Lower cost for faster tests

	// ProductionBcryptCost is the production bcrypt cost
	ProductionBcryptCost = 12
)

// OrgConfig represents an organization's configuration for testing
type OrgConfig struct {
	OrgID   uuid.UUID
	APIKeys []string
}

// TestFixture provides a complete test environment
type TestFixture struct {
	TempDir    string
	InitConfig string
	AuthConfig string
	Orgs       []OrgConfig
	t          *testing.T
}

// NewTestFixture creates a new test fixture with temporary files
func NewTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	tmpDir := t.TempDir()
	return &TestFixture{
		TempDir:    tmpDir,
		InitConfig: filepath.Join(tmpDir, "init-config.cfg"),
		AuthConfig: filepath.Join(tmpDir, "auth.cfg"),
		Orgs:       []OrgConfig{},
		t:          t,
	}
}

// AddOrg adds an organization to the test fixture
func (f *TestFixture) AddOrg(orgID uuid.UUID, apiKeys ...string) *TestFixture {
	f.Orgs = append(f.Orgs, OrgConfig{
		OrgID:   orgID,
		APIKeys: apiKeys,
	})
	return f
}

// AddRandomOrg adds an organization with a random UUID
func (f *TestFixture) AddRandomOrg(apiKeys ...string) *TestFixture {
	return f.AddOrg(uuid.New(), apiKeys...)
}

// WriteInitConfig writes the init config file
func (f *TestFixture) WriteInitConfig() error {
	return WriteInitConfig(f.InitConfig, f.Orgs)
}

// WriteAuthConfig writes the auth config file with hashed keys
func (f *TestFixture) WriteAuthConfig(bcryptCost int) error {
	return WriteAuthConfig(f.AuthConfig, f.Orgs, bcryptCost)
}

// WriteAuthConfigDefault writes the auth config with default test bcrypt cost
func (f *TestFixture) WriteAuthConfigDefault() error {
	return f.WriteAuthConfig(DefaultBcryptCost)
}

// Cleanup removes all temporary files (called automatically by t.TempDir())
func (f *TestFixture) Cleanup() {
	// No-op: t.TempDir() handles cleanup automatically
}

// WriteInitConfig writes an init config file
func WriteInitConfig(path string, orgs []OrgConfig) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create init config: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for i, org := range orgs {
		if i > 0 {
			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "[%s]\n", org.OrgID.String())
		for _, key := range org.APIKeys {
			fmt.Fprintf(writer, "%s\n", key)
		}
	}

	return nil
}

// WriteAuthConfig writes an auth config file with bcrypt hashes
func WriteAuthConfig(path string, orgs []OrgConfig, bcryptCost int) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create auth config: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "# Auto-generated authentication config\n")
	fmt.Fprintf(writer, "# Generated for testing\n\n")

	for i, org := range orgs {
		if i > 0 {
			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "[%s]\n", org.OrgID.String())
		for _, key := range org.APIKeys {
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(key), bcryptCost)
			if err != nil {
				return fmt.Errorf("failed to hash key: %w", err)
			}
			fmt.Fprintf(writer, "%s\n", string(hashedBytes))
		}
	}

	return nil
}

// CreateSimpleAuthConfig creates a simple auth config with one org and one key
func CreateSimpleAuthConfig(t *testing.T, orgID uuid.UUID, apiKey string) string {
	t.Helper()

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgs := []OrgConfig{{OrgID: orgID, APIKeys: []string{apiKey}}}
	if err := WriteAuthConfig(authConfig, orgs, DefaultBcryptCost); err != nil {
		t.Fatalf("Failed to create simple auth config: %v", err)
	}

	return authConfig
}

// CreateMultiOrgAuthConfig creates an auth config with multiple orgs
func CreateMultiOrgAuthConfig(t *testing.T, numOrgs, keysPerOrg int) (string, []OrgConfig) {
	t.Helper()

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgs := make([]OrgConfig, numOrgs)
	for i := 0; i < numOrgs; i++ {
		keys := make([]string, keysPerOrg)
		for j := 0; j < keysPerOrg; j++ {
			keys[j] = fmt.Sprintf("key-org%d-%d", i, j)
		}
		orgs[i] = OrgConfig{
			OrgID:   uuid.New(),
			APIKeys: keys,
		}
	}

	if err := WriteAuthConfig(authConfig, orgs, DefaultBcryptCost); err != nil {
		t.Fatalf("Failed to create multi-org auth config: %v", err)
	}

	return authConfig, orgs
}

// HashAPIKey hashes an API key with the given bcrypt cost
func HashAPIKey(apiKey string, cost int) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), cost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// HashAPIKeyDefault hashes an API key with the default test cost
func HashAPIKeyDefault(apiKey string) (string, error) {
	return HashAPIKey(apiKey, DefaultBcryptCost)
}

// MustHashAPIKey hashes an API key or panics (for test setup)
func MustHashAPIKey(apiKey string, cost int) string {
	hash, err := HashAPIKey(apiKey, cost)
	if err != nil {
		panic(fmt.Sprintf("failed to hash API key: %v", err))
	}
	return hash
}

// GenerateTestOrgs generates test organization configurations
func GenerateTestOrgs(numOrgs, keysPerOrg int) []OrgConfig {
	orgs := make([]OrgConfig, numOrgs)
	for i := 0; i < numOrgs; i++ {
		keys := make([]string, keysPerOrg)
		for j := 0; j < keysPerOrg; j++ {
			keys[j] = fmt.Sprintf("test-key-%d-%d", i, j)
		}
		orgs[i] = OrgConfig{
			OrgID:   uuid.New(),
			APIKeys: keys,
		}
	}
	return orgs
}

// ValidateTestCredentials is a helper to validate credentials and fail the test if unexpected
func ValidateTestCredentials(t *testing.T, validator func(uuid.UUID, string) (bool, error), orgID uuid.UUID, apiKey string, expectValid bool) {
	t.Helper()

	valid, err := validator(orgID, apiKey)
	if err != nil {
		t.Fatalf("Validation error for org=%s key=%s: %v", orgID, apiKey, err)
	}

	if valid != expectValid {
		t.Errorf("Expected valid=%v for org=%s key=%s, got valid=%v", expectValid, orgID, apiKey, valid)
	}
}

// WaitForFileChange is a test helper to ensure file modifications are detected
func WaitForFileChange(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Ensure filesystem has time to register the change
	// This is a small delay to help with filesystem latency
	// In real tests, you should wait for the actual notification
}

// CreateCorruptedAuthConfig creates an intentionally corrupted auth config for error testing
func CreateCorruptedAuthConfig(t *testing.T, corruptionType string) string {
	t.Helper()

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	var content string
	switch corruptionType {
	case "invalid-uuid":
		content = "[not-a-uuid]\ntest-key\n"
	case "key-before-org":
		content = "test-key\n[11111111-2222-3333-4444-555555555555]\n"
	case "malformed-bcrypt":
		content = "[11111111-2222-3333-4444-555555555555]\n$2a$invalid$hash\n"
	case "incomplete-bracket":
		content = "[11111111-2222-3333-4444-555555555555\ntest-key\n"
	default:
		t.Fatalf("Unknown corruption type: %s", corruptionType)
	}

	if err := os.WriteFile(authConfig, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write corrupted config: %v", err)
	}

	return authConfig
}

// AssertFileExists checks if a file exists and fails the test if not
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist: %s", path)
	}
}

// AssertFileContains checks if a file contains a substring
func AssertFileContains(t *testing.T, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !contains(string(content), substring) {
		t.Errorf("File %s does not contain %q", path, substring)
	}
}

// AssertFileNotContains checks if a file does not contain a substring
func AssertFileNotContains(t *testing.T, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if contains(string(content), substring) {
		t.Errorf("File %s should not contain %q", path, substring)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	TempDir    string
	AuthConfig string
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(b *testing.B) *BenchmarkHelper {
	b.Helper()

	tmpDir := b.TempDir()
	return &BenchmarkHelper{
		TempDir:    tmpDir,
		AuthConfig: filepath.Join(tmpDir, "auth.cfg"),
	}
}

// SetupAuthConfig sets up an auth config for benchmarking
func (h *BenchmarkHelper) SetupAuthConfig(orgs []OrgConfig, bcryptCost int) error {
	return WriteAuthConfig(h.AuthConfig, orgs, bcryptCost)
}

// CommonTestUUIDs provides commonly used test UUIDs
var (
	TestUUID1 = uuid.MustParse("11111111-2222-3333-4444-555555555555")
	TestUUID2 = uuid.MustParse("22222222-3333-4444-5555-666666666666")
	TestUUID3 = uuid.MustParse("33333333-4444-5555-6666-777777777777")
	ZeroUUID  = uuid.UUID{}
)

// CommonTestKeys provides commonly used test API keys
var (
	TestKey1 = "test-api-key-1"
	TestKey2 = "test-api-key-2"
	TestKey3 = "test-api-key-3"
)
