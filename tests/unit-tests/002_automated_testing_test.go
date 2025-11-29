// tests/unit-tests/002-automated-testing-test.go
package unit_tests_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/eterrain/tf-backend-service/tests/testutil"
)

// TestFixturesAvailable validates that test fixtures are accessible
// T008: Write failing unit test for test fixture availability
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
// T009: Write failing unit test demonstrating table-driven testing pattern
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
