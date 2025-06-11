package pgclient

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

type (
	// MockDB is a mock implementation of the DBTX interface for testing
	MockDB struct {
		ExecFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
		QueryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
		QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	}
	// MockRow is a mock implementation of pgx.Row for testing
	MockRow struct {
		ScanFunc func(dest ...interface{}) error
	}
)

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return m.ExecFunc(ctx, sql, args...)
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return m.QueryFunc(ctx, sql, args...)
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return m.QueryRowFunc(ctx, sql, args...)
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return m.ScanFunc(dest...)
}

func deferTestCloseConn(t *testing.T, conn *pgx.Conn) {
	t.Helper()
	assert.NoError(t, conn.Close(t.Context()))
}

func loadTestPostgresqlClient(t *testing.T, runOpts test.PostgresContainerRunOptions) PostgresqlClient {
	t.Helper()

	pgContainer := test.LoadPostgresTestContainer(t, runOpts, false)

	// get the endpoint (with format <host>:<port>) from the container context
	endpoint, err := pgContainer.Endpoint(t.Context(), "")
	assert.NoError(t, err)

	endpointParts := strings.Split(endpoint, ":")
	if len(endpointParts) != 2 {
		t.Fatalf("Invalid endpoint format: %s", endpoint)
	}

	// parse the port as an integer
	port, err := strconv.Atoi(endpointParts[1])
	assert.NoError(t, err)

	// create a connection
	connConfig := &ConnConfig{
		Host:     endpointParts[0],
		Port:     port,
		Username: runOpts.Username,
		Password: runOpts.Password,
		Database: runOpts.Database,
		SSLMode:  "disable", // Use disable for testing purposes
	}

	client, err := NewPostgresqlClient(t.Context(), connConfig)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	return client
}
