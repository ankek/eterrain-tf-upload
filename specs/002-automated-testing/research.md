# Research: Automated Testing Infrastructure

**Feature**: 002-automated-testing
**Phase**: Phase 0 - Research & Technology Selection
**Date**: 2025-11-27

## Overview

This document captures research decisions for implementing automated testing infrastructure for the Terraform backend service. All decisions align with constitution Section IV (Comprehensive Testing), Section VII (DRY), and Section VIII (KISS).

---

## Decision 1: Testing Framework

**Chosen**: Go standard library `testing` package + optional `testify/assert`

**Rationale**:
- **Standard Library First** (KISS principle): Go's built-in `testing` package provides all core functionality (test discovery, execution, benchmarks, subtests)
- **No Complex Dependencies**: Avoids heavyweight test frameworks that add complexity
- **Testify as Optional Enhancement**: `testify/assert` provides better assertion messages while remaining lightweight
- **Industry Standard**: Used by major Go projects (Kubernetes, Docker, Terraform itself)
- **Built-in Tooling**: `go test` command integrates seamlessly with coverage, profiling, and CI/CD

**Alternatives Considered**:
- **Ginkgo/Gomega (BDD framework)**: Rejected - adds significant complexity and non-standard syntax (violates KISS)
- **GoConvey**: Rejected - web UI and DSL add unnecessary overhead (violates KISS)
- **Pure standard library only**: Considered - but testify assertions significantly improve error messages with minimal cost

**Implementation**:
```go
// Unit test example using standard library + testify
package auth_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestValidateAPIKey(t *testing.T) {
    tests := []struct {
        name    string
        apiKey  string
        want    bool
    }{
        {"valid key", "valid-api-key-12345", true},
        {"empty key", "", false},
        {"short key", "abc", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ValidateAPIKey(tt.apiKey)
            assert.Equal(t, tt.want, got, "ValidateAPIKey(%q)", tt.apiKey)
        })
    }
}
```

---

## Decision 2: Integration Test Database Strategy

**Chosen**: Isolated test database with transaction-based cleanup

**Rationale**:
- **Isolation** (Constitution Section IV): Each test uses a separate test database or transaction scope
- **No Production Impact**: Test database completely separate from dev/prod databases
- **Fast Cleanup**: Rollback transactions instead of manual cleanup reduces test time
- **Parallel Execution**: Different test files can use separate database connections

**Alternatives Considered**:
- **In-memory SQLite**: Rejected - MySQL-specific syntax and behavior wouldn't be tested
- **Docker containers per test**: Rejected - too slow (violates <5 second constraint)
- **Shared test database with manual cleanup**: Rejected - risks test interdependence and data pollution

**Implementation Pattern**:
```go
// Integration test with transaction rollback
func TestDatabaseOperations(t *testing.T) {
    // Setup: Create test database connection
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    // Start transaction
    tx, err := db.Begin()
    require.NoError(t, err)
    defer tx.Rollback()  // Always rollback - no persist to disk

    // Test database operations using tx
    // ... test code ...

    // Transaction auto-rolls back via defer
}
```

**Environment Configuration**:
- `TEST_DB_HOST`, `TEST_DB_PORT`, `TEST_DB_USER`, `TEST_DB_PASSWORD`, `TEST_DB_NAME`
- Default: `localhost:3306`, user `test_user`, database `eterrain_test`
- Test database must be separate MySQL instance or dedicated test schema

---

## Decision 3: Test Utilities and Shared Code (DRY Compliance)

**Chosen**: Centralized `./tests/testutil/` package with reusable helpers

**Rationale**:
- **DRY Principle** (Constitution Section VII): Common test setup/teardown logic defined once
- **Consistency**: All tests use same fixtures, database setup, and assertions
- **Maintainability**: Bug fixes in test utilities apply to all tests

**Utilities Provided**:
1. **fixtures.go**: Common test data (org IDs, API keys, sample requests)
2. **database.go**: Test database connection, setup, teardown, transaction management
3. **assertions.go**: Custom assertions for domain-specific checks
4. **mocks.go**: Mock implementations of interfaces (Storage, CredentialStore, etc.)

**Implementation Example**:
```go
// testutil/database.go
package testutil

import (
    "database/sql"
    "testing"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    dsn := GetTestDSN()  // Reads TEST_DB_* env vars
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
```

---

## Decision 4: Test Numbering and Organization

**Chosen**: Test files numbered to match specification features (XXX-feature-test.go)

**Rationale**:
- **Constitution Mandate** (Section IV): "Test files MUST follow the same numbering strategy as specification features"
- **Traceability**: Easy to find tests for a specific feature
- **Maintainability**: Clear mapping between specs, implementation, and tests

