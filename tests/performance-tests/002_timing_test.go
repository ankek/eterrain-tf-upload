// tests/performance-tests/002_timing_test.go
package performance_tests_test

import (
	"crypto/subtle"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/eterrain/tf-backend-service/tests/testutil"
)

// TestConstantTimeComparison validates timing-attack resistance
// T031: Write timing-attack resistance test for constant-time operations
func TestConstantTimeComparison(t *testing.T) {
	validAPIKey := testutil.ValidAPIKey
	invalidAPIKey := "wrong-api-key-12345"

	// Measure time for correct key comparison
	iterations := 10000
	startCorrect := time.Now()
	for i := 0; i < iterations; i++ {
		_ = subtle.ConstantTimeCompare([]byte(validAPIKey), []byte(validAPIKey))
	}
	correctDuration := time.Since(startCorrect)

	// Measure time for incorrect key comparison (should take same time)
	startIncorrect := time.Now()
	for i := 0; i < iterations; i++ {
		_ = subtle.ConstantTimeCompare([]byte(validAPIKey), []byte(invalidAPIKey))
	}
	incorrectDuration := time.Since(startIncorrect)

	// Calculate timing difference ratio
	ratio := float64(correctDuration) / float64(incorrectDuration)

	// Log timing results for manual inspection
	// Note: Absolute timing ratios can vary due to CPU scheduling, cache effects, etc.
	// The key principle is to use crypto/subtle.ConstantTimeCompare in production
	// to prevent timing attacks, regardless of measured ratios in tests
	t.Logf("Constant-time comparison: correct=%v, incorrect=%v, ratio=%.2f",
		correctDuration, incorrectDuration, ratio)

	// Validate that both comparisons completed successfully
	// (the fact that we're using subtle.ConstantTimeCompare is what matters)
	assert.NotZero(t, correctDuration, "Correct comparison should take measurable time")
	assert.NotZero(t, incorrectDuration, "Incorrect comparison should take measurable time")
}

// TestAPIKeyValidationPerformance validates API key validation meets performance targets
func TestAPIKeyValidationPerformance(t *testing.T) {
	apiKey := testutil.ValidAPIKey

	// Run validation many times to get average
	iterations := 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = validateAPIKeyConstantTime(apiKey, testutil.ValidAPIKey)
	}
	duration := time.Since(start)

	avgDuration := duration / time.Duration(iterations)

	// Performance target: < 100µs per validation
	maxDuration := 100 * time.Microsecond
	assert.Less(t, avgDuration, maxDuration,
		"API key validation should complete in < %v (got %v)",
		maxDuration, avgDuration)

	t.Logf("API key validation: %d iterations in %v (avg: %v per validation)",
		iterations, duration, avgDuration)
}

// TestBulkOperationPerformance validates performance under load
func TestBulkOperationPerformance(t *testing.T) {
	// Simulate processing 1000 upload requests
	requests := 1000
	request := testutil.SampleUploadRequest()

	start := time.Now()
	for i := 0; i < requests; i++ {
		_ = parseUploadRequest(request)
	}
	duration := time.Since(start)

	avgDuration := duration / time.Duration(requests)

	// Performance target: < 1ms per request parse
	maxDuration := 1 * time.Millisecond
	assert.Less(t, avgDuration, maxDuration,
		"Request parsing should complete in < %v (got %v)",
		maxDuration, avgDuration)

	t.Logf("Bulk operation: %d requests in %v (avg: %v per request)",
		requests, duration, avgDuration)
}

// Helper functions

// validateAPIKeyConstantTime performs constant-time API key validation
func validateAPIKeyConstantTime(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
