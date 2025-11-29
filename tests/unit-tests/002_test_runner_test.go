// tests/unit-tests/002_test_runner_test.go
package unit_tests_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findRepoRoot walks up the directory tree to find the repository root
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod or hit root
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// TestMakeTestAllExecutesAllCategories verifies that `make test-all` runs all test categories
// T037: Write test to verify `make test-all` executes all test categories
func TestMakeTestAllExecutesAllCategories(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root - skipping make test-all validation")
	}

	// Skip if Makefile doesn't exist
	makefilePath := filepath.Join(repoRoot, "Makefile.testing")
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		t.Skip("Makefile.testing not found - skipping make test-all validation")
	}

	// Run make test-all from repository root
	cmd := exec.Command("make", "-f", "Makefile.testing", "test-all")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()

	// Note: This command may fail if integration tests require database
	// The key is to verify it ATTEMPTS to run all categories
	outputStr := string(output)

	// Verify all test categories are mentioned in output
	testCategories := []string{
		"Running unit tests",
		"Running integration tests",
		"Running edge case tests",
		"Running performance tests",
	}

	for _, category := range testCategories {
		assert.Contains(t, outputStr, category,
			"make test-all should execute %s", category)
	}

	// Log the output for debugging
	t.Logf("make test-all output:\n%s", outputStr)

	// Log the error if any (may be expected if DB not set up)
	if err != nil {
		t.Logf("make test-all exited with error (may be expected if DB not configured): %v", err)
	}
}

// TestMakefileTargetsExist verifies that all required Makefile targets exist
func TestMakefileTargetsExist(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root - skipping Makefile validation")
	}

	// Read Makefile content
	makefilePath := filepath.Join(repoRoot, "Makefile.testing")
	content, err := os.ReadFile(makefilePath)
	if os.IsNotExist(err) {
		t.Skip("Makefile.testing not found - skipping Makefile validation")
	}
	require.NoError(t, err, "Should be able to read Makefile.testing")

	makefileContent := string(content)

	// Required targets
	requiredTargets := []string{
		"test-unit:",
		"test-integration:",
		"test-edge:",
		"test-performance:",
		"test-all:",
		"coverage:",
	}

	for _, target := range requiredTargets {
		assert.Contains(t, makefileContent, target,
			"Makefile should contain target %s", target)
	}
}

// TestTestDirectoryStructure verifies the test directory structure exists
func TestTestDirectoryStructure(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root")
	}

	requiredDirs := []string{
		"tests/unit-tests",
		"tests/integration-tests",
		"tests/edge-case-tests",
		"tests/performance-tests",
		"tests/testutil",
		"tests/scripts",
	}

	for _, dir := range requiredDirs {
		fullPath := filepath.Join(repoRoot, dir)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "Directory %s should exist", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)
	}
}

// TestTestUtilPackageAvailable verifies testutil package files exist
func TestTestUtilPackageAvailable(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root")
	}

	requiredFiles := []string{
		"tests/testutil/fixtures.go",
		"tests/testutil/database.go",
		"tests/testutil/assertions.go",
		"tests/testutil/performance.go",
	}

	for _, file := range requiredFiles {
		fullPath := filepath.Join(repoRoot, file)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "File %s should exist", file)
		assert.False(t, info.IsDir(), "%s should be a file", file)
	}
}

// TestGitignoreContainsCoverageArtifacts verifies coverage files are in .gitignore
func TestGitignoreContainsCoverageArtifacts(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root")
	}

	// Read .gitignore
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if os.IsNotExist(err) {
		t.Skip(".gitignore not found - skipping validation")
	}
	require.NoError(t, err, "Should be able to read .gitignore")

	gitignoreContent := string(content)

	// Verify coverage artifacts are ignored
	coverageArtifacts := []string{
		"coverage.out",
		"coverage.html",
	}

	for _, artifact := range coverageArtifacts {
		assert.Contains(t, gitignoreContent, artifact,
			".gitignore should contain %s", artifact)
	}
}

// TestEnvTestFileExists verifies .env.test template exists
func TestEnvTestFileExists(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root")
	}

	envPath := filepath.Join(repoRoot, ".env.test")
	info, err := os.Stat(envPath)
	require.NoError(t, err, ".env.test file should exist")
	assert.False(t, info.IsDir(), ".env.test should be a file")

	// Read content
	content, err := os.ReadFile(envPath)
	require.NoError(t, err, "Should be able to read .env.test")

	envContent := string(content)

	// Verify required environment variables
	requiredVars := []string{
		"TEST_DB_HOST",
		"TEST_DB_PORT",
		"TEST_DB_USER",
		"TEST_DB_PASSWORD",
		"TEST_DB_NAME",
	}

	for _, varName := range requiredVars {
		assert.Contains(t, envContent, varName,
			".env.test should contain %s", varName)
	}
}

// TestDatabaseSetupScriptExists verifies database setup SQL script exists
func TestDatabaseSetupScriptExists(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skip("Could not find repository root")
	}

	scriptPath := filepath.Join(repoRoot, "tests/scripts/setup-test-db.sql")
	info, err := os.Stat(scriptPath)
	require.NoError(t, err, "tests/scripts/setup-test-db.sql should exist")
	assert.False(t, info.IsDir(), "setup-test-db.sql should be a file")

	// Read content
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read setup-test-db.sql")

	scriptContent := strings.ToUpper(string(content))

	// Verify SQL contains required statements
	assert.Contains(t, scriptContent, "CREATE DATABASE", "Setup script should create database")
	assert.Contains(t, scriptContent, "CREATE USER", "Setup script should create user")
	assert.Contains(t, scriptContent, "GRANT", "Setup script should grant privileges")
}
