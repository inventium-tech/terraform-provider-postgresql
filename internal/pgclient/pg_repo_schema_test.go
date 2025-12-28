package pgclient

import (
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/stretchr/testify/assert"
)

// Integration tests for SchemaRepo.
func TestSchemaRepo_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var (
		err    error
		exists bool
		model  *SchemaModel
	)

	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	assert.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
	})

	ctx := t.Context()
	repo := NewSchemaRepo()

	// Test schema parameters
	testSchemaParams := SchemaCreateParams{
		Name:        "test_schema",
		Owner:       "postgres",
		IfNotExists: false,
	}

	// Test Create
	t.Run("Create schema", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testSchemaParams)
		assert.NoError(t, err)
	})

	// Test Exists
	t.Run("Schema exists", func(t *testing.T) {
		exists, err = repo.Exists(ctx, pgConn, "test_schema")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test GetOne
	t.Run("Get schema", func(t *testing.T) {
		model, err = repo.GetOne(ctx, pgConn, "test_schema")
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "test_schema", model.Name.String)
		assert.Equal(t, "postgres", model.Owner.String)
	})

	// Test Update
	t.Run("Update schema owner", func(t *testing.T) {
		// First create a new user to transfer ownership
		_, err := pgConn.Exec(ctx, "CREATE ROLE test_schema_owner LOGIN PASSWORD 'password'")
		assert.NoError(t, err)

		newOwner := "test_schema_owner"
		updateParams := SchemaUpdateParams{
			Owner: &newOwner,
		}

		err = repo.Update(ctx, pgConn, "test_schema", updateParams)
		assert.NoError(t, err)

		// Verify update
		model, err = repo.GetOne(ctx, pgConn, "test_schema")
		assert.NoError(t, err)
		assert.Equal(t, "test_schema_owner", model.Owner.String)

		// Cleanup: transfer back to postgres
		pgOwner := "postgres"
		err = repo.Update(ctx, pgConn, "test_schema", SchemaUpdateParams{Owner: &pgOwner})
		assert.NoError(t, err)

		// Drop the test user
		_, err = pgConn.Exec(ctx, "DROP ROLE test_schema_owner")
		assert.NoError(t, err)
	})

	// Test Drop
	t.Run("Drop schema", func(t *testing.T) {
		err = repo.Drop(ctx, pgConn, "test_schema", false)
		assert.NoError(t, err)

		// Verify schema no longer exists
		exists, err = repo.Exists(ctx, pgConn, "test_schema")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test Drop with CASCADE
	t.Run("Create and drop schema with cascade", func(t *testing.T) {
		testSchemaParams.Name = "test_schema_cascade"
		err = repo.Create(ctx, pgConn, testSchemaParams)
		assert.NoError(t, err)

		// Create a table in the schema
		_, err = pgConn.Exec(ctx, "CREATE TABLE test_schema_cascade.test_table (id INT)")
		assert.NoError(t, err)

		// Drop with CASCADE
		err = repo.Drop(ctx, pgConn, "test_schema_cascade", true)
		assert.NoError(t, err)

		// Verify schema no longer exists
		exists, err = repo.Exists(ctx, pgConn, "test_schema_cascade")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test public schema cannot be dropped
	t.Run("Cannot drop public schema", func(t *testing.T) {
		err = repo.Drop(ctx, pgConn, "public", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot drop the public schema")
	})
}

func TestSchemaRepo_List(t *testing.T) {
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

	t.Cleanup(func() {
	})

	ctx := t.Context()
	repo := NewSchemaRepo()

	// Create some test schemas
	testSchemas := []string{"list_test_1", "list_test_2", "other_schema"}
	for _, name := range testSchemas {
		err := repo.Create(ctx, pgConn, SchemaCreateParams{Name: name, Owner: "postgres"})
		assert.NoError(t, err)
	}

	t.Cleanup(func() {
		for _, name := range testSchemas {
			_ = repo.Drop(ctx, pgConn, name, false)
		}
	})

	// Test list all schemas (excluding system)
	t.Run("List all user schemas", func(t *testing.T) {
		schemas, err := repo.List(ctx, pgConn, SchemaListParams{
			IncludeSystemSchemas: false,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(schemas), 3) // At least our 3 test schemas

		// Verify test schemas are included
		schemaNames := make([]string, len(schemas))
		for i, s := range schemas {
			schemaNames[i] = s.Name.String
		}
		assert.Contains(t, schemaNames, "list_test_1")
		assert.Contains(t, schemaNames, "list_test_2")
		assert.Contains(t, schemaNames, "other_schema")
	})

	// Test LIKE ANY filter
	t.Run("List schemas with LIKE ANY", func(t *testing.T) {
		schemas, err := repo.List(ctx, pgConn, SchemaListParams{
			IncludeSystemSchemas: false,
			LikeAnyPatterns:      []string{"list_test_%"},
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(schemas), 2)

		for _, s := range schemas {
			assert.Contains(t, s.Name.String, "list_test_")
		}
	})

	// Test regex filter
	t.Run("List schemas with regex", func(t *testing.T) {
		schemas, err := repo.List(ctx, pgConn, SchemaListParams{
			IncludeSystemSchemas: false,
			RegexPattern:         "^list_test_[12]$",
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(schemas))
	})
}
