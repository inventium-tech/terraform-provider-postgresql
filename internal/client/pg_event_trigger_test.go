package client

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/postgres"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

const (
	testEventTriggerDb   = "test_event_trigger_db"
	testEventTriggerUser = "test_event_trigger_user"
)

func testPrepareEventTriggerTestCase(t *testing.T) (context.Context, *sql.DB) {
	runOpts := test.PostgresContainerRunOptions{
		Database: testEventTriggerDb,
		Username: testEventTriggerUser,
	}
	pgContainer := test.LoadPostgresTestContainer(t, runOpts, false)
	connString := test.GetPostgresConnectionString(t, pgContainer)
	ctx := context.TODO()

	db, err := postgres.Open(ctx, connString)
	assert.NoError(t, err)
	return ctx, db
}

func mockEventTriggerCreateParams(t *testing.T) EventTriggerCreateParams {
	t.Helper()
	return EventTriggerCreateParams{
		Name:     "test_trigger",
		Event:    "ddl_command_start",
		ExecFunc: "test_func",
		Enabled:  true,
		Tags:     []string{"CREATE TABLE"},
		Comment:  "test comment",
	}
}

func mockUserFunctionCreateParamsForEventTrigger(t *testing.T) UserFunctionCreateParams {
	t.Helper()
	params := mockUserFunctionCreateParams(t)
	params.Body = "BEGIN RAISE NOTICE 'DDL command executed'; END;"
	params.Args = map[string]string{}
	params.Returns = "event_trigger"
	return params
}

func TestEventTriggerSQL_Create(t *testing.T) {
	ctx, db := testPrepareEventTriggerTestCase(t)
	defer db.Close()

	userFunctionRepo := NewUserFunctionRepository(db)
	eventTriggerRepo := NewEventTriggerRepository(db)

	tests := []struct {
		name         string
		createParams func(t *testing.T) EventTriggerCreateParams
		setup        func(t *testing.T)
		wantErr      bool
		errMsg       string
	}{
		{
			name: "Success",
			createParams: func(t *testing.T) EventTriggerCreateParams {
				validParams := mockEventTriggerCreateParams(t)
				validParams.ExecFunc = mockUserFunctionCreateParams(t).Name
				return validParams
			},
			setup: func(t *testing.T) {
				execFunc := mockUserFunctionCreateParamsForEventTrigger(t)
				err := userFunctionRepo.Create(ctx, execFunc)
				assert.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:         "FailEventTriggerExists",
			createParams: mockEventTriggerCreateParams,
			wantErr:      true,
			errMsg:       "pq: event trigger \"test_trigger\" already exists",
		},
		{
			name: "FailExecFunctionNotFound",
			createParams: func(t *testing.T) EventTriggerCreateParams {
				invalidEvent := mockEventTriggerCreateParams(t)
				invalidEvent.Name = "invalid_event_trigger"
				invalidEvent.ExecFunc = "test_invalid_function"
				return invalidEvent
			},
			wantErr: true,
			errMsg:  "pq: function test_invalid_function() does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			var params EventTriggerCreateParams
			if tt.createParams != nil {
				params = tt.createParams(t)
			}

			err := eventTriggerRepo.Create(context.Background(), params)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventTriggerSQL_Get(t *testing.T) {
	ctx, db := testPrepareEventTriggerTestCase(t)
	defer db.Close()

	userFunctionRepo := NewUserFunctionRepository(db)
	eventTriggerRepo := NewEventTriggerRepository(db)

	eventTriggerCreateParams := mockEventTriggerCreateParams(t)
	userFuncCreateParams := mockUserFunctionCreateParamsForEventTrigger(t)

	testMatrix := []struct {
		name      string
		nameParam string
		setup     func(t *testing.T)
		wantErr   bool
		result    *EventTriggerModel
		errMsg    string
	}{
		{
			name:      "Success",
			nameParam: "test_trigger",
			setup: func(t *testing.T) {
				err := userFunctionRepo.Create(ctx, userFuncCreateParams)
				assert.NoError(t, err)

				eventTriggerParams := eventTriggerCreateParams
				eventTriggerParams.Tags = []string{}
				eventTriggerParams.ExecFunc = userFuncCreateParams.Name
				err = eventTriggerRepo.Create(ctx, eventTriggerParams)
				assert.NoError(t, err)
			},
			result: &EventTriggerModel{
				Name:     eventTriggerCreateParams.Name,
				Event:    eventTriggerCreateParams.Event,
				Tags:     nil,
				ExecFunc: userFuncCreateParams.Name,
				Enabled:  eventTriggerCreateParams.Enabled,
				Database: testEventTriggerDb,
				Owner:    testEventTriggerUser,
				Comment:  eventTriggerCreateParams.Comment,
			},
			wantErr: false,
		},
	}

	for _, tt := range testMatrix {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			m, err := eventTriggerRepo.Get(ctx, "test_trigger")
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				if tt.result != nil {
					assert.Equal(t, tt.result, m)
				}
				assert.NoError(t, err)
			}
		})
	}
}
