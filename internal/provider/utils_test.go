package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"os"
	"strings"
	"terraform-provider-postgresql/internal/client"
	"testing"
)

type postgresTestContainerConfig struct {
	image    string
	username string
	password string
	database string
}

const (
	testPGDefaultImage    = "postgres:16-alpine"
	testPGDefaultDb       = "test_tf_provider"
	testPGDefaultUsername = "tester"
	testPGDefaultPassword = "tester"
)

func loadPostgresTestContainer(t *testing.T, config postgresTestContainerConfig) client.PGClient {
	ctx := context.Background()

	if config.image == "" {
		config.image = testPGDefaultImage
	}
	if config.username == "" {
		config.username = testPGDefaultUsername
	}
	if config.password == "" {
		config.password = testPGDefaultPassword
	}
	if config.database == "" {
		config.database = testPGDefaultDb

	}

	opts := []testcontainers.ContainerCustomizer{
		postgres.WithDatabase(config.database),
		postgres.WithUsername(config.username),
		postgres.WithPassword(config.password),
		postgres.BasicWaitStrategies(),
	}

	pgContainer, err := postgres.Run(ctx, config.image, opts...)
	assert.NoError(t, err)

	endpoint, err := pgContainer.Endpoint(ctx, "")
	assert.NoError(t, err)

	connUrl := strings.Split(endpoint, ":")
	pgHost, pgPort := connUrl[0], connUrl[1]

	pgClient := client.NewPGClient(client.PGClientConfig{
		Host:     pgHost,
		Port:     pgPort,
		Username: config.username,
		Password: config.password,
		Database: config.database,
		Scheme:   "postgres",
		SSLMode:  "disable",
	})

	assert.NoError(t, os.Setenv("POSTGRES_HOST", pgHost))
	assert.NoError(t, os.Setenv("POSTGRES_PORT", pgPort))
	assert.NoError(t, os.Setenv("POSTGRES_USER", config.username))
	assert.NoError(t, os.Setenv("POSTGRES_PASSWORD", config.username))
	assert.NoError(t, os.Setenv("POSTGRES_DATABASE", config.database))
	assert.NoError(t, os.Setenv("POSTGRES_PASSWORD", config.password))
	assert.NoError(t, os.Setenv("POSTGRES_SCHEME", "postgres"))
	assert.NoError(t, os.Setenv("POSTGRES_SSLMODE", "disable"))

	return pgClient
}

func createTestFunction(t *testing.T, c client.PGClient, dbName, fName, fBody, fReturns, fLang string) {
	ctx := context.TODO()

	// CREATE RESOURCE
	txn, err := c.CreateTransaction(ctx, dbName)
	if err != nil {
		t.Fatalf("could not create transaction for db %s: %v", dbName, err)
	}
	defer c.DeferredRollback(txn)

	createFunctionQuery := c.CreateFunctionQuery(fName, fReturns, fBody, fLang, true)

	if _, err = txn.ExecContext(ctx, createFunctionQuery); err != nil {
		t.Fatalf("could not create test function: '%s' in db %s: %v", fName, dbName, err)
	}

	if err = txn.Commit(); err != nil {
		t.Fatalf("could not commit transaction for db %s: %v", dbName, err)
	}
}

func createTestEventTrigger(t *testing.T, c client.PGClient, m eventTriggerResModel) {
	ctx := context.TODO()
	diags := diag.Diagnostics{}

	diags.Append(m.Create(ctx, c)...)
	if diags.HasError() {
		t.Fatal("could not create test event trigger")
	}
}
