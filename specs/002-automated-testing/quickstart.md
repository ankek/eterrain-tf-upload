# Quickstart: Automated Testing Infrastructure

**Feature**: 002-automated-testing
**Phase**: Phase 1 - Design & Contracts
**Date**: 2025-11-27

## Goal

Get the automated testing infrastructure up and running in under 10 minutes. This guide walks you through:
1. Creating the test directory structure
2. Writing your first test
3. Running tests
4. Validating coverage

---

## Prerequisites

- ✅ Go 1.25+ installed (`go version`)
- ✅ MySQL 8.4+ running (for integration tests)
- ✅ Repository cloned and on `002-automated-testing` branch

---

## Step 1: Create Test Directory Structure (2 minutes)

Create the four test category directories and the shared utilities package:

```bash
# From repository root
mkdir -p tests/unit-tests
mkdir -p tests/integration-tests
mkdir -p tests/edge-case-tests
mkdir -p tests/performance-tests
mkdir -p tests/testutil
```

Verify the structure:

```bash
tree tests/
# Expected output:
# tests/
# ├── edge-case-tests/
# ├── integration-tests/
# ├── performance-tests/
# ├── unit-tests/
# └── testutil/
```

---

## Step 2: Create Shared Test Utilities (3 minutes)

### Create `tests/testutil/fixtures.go`

```go
// tests/testutil/fixtures.go
package testutil

// Common test data constants
const (
	// Valid test organization ID
	ValidOrgID = "11111111-2222-3333-4444-555555555555"

	// Valid test API key
	ValidAPIKey = "demo-api-key-12345"

	// Invalid test data
	InvalidOrgID  = "invalid-uuid"
	InvalidAPIKey = "short"
	EmptyString   = ""
)

// SampleUploadRequest returns a valid upload request for testing
func SampleUploadRequest() map[string]interface{} {
	return map[string]interface{}{
		"resource_type": "vm_instance",
		"resource_name": "web-server-01",
		"status":        "running",
		"region":        "us-east-1",
	}
}
```

### Create `tests/testutil/database.go`

```go
// tests/testutil/database.go
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// GetTestDSN returns the Data Source Name for test database
func GetTestDSN() string {
	host := getEnv("TEST_DB_HOST", "localhost")
	port := getEnv("TEST_DB_PORT", "3306")
	user := getEnv("TEST_DB_USER", "test_user")
	password := getEnv("TEST_DB_PASSWORD", "test_password")
	dbname := getEnv("TEST_DB_NAME", "eterrain_test")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		user, password, host, port, dbname)
}

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := GetTestDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Test database ping failed: %v", err)
	}

	return db
}

// TeardownTestDB closes the database connection
func TeardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

---

## Step 3: Write Your First Unit Test (2 minutes)

Create `tests/unit-tests/002-automated-testing-test.go`:

```go
// tests/unit-tests/002-automated-testing-test.go
package unit_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourrepo/eterrain-tf-upload/tests/testutil"
)

// TestFixturesAvailable validates that test fixtures are accessible
func TestFixturesAvailable(t *testing.T) {
	t.Parallel() // Mark as safe for parallel execution

	// Validate org ID fixture
	orgID := testutil.ValidOrgID
	assert.NotEmpty(t, orgID, "ValidOrgID fixture should not be empty")
	assert.Len(t, orgID, 36, "ValidOrgID should be UUID format (36 chars)")

	// Validate API key fixture
	apiKey := testutil.ValidAPIKey
	assert.NotEmpty(t, apiKey, "ValidAPIKey fixture should not be empty")

	// Validate sample request
	request := testutil.SampleUploadRequest()
	assert.Contains(t, request, "resource_type", "Sample request should contain resource_type")
	assert.Equal(t, "vm_instance", request["resource_type"])
}

// TestTableDrivenExample demonstrates table-driven testing pattern
func TestTableDrivenExample(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid input",
			input:   "valid-data",
			want:    true,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid input",
			input:   "!!!",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Example validation function (replace with real implementation)
			got := len(tt.input) > 0 && tt.input[0] != '!'
			err := error(nil)
			if !got {
				err = fmt.Errorf("invalid input")
			}

			assert.Equal(t, tt.want, got, "validation result mismatch")
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}
```

---

## Step 4: Run Your First Test (1 minute)

```bash
# Run the unit test
go test ./tests/unit-tests/... -v

# Expected output:
# === RUN   TestFixturesAvailable
# --- PASS: TestFixturesAvailable (0.00s)
# === RUN   TestTableDrivenExample
# === RUN   TestTableDrivenExample/valid_input
# === RUN   TestTableDrivenExample/empty_input
# === RUN   TestTableDrivenExample/invalid_input
# --- PASS: TestTableDrivenExample (0.00s)
# PASS
# ok      ./tests/unit-tests    0.234s
```

✅ **Success!** You've created and run your first test.

---

## Step 5: Write an Integration Test (2 minutes)

Create `tests/integration-tests/002-database-setup-test.go`:

```go
// tests/integration-tests/002-database-setup-test.go
package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourrepo/eterrain-tf-upload/tests/testutil"
)

// TestDatabaseConnection validates test database connectivity
func TestDatabaseConnection(t *testing.T) {
	// Setup: Create test database connection
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Test: Verify connection is active
	err := db.Ping()
	require.NoError(t, err, "Database ping should succeed")

	// Test: Query database version
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	require.NoError(t, err, "Version query should succeed")
	assert.NotEmpty(t, version, "Database version should not be empty")
	assert.Contains(t, version, "MySQL", "Should be MySQL database")

	t.Logf("Test database version: %s", version)
}

