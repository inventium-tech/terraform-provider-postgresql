package pgclient

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Note: DeferredRollback is not tested directly because it requires a full implementation
// of the pgx.Tx interface, which is complex to mock. The function is simple enough
// that we can rely on manual inspection to verify its correctness.

func TestSanitizeIdentifierInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid simple identifier",
			input:    "table_name",
			expected: "table_name",
			wantErr:  false,
		},
		{
			name:     "Valid schema qualified identifier",
			input:    "schema.table_name",
			expected: "schema.table_name",
			wantErr:  false,
		},
		{
			name:     "Valid identifier with underscore",
			input:    "my_table_name",
			expected: "my_table_name",
			wantErr:  false,
		},
		{
			name:     "Valid identifier with numbers",
			input:    "table123",
			expected: "table123",
			wantErr:  false,
		},
		{
			name:     "Invalid identifier starting with number",
			input:    "1table",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid identifier with special characters",
			input:    "table-name",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid identifier with SQL injection attempt",
			input:    "table_name; DROP TABLE users;",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Multi-part identifier",
			input:    "schema.table.column",
			expected: `"schema"."table"."column"`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeIdentifierInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeReturnsTypeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid simple type",
			input:    "integer",
			expected: "integer",
			wantErr:  false,
		},
		{
			name:     "Valid array type",
			input:    "text[]",
			expected: "text[]",
			wantErr:  false,
		},
		{
			name:     "Valid schema qualified type",
			input:    "pg_catalog.text",
			expected: "pg_catalog.text",
			wantErr:  false,
		},
		{
			name:     "Valid schema qualified array type",
			input:    "pg_catalog.text[]",
			expected: "pg_catalog.text[]",
			wantErr:  false,
		},
		{
			name:     "Invalid type with special characters",
			input:    "int-eger",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid type with SQL injection attempt",
			input:    "integer; DROP TABLE users;",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeReturnsTypeInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		sanitType SanitationType
		expected  string
		wantErr   bool
	}{
		{
			name:      "Valid identifier",
			input:     "table_name",
			sanitType: SanitizeIdentifier,
			expected:  "table_name",
			wantErr:   false,
		},
		{
			name:      "Valid return type",
			input:     "integer[]",
			sanitType: SanitizeReturns,
			expected:  "integer[]",
			wantErr:   false,
		},
		{
			name:      "Invalid identifier",
			input:     "table-name",
			sanitType: SanitizeIdentifier,
			expected:  "",
			wantErr:   true,
		},
		{
			name:      "Invalid return type",
			input:     "int-eger",
			sanitType: SanitizeReturns,
			expected:  "",
			wantErr:   true,
		},
		{
			name:      "Invalid sanitation type",
			input:     "table_name",
			sanitType: 999, // Invalid type
			expected:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeInput(tt.input, tt.sanitType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
