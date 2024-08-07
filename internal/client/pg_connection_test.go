package client

import (
	"database/sql"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockPgConnector is a mock implementation of the PgConnector interface.
type mockPgConnector struct{}

// Ensure mockPgConnector implements PgConnector.
var _ PgConnector = &mockPgConnector{}

func (m *mockPgConnector) EventTriggerRepository() EventTriggerRepository {
	return nil
}

func (m *mockPgConnector) UserFunctionRepository() UserFunctionRepository {
	return nil
}

func mockPgConnectionOpts() *PgConnectionOpts {
	return &PgConnectionOpts{
		Host:        "localhost",
		Port:        5432,
		Username:    "tester",
		Password:    "tester123",
		Database:    "tester_db",
		Scheme:      "postgres",
		SSLMode:     "disable",
		MaxOpenConn: 10,
		MaxIdleConn: 5,
	}
}

func TestPgConnectionOpts_String(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(opts *PgConnectionOpts)
		expected string
	}{
		{
			name: "ValidConnectionString",
			modify: func(opts *PgConnectionOpts) {
				opts.Username = "user"
				opts.Password = "password"
			},
			expected: "postgres://user:password@localhost:5432/tester_db?sslmode=disable",
		},
		{
			name: "WithSpecialCharacters",
			modify: func(opts *PgConnectionOpts) {
				opts.Username = "user@name"
				opts.Password = "pass@word"
			},
			expected: "postgres://user%40name:pass%40word@localhost:5432/tester_db?sslmode=disable",
		},
		{
			name: "WithDifferentScheme",
			modify: func(opts *PgConnectionOpts) {
				opts.Scheme = "awspostgres"
			},
			expected: "awspostgres://tester:tester123@localhost:5432/tester_db?",
		},
		{
			name: "WithEmptySSLMode",
			modify: func(opts *PgConnectionOpts) {
				opts.SSLMode = ""
			},
			expected: "postgres://tester:tester123@localhost:5432/tester_db?sslmode=",
		},
		{
			name: "WithEmptyParams",
			modify: func(opts *PgConnectionOpts) {
				opts.SSLMode = ""
				opts.Scheme = "gcppostgres"
			},
			expected: "gcppostgres://tester:tester123@localhost:5432/tester_db?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create opts here to make sure the modify function can alter it on every test case
			opts := mockPgConnectionOpts()
			if tt.modify != nil {
				tt.modify(opts)
			}
			assert.Equal(t, tt.expected, opts.String())
		})
	}
}

func TestPgConnectionOpts_Validation(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	tests := []struct {
		name      string
		opts      PgConnectionOpts
		expectErr bool
	}{
		{
			name:      "ValidOptions",
			opts:      *mockPgConnectionOpts(),
			expectErr: false,
		},
		{
			name: "MissingHost",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Host = ""
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "InvalidPort",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Port = 0
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "MissingUsername",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Username = ""
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "MissingPassword",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Password = ""
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "MissingDatabase",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Database = ""
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "InvalidScheme",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.Scheme = "invalid"
				return opts
			}(),
			expectErr: true,
		},
		{
			name: "MissingSSLMode",
			opts: func() PgConnectionOpts {
				opts := *mockPgConnectionOpts()
				opts.SSLMode = ""
				return opts
			}(),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.opts)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPgConnector(t *testing.T) {
	t.Run("NilDB", func(t *testing.T) {
		conn, err := NewPgConnector(nil)
		assert.Error(t, err)
		assert.Nil(t, conn)
	})

	t.Run("ValidDB", func(t *testing.T) {
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dbname?sslmode=disable")
		require.NoError(t, err)
		defer db.Close()

		conn, err := NewPgConnector(db)
		assert.NoError(t, err)
		assert.NotNil(t, conn)
	})
}

func TestPgConnection_EventTriggerRepository(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	conn, err := NewPgConnector(db)
	require.NoError(t, err)

	// event trigger is created on first call
	repoFirstCall := conn.EventTriggerRepository()
	assert.NotNil(t, repoFirstCall)

	// event trigger should be exact same on another call
	repoSecondCall := conn.EventTriggerRepository()
	assert.Equal(t, repoFirstCall, repoSecondCall)

	// forced to nil should create a new one
	conn.(*pgConnection).eventTriggerRepository = nil //nolint:forcetypeassert
	repoThirdCall := conn.EventTriggerRepository()
	assert.NotNil(t, repoThirdCall)
}
