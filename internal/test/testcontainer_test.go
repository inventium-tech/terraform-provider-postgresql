package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestGetPostgresConnectionString(t *testing.T) {
	pgContainer := LoadPostgresTestContainer(t, PostgresContainerRunOptions{}, false)
	t.Cleanup(func() {
		assert.NoError(t, pgContainer.Terminate(context.Background()))
	})

	connString := GetPostgresConnectionString(t, pgContainer)

	u, err := url.Parse(connString)
	assert.NoError(t, err)
	assert.Equal(t, "disable", u.Query().Get("sslmode"))
}
