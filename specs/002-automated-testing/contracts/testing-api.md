# Testing API Contracts

**Feature**: 002-automated-testing
**Phase**: Phase 1 - Design & Contracts
**Date**: 2025-11-27

## Overview

This document defines the "API" contracts for the automated testing infrastructure. Since this feature focuses on testing infrastructure rather than HTTP endpoints, the "API" here refers to the public interfaces and commands that developers use to interact with the testing system.

---

## Contract 1: Test Execution Commands

### Run All Tests

**Command**: `make test-all`

**Description**: Execute all test categories (unit, integration, edge-case, performance) in sequence

**Prerequisites**:
- Go 1.25+ installed
- Test database available (for integration tests)
- Environment variables configured (TEST_DB_* variables)

**Input**: None

**Output**:
```
Running unit tests...
ok      ./tests/unit-tests    0.234s  coverage: 85.2% of statements

Running integration tests...
ok      ./tests/integration-tests    1.543s  coverage: 78.5% of statements

Running edge case tests...
ok      ./tests/edge-case-tests    0.456s

Running performance tests...
BenchmarkAuthPerformance-4    1000000    1234 ns/op    512 B/op    8 allocs/op
ok      ./tests/performance-tests    2.123s
```

**Success Criteria**: All tests pass (exit code 0)

**Failure Handling**: Non-zero exit code, detailed error messages showing which tests failed

---

### Run Unit Tests Only

**Command**: `make test-unit` or `go test ./tests/unit-tests/... -v`

**Description**: Execute unit tests that test individual functions and components in isolation

**Prerequisites**:
- Go 1.25+ installed
- No external dependencies required

**Input**: None

**Output**:
```
=== RUN   TestValidateAPIKey
=== RUN   TestValidateAPIKey/valid_key
=== RUN   TestValidateAPIKey/empty_key
--- PASS: TestValidateAPIKey (0.00s)
    --- PASS: TestValidateAPIKey/valid_key (0.00s)
    --- PASS: TestValidateAPIKey/empty_key (0.00s)
PASS
coverage: 85.2% of statements
ok      ./tests/unit-tests    0.234s
```

**Success Criteria**: All unit tests pass, coverage >= 80% for critical paths

**Failure Handling**: Non-zero exit code, detailed failure messages with expected vs actual values

---

### Run Integration Tests Only

**Command**: `make test-integration` or `go test ./tests/integration-tests/... -v`

**Description**: Execute integration tests that validate database operations and multi-component interactions

**Prerequisites**:
- Go 1.25+ installed
- Test database running (MySQL 8.4+)
- Environment variables: `TEST_DB_HOST`, `TEST_DB_PORT`, `TEST_DB_USER`, `TEST_DB_PASSWORD`, `TEST_DB_NAME`

**Input**: None

**Output**:
```
=== RUN   TestDatabaseCRUD
=== RUN   TestDatabaseCRUD/create
=== RUN   TestDatabaseCRUD/read
=== RUN   TestDatabaseCRUD/update
=== RUN   TestDatabaseCRUD/delete
--- PASS: TestDatabaseCRUD (1.23s)
PASS
ok      ./tests/integration-tests    1.543s
```

**Success Criteria**: All integration tests pass, database operations validated, cleanup successful

**Failure Handling**:
- Database connection errors: Clear message about TEST_DB_* configuration
- Test failures: SQL error messages, stack traces, cleanup status

---

### Run Edge Case Tests Only

**Command**: `make test-edge` or `go test ./tests/edge-case-tests/... -v`

**Description**: Execute tests for boundary conditions, error scenarios, and edge cases

**Prerequisites**:
- Go 1.25+ installed
- No external dependencies required

**Input**: None

**Output**:
```
=== RUN   TestValidationEdgeCases
=== RUN   TestValidationEdgeCases/empty_string
=== RUN   TestValidationEdgeCases/max_size
=== RUN   TestValidationEdgeCases/null_input
--- PASS: TestValidationEdgeCases (0.12s)
PASS
ok      ./tests/edge-case-tests    0.456s
```

**Success Criteria**: All edge case tests pass, boundary conditions validated

**Failure Handling**: Non-zero exit code, detailed failure messages for each edge case

---

### Run Performance Tests Only

**Command**: `make test-performance` or `go test ./tests/performance-tests/... -v -bench=. -benchmem`

**Description**: Execute performance tests that measure execution time and validate timing constraints

**Prerequisites**:
- Go 1.25+ installed
- No external dependencies required

**Input**: None

**Output**:
```
goos: linux
goarch: amd64
BenchmarkAuthPerformance-4        1000000     1234 ns/op    512 B/op    8 allocs/op
BenchmarkRateLimiting-4            500000     2456 ns/op   1024 B/op   16 allocs/op
PASS
ok      ./tests/performance-tests    2.123s
```

**Success Criteria**: Benchmarks complete, performance targets met (e.g., <2ms per auth operation)

**Failure Handling**: Performance degradation warnings, timing attack vulnerabilities identified

---

## Contract 2: Code Coverage Commands

### Generate Coverage Report

**Command**: `make coverage`

**Description**: Generate code coverage report for critical paths (auth, validation, storage)

**Prerequisites**:
- Go 1.25+ installed
- Tests must be executable

**Input**: None

**Output**:
```
Generating coverage report...
ok      ./internal/auth       0.234s  coverage: 92.5% of statements
ok      ./internal/validation 0.123s  coverage: 88.3% of statements
ok      ./internal/storage    0.456s  coverage: 81.7% of statements

total:                                    coverage: 87.5% of statements
```

**Success Criteria**: Critical paths achieve >= 80% coverage, security-critical code achieves 100% coverage

**Failure Handling**: Warning if coverage < 80%, error if coverage < 70%

