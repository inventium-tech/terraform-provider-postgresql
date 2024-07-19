package client

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gocloud.dev/postgres"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
)

type PGClient interface {
	GetConfig() PGClientConfig
	GetConnection(ctx context.Context, targetDB ...string) (*PGClientConnection, error)
	CreateTransaction(ctx context.Context, targetDB ...string) (*sql.Tx, error)
	DeferredRollback(txn *sql.Tx)

	PGCommonQueries
	PGDatabaseQueries
	PGEventTriggerQueries
	PGFunctionQueries
	PGRoleQueries
}

type PGClientConnection struct {
	*sql.DB
	config PGClientConfig
}

type PGClientConfigOpts func(*PGClientConfig) error

type PGClientConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	Database    string
	Scheme      string
	SSLMode     string
	MaxOpenConn int
	MaxIdleConn int
}

type pgClientModel struct {
	lock     sync.RWMutex
	connPool map[string]*PGClientConnection
	config   PGClientConfig
}

var (
	_ PGClient = &pgClientModel{}
)

func NewPGClient(cfg PGClientConfig) PGClient {
	return &pgClientModel{
		connPool: make(map[string]*PGClientConnection),
		config:   cfg,
	}
}

func NewPGClientConfig(opts ...PGClientConfigOpts) (PGClientConfig, error) {
	config := PGClientConfig{
		Scheme:      "postgres",
		SSLMode:     "prefer",
		MaxOpenConn: 0,
		MaxIdleConn: 5,
	}
	for _, optFn := range opts {
		if err := optFn(&config); err != nil {
			return config, err
		}

	}
	if err := config.Validate(); err != nil {
		return config, err
	}
	return config, nil
}

func (cfg PGClientConfig) String() string {
	params := map[string][]string{
		"sslmode": {cfg.SSLMode},
	}

	// sslmode and connect_timeout are not allowed with gocloud
	// (TLS is provided by gocloud directly)
	//if cfg.Scheme == "postgres" {
	//	params["sslmode"] = []string{cfg.SSLMode}
	//}

	return fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s?%v",
		cfg.Scheme,
		url.PathEscape(cfg.Username),
		url.PathEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.Database,
		url.Values(params).Encode(),
	)
}

func (cfg PGClientConfig) Validate() error {
	if cfg.Host == "" {
		return fmt.Errorf("host is required in PGClientConfig")
	}

	if cfg.Port == "" {
		return fmt.Errorf("port is required in PGClientConfig")
	}

	if cfg.Username == "" {
		return fmt.Errorf("username is required in PGClientConfig")
	}

	if cfg.Password == "" {
		return fmt.Errorf("password is required in PGClientConfig")
	}

	if cfg.Database == "" {
		return fmt.Errorf("database is required in PGClientConfig")
	}

	if cfg.Scheme == "" {
		return fmt.Errorf("scheme is required in PGClientConfig")
	}

	if cfg.SSLMode == "" {
		return fmt.Errorf("sslmode is required in PGClientConfig")
	}

	return nil

}

func (cli *pgClientModel) GetConfig() PGClientConfig {
	return cli.config
}

func (cli *pgClientModel) GetConnection(ctx context.Context, targetDB ...string) (*PGClientConnection, error) {
	config, err := cli.getTargetConnConfig(targetDB)
	if err != nil {
		return nil, err
	}

	cli.lock.Lock()
	defer cli.lock.Unlock()

	connString := config.String()
	conn, ok := cli.connPool[connString]
	if ok {
		return conn, nil
	}

	db, err := postgres.Open(ctx, connString)
	if err != nil {
		sanitizeErr := strings.Replace(err.Error(), config.Password, "****", -1)
		return nil, fmt.Errorf("error connecting to database '%s'. Error: %s", config.Database, sanitizeErr)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database '%s'. Error: %s", config.Database, err.Error())
	}

	db.SetMaxOpenConns(config.MaxOpenConn)
	db.SetMaxIdleConns(config.MaxIdleConn)

	conn = &PGClientConnection{
		DB:     db,
		config: config,
	}

	return conn, nil
}

func (cli *pgClientModel) CreateTransaction(ctx context.Context, targetDB ...string) (*sql.Tx, error) {
	//cli.logger.Debug(ctx, "Creating transaction")
	config, err := cli.getTargetConnConfig(targetDB)
	if err != nil {
		return nil, err
	}

	conn, err := cli.GetConnection(ctx, targetDB...)
	if err != nil {
		return nil, err
	}

	txn, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction in database '%s'. Error: %s", config.Database, err.Error())
	}

	return txn, nil
}

func (cli *pgClientModel) DeferredRollback(txn *sql.Tx) {
	err := txn.Rollback()
	switch {
	case errors.Is(err, sql.ErrTxDone):
		tfLog := os.Getenv("TF_LOG")
		if slices.Contains([]string{"DEBUG", "TRACE"}, tfLog) {
			fmt.Println("[DEBUG] transaction has already been committed or rolled back")
		}
		break
	case err != nil:
		fmt.Printf("[ERROR]: error rolling back transaction.\nError: %v\n", err)
		break
	}

}

func (cli *pgClientModel) getTargetConnConfig(targetDB []string) (PGClientConfig, error) {
	targetCfg := cli.config
	switch len(targetDB) {
	case 0:
		return targetCfg, nil
	case 1:
		targetCfg.Database = targetDB[0]
		break
	default:
		return targetCfg, fmt.Errorf("only one connection configuration is allowed")
	}

	return targetCfg, nil
}

func WithMaxOpenConn(maxOpenConn int) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.MaxOpenConn = maxOpenConn
		return nil
	}
}

func WithMaxIdleConn(maxIdleConn int) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.MaxIdleConn = maxIdleConn
		return nil
	}
}

func WithSSLMode(sslMode string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.SSLMode = sslMode
		return nil
	}
}

func WithScheme(scheme string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Scheme = scheme
		return nil
	}
}

func WithPGClientConfigHost(host string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Host = host
		return nil
	}
}

func WithPort(port string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Port = port
		return nil
	}
}

func WithUsername(username string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Username = username
		return nil
	}
}

func WithPassword(password string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Password = password
		return nil
	}
}

func WithDatabase(database string) PGClientConfigOpts {
	return func(cfg *PGClientConfig) error {
		cfg.Database = database
		return nil
	}
}
