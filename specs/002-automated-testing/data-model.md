# Data Model: Automated Testing Infrastructure

**Feature**: 002-automated-testing
**Phase**: Phase 1 - Design & Contracts
**Date**: 2025-11-27

## Overview

This document defines the data structures and entities for the automated testing infrastructure. The testing system itself doesn't persist data to databases, but it manages test artifacts, configurations, and execution results.

---

## Entity 1: Test Suite

**Purpose**: Collection of all tests organized by category (unit, integration, edge-case, performance)

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Category` | string | Test category (unit, integration, edge-case, performance) | Required, one of 4 categories |
| `FeatureNumber` | string | Feature specification number (e.g., "001", "002") | Required, 3-digit format |
| `TestFiles` | []string | List of test file paths in this suite | Required, min 1 file |
| `TotalTests` | int | Total number of test cases in suite | Auto-calculated |
| `PassedTests` | int | Number of tests that passed | Auto-calculated |
| `FailedTests` | int | Number of tests that failed | Auto-calculated |
| `ExecutionTime` | duration | Total execution time for suite | Auto-calculated |

**Relationships**:
- Contains multiple `TestCase` entities
- Associated with one `FeatureNumber`

**File Location**:
- Not persisted to database
- Exists as runtime structure during test execution
- Represented in test output and coverage reports

---

## Entity 2: Test Case

**Purpose**: Individual test function that validates specific behavior or scenario

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Name` | string | Test function name (e.g., "TestValidateAPIKey") | Required, starts with "Test" or "Benchmark" |
| `FilePath` | string | Path to test file | Required, ends with "_test.go" |
| `Category` | string | Test category (unit, integration, edge-case, performance) | Required, one of 4 categories |
| `Status` | string | Test result (pass, fail, skip) | Required, one of 3 statuses |
| `Duration` | duration | Execution time for this test case | Auto-calculated |
| `ErrorMessage` | string | Error message if test failed | Optional, present only if Status=fail |
| `StackTrace` | string | Stack trace if test failed | Optional, present only if Status=fail |

**Relationships**:
- Belongs to one `TestSuite`
- May have multiple `TestAssertion` entities

**File Location**:
- Not persisted to database
- Exists as runtime structure during test execution
- Captured in test output (`go test -v` output)

---

## Entity 3: Test Fixture

**Purpose**: Reusable test data, setup, and teardown logic used across multiple tests

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Name` | string | Fixture identifier (e.g., "ValidOrgID", "TestAPIKey") | Required, unique within testutil package |
| `Type` | string | Data type (string, UUID, struct, etc.) | Required |
| `Value` | interface{} | Actual fixture value | Required |
| `Description` | string | Purpose of this fixture | Optional |

**Relationships**:
- Used by multiple `TestCase` entities
- Defined in `./tests/testutil/fixtures.go`

**Example**:
```go
// testutil/fixtures.go
package testutil

const (
    ValidOrgID     = "11111111-2222-3333-4444-555555555555"
    ValidAPIKey    = "demo-api-key-12345"
    InvalidOrgID   = "invalid-uuid"
    EmptyString    = ""
)

var SampleUploadRequest = map[string]interface{}{
    "resource_type": "vm_instance",
    "resource_name": "web-server-01",
    "status":        "running",
    "region":        "us-east-1",
}
```

---

## Entity 4: Test Helper

**Purpose**: Shared utility functions in `./tests/testutil/` for common test operations (assertions, mocks, setup)

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `FunctionName` | string | Helper function name (e.g., "SetupTestDB") | Required |
| `Package` | string | Package within testutil (database, assertions, mocks) | Required |
| `Purpose` | string | What this helper does | Required |
| `Parameters` | []Parameter | Function parameters | Optional |
| `ReturnTypes` | []Type | Function return types | Optional |

**Relationships**:
- Used by multiple `TestCase` entities
- Organized into packages within `./tests/testutil/`

**Example**:
```go
// testutil/database.go
package testutil

import (
    "database/sql"
    "testing"
)

// SetupTestDB creates a test database connection
// Helper for integration tests
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    // Implementation...
}

// TeardownTestDB closes the database connection
// Helper for integration tests
func TeardownTestDB(t *testing.T, db *sql.DB) {
    t.Helper()
    // Implementation...
}
```

---

## Entity 5: Test Database

**Purpose**: Isolated database instance or transaction scope used for integration testing

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Host` | string | Test database host | Default: "localhost" |
| `Port` | int | Test database port | Default: 3306 |
| `User` | string | Test database user | Default: "test_user" |
| `Password` | string | Test database password | From env: TEST_DB_PASSWORD |
| `Database` | string | Test database name | Default: "eterrain_test" |
| `Connection` | *sql.DB | Active database connection | Managed by testutil |
| `Transaction` | *sql.Tx | Active transaction (for rollback cleanup) | Optional, for transactional tests |

**Relationships**:
- Used by integration tests only
- Completely separate from development/production databases
- Managed by `testutil.SetupTestDB()` and `testutil.TeardownTestDB()`

**Environment Configuration**:
- `TEST_DB_HOST`: Database host (default: "localhost")
- `TEST_DB_PORT`: Database port (default: "3306")
- `TEST_DB_USER`: Database user (default: "test_user")
- `TEST_DB_PASSWORD`: Database password (required)
- `TEST_DB_NAME`: Database name (default: "eterrain_test")

