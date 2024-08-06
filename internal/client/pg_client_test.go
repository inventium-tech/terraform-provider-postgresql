package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func TestPgClientPool_GetConnection(t *testing.T) {
	targetDb := "test_client_connection"
	runOpts := test.PostgresContainerRunOptions{Database: targetDb}
	pgContainer := test.LoadPostgresTestContainer(t, runOpts, false)
	connString := test.GetPostgresConnectionString(t, pgContainer)

	setupInitOpts := func(pool *pgClientPool) {
		opts := &PgConnectionOpts{}
		assert.NoError(t, opts.FromConnectionString(connString))

		pool.initConfig = *opts
	}
	tests := []struct {
		name     string
		targetDb string
		setup    func(pool *pgClientPool)
		wantErr  bool
	}{
		{
			name: "ValidOpts",
			// TODO: create a new database for this test
			targetDb: targetDb,
			setup:    setupInitOpts,
			wantErr:  false,
		},
		{
			name:     "InvalidOpts",
			targetDb: "invalid_db",
			setup:    setupInitOpts,
			wantErr:  true,
		},
		{
			name:     "ExistingConnection",
			targetDb: targetDb,
			setup:    setupInitOpts,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()

			pool := &pgClientPool{
				connPool:   make(map[string]PgConnector),
				initConfig: PgConnectionOpts{},
			}

			if tt.setup != nil {
				tt.setup(pool)
			}

			conn, err := pool.GetConnection(ctx, tt.targetDb)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, conn)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conn)
			}
		})
	}
}
