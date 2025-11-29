// tests/integration-tests/002_database_setup_test.go
package integration_tests_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/eterrain/tf-backend-service/tests/testutil"
)

// TestDatabaseConnection validates test database connectivity
// T014: Write failing integration test for database connectivity
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
// T015: Write failing integration test for transaction-based cleanup
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
