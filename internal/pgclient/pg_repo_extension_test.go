package pgclient

import (
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/stretchr/testify/assert"
)

// Integration tests for ExtensionRepo.
func TestExtensionRepo_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var (
		err    error
		exists bool
		model  *ExtensionModel
	)

	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	assert.NoError(t, err)

	ctx := t.Context()
	repo := NewExtensionRepo()

	// Test extension parameters
	testExtParams := ExtensionCreateParams{
		Name:    "uuid-ossp",
		Cascade: false,
	}

	// Test Create
	t.Run("Create extension", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testExtParams)
		assert.NoError(t, err)
	})

	// Test Exists
	t.Run("Extension exists", func(t *testing.T) {
		exists, err = repo.Exists(ctx, pgConn, "uuid-ossp")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test GetOne
	t.Run("Get extension", func(t *testing.T) {
		model, err = repo.GetOne(ctx, pgConn, "uuid-ossp")
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "uuid-ossp", model.Name.String)
		assert.NotEmpty(t, model.Version.String)
	})

	// Test GetAll
	t.Run("Get all extensions", func(t *testing.T) {
		extensions, err := repo.GetAll(ctx, pgConn)
		assert.NoError(t, err)
		assert.NotEmpty(t, extensions)

		// Find our test extension
		found := false
		for _, ext := range extensions {
			if ext.Name.String == "uuid-ossp" {
				found = true
				break
			}
		}
		assert.True(t, found, "uuid-ossp should be in the list")
	})

	// Test GetAvailable
	t.Run("Get available extension", func(t *testing.T) {
		available, err := repo.GetAvailable(ctx, pgConn, "uuid-ossp")
		assert.NoError(t, err)
		assert.NotNil(t, available)
		assert.Equal(t, "uuid-ossp", available.Name.String)
		assert.NotEmpty(t, available.DefaultVersion.String)
	})

	// Test Update - version (update to latest)
	t.Run("Update extension version", func(t *testing.T) {
		updateParams := ExtensionUpdateParams{
			Version: new(""),
		}

		err = repo.Update(ctx, pgConn, "uuid-ossp", updateParams)
		assert.NoError(t, err)
	})

	// Test Drop
	t.Run("Drop extension", func(t *testing.T) {
		err = repo.Drop(ctx, pgConn, "uuid-ossp", false)
		assert.NoError(t, err)

		// Verify extension no longer exists
		exists, err = repo.Exists(ctx, pgConn, "uuid-ossp")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test Create with schema
	t.Run("Create extension with schema", func(t *testing.T) {
		testExtParams.Name = "pg_trgm"
		testExtParams.Schema = "public"

		err = repo.Create(ctx, pgConn, testExtParams)
		assert.NoError(t, err)

		model, err = repo.GetOne(ctx, pgConn, "pg_trgm")
		assert.NoError(t, err)
		assert.Equal(t, "public", model.Schema.String)

		// Cleanup
		err = repo.Drop(ctx, pgConn, "pg_trgm", false)
		assert.NoError(t, err)
	})

	// Test Drop with CASCADE
	t.Run("Create and drop extension with cascade", func(t *testing.T) {
		testExtParams.Name = "citext"
		testExtParams.Cascade = false

		err = repo.Create(ctx, pgConn, testExtParams)
		assert.NoError(t, err)

		// Drop with cascade
		err = repo.Drop(ctx, pgConn, "citext", true)
		assert.NoError(t, err)

		// Verify extension no longer exists
		exists, err = repo.Exists(ctx, pgConn, "citext")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test IF NOT EXISTS (idempotency)
	t.Run("Create extension twice (idempotent)", func(t *testing.T) {
		testExtParams.Name = "hstore"

		err = repo.Create(ctx, pgConn, testExtParams)
		assert.NoError(t, err)

		// Create again - should not error
		err = repo.Create(ctx, pgConn, testExtParams)
		assert.NoError(t, err)

		// Cleanup
		err = repo.Drop(ctx, pgConn, "hstore", false)
		assert.NoError(t, err)
	})
}

func TestExtensionRepo_Exists_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	assert.NoError(t, err)

	ctx := t.Context()
	repo := NewExtensionRepo()

	exists, err := repo.Exists(ctx, pgConn, "nonexistent_extension")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestValidateExtensionVersion is a pure unit test — no database required.
func TestValidateExtensionVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid semver", "1.0.0", false},
		{"valid single number", "2", false},
		{"valid with dash", "1.2-beta", false},
		{"valid with underscore", "2.1_stable", false},
		{"valid alphanumeric", "1a2b3c", false},
		{"valid multi-dot", "2.1.4.1", false},
		{"starts with dot", ".1.0", true},
		{"starts with dash", "-1.0", true},
		{"contains single-quote", "1.0'--", true},
		{"contains semicolon", "1.0;DROP", true},
		{"contains space", "1.0 beta", true},
		{"contains parenthesis", "1.0()", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExtensionVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestQuoteStringLiteral is a pure unit test — no database required.
func TestQuoteStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple version", "1.0.0", "'1.0.0'"},
		{"no special chars", "abc", "'abc'"},
		{"single quote escaped", "it's", "'it''s'"},
		{"multiple quotes", "a'b'c", "'a''b''c'"},
		{"empty string", "", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteStringLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
