package client

import (
	"context"
	"fmt"
	"gocloud.dev/postgres"
	"strings"
	"sync"
)

type pgClientPool struct {
	lock       sync.RWMutex
	connPool   map[string]PgConnector
	initConfig PgConnectionOpts
}

type PgClient interface {
	GetConnection(ctx context.Context, db ...string) (PgConnector, error)
	GetInitConfig() PgConnectionOpts
}

var _ PgClient = &pgClientPool{}

func NewPgClient(opts PgConnectionOpts) PgClient {
	return &pgClientPool{
		connPool:   make(map[string]PgConnector),
		initConfig: opts,
	}
}

func (p *pgClientPool) GetInitConfig() PgConnectionOpts {
	return p.initConfig
}

func (p *pgClientPool) GetConnection(ctx context.Context, targetDb ...string) (PgConnector, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	connOpts := p.initConfig
	if len(targetDb) > 0 {
		connOpts.Database = targetDb[0]
	}
	connString := connOpts.String()
	conn, ok := p.connPool[connString]
	if ok {
		return conn, nil
	}

	db, err := postgres.Open(ctx, connString)
	if err != nil {
		sanitizeErr := strings.Replace(err.Error(), connOpts.Password, "****", -1)
		return nil, fmt.Errorf("error connecting to database '%s'. Error: %s", connOpts.Database, sanitizeErr)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database '%s'. Error: %s", connOpts.Database, err.Error())
	}

	db.SetMaxOpenConns(connOpts.MaxOpenConn)
	db.SetMaxIdleConns(connOpts.MaxIdleConn)

	pgConnector, err := NewPgConnector(db)
	if err != nil {
		return nil, fmt.Errorf("error create a client connector: %v", err)
	}

	return pgConnector, nil
}
