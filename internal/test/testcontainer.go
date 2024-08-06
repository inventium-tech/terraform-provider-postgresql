package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	testPGDefaultImage    = "postgres:16-alpine"
	testPGDefaultDb       = "test_tf_provider"
	testPGDefaultUsername = "tester"
	testPGDefaultPassword = "tester"
)

type PostgresContainerRunOptions struct {
	Image    string
	Username string
	Password string
	Database string
}

func LoadPostgresTestContainer(t *testing.T, config PostgresContainerRunOptions, setEnVars bool) *postgres.PostgresContainer {
	t.Helper()
	ctx := context.TODO()

	if config.Image == "" {
		config.Image = testPGDefaultImage
	}
	if config.Username == "" {
		config.Username = testPGDefaultUsername
	}
	if config.Password == "" {
		config.Password = testPGDefaultPassword
	}
	if config.Database == "" {
		config.Database = testPGDefaultDb
	}

	opts := []testcontainers.ContainerCustomizer{
		postgres.WithDatabase(config.Database),
		postgres.WithUsername(config.Username),
		postgres.WithPassword(config.Password),
		postgres.BasicWaitStrategies(),
	}

	pgContainer, err := postgres.Run(ctx, config.Image, opts...)
	assert.NoError(t, err)

	if setEnVars {
		endpoint, endpointErr := pgContainer.Endpoint(ctx, "")
		assert.NoError(t, endpointErr)

		connUrl := strings.Split(endpoint, ":")
		pgHost, pgPort := connUrl[0], connUrl[1]

		assert.NoError(t, os.Setenv("POSTGRES_HOST", pgHost))
		assert.NoError(t, os.Setenv("POSTGRES_PORT", pgPort))
		assert.NoError(t, os.Setenv("POSTGRES_USER", config.Username))
		assert.NoError(t, os.Setenv("POSTGRES_PASSWORD", config.Password))
		assert.NoError(t, os.Setenv("POSTGRES_DATABASE", config.Database))
		assert.NoError(t, os.Setenv("POSTGRES_SCHEME", "postgres"))
		assert.NoError(t, os.Setenv("POSTGRES_SSLMODE", "disable"))
	}

	return pgContainer
}

func GetPostgresConnectionString(t *testing.T, container *postgres.PostgresContainer) string {
	t.Helper()

	ctx := context.TODO()
	connString, err := container.ConnectionString(ctx)
	assert.NoError(t, err)

	u, err := url.Parse(connString)
	assert.NoError(t, err)

	queryParams := u.Query()

	if queryParams.Get("sslmode") == "" {
		queryParams.Set("sslmode", "disable")
	}

	return connString + queryParams.Encode()
}
