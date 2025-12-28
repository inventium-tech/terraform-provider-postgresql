package pgclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"terraform-provider-postgresql/internal/test"
)

// Unit tests for EventTriggerRepo helper functions

func TestEventTriggerRepo_SetEventTriggerStatus(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    string
	}{
		{
			name:    "Enable event trigger",
			enabled: true,
			want:    "ENABLE",
		},
		{
			name:    "Disable event trigger",
			enabled: false,
			want:    "DISABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock DB that verifies the correct SQL is executed
			executed := false
			mockDB := &MockDB{
				ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
					expectedSQL := fmt.Sprintf(alterEventTriggerStatusQuery, "test_trigger", tt.want)
					assert.Equal(t, expectedSQL, sql)
					executed = true
					return pgconn.CommandTag{}, nil
				},
			}

			repo := &eventTriggerRepo{}
			err := repo.setEventTriggerStatus(t.Context(), mockDB, "test_trigger", tt.enabled)

			require.NoError(t, err)
			require.True(t, executed, "ExecFunc should have been called")
		})
	}
}

func TestEventTriggerRepo_FormatTagsAsString(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "Single tag",
			tags:     []string{"CREATE_TABLE"},
			expected: "'CREATE_TABLE'",
		},
		{
			name:     "Multiple tags",
			tags:     []string{"CREATE_TABLE", "DROP_TABLE", "ALTER_TABLE"},
			expected: "'CREATE_TABLE', 'DROP_TABLE', 'ALTER_TABLE'",
		},
		{
			name:     "Empty tags",
			tags:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &eventTriggerRepo{}
			result := repo.formatTagsAsString(tt.tags)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Integration tests for EventTriggerRepo

func TestEventTriggerRepo_Integration(t *testing.T) {
	var err error

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_event_trigger_db",
		Username: "test_event_trigger_user",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	require.NoError(t, err, "Failed to get database connection")

	// Clean-up database connection after test completion
	t.Cleanup(func() {
	})

	ctx := t.Context()
	repo := NewEventTriggerRepo()

	// Create a test function that will be used by the event trigger
	_, err = pgConn.Exec(ctx, `
		CREATE OR REPLACE FUNCTION test_event_trigger_func()
		RETURNS event_trigger
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'Event trigger fired: %', tg_event;
		END;
		$$;
	`)
	require.NoError(t, err, "Failed to create test function")

	// Clean up the function when done
	t.Cleanup(func() {
		_, cleanupErr := pgConn.Exec(ctx, `DROP FUNCTION IF EXISTS test_event_trigger_func();`)
		if cleanupErr != nil {
			t.Logf("Failed to clean up function: %s", cleanupErr)
		}
	})

	// Test event trigger parameters
	testEventTriggerParams := EventTriggerCreateParams{
		Name:     "test_event_trigger",
		Event:    "ddl_command_start",
		Tags:     []string{"CREATE TABLE", "ALTER TABLE"},
		ExecFunc: "test_event_trigger_func",
		Enabled:  true,
		Database: runOpts.Database,
		Comment:  "Test event trigger",
	}

	// Test Create
	t.Run("Create event trigger", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testEventTriggerParams)
		require.NoError(t, err, "Failed to create event trigger")
	})

	// Test GetOne
	t.Run("Get event trigger", func(t *testing.T) {
		var model *EventTriggerModel

		model, err = repo.GetOne(ctx, pgConn, "test_event_trigger")
		require.NoError(t, err, "Failed to get event trigger")
		require.NotNil(t, model, "Event trigger model should not be nil")

		// Verify model properties
		assert.Equal(t, "test_event_trigger", model.Name.String)
		assert.Equal(t, "ddl_command_start", model.Event.String)
		assert.True(t, model.Enabled.Bool)
		assert.Equal(t, "test_event_trigger_func", model.ExecFunc.String)
		assert.Equal(t, runOpts.Database, model.Database.String)
		assert.Equal(t, runOpts.Username, model.Owner.String)
		assert.Equal(t, "Test event trigger", model.Comment.String)

		// Check tags
		require.Len(t, model.Tags.Elements, 2, "Should have 2 tags")
		tagValues := []string{model.Tags.Elements[0].String, model.Tags.Elements[1].String}
		assert.Contains(t, tagValues, "CREATE TABLE")
		assert.Contains(t, tagValues, "ALTER TABLE")
	})

	// Test Update
	t.Run("Update event trigger", func(t *testing.T) {
		var model *EventTriggerModel

		// Test updating name
		t.Run("Update name", func(t *testing.T) {
			newName := "updated_event_trigger"
			updateParams := EventTriggerUpdateParams{
				Name: &newName,
			}

			err = repo.Update(ctx, pgConn, "test_event_trigger", updateParams)
			require.NoError(t, err, "Failed to update event trigger name")

			// Verify event trigger exists with the new name
			model, err = repo.GetOne(ctx, pgConn, newName)
			require.NoError(t, err, "Failed to get updated event trigger")
			assert.Equal(t, newName, model.Name.String)
		})

		// Test updating comment
		t.Run("Update comment", func(t *testing.T) {
			newComment := "Updated comment"
			updateCommentParams := EventTriggerUpdateParams{
				Comment: &newComment,
			}

			err = repo.Update(ctx, pgConn, "updated_event_trigger", updateCommentParams)
			require.NoError(t, err, "Failed to update event trigger comment")

			// Verify comment was updated
			model, err = repo.GetOne(ctx, pgConn, "updated_event_trigger")
			require.NoError(t, err, "Failed to get event trigger after comment update")
			assert.Equal(t, "updated_event_trigger", model.Name.String)
			assert.Equal(t, newComment, model.Comment.String)
		})

		// Test updating enabled status
		t.Run("Update enabled status", func(t *testing.T) {
			enabled := false
			updateEnabledParams := EventTriggerUpdateParams{
				Enabled: &enabled,
			}

			err = repo.Update(ctx, pgConn, "updated_event_trigger", updateEnabledParams)
			require.NoError(t, err, "Failed to update event trigger enabled status")

			// Verify enabled status was updated
			model, err = repo.GetOne(ctx, pgConn, "updated_event_trigger")
			require.NoError(t, err, "Failed to get event trigger after status update")
			assert.False(t, model.Enabled.Bool)
		})
	})

	// Test Drop
	t.Run("Drop event trigger", func(t *testing.T) {
		err = repo.Drop(ctx, pgConn, "updated_event_trigger")
		require.NoError(t, err, "Failed to drop event trigger")

		// Verify event trigger no longer exists
		_, err = repo.GetOne(ctx, pgConn, "updated_event_trigger")
		require.Error(t, err, "Getting dropped event trigger should return an error")
	})

	// Test Exists method
	t.Run("Exists method", func(t *testing.T) {
		var exists bool

		// Create a new event trigger for testing
		newTriggerParams := EventTriggerCreateParams{
			Name:     "exists_test_trigger",
			Event:    "ddl_command_start",
			Tags:     []string{"CREATE TABLE"},
			ExecFunc: "test_event_trigger_func",
			Enabled:  true,
			Database: runOpts.Database,
		}

		err = repo.Create(ctx, pgConn, newTriggerParams)
		require.NoError(t, err, "Failed to create test event trigger")

		// Test that the event trigger exists
		exists, err = repo.Exists(ctx, pgConn, "exists_test_trigger")
		require.NoError(t, err, "Failed to check if event trigger exists")
		assert.True(t, exists, "Event trigger should exist")

		// Clean up
		err = repo.Drop(ctx, pgConn, "exists_test_trigger")
		require.NoError(t, err, "Failed to drop test event trigger")

		// Verify it no longer exists
		exists, err = repo.Exists(ctx, pgConn, "exists_test_trigger")
		require.NoError(t, err, "Failed to check if event trigger exists after drop")
		assert.False(t, exists, "Event trigger should no longer exist")
	})
}