// TestTransactionRollback validates transaction-based test cleanup
func TestTransactionRollback(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Start transaction
	tx, err := db.Begin()
	require.NoError(t, err, "Transaction start should succeed")
	defer tx.Rollback() // Ensure rollback even if test fails

	// Perform database operation within transaction
	// (This is example - replace with real schema)
	_, err = tx.Exec("CREATE TABLE IF NOT EXISTS test_table (id INT PRIMARY KEY, name VARCHAR(50))")
	require.NoError(t, err, "Table creation should succeed")

	_, err = tx.Exec("INSERT INTO test_table (id, name) VALUES (?, ?)", 1, "test-data")
	require.NoError(t, err, "Insert should succeed")

	// Query within transaction
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
	require.NoError(t, err, "Count query should succeed")
	assert.Equal(t, 1, count, "Should have 1 row")

	// Transaction will rollback via defer - no data persisted
	t.Log("Transaction will rollback - no data persisted to disk")
}
```

Run the integration test:

```bash
# Set test database password (required)
export TEST_DB_PASSWORD=your_test_password

# Run integration test
go test ./tests/integration-tests/... -v

# Expected output:
# === RUN   TestDatabaseConnection
# --- PASS: TestDatabaseConnection (0.12s)
#     002-database-setup-test.go:XX: Test database version: 8.4.0-MySQL
# === RUN   TestTransactionRollback
# --- PASS: TestTransactionRollback (0.23s)
# PASS
# ok      ./tests/integration-tests    0.543s
```

---

## Step 6: Check Code Coverage (1 minute)

```bash
# Run tests with coverage
go test ./tests/unit-tests/... -cover

# Expected output:
# PASS
# coverage: 75.0% of statements
# ok      ./tests/unit-tests    0.234s

# Generate HTML coverage report
go test ./tests/unit-tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Open coverage.html in browser to see covered/uncovered code
```

---

## Step 7: Create Makefile for Convenience (Optional)

Create `Makefile` in repository root:

```makefile
.PHONY: test test-unit test-integration test-edge test-performance test-all coverage

# Run all tests
test-all: test-unit test-integration test-edge test-performance

# Unit tests only
test-unit:
	@echo "Running unit tests..."
	go test ./tests/unit-tests/... -v -cover

# Integration tests (requires test database)
test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration-tests/... -v -cover

# Edge case tests
test-edge:
	@echo "Running edge case tests..."
	go test ./tests/edge-case-tests/... -v

# Performance tests
test-performance:
	@echo "Running performance tests..."
	go test ./tests/performance-tests/... -v -bench=. -benchmem

# Coverage report for critical paths
coverage:
	@echo "Generating coverage report..."
	go test ./tests/unit-tests/... ./tests/integration-tests/... \
		-coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	go tool cover -func=coverage.out | grep total
```

Now you can run tests with simple commands:

```bash
make test-unit           # Run unit tests
make test-integration    # Run integration tests
make test-all            # Run all tests
make coverage            # Generate coverage report
```

---

## Next Steps

✅ **You've completed the quickstart!** Your testing infrastructure is ready.

**What to do next:**

1. **Write tests for existing code**: Start with authentication and validation functions
2. **Follow TDD workflow**: Write tests BEFORE implementing new features
3. **Check coverage**: Aim for 80%+ on critical paths, 100% on security code
4. **Add CI/CD**: Integrate `make test-all` into your CI/CD pipeline

**Example TDD Workflow:**

```bash
# 1. Write failing test
vim tests/unit-tests/003-new-feature-test.go
go test ./tests/unit-tests/... -v  # RED: Test fails

# 2. Implement feature
vim internal/mypackage/myfeature.go

# 3. Run test again
go test ./tests/unit-tests/... -v  # GREEN: Test passes

# 4. Refactor if needed
# ... refactor code ...
go test ./tests/unit-tests/... -v  # Still GREEN

# 5. Check coverage
make coverage  # Verify >= 80%
```

---

## Troubleshooting

### Issue: `go: cannot find module providing package github.com/stretchr/testify/assert`

**Solution**: Install testify dependency

```bash
go get github.com/stretchr/testify/assert
go mod tidy
```

### Issue: Integration tests fail with "Access denied for user 'test_user'"

**Solution**: Create test database and user

```sql
CREATE DATABASE IF NOT EXISTS eterrain_test;
CREATE USER IF NOT EXISTS 'test_user'@'localhost' IDENTIFIED BY 'test_password';
GRANT ALL PRIVILEGES ON eterrain_test.* TO 'test_user'@'localhost';
FLUSH PRIVILEGES;
```

### Issue: Tests fail with "no such table"

**Solution**: Integration tests use transaction-based cleanup. Create tables within each test or use test fixtures.

### Issue: Coverage is 0%

**Solution**: Make sure you're testing actual implementation code, not just test utilities. Coverage measures code in `internal/` and `cmd/`, not `tests/`.

---

## Summary

**Time invested**: ~10 minutes
**What you built**:
- ✅ Test directory structure (`./tests/`)
- ✅ Shared test utilities (`testutil` package)
- ✅ First unit test with fixtures and table-driven testing
- ✅ First integration test with database connection and transactions
- ✅ Coverage reporting
- ✅ Makefile for convenient test execution

**You're ready to implement test-first development!** 🎉
