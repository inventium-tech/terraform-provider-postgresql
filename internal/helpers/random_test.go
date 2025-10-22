package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"unicode"
)

func TestRandomString(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		wantLen  int
		wantType string
	}{
		{
			name:     "positive length",
			length:   10,
			wantLen:  10,
			wantType: "alphanumeric",
		},
		{
			name:     "zero length",
			length:   0,
			wantLen:  0,
			wantType: "empty",
		},
		{
			name:     "negative length",
			length:   -5,
			wantLen:  0,
			wantType: "empty",
		},
		{
			name:     "large length",
			length:   100,
			wantLen:  100,
			wantType: "alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandomString(tt.length)

			// Check length
			assert.Equal(t, tt.wantLen, len(result), "RandomString() length = %v, want %v", len(result), tt.wantLen)

			// Check content type
			switch tt.wantType {
			case "empty":
				assert.Empty(t, result, "RandomString() should return empty string")
			case "alphanumeric":
				for _, char := range result {
					isAlphanumeric := unicode.IsLetter(char) || unicode.IsDigit(char)
					assert.True(t, isAlphanumeric, "RandomString() contains non-alphanumeric character: %c", char)
				}
			}
		})
	}
}

func TestRandomString_Uniqueness(t *testing.T) {
	// Generate multiple strings of the same length and verify they're different
	length := 20
	iterations := 10
	results := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		results[i] = RandomString(length)
		assert.Equal(t, length, len(results[i]), "RandomString() length = %v, want %v", len(results[i]), length)
	}

	// Check that all generated strings are unique
	for i := 0; i < iterations; i++ {
		for j := i + 1; j < iterations; j++ {
			assert.NotEqual(t, results[i], results[j],
				"RandomString() generated duplicate strings: %s and %s", results[i], results[j])
		}
	}
}
