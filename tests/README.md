# Test Infrastructure

This directory contains automated tests for the eterrain-tf-upload service, organized by test type following test-first development principles.

## Directory Structure

```
tests/
├── unit-tests/             # Unit tests for individual functions
├── integration-tests/      # Integration tests for database operations
├── edge-case-tests/        # Boundary conditions and error scenarios
├── performance-tests/      # Performance and load tests
├── testutil/               # Shared test utilities and fixtures
└── scripts/                # Database setup scripts
```

## Test Naming Convention

Test files follow the naming pattern: `XXX_feature_name_test.go`

- `XXX`: Matches the specification number (e.g., `002` for feature 002-automated-testing)
- Feature name describes what's being tested
- Must end with `_test.go` (Go requirement)

## Quick Start

### 1. Run Unit Tests

Unit tests require no external dependencies:

```bash
# Using Makefile
make -f Makefile.testing test-unit

# Or directly with go test
go test ./tests/unit-tests/... -v -cover
```

### 2. Setup Integration Tests

Integration tests require a MySQL test database:

**Step 1: Create Test Database**

```bash
# Run the SQL setup script as MySQL root user
mysql -u root -p < tests/scripts/setup-test-db.sql
```

This creates:
- Database: `eterrain_test`
- User: `test_user` with password `test_password`
- Grants necessary permissions

**Step 2: Configure Environment Variables**

Copy `.env.test` and update with your MySQL credentials:

```bash
cp .env.test .env.test.local
# Edit .env.test.local with your actual password
export $(cat .env.test.local | xargs)
```

**Step 3: Run Integration Tests**

```bash
# Using Makefile
make -f Makefile.testing test-integration

# Or directly with go test
go test ./tests/integration-tests/... -v -cover
```

### 3. Run All Tests

```bash
make -f Makefile.testing test-all
```

## Test-First Development Workflow

Follow the TDD red-green-refactor cycle:

### 1. RED: Write Failing Test

```go
// tests/unit-tests/003_new_feature_test.go
func TestNewFeature(t *testing.T) {
    result := mypackage.NewFunction("input")
    assert.Equal(t, "expected", result)
}
```

Run test (it should fail):
```bash
go test ./tests/unit-tests/... -v  # RED: Test fails
```

### 2. GREEN: Implement Minimum Code

```go
// internal/mypackage/myfeature.go
func NewFunction(input string) string {
    return "expected"
}
```

Run test (it should pass):
```bash
go test ./tests/unit-tests/... -v  # GREEN: Test passes
```

### 3. REFACTOR: Improve Code

Refactor while keeping tests green:
```bash
go test ./tests/unit-tests/... -v  # Still GREEN
```

### 4. CHECK COVERAGE

Ensure >= 80% coverage for critical paths:
```bash
make -f Makefile.testing coverage
```

## Available Test Commands

| Command | Description |
|---------|-------------|
| `make -f Makefile.testing test-unit` | Run unit tests only |
| `make -f Makefile.testing test-integration` | Run integration tests (requires DB) |
| `make -f Makefile.testing test-edge` | Run edge case tests |
| `make -f Makefile.testing test-performance` | Run performance tests |
| `make -f Makefile.testing test-all` | Run all test categories |
| `make -f Makefile.testing coverage` | Generate coverage report |

## Test Utilities (testutil package)

The `testutil` package provides shared fixtures and helpers:

### Fixtures (testutil/fixtures.go)

```go
import "github.com/eterrain/tf-backend-service/tests/testutil"

func TestExample(t *testing.T) {
    // Use valid test data
    orgID := testutil.ValidOrgID
    apiKey := testutil.ValidAPIKey
    request := testutil.SampleUploadRequest()
}
```

### Database Helpers (testutil/database.go)

```go
func TestDatabaseOperation(t *testing.T) {
    // Setup test database connection
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    // Use transaction-based cleanup
    tx, _ := db.Begin()
    defer tx.Rollback()

    // Perform database operations
    // Changes will rollback automatically
}
```

## Troubleshooting

### "connection refused" errors

Integration tests require MySQL to be running:

```bash
# Check if MySQL is running
sudo systemctl status mysql

# Start MySQL if needed
sudo systemctl start mysql
```

### "Access denied for user 'test_user'"

Run the database setup script:

```bash
mysql -u root -p < tests/scripts/setup-test-db.sql
```

### Coverage is 0%

Coverage measures code in `internal/` and `cmd/`, not `tests/`. Write tests that exercise application code, not just test utilities.

## Best Practices

1. ✅ **Write tests BEFORE implementation** (TDD)
2. ✅ **Use table-driven tests** for multiple test cases
3. ✅ **Mark independent tests with `t.Parallel()`**
4. ✅ **Use transactions for integration test cleanup**
5. ✅ **Aim for 80%+ coverage on critical paths**
6. ✅ **100% coverage on security code** (auth, validation, rate limiting)
7. ✅ **Run full test suite before commits** (`make test-all`)

## Coverage Targets

- **Security code**: 100% (authentication, validation, rate limiting)
- **Critical paths**: 80% minimum
- **Test suite execution**: < 30 seconds for typical feature
- **Unit tests**: < 100ms per test
- **Integration tests**: < 5 seconds per test

## Contributing

When adding new features:

1. Create test file: `tests/unit-tests/XXX_feature_name_test.go`
2. Write failing tests (RED phase)
3. Implement minimum code to pass tests (GREEN phase)
4. Refactor while keeping tests green
5. Verify coverage >= 80%
6. Run `make -f Makefile.testing test-all` before committing
