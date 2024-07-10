package uuid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		uuid          string
		expectedError bool
	}{
		{"123e4567-e89b-12d3-a456-426614174000", false}, // valid UUID
		{"123e4567-e89b-12d3-a456-42661417400", true},   // invalid UUID, too short
		{"123e4567-e89b-12d3-a456-4266141740000", true}, // invalid UUID, too long
		{"g23e4567-e89b-12d3-a456-426614174000", true},  // invalid UUID, contains non-hex character
		{"123e4567e89b12d3a456426614174000", true},      // invalid UUID, missing hyphens
		{"", true}, // invalid UUID, empty string
	}

	for _, tt := range tests {
		err := Validate(tt.uuid)
		if tt.expectedError {
			assert.Error(t, err, "expected error for UUID: %s", tt.uuid)
		} else {
			assert.NoError(t, err, "did not expect error for UUID: %s", tt.uuid)
		}
	}
}
