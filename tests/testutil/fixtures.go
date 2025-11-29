// Package testutil provides shared test utilities, fixtures, and helpers for the eterrain test suite.
//
// This package centralizes common test data, database setup/teardown functions, and reusable
// assertions to support test-first development across all test categories (unit, integration,
// edge-case, and performance tests).
//
// Usage:
//
//	import "github.com/eterrain/tf-backend-service/tests/testutil"
//
//	func TestMyFeature(t *testing.T) {
//	    // Use valid test fixtures
//	    orgID := testutil.ValidOrgID
//	    apiKey := testutil.ValidAPIKey
//
//	    // Use sample test data
//	    request := testutil.SampleUploadRequest()
//	    // ... test implementation ...
//	}
package testutil

// Common test data constants for authentication and API testing
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
