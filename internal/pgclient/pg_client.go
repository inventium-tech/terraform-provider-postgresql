package pgclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"terraform-provider-postgresql/internal/helpers"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresqlClient interface {
	GetInitConfig() *ConnConfig
	GetConnection(ctx context.Context, targetDB ...string) (DBTX, error)
	GetPool(ctx context.Context, targetDB ...string) (*pgxpool.Pool, error)
	AcquireConn(ctx context.Context, targetDB ...string) (*pgxpool.Conn, error)
	Close() error
}

func NewPostgresqlClient(ctx context.Context, connConfig *ConnConfig) (PostgresqlClient, error) {
	validator := helpers.GetSafeValidator()
	if err := validator.Struct(connConfig); err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}

	client := &pgClientImpl{
		initConfig: connConfig,
		pools:      make(map[string]*pgxpool.Pool),
	}

	// Test the connection by creating initial pool
	_, err := client.getOrCreatePool(ctx, connConfig.Database)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type pgClientImpl struct {
	initConfig *ConnConfig
	pools      map[string]*pgxpool.Pool
	mu         sync.RWMutex
}

func (p *pgClientImpl) GetConnection(ctx context.Context, targetDb ...string) (DBTX, error) {
	pool, err := p.GetPool(ctx, targetDb...)
	if err != nil {
		return nil, err
	}
	// Return the pool itself, which implements DBTX interface
	// The pool handles connection management internally
	return pool, nil
}

func (p *pgClientImpl) GetPool(ctx context.Context, targetDb ...string) (*pgxpool.Pool, error) {
	database := p.initConfig.Database
	if len(targetDb) > 0 && targetDb[0] != "" {
		database = targetDb[0]
	}

	return p.getOrCreatePool(ctx, database)
}

func (p *pgClientImpl) AcquireConn(ctx context.Context, targetDb ...string) (*pgxpool.Conn, error) {
	pool, err := p.GetPool(ctx, targetDb...)
	if err != nil {
		return nil, err
	}

	// Acquire a connection from the pool
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("error acquiring connection from pool: %w", err)
	}

	return conn, nil
}

func (p *pgClientImpl) getOrCreatePool(ctx context.Context, database string) (*pgxpool.Pool, error) {
	poolKey := fmt.Sprintf("%s:%s@%s", p.initConfig.Host, p.initConfig.Username, database)

	// Fast path: read lock to check if pool exists
	p.mu.RLock()
	if pool, ok := p.pools[poolKey]; ok {
		p.mu.RUnlock()
		return pool, nil
	}
	p.mu.RUnlock()

	// Slow path: write lock to create pool
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have created it)
	if pool, ok := p.pools[poolKey]; ok {
		return pool, nil
	}

	// Create connection string for this database
	connOpts := &ConnConfig{
		Host:     p.initConfig.Host,
		Port:     p.initConfig.Port,
		Username: p.initConfig.Username,
		Password: p.initConfig.Password,
		Database: database,
		SSLMode:  p.initConfig.SSLMode,
	}

	// Parse config and create pool
	poolConfig, err := pgxpool.ParseConfig(connOpts.String())
	if err != nil {
		sanitizeErr := strings.ReplaceAll(err.Error(), connOpts.Password, "****")
		return nil, fmt.Errorf("error parsing connection config for database '%s': %s", database, sanitizeErr)
	}

	// Configure pool settings
	// These are reasonable defaults for a Terraform provider
	poolConfig.MinConns = 1        // Keep at least 1 connection alive
	poolConfig.MaxConns = 10       // Allow up to 10 concurrent connections per database
	poolConfig.MaxConnIdleTime = 0 // Don't close idle connections (keep warm)
	poolConfig.MaxConnLifetime = 0 // Don't expire connections based on lifetime
	// Note: HealthCheckPeriod defaults to 1 minute, which is reasonable for Terraform

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		sanitizeErr := strings.ReplaceAll(err.Error(), connOpts.Password, "****")
		return nil, fmt.Errorf("error creating connection pool for database '%s': %s", database, sanitizeErr)
	}

	// Test the pool with a ping
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error pinging database '%s': %s", database, err.Error())
	}

	p.pools[poolKey] = pool
	return pool, nil
}

func (p *pgClientImpl) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, pool := range p.pools {
		pool.Close()
		delete(p.pools, key)
	}
	return nil
}

func (p *pgClientImpl) GetInitConfig() *ConnConfig {
	return p.initConfig
}
