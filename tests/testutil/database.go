// Database setup and teardown helpers for integration tests.
//
// This file provides functions to create and manage test database connections
// with automatic cleanup. Integration tests should use SetupTestDB/TeardownTestDB
// to ensure proper resource management and test isolation.
//
// Configuration is managed through environment variables (see .env.test):
//   - TEST_DB_HOST: Database host (default: localhost)
//   - TEST_DB_PORT: Database port (default: 3306)
//   - TEST_DB_USER: Database user (default: test_user)
//   - TEST_DB_PASSWORD: Database password (default: test_password)
//   - TEST_DB_NAME: Database name (default: eterrain_test)
//
// Usage:
//
//	func TestDatabaseOperation(t *testing.T) {
//	    db := testutil.SetupTestDB(t)
//	    defer testutil.TeardownTestDB(t, db)
//
//	    // Use transaction-based cleanup
//	    tx, _ := db.Begin()
//	    defer tx.Rollback()
//
//	    // Perform database operations within transaction
//	    // Changes will rollback automatically
//	}
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// GetTestDSN returns the Data Source Name for test database.
// Reads configuration from environment variables with sensible defaults.
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
