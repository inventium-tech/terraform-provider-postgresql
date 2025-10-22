package pgclient

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"sync"
	"terraform-provider-postgresql/internal/helpers"
)

type PostgresqlClient interface {
	GetInitConfig() *ConnConfig
	GetConnection(ctx context.Context, targetDB ...string) (*pgx.Conn, error)
}

func NewPostgresqlClient(ctx context.Context, connConfig *ConnConfig) (PostgresqlClient, error) {
	validator := helpers.GetSafeValidator()
	if err := validator.Struct(connConfig); err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}

	client := &pgClientImpl{
		connPool:   make(map[string]*pgx.Conn),
		initConfig: connConfig,
	}

	if _, err := client.GetConnection(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

type pgClientImpl struct {
	lock       sync.RWMutex
	connPool   map[string]*pgx.Conn
	initConfig *ConnConfig
}

func (p *pgClientImpl) GetConnection(ctx context.Context, targetDb ...string) (*pgx.Conn, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	connOpts := p.initConfig
	if len(targetDb) > 0 && targetDb[0] != "" {
		connOpts.Database = targetDb[0]
	}

	if conn, ok := p.connPool[connOpts.ID()]; ok {
		return conn, nil
	}

	conn, err := pgx.Connect(ctx, connOpts.String())
	if err != nil {
		sanitizeErr := strings.ReplaceAll(err.Error(), connOpts.Password, "****")
		return nil, fmt.Errorf("error connecting to database '%s'. Error: %s", connOpts.Database, sanitizeErr)
	}

	if err = conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error pinging database '%s'. Error: %s", connOpts.Database, err.Error())
	}

	p.connPool[connOpts.ID()] = conn
	return conn, nil
}

func (p *pgClientImpl) GetInitConfig() *ConnConfig {
	return p.initConfig
}
