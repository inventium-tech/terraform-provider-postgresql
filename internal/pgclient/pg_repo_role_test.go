package pgclient

import (
	"github.com/stretchr/testify/assert"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

// Integration tests for RoleRepo.
func TestRoleRepo_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var (
		err    error
		exists bool
		model  *RoleModel
	)

	runOpts := test.PostgresContainerRunOptions{
		Database: "test_role_db",
		Username: "test_role_user",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	assert.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		if err = pgConn.Close(t.Context()); err != nil {
			t.Logf("failed to close connection: %s", err)
		}
	})

	ctx := t.Context()
	repo := NewRoleRepo()

	// Test role parameters
	testRoleParams := RoleCreateParams{
		Name:            "test_role",
		Superuser:       false,
		Inherit:         true,
		CreateRole:      false,
		CreateDB:        false,
		Login:           true,
		Replication:     false,
		BypassRLS:       false,
		ConnectionLimit: 5,
		ValidUntil:      "infinity",
		Comment:         "Test role",
	}

	// Test Create
	t.Run("Create role", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testRoleParams)
		assert.NoError(t, err)
	})

	// Test Exists
	t.Run("Role exists", func(t *testing.T) {
		exists, err = repo.Exists(ctx, pgConn, "test_role")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test GetOne
	t.Run("Get role", func(t *testing.T) {
		model, err = repo.GetOne(ctx, pgConn, "test_role")
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "test_role", model.Name.String)
		assert.False(t, model.Superuser.Bool)
		assert.True(t, model.Inherit.Bool)
		assert.False(t, model.CreateRole.Bool)
		assert.False(t, model.CreateDB.Bool)
		assert.True(t, model.Login.Bool)
		assert.False(t, model.Replication.Bool)
		assert.False(t, model.BypassRLS.Bool)
		assert.Equal(t, int32(5), model.ConnectionLimit.Int32)
		assert.Equal(t, "Test role", model.Comment.String)
	})

	// Test Update
	t.Run("Update role", func(t *testing.T) {
		// Update the role name
		newName := "updated_role"
		updateParams := RoleUpdateParams{
			Name: &newName,
		}

		err = repo.Update(ctx, pgConn, "test_role", updateParams)
		assert.NoError(t, err)

		// Verify original role no longer exists
		exists, err = repo.Exists(ctx, pgConn, "test_role")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Verify role exists with new name
		exists, err = repo.Exists(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Update the role comment
		newComment := "Updated comment"
		updateCommentParams := RoleUpdateParams{
			Comment: &newComment,
		}

		err = repo.Update(ctx, pgConn, newName, updateCommentParams)
		assert.NoError(t, err)

		// Verify comment was updated
		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.Equal(t, newName, model.Name.String)
		assert.Equal(t, newComment, model.Comment.String)

		// Update boolean options
		superuser := true
		createDB := true
		updateBoolParams := RoleUpdateParams{
			Superuser: &superuser,
			CreateDB:  &createDB,
		}

		err = repo.Update(ctx, pgConn, newName, updateBoolParams)
		assert.NoError(t, err)

		// Verify boolean options were updated
		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.True(t, model.Superuser.Bool)
		assert.True(t, model.CreateDB.Bool)

		// Update connection limit
		connectionLimit := int32(10)
		updateConnLimitParams := RoleUpdateParams{
			ConnectionLimit: &connectionLimit,
		}

		err = repo.Update(ctx, pgConn, newName, updateConnLimitParams)
		assert.NoError(t, err)

		// Verify connection limit was updated
		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.Equal(t, int32(10), model.ConnectionLimit.Int32)
	})

	// Test Drop
	t.Run("Drop role", func(t *testing.T) {
		newName := "updated_role"
		err = repo.Drop(ctx, pgConn, newName)
		assert.NoError(t, err)

		// Verify role no longer exists
		exists, err = repo.Exists(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test creating role with password
	t.Run("Create role with password", func(t *testing.T) {
		password := "test_password"
		roleWithPasswordParams := RoleCreateParams{
			Name:     "role_with_password",
			Password: &password,
			Login:    true,
		}

		err = repo.Create(ctx, pgConn, roleWithPasswordParams)
		assert.NoError(t, err)

		// Verify role exists
		exists, err = repo.Exists(ctx, pgConn, "role_with_password")
		assert.NoError(t, err)
		assert.True(t, exists)

		// Clean up
		err = repo.Drop(ctx, pgConn, "role_with_password")
		assert.NoError(t, err)
	})

	// Test useBooleanOption helper method
	t.Run("Test useBooleanOption helper", func(t *testing.T) {
		r := &roleRepo{}

		// Test with nil value
		result := r.useBooleanOption("SUPERUSER", nil)
		assert.Equal(t, "", result)

		// Test with true value
		trueVal := true
		result = r.useBooleanOption("SUPERUSER", &trueVal)
		assert.Equal(t, "SUPERUSER", result)

		// Test with false value
		falseVal := false
		result = r.useBooleanOption("SUPERUSER", &falseVal)
		assert.Equal(t, "NOSUPERUSER", result)
	})
}
