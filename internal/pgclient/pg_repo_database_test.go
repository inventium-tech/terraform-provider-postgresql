package pgclient

import (
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/stretchr/testify/assert"
)

// Integration tests for DatabaseRepo.
func TestDatabaseRepo_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var (
		err    error
		exists bool
		model  *DatabaseModel
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
	repo := NewDatabaseRepo()

	// Test database parameters
	allowConn := true
	isTemplate := false
	testDBParams := DatabaseCreateParams{
		Name:             "test_db",
		Owner:            "postgres",
		Encoding:         "UTF8",
		Template:         "template1",
		ConnectionLimit:  10,
		AllowConnections: &allowConn,
		IsTemplate:       &isTemplate,
		Comment:          "Test database",
	}

	// Test Create
	t.Run("Create database", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testDBParams)
		assert.NoError(t, err)
	})

	// Test Exists
	t.Run("Database exists", func(t *testing.T) {
		exists, err = repo.Exists(ctx, pgConn, "test_db")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test GetOne
	t.Run("Get database", func(t *testing.T) {
		model, err = repo.GetOne(ctx, pgConn, "test_db")
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "test_db", model.Name.String)
		assert.Equal(t, "postgres", model.Owner.String)
		assert.Equal(t, "UTF8", model.Encoding.String)
		assert.True(t, model.AllowConnections.Bool)
		assert.False(t, model.IsTemplate.Bool)
		assert.Equal(t, int32(10), model.ConnectionLimit.Int32)
		assert.Equal(t, "Test database", model.Comment.String)
	})

	// Test Update
	t.Run("Update database", func(t *testing.T) {
		newLimit := int32(20)
		newComment := "Updated test database"
		updateParams := DatabaseUpdateParams{
			ConnectionLimit: &newLimit,
			Comment:         &newComment,
		}

		err = repo.Update(ctx, pgConn, "test_db", updateParams)
		assert.NoError(t, err)

		// Verify update
		model, err = repo.GetOne(ctx, pgConn, "test_db")
		assert.NoError(t, err)
		assert.Equal(t, int32(20), model.ConnectionLimit.Int32)
		assert.Equal(t, "Updated test database", model.Comment.String)
	})

	// Test TerminateConnections
	t.Run("Terminate connections", func(t *testing.T) {
		err = repo.TerminateConnections(ctx, pgConn, "test_db")
		assert.NoError(t, err)
	})

	// Test Drop
	t.Run("Drop database", func(t *testing.T) {
		err = repo.Drop(ctx, pgConn, "test_db", false)
		assert.NoError(t, err)

		// Verify database no longer exists
		exists, err = repo.Exists(ctx, pgConn, "test_db")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test Drop with force_drop
	t.Run("Create and force drop database", func(t *testing.T) {
		// Create a new database
		testDBParams.Name = "test_db_force"
		err = repo.Create(ctx, pgConn, testDBParams)
		assert.NoError(t, err)

		// Force drop the database
		err = repo.Drop(ctx, pgConn, "test_db_force", true)
		assert.NoError(t, err)

		// Verify database no longer exists
		exists, err = repo.Exists(ctx, pgConn, "test_db_force")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestDatabaseRepo_Exists_NotFound(t *testing.T) {
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
	repo := NewDatabaseRepo()

	// Test non-existent database
	exists, err := repo.Exists(ctx, pgConn, "nonexistent_db")
	assert.NoError(t, err)
	assert.False(t, exists)
}