**Naming Convention**:
- Unit tests: `./tests/unit-tests/002-automated-testing-test.go`
- Integration tests: `./tests/integration-tests/002-test-execution-test.go`
- Edge case tests: `./tests/edge-case-tests/002-test-isolation-test.go`
- Performance tests: `./tests/performance-tests/002-test-speed-test.go`

**File Organization**:
- Each feature gets test files in each category (unit, integration, edge-case, performance)
- Test files within a feature use descriptive suffixes (`-test.go`)
- Shared utilities are unnumbered in `testutil/` package

---

## Decision 5: Code Coverage Tooling

**Chosen**: Go built-in coverage tools (`go test -cover`, `-coverprofile`)

**Rationale**:
- **Standard Tooling** (KISS principle): No additional dependencies
- **Constitutional Requirement**: 80% coverage for critical paths (auth, validation, storage)
- **CI/CD Integration**: Coverage reports in standard format for automated enforcement

**Usage**:
```bash
# Run tests with coverage
go test ./tests/unit-tests/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Enforce 80% minimum coverage (CI/CD check)
go test ./internal/auth/... -coverprofile=auth_coverage.out
go tool cover -func=auth_coverage.out | grep total | awk '{print $3}' # Must be >= 80%
```

**Coverage Targets**:
- **100% coverage**: Security-critical paths (auth, validation, rate limiting)
- **80% minimum**: All critical paths (storage, middleware, handlers)
- **No coverage requirement**: Main function, initialization code, vendor code

---

## Decision 6: Performance Testing Approach

**Chosen**: Go benchmarks + table-driven load tests

**Rationale**:
- **Built-in Benchmarks**: `testing.B` provides standard benchmarking framework
- **Timing Attack Resistance**: Can verify constant-time operations
- **Load Simulation**: Table-driven tests can simulate concurrent requests

**Implementation Pattern**:
```go
// Benchmark for performance-critical function
func BenchmarkValidateAPIKey(b *testing.B) {
    apiKey := "test-api-key-12345"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ValidateAPIKey(apiKey)
    }
}

// Timing attack resistance test
func TestAuthConstantTime(t *testing.T) {
    validKey := "valid-key"
    invalidKey := "invalid"

    // Measure time for valid key
    validDurations := measureAuthTime(t, validKey, 1000)

    // Measure time for invalid key
    invalidDurations := measureAuthTime(t, invalidKey, 1000)

    // Verify times are within acceptable variance (constant-time)
    assert.InDelta(t, avg(validDurations), avg(invalidDurations), 0.1,
        "Auth time should be constant regardless of key validity")
}
```

---

## Decision 7: Test Execution Strategy

**Chosen**: Parallel test execution with isolated resources

**Rationale**:
- **Speed** (<30 second constraint): Parallel execution leverages multiple cores
- **Safety**: Each test is independent with isolated resources
- **Go Native**: `t.Parallel()` marker enables parallel execution

**Implementation**:
```go
func TestParallelSafe(t *testing.T) {
    t.Parallel()  // Mark test as safe for parallel execution

    // Test code with isolated resources
    // No shared state, no global variables
}
```

**Execution Commands**:
```bash
# Run all tests in parallel
go test ./tests/... -v -parallel 4

# Run specific test category
go test ./tests/unit-tests/... -v
go test ./tests/integration-tests/... -v

# Run with race detector (detect concurrent access bugs)
go test ./tests/... -race
```

---

## Decision 8: Continuous Integration Integration

**Chosen**: Makefile targets for standardized test execution

**Rationale**:
- **Consistency**: Same commands work locally and in CI/CD
- **DRY**: Test execution logic defined once
- **Documentation**: Makefile serves as runnable documentation

**Makefile Targets**:
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
	go test ./internal/auth/... ./internal/validation/... ./internal/storage/... \
		-coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | grep total
```

---

## Summary of Research Decisions

| Decision | Choice | Rationale | Constitution Alignment |
|----------|--------|-----------|----------------------|
| Testing Framework | Go `testing` + `testify/assert` | Standard library, simple, industry-standard | Section VIII (KISS) |
| Integration DB | Isolated DB + transactions | Fast cleanup, parallel execution | Section IV (Testing) |
| Test Utilities | Centralized `testutil/` package | Reusable helpers, consistency | Section VII (DRY) |
| Test Numbering | Match spec numbers (XXX-feature-test.go) | Traceability, constitutional mandate | Section IV (Testing) |
| Code Coverage | Go built-in tools | No dependencies, CI/CD integration | Section IV (80% coverage) |
| Performance Tests | Go benchmarks + load tests | Built-in, timing attack verification | Section IV (Testing) |
| Test Execution | Parallel with `t.Parallel()` | Speed, safety, Go native | Section IV (Testing) |
| CI Integration | Makefile targets | Consistency, DRY, documentation | Section VII (DRY) |

**Phase 0 Complete**: All technology decisions made. No unresolved questions. Ready for Phase 1 design.