**Artifacts**:
- `coverage.out`: Machine-readable coverage data
- `coverage.html`: Human-readable HTML report

---

### View HTML Coverage Report

**Command**: `go tool cover -html=coverage.out`

**Description**: Open interactive HTML coverage report in browser

**Prerequisites**:
- `coverage.out` file generated from previous coverage run
- Web browser available

**Input**: None

**Output**: Opens browser showing color-coded coverage:
- Green: Covered lines
- Red: Uncovered lines
- Gray: Non-executable lines

**Success Criteria**: Report displays, easy to identify uncovered code

---

## Contract 3: Test Database Setup

### Initialize Test Database

**Command**: `make setup-test-db` (custom Makefile target)

**Description**: Create and configure isolated test database for integration tests

**Prerequisites**:
- MySQL 8.4+ server running
- Admin credentials for database creation

**Input**: None (uses TEST_DB_* environment variables)

**Output**:
```
Creating test database 'eterrain_test'...
Creating test user 'test_user'...
Granting permissions...
Running migrations on test database...
Test database ready!
```

**Success Criteria**: Test database created, user configured, migrations applied

**Failure Handling**: Clear error messages for connection issues, permission problems, migration failures

**SQL Operations**:
```sql
CREATE DATABASE IF NOT EXISTS eterrain_test;
CREATE USER IF NOT EXISTS 'test_user'@'localhost' IDENTIFIED BY 'test_password';
GRANT ALL PRIVILEGES ON eterrain_test.* TO 'test_user'@'localhost';
FLUSH PRIVILEGES;
```

---

### Teardown Test Database

**Command**: `make teardown-test-db` (custom Makefile target)

**Description**: Drop test database and clean up test user (for complete cleanup)

**Prerequisites**:
- Test database exists
- Admin credentials for database deletion

**Input**: None (uses TEST_DB_* environment variables)

**Output**:
```
Dropping test database 'eterrain_test'...
Removing test user 'test_user'...
Test database cleanup complete!
```

**Success Criteria**: Test database removed, user deleted

**Failure Handling**: Warnings if database/user doesn't exist (non-fatal)

---

## Contract 4: Test Utilities Public Interface

### testutil.SetupTestDB(t *testing.T) *sql.DB

**Description**: Create test database connection for integration tests

**Parameters**:
- `t *testing.T`: Testing context for error reporting

**Returns**:
- `*sql.DB`: Database connection configured for test database

**Behavior**:
- Reads TEST_DB_* environment variables
- Creates connection to test database
- Verifies connection with Ping()
- Fails test if connection cannot be established

**Usage Example**:
```go
func TestDatabaseOperations(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    // Test code using db
}
```

---

### testutil.TeardownTestDB(t *testing.T, db *sql.DB)

**Description**: Close test database connection and cleanup

**Parameters**:
- `t *testing.T`: Testing context for error reporting
- `db *sql.DB`: Database connection to close

**Returns**: None

**Behavior**:
- Closes database connection
- Reports error if close fails (non-fatal)

---

### testutil.GetValidOrgID() string

**Description**: Get valid organization ID for testing

**Parameters**: None

**Returns**:
- `string`: Valid UUID for test organization

**Behavior**:
- Returns constant: `"11111111-2222-3333-4444-555555555555"`
- Consistent across all tests

---

### testutil.GetValidAPIKey() string

**Description**: Get valid API key for testing

**Parameters**: None

**Returns**:
- `string`: Valid API key for test authentication

**Behavior**:
- Returns constant: `"demo-api-key-12345"`
- Consistent across all tests

---

## Contract 5: Test Execution Environment

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TEST_DB_HOST` | No | `localhost` | Test database host |
| `TEST_DB_PORT` | No | `3306` | Test database port |
| `TEST_DB_USER` | No | `test_user` | Test database user |
| `TEST_DB_PASSWORD` | Yes (integration tests) | None | Test database password |
| `TEST_DB_NAME` | No | `eterrain_test` | Test database name |

**Configuration Example** (`.env.test`):
```bash
TEST_DB_HOST=localhost
TEST_DB_PORT=3306
TEST_DB_USER=test_user
TEST_DB_PASSWORD=secure_test_password
TEST_DB_NAME=eterrain_test
```

---

## Contract 6: Test Output Formats

### Standard Output (Console)

**Format**: Human-readable test results with pass/fail status

**Example**:
```
=== RUN   TestValidateAPIKey
--- PASS: TestValidateAPIKey (0.00s)
PASS
ok      ./tests/unit-tests    0.234s
```

---

### JSON Output (CI/CD)

**Format**: Machine-parseable JSON for automated processing

**Command**: `go test ./tests/... -json`

**Example**:
```json
{
  "Time": "2025-11-27T10:00:00Z",
  "Action": "pass",
  "Package": "./tests/unit-tests",
  "Test": "TestValidateAPIKey",
  "Elapsed": 0.234
}
```

---

### Coverage Output

**Format**: Coverage data in `coverage.out` file

**Command**: `go test ./tests/... -coverprofile=coverage.out`

**Example** (`coverage.out`):
```
mode: set
github.com/yourrepo/internal/auth/validate.go:10.45,12.16 2 1
github.com/yourrepo/internal/auth/validate.go:12.16,14.3 1 1
```

---

## Summary

The testing infrastructure provides six primary contracts:

1. **Test Execution Commands**: Run tests by category or all together
2. **Code Coverage Commands**: Generate and view coverage reports
3. **Test Database Setup**: Initialize and teardown test database
4. **Test Utilities Interface**: Public functions in `testutil` package
5. **Test Environment**: Environment variable configuration
6. **Test Output Formats**: Console, JSON, and coverage formats

All contracts follow Go testing conventions and constitution requirements for simplicity (KISS) and consistency (DRY).
