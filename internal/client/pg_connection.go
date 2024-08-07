package client

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

const (
	defaultMaxOpenConn = 0
	defaultMaxIdleConn = 5
)

type pgConnection struct {
	*sql.DB

	eventTriggerRepository EventTriggerRepository
	userFunctionRepository UserFunctionRepository
}

type PgConnectionOpts struct {
	Host        string `json:"host" validate:"required,hostname"`
	Port        int    `json:"port" validate:"required,gt=0"`
	Username    string `json:"username" validate:"required"`
	Password    string `json:"password" validate:"required,alphanumunicode"`
	Database    string `json:"database" validate:"required"`
	Scheme      string `json:"scheme" validate:"required,oneof=postgres awspostgres gcppostgres"`
	SSLMode     string `json:"ssl_mode" validate:"required_if=Scheme postgres"`
	MaxOpenConn int    `json:"max_open_conn" validate:"min=0"`
	MaxIdleConn int    `json:"max_idle_conn" validate:"min=0"`
}

type PgConnectionOptsFn func(*PgConnectionOpts) error

type PgConnector interface {
	EventTriggerRepository() EventTriggerRepository
	UserFunctionRepository() UserFunctionRepository
}

var _ PgConnector = &pgConnection{}

func NewPgConnectionOpts(ctx context.Context, configFn ...PgConnectionOptsFn) (*PgConnectionOpts, error) {
	opts := &PgConnectionOpts{}

	for _, fn := range configFn {
		if err := fn(opts); err != nil {
			return nil, err
		}
	}

	validate := GetValidatorFromCtx(ctx)
	if err := validate.Struct(opts); err != nil {
		return nil, WrapPgError(err, err.Error())
	}

	return opts, nil
}

func (o *PgConnectionOpts) String() string {
	params := map[string][]string{}

	// sslmode and connect_timeout are not allowed with gocloud
	// (TLS is provided by gocloud directly)
	if o.Scheme == "postgres" {
		params["sslmode"] = []string{o.SSLMode}
	}

	userInfo := url.UserPassword(o.Username, o.Password)

	return fmt.Sprintf(
		"%s://%s@%s:%v/%s?%v",
		o.Scheme,
		userInfo.String(),
		o.Host,
		o.Port,
		o.Database,
		url.Values(params).Encode(),
	)
}

func (o *PgConnectionOpts) FromConnectionString(connString string) error {
	u, err := url.Parse(connString)
	if err != nil {
		return err
	}

	var port int

	switch len(u.Port()) {
	case 0:
		port = 5432
	default:
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return err
		}
	}
	var sslmode string
	if sslmode = u.Query().Get("sslmode"); sslmode == "" {
		sslmode = "disable"
	}

	o.Scheme = u.Scheme
	o.Host = u.Hostname()
	o.Port = port
	o.Username = u.User.Username()
	o.Password, _ = u.User.Password()
	o.Database = u.Path[1:] // database name is the path without the '/' prefix
	o.SSLMode = sslmode
	o.MaxOpenConn = defaultMaxOpenConn
	o.MaxIdleConn = defaultMaxIdleConn

	return nil
}

func NewPgConnector(db *sql.DB) (PgConnector, error) {
	if db == nil {
		return nil, errors.New("database connection param is nil")
	}

	return &pgConnection{
		DB: db,
	}, nil
}

func (p *pgConnection) EventTriggerRepository() EventTriggerRepository {
	if p.eventTriggerRepository == nil {
		p.eventTriggerRepository = NewEventTriggerRepository(p.DB)
	}
	return p.eventTriggerRepository
}

func (p *pgConnection) UserFunctionRepository() UserFunctionRepository {
	if p.userFunctionRepository == nil {
		p.userFunctionRepository = NewUserFunctionRepository(p.DB)
	}
	return p.userFunctionRepository
}

func WithMaxOpenConn(maxOpenConn int) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.MaxOpenConn = maxOpenConn
		return nil
	}
}

func WithMaxIdleConn(maxIdleConn int) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.MaxIdleConn = maxIdleConn
		return nil
	}
}

func WithSSLMode(sslMode string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.SSLMode = sslMode
		return nil
	}
}

func WithScheme(scheme string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Scheme = scheme
		return nil
	}
}

func WithHost(scheme string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Host = scheme
		return nil
	}
}

func WithPort(port int) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Port = port
		return nil
	}
}

func WithUsername(username string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Username = username
		return nil
	}
}

func WithPassword(password string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Password = password
		return nil
	}
}

func WithDatabase(database string) PgConnectionOptsFn {
	return func(o *PgConnectionOpts) error {
		o.Database = database
		return nil
	}
}
