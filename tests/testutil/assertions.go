// Custom assertion helpers for edge case testing.
//
// This file provides specialized assertion functions for common edge case scenarios
// such as boundary value testing, null handling, and error condition validation.
//
// Usage:
//
//	import "github.com/eterrain/tf-backend-service/tests/testutil"
//
//	func TestEdgeCase(t *testing.T) {
//	    testutil.AssertValidUUID(t, orgID, "Organization ID")
//	    testutil.AssertNonEmpty(t, apiKey, "API Key")
//	    testutil.AssertMaxLength(t, name, 255, "Resource Name")
//	}
package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertValidUUID verifies that a string is a valid UUID format (36 characters)
func AssertValidUUID(t *testing.T, value string, fieldName string) bool {
	t.Helper()
	if !assert.Len(t, value, 36, "%s should be UUID format (36 chars)", fieldName) {
		return false
	}
	// Could add additional UUID format validation here
	return true
}

// AssertNonEmpty verifies that a string is not empty
func AssertNonEmpty(t *testing.T, value string, fieldName string) bool {
	t.Helper()
	return assert.NotEmpty(t, value, "%s should not be empty", fieldName)
}

// AssertMaxLength verifies that a string does not exceed maximum length
func AssertMaxLength(t *testing.T, value string, maxLen int, fieldName string) bool {
	t.Helper()
	return assert.LessOrEqual(t, len(value), maxLen, "%s should not exceed %d characters", fieldName, maxLen)
}

// AssertMinLength verifies that a string meets minimum length requirement
func AssertMinLength(t *testing.T, value string, minLen int, fieldName string) bool {
	t.Helper()
	return assert.GreaterOrEqual(t, len(value), minLen, "%s should be at least %d characters", fieldName, minLen)
}

// AssertNotNil verifies that a map or pointer is not nil
func AssertNotNil(t *testing.T, value interface{}, fieldName string) bool {
	t.Helper()
	return assert.NotNil(t, value, "%s should not be nil", fieldName)
}

// AssertMapHasKey verifies that a map contains a required key
func AssertMapHasKey(t *testing.T, m map[string]interface{}, key string) bool {
	t.Helper()
	_, exists := m[key]
	return assert.True(t, exists, "Map should contain key '%s'", key)
}

// AssertMapKeyNotNil verifies that a map key exists and its value is not nil
func AssertMapKeyNotNil(t *testing.T, m map[string]interface{}, key string) bool {
	t.Helper()
	value, exists := m[key]
	if !assert.True(t, exists, "Map should contain key '%s'", key) {
		return false
	}
	return assert.NotNil(t, value, "Map key '%s' should not have nil value", key)
}

// AssertInRange verifies that an integer value is within specified range
func AssertInRange(t *testing.T, value int, min int, max int, fieldName string) bool {
	t.Helper()
	if !assert.GreaterOrEqual(t, value, min, "%s should be >= %d", fieldName, min) {
		return false
	}
	return assert.LessOrEqual(t, value, max, "%s should be <= %d", fieldName, max)
}

// AssertValidEnum verifies that a string value is one of the allowed enum values
func AssertValidEnum(t *testing.T, value string, validValues []string, fieldName string) bool {
	t.Helper()
	for _, valid := range validValues {
		if value == valid {
			return true
		}
	}
	assert.Fail(t, "%s should be one of %v, got '%s'", fieldName, validValues, value)
	return false
}
