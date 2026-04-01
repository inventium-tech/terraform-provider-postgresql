package pgclient

import (
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/stretchr/testify/assert"
)

// TestBuildDesiredMembershipMap is a pure unit test — no database required.
// It specifically covers the regression where in_role memberships were being
// accidentally revoked during an Update when role or admin changed, because
// they were not included in the desired membership map passed to syncMembership.
func TestBuildDesiredMembershipMap(t *testing.T) {
	tests := []struct {
		name       string
		roles      []string
		adminRoles []string
		want       map[string]bool
		wantErr    bool
	}{
		{
			name:       "nil inputs returns empty map",
			roles:      nil,
			adminRoles: nil,
			want:       map[string]bool{},
		},
		{
			name:  "roles only",
			roles: []string{"parent_role", "member_role"},
			want:  map[string]bool{"parent_role": false, "member_role": false},
		},
		{
			name:       "admin roles only",
			adminRoles: []string{"admin_role"},
			want:       map[string]bool{"admin_role": true},
		},
		{
			name:       "roles and admin roles combined",
			roles:      []string{"member_role"},
			adminRoles: []string{"admin_role"},
			want:       map[string]bool{"member_role": false, "admin_role": true},
		},
		{
			// Regression: when the Terraform Update handler includes in_role entries
			// alongside plan roles, buildDesiredMembershipMap must retain them so that
			// syncMembership does NOT revoke them.
			name:       "in_role entries preserved when merged with plan roles",
			roles:      []string{"plan_role", "in_role_parent"},
			adminRoles: []string{"admin_role"},
			want: map[string]bool{
				"plan_role":      false,
				"in_role_parent": false, // must survive — was previously wiped on update
				"admin_role":     true,
			},
		},
		{
			name:  "empty string roles are skipped",
			roles: []string{"", "valid_role"},
			want:  map[string]bool{"valid_role": false},
		},
		{
			name:       "admin wins over role when same name in both",
			roles:      []string{"shared_role"},
			adminRoles: []string{"shared_role"},
			// adminRoles is processed last, so it overwrites the non-admin entry.
			want: map[string]bool{"shared_role": true},
		},
		{
			name:    "invalid role name returns error",
			roles:   []string{"invalid name with spaces"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDesiredMembershipMap(tt.roles, tt.adminRoles)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

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

	ctx := t.Context()
	repo := NewRoleRepo()

	baseRoles := []string{"parent_role", "admin_role", "member_role"}

	for _, roleName := range baseRoles {
		err = repo.Create(ctx, pgConn, RoleCreateParams{Name: roleName})
		assert.NoError(t, err)
	}

	t.Cleanup(func() {
		cleanupRoles := append(baseRoles, "test_role", "updated_role")
		for _, roleName := range cleanupRoles {
			_ = repo.Drop(ctx, pgConn, roleName)
		}
	})

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
		InRole:          []string{"parent_role"},
		Roles:           []string{"member_role"},
		AdminRoles:      []string{"admin_role"},
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
		assert.ElementsMatch(t, []string{"parent_role", "member_role"}, model.Roles)
		assert.ElementsMatch(t, []string{"admin_role"}, model.AdminRoles)
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
		updateBoolParams := RoleUpdateParams{
			Superuser: new(true),
			CreateDB:  new(true),
		}

		err = repo.Update(ctx, pgConn, newName, updateBoolParams)
		assert.NoError(t, err)

		// Verify boolean options were updated
		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.True(t, model.Superuser.Bool)
		assert.True(t, model.CreateDB.Bool)

		// Update connection limit
		updateConnLimitParams := RoleUpdateParams{
			ConnectionLimit: new(int32(10)),
		}

		err = repo.Update(ctx, pgConn, newName, updateConnLimitParams)
		assert.NoError(t, err)

		// Verify connection limit was updated
		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.Equal(t, int32(10), model.ConnectionLimit.Int32)
	})

	// Test membership updates
	t.Run("Update memberships", func(t *testing.T) {
		newName := "updated_role"
		updateMembershipParams := RoleUpdateParams{
			Roles:      []string{"parent_role"},
			AdminRoles: []string{"member_role"},
		}

		err = repo.Update(ctx, pgConn, newName, updateMembershipParams)
		assert.NoError(t, err)

		model, err = repo.GetOne(ctx, pgConn, newName)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"parent_role"}, model.Roles)
		assert.ElementsMatch(t, []string{"member_role"}, model.AdminRoles)
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
		roleWithPasswordParams := RoleCreateParams{
			Name:     "role_with_password",
			Password: new("test_password"),
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
		result = r.useBooleanOption("SUPERUSER", new(true))
		assert.Equal(t, "SUPERUSER", result)

		// Test with false value
		result = r.useBooleanOption("SUPERUSER", new(false))
		assert.Equal(t, "NOSUPERUSER", result)
	})
}
