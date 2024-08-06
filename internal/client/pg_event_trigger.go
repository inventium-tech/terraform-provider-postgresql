package client

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"strings"
)

const eventTriggerObject = "EVENT TRIGGER"

type eventTriggerSQL struct {
	db *sql.DB
}

type EventTriggerModel struct {
	Name     string   `json:"name"`
	Event    string   `json:"event"`
	Tags     []string `json:"tags"`
	ExecFunc string   `json:"exec_func"`
	Enabled  bool     `json:"enabled"`
	Database string   `json:"database"`
	Owner    string   `json:"owner"`
	Comment  string   `json:"comment"`
}

type EventTriggerRepository interface {
	Create(ctx context.Context, params EventTriggerCreateParams) error
	Drop(ctx context.Context, name string) error
	Get(ctx context.Context, name string) (*EventTriggerModel, error)
	Update(ctx context.Context, params EventTriggerUpdateParams) (*EventTriggerModel, error)
	Exists(ctx context.Context, name string) (bool, error)
	Scan(row *sql.Row) (*EventTriggerModel, error)
}

type EventTriggerCreateParams struct {
	Name     string   `validate:"required"`
	Event    string   `validate:"required,oneof=ddl_command_start ddl_command_end sql_drop table_rewrite"`
	ExecFunc string   `validate:"required"`
	Enabled  bool     `validate:"boolean"`
	Tags     []string `validate:"unique"`
	Comment  string
}

type EventTriggerUpdateParams struct {
	Name    string  `validate:"required"`
	NewName *string `validate:"required_without_all=Enabled Owner"`
	Enabled *bool   `validate:"required_without_all=NewName Owner Comment"`
	Owner   *string `validate:"required_without_all=NewName Enabled Comment"`
	Comment *string `validate:"required_without_all=NewName Enabled Owner"`
}

func NewEventTriggerRepository(db *sql.DB) EventTriggerRepository {
	return &eventTriggerSQL{
		db: db,
	}
}

func (e *eventTriggerSQL) Create(ctx context.Context, params EventTriggerCreateParams) error {
	validate := GetValidatorFromCtx(ctx)
	if err := validate.Struct(params); err != nil {
		return PgErrWithMetadata(err, "operation", opStructValidation)
	}

	txn, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return PgErrWithMetadata(err, "operation", opCreateEventTrigger, "pg_cmd", opStartTransaction)
	}
	defer DeferredRollback(txn)

	whenClause := ""
	if len(params.Tags) > 0 {
		whenClause = fmt.Sprintf("WHEN TAG IN (%s)", pgQuoteListOfLiterals(params.Tags))
	}

	createQuery := `
		CREATE EVENT TRIGGER %s
			ON %s
			%s
		EXECUTE FUNCTION %s();`

	err = WithQueryExecHandler(txn.ExecContext(ctx, fmt.Sprintf(createQuery, params.Name, params.Event, whenClause, params.ExecFunc)))
	if err != nil {
		return PgErrWithMetadata(err, "operation", opCreateEventTrigger)
	}

	err = CreateComment(ctx, txn, eventTriggerObject, params.Name, params.Comment)
	if err != nil {
		return PgErrWithMetadata(err, "operation", opCreateEventTrigger)
	}

	if err = txn.Commit(); err != nil {
		return PgErrWithMetadata(err, "operation", opCreateEventTrigger, "pg_cmd", opCommitTransaction)
	}
	return nil
}

func (e *eventTriggerSQL) Drop(ctx context.Context, name string) error {
	txn, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return PgErrWithMetadata(err, "operation", opDropEventTrigger, "pg_cmd", opStartTransaction)
	}
	defer DeferredRollback(txn)

	err = DropObject(ctx, txn, eventTriggerObject, name)
	if err != nil {
		return PgErrWithMetadata(err, "operation", opDropEventTrigger)
	}

	if err = txn.Commit(); err != nil {
		return PgErrWithMetadata(err, "operation", opCreateEventTrigger, "pg_cmd", opCommitTransaction)
	}
	return nil
}

