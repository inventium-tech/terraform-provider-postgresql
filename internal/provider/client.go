package provider

//
//import (
//	"context"
//	"database/sql"
//	"fmt"
//	"github.com/hashicorp/terraform-plugin-framework/diag"
//	"github.com/hashicorp/terraform-plugin-log/tflog"
//	"gocloud.dev/postgres"
//	"net/url"
//	"sync"
//	"time"
//
//	_ "github.com/lib/pq" // PostgreSQL db
//)
//
//type postgresClient struct {
//	lock     sync.RWMutex
//	connPool map[string]*pgClientConnection
//	config   *PostgresqlProviderConfig
//}
//
//type postgresClientConnector interface {
//	GetConnectionString(overrides ...postgresConnectorBuilder) (*PostgresConnectionString, diag.Diagnostics)
//	GetConnection(overrides ...postgresConnectorBuilder) (*pgClientConnection, diag.Diagnostics)
//	StartTransaction(overrides ...postgresConnectorBuilder) (*sql.Tx, diag.Diagnostics)
//}
//
//type pgClientHelperQueries interface {
//	isConnUserSuperuser(conn *sql.DB) (bool, error)
//}
//
//type pgClientConnection struct {
//	*sql.DB
//	limits     int64
//	connString *PostgresConnectionString
//}
//type PostgresConnectionString struct {
//	Host     string
//	Port     int64
//	Username string
//	Password string
//	Database string
//	Scheme   string
//	Params   map[string][]string
//}
//
//type postgresConnectorBuilder func(*PostgresConnectionString) error
//
//var (
//	_ postgresClientConnector = &postgresClient{}
//)
//
//func NewPostgresqlClient(cfg *PostgresqlProviderConfig) *postgresClient {
//	return &postgresClient{
//		connPool: make(map[string]*pgClientConnection),
//		config:   cfg,
//	}
//}
//
//func (pgC *postgresClient) GetConnectionString(overrides ...postgresConnectorBuilder) (*PostgresConnectionString, diag.Diagnostics) {
//	diags := diag.Diagnostics{}
//	params := map[string][]string{}
//	// sslmode and connect_timeout are not allowed with gocloud
//	// (TLS is provided by gocloud directly)
//	if pgC.config.Scheme.ValueString() == "postgres" {
//		params["sslmode"] = []string{pgC.config.SSLMode.ValueString()}
//	}
//
//	pgCs := &PostgresConnectionString{
//		Host:     pgC.config.Host.ValueString(),
//		Port:     pgC.config.Port.ValueInt64(),
//		Username: pgC.config.Username.ValueString(),
//		Password: pgC.config.Password.ValueString(),
//		Database: pgC.config.Database.ValueString(),
//		Scheme:   pgC.config.Scheme.ValueString(),
//		Params:   params,
//	}
//
//	for _, overrideFn := range overrides {
//		if err := overrideFn(pgCs); err != nil {
//			diags.AddError(
//				"Error building connection string",
//				err.Error(),
//			)
//			return nil, diags
//		}
//	}
//
//	return pgCs, nil
//}
//
//func (pgC *postgresClient) GetConnection(overrides ...postgresConnectorBuilder) (*pgClientConnection, diag.Diagnostics) {
//	tflog.Info(context.TODO(), "GET a connection")
//	diags := diag.Diagnostics{}
//	pgC.lock.RLock()
//
//	connString, diags := pgC.GetConnectionString(overrides...)
//	if diags.HasError() {
//		pgC.lock.RUnlock()
//		return nil, diags
//	}
//	connStringEncoded := connString.String()
//
//	conn, ok := pgC.connPool[connStringEncoded]
//	pgC.lock.RUnlock()
//	if ok {
//		return conn, nil
//	}
//
//	pgC.lock.Lock()
//	defer pgC.lock.Unlock()
//	conn, ok = pgC.connPool[connStringEncoded]
//	if ok {
//		return conn, nil
//	}
//
//	ctx, _ := context.WithTimeout(context.Background(), 10*time.Minute)
//	db, err := postgres.Open(ctx, connStringEncoded)
//	if err != nil {
//		diags.AddError(
//			fmt.Sprintf("Error connecting to database %s.", connString.Database),
//			err.Error(),
//		)
//		return nil, diags
//	}
//
//	conn = &pgClientConnection{
//		DB:         db,
//		connString: connString,
//	}
//	pgC.connPool[connStringEncoded] = conn
//
//	return conn, diags
//}
//
//func (pgC *postgresClient) StartTransaction(overrides ...postgresConnectorBuilder) (*sql.Tx, diag.Diagnostics) {
//	tflog.Debug(context.TODO(), "Starting Postgresql Transaction")
//	diags := diag.Diagnostics{}
//	conn, diags := pgC.GetConnection(overrides...)
//	if diags.HasError() {
//		return nil, diags
//	}
//
//	txn, err := conn.Begin()
//	if err != nil {
//		diags.AddError(
//			fmt.Sprintf("Error starting transaction in Database '%s'", conn.connString.Database),
//			err.Error(),
//		)
//	}
//
//	return txn, diags
//}
//
//func (pgCS *PostgresConnectionString) String() string {
//	return fmt.Sprintf(
//		"%s://%s:%s@%s:%d/%s?%v",
//		pgCS.Scheme,
//		url.PathEscape(pgCS.Username),
//		url.PathEscape(pgCS.Password),
//		pgCS.Host,
//		pgCS.Port,
//		pgCS.Database,
//		url.Values(pgCS.Params).Encode(),
//	)
//}
//
//func withConnStringDatabase(db string) postgresConnectorBuilder {
//	return func(pgCS *PostgresConnectionString) error {
//		pgCS.Database = db
//		return nil
//	}
//}
//
//func withConnStringUsername(u string) postgresConnectorBuilder {
//	return func(pgCS *PostgresConnectionString) error {
//		pgCS.Username = u
//		return nil
//	}
//}
//
//func withConnStringPasswd(p string) postgresConnectorBuilder {
//	return func(pgCS *PostgresConnectionString) error {
//		pgCS.Password = p
//		return nil
//	}
//}