**State Transitions**:
1. **Setup**: Connection created, transaction started
2. **Testing**: Test performs database operations within transaction
3. **Teardown**: Transaction rolled back, connection closed
4. **Result**: No data persisted to disk, clean state for next test

---

## Entity 6: Test Report

**Purpose**: Output from test execution showing pass/fail status, execution time, and failure diagnostics

**Attributes**:

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Timestamp` | datetime | When test run started | Auto-generated |
| `Category` | string | Test category or "all" for full suite | Required |
| `TotalTests` | int | Total number of tests executed | Auto-calculated |
| `PassedTests` | int | Number of tests that passed | Auto-calculated |
| `FailedTests` | int | Number of tests that failed | Auto-calculated |
| `SkippedTests` | int | Number of tests skipped | Auto-calculated |
| `TotalDuration` | duration | Total execution time | Auto-calculated |
| `CoveragePercent` | float | Code coverage percentage | Auto-calculated from -cover |
| `FailureDetails` | []TestCase | Details of failed tests | Optional, present if failures exist |

**Relationships**:
- Contains multiple `TestCase` entities
- Generated by `go test` command execution

**Output Formats**:
1. **Console**: Standard `go test -v` output
2. **JSON**: `go test -json` for CI/CD integration
3. **Coverage**: `coverage.out` file for coverage reports
4. **HTML**: `coverage.html` for visual coverage inspection

**Example Output**:
```
=== RUN   TestValidateAPIKey
=== RUN   TestValidateAPIKey/valid_key
=== RUN   TestValidateAPIKey/empty_key
=== RUN   TestValidateAPIKey/short_key
--- PASS: TestValidateAPIKey (0.00s)
    --- PASS: TestValidateAPIKey/valid_key (0.00s)
    --- PASS: TestValidateAPIKey/empty_key (0.00s)
    --- PASS: TestValidateAPIKey/short_key (0.00s)
PASS
coverage: 85.2% of statements
ok      github.com/yourrepo/tests/unit-tests    0.234s
```

---

## Data Flow Diagrams

### Test Execution Flow

```
1. Developer runs: go test ./tests/unit-tests/...

2. Go test runner:
   - Discovers test files (*_test.go)
   - Loads TestSuite for category
   - Executes each TestCase

3. TestCase execution:
   - Setup (uses TestHelper functions)
   - Run test logic (uses TestFixture data)
   - Assertions (uses testify/assert)
   - Teardown (cleanup via defer)

4. Test runner generates TestReport:
   - Aggregates pass/fail counts
   - Calculates coverage
   - Outputs results

5. Results available:
   - Console output (human-readable)
   - JSON output (CI/CD parsing)
   - Coverage files (coverage.out, coverage.html)
```

### Integration Test Database Flow

```
1. Integration test starts:
   - testutil.SetupTestDB(t) creates TestDatabase connection

2. Transaction-based testing:
   - tx, _ := db.Begin() starts transaction
   - Test performs CRUD operations within tx
   - defer tx.Rollback() ensures cleanup

3. Test completes:
   - Transaction rolls back (no persist)
   - testutil.TeardownTestDB(t, db) closes connection

4. Next test:
   - Clean database state
   - No data pollution from previous test
```

---

## File Organization

### Test Directory Structure

```
tests/
├── unit-tests/
│   ├── 001-baseline-documentation-test.go
│   ├── 002-automated-testing-test.go
│   └── ...
│
├── integration-tests/
│   ├── 001-database-ops-test.go
│   ├── 002-api-workflow-test.go
│   └── ...
│
├── edge-case-tests/
│   ├── 001-validation-edge-test.go
│   ├── 002-security-edge-test.go
│   └── ...
│
├── performance-tests/
│   ├── 001-auth-timing-test.go
│   ├── 002-rate-limit-test.go
│   └── ...
│
└── testutil/
    ├── fixtures.go       # TestFixture definitions
    ├── database.go       # TestDatabase helpers
    ├── assertions.go     # Custom assertions
    └── mocks.go          # Mock implementations
```

---

## Validation Rules Summary

### Test File Naming

- **Format**: `XXX-feature-description-test.go`
- **XXX**: 3-digit feature number (001, 002, etc.)
- **Suffix**: Must end with `_test.go` (Go convention)

### Test Function Naming

- **Unit/Integration/Edge**: Start with `Test` prefix
- **Performance**: Start with `Benchmark` prefix
- **Examples**: `TestValidateAPIKey`, `BenchmarkAuthPerformance`

### Test Categories

- **unit**: Individual function/component testing
- **integration**: Multi-component and database testing
- **edge-case**: Boundary conditions and error scenarios
- **performance**: Speed and timing tests

### Coverage Requirements

- **100% coverage**: Security-critical code (auth, validation, rate limiting)
- **80% minimum**: Critical paths (storage, handlers, middleware)
- **No requirement**: Main function, init code, vendor code

---

## Summary

The automated testing infrastructure manages six primary entities:

1. **Test Suite**: Organization of tests by category and feature
2. **Test Case**: Individual test functions with pass/fail status
3. **Test Fixture**: Reusable test data in `testutil/fixtures.go`
4. **Test Helper**: Utility functions in `testutil/*.go`
5. **Test Database**: Isolated database for integration tests
6. **Test Report**: Execution results and coverage metrics

All entities follow constitution requirements for test-first development, DRY principles (shared utilities), and KISS principles (standard Go testing library).