func (e *eventTriggerSQL) Get(ctx context.Context, name string) (*EventTriggerModel, error) {
	readQuery := `
		SELECT evtname												 as "name",					
			   e.evtevent                                            as "event",
			   e.evttags                                             as "tags",
			   e.evtenabled                                          as "evtEnabled",
			   p.proname                                             as "exec_func",
			   pg_catalog.current_database()                         as "database",
			   pg_catalog.pg_get_userbyid(e.evtowner)                as "owner",
			   pg_catalog.obj_description(e.oid, 'pg_event_trigger') as "comment"
		FROM pg_catalog.pg_event_trigger e
				 LEFT JOIN pg_catalog.pg_proc p ON p.oid = e.evtfoid
				 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE e.evtname = %s;`

	row := e.db.QueryRowContext(ctx, fmt.Sprintf(readQuery, pq.QuoteLiteral(name)))
	model, err := e.Scan(row)
	if err != nil {
		return nil, PgErrWithMetadata(err, "operation", opGetEventTrigger)
	}
	return model, nil
}

func (e *eventTriggerSQL) Update(ctx context.Context, params EventTriggerUpdateParams) (*EventTriggerModel, error) {
	validate := GetValidatorFromCtx(ctx)
	if err := validate.Struct(params); err != nil {
		return nil, PgErrWithMetadata(err, "operation", opStructValidation)
	}

	txn, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, PgErrWithMetadata(err, "operation", opUpdateEventTrigger, "pg_cmd", opStartTransaction)
	}
	defer DeferredRollback(txn)

	var operations []string

	if params.NewName != nil {
		operations = append(operations, fmt.Sprintf("RENAME TO %s", pq.QuoteIdentifier(*params.NewName)))
	}
	if params.Enabled != nil {
		enabledText := "ALWAYS"
		if !*params.Enabled {
			enabledText = "DISABLE"
		}
		operations = append(operations, fmt.Sprintf("ENABLE %s", enabledText))
	}
	if params.Owner != nil {
		operations = append(operations, fmt.Sprintf("OWNER TO %s", pq.QuoteIdentifier(*params.Owner)))
	}

	updateQuery := `ALTER EVENT TRIGGER %s %s;`

	if (len(operations)) == 0 {
		err = WithQueryExecHandler(txn.ExecContext(ctx, fmt.Sprintf(updateQuery, *params.NewName, strings.Join(operations, " "))))
		if err != nil {
			return nil, PgErrWithMetadata(err, "operation", opUpdateEventTrigger)
		}
	}

	if params.Comment != nil {
		err = CreateComment(ctx, txn, "EVENT TRIGGER", params.Name, *params.Comment)
		if err != nil {
			return nil, PgErrWithMetadata(err, "operation", opUpdateEventTrigger)
		}
	}

	if err = txn.Commit(); err != nil {
		return nil, PgErrWithMetadata(err, "operation", opCreateEventTrigger, "pg_cmd", opCommitTransaction)
	}

	return e.Get(ctx, params.Name)
}

func (e *eventTriggerSQL) Exists(ctx context.Context, name string) (bool, error) {
	existsQuery := `
		SELECT EXISTS (
			SELECT 1 
			FROM pg_catalog.pg_event_trigger e
			WHERE e.evtname = %s);`

	var exists bool
	row := e.db.QueryRowContext(ctx, fmt.Sprintf(existsQuery, pq.QuoteLiteral(name)))
	err := row.Scan(&exists)
	if err != nil {
		return false, PgErrWithMetadata(err, "operation", opExistsEventTrigger, "pg_cmd", opQueryRow)
	}

	return exists, nil
}

func (e *eventTriggerSQL) Scan(row *sql.Row) (*EventTriggerModel, error) {
	var eventTrigger EventTriggerModel

	var comment sql.NullString
	var enabledRaw string

	err := row.Scan(
		&eventTrigger.Name,
		&eventTrigger.Event,
		(*pq.StringArray)(&eventTrigger.Tags),
		&enabledRaw,
		&eventTrigger.ExecFunc,
		&eventTrigger.Database,
		&eventTrigger.Owner,
		&comment,
	)
	if err != nil {
		return nil, PgErrWithMetadata(err, "operation", opScanRowResult, "model", "event_trigger")
	}

	//eventTrigger.Tags = tags.StringSlice()
	eventTrigger.Comment = comment.String

	// evtenabled: Controls in which session_replication_role modes the event trigger fires.
	// O = trigger fires in “origin” and “local” modes
	// D = trigger is disabled
	// R = trigger fires in “replica” mode
	// A = trigger fires always.
	eventTrigger.Enabled = enabledRaw != "D"

	return &eventTrigger, nil
}
