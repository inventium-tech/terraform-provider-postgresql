package client

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/postgres"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func mockUserFunctionCreateParams(t *testing.T) UserFunctionCreateParams {
	t.Helper()
	return UserFunctionCreateParams{
		Name:    "test_function",
		Args:    map[string]string{"arg1": "TEXT"},
		Returns: "TEXT",
		Lang:    "plpgsql",
		Body:    "BEGIN RETURN 'Hello, arg1!'; END;",
		Replace: true,
	}
}

func TestUserFunctionSQL_Create(t *testing.T) {
	targetDb := "test_user_function"
	runOpts := test.PostgresContainerRunOptions{Database: targetDb}
	pgContainer := test.LoadPostgresTestContainer(t, runOpts, false)
	connString := test.GetPostgresConnectionString(t, pgContainer)
	ctx := context.TODO()

	db, err := postgres.Open(ctx, connString)
	assert.NoError(t, err)
	defer db.Close()

	repo := NewUserFunctionRepository(db)

	tests := []struct {
		name         string
		createParams func(t *testing.T) UserFunctionCreateParams
		setup        func(t *testing.T)
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "Success",
			createParams: mockUserFunctionCreateParams,
			wantErr:      false,
		},
		{
			name: "FailInvalidFuncArgsType",
			createParams: func(t *testing.T) UserFunctionCreateParams {
				invalidFuncParamsType := mockUserFunctionCreateParams(t)
				invalidFuncParamsType.Args = map[string]string{"arg1": "invalid_type"}
				return invalidFuncParamsType
			},
			wantErr: true,
			errMsg:  fmt.Sprintf(msgErrorCreatingObject, functionObjectType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			var params UserFunctionCreateParams
			if tt.createParams != nil {
				params = tt.createParams(t)
			}

			err := repo.Create(context.Background(), params)
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
