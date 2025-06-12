package pgclient

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"terraform-provider-postgresql/internal/helpers"
)

// queries.
const (
	createEventTriggerQuery = `
	CREATE EVENT TRIGGER %s
    	ON %s
    	%s
	EXECUTE FUNCTION %s();`

	selectEventTriggerQuery = `
	SELECT e.evtname                                         AS name,
       e.evtevent                                            AS event,
       e.evttags                                             AS tags,
       e.evtenabled                                          AS enabled,
       p.proname                                             AS exec_func,
       pg_catalog.current_database()                         AS database,
       pg_catalog.pg_get_userbyid(e.evtowner)                AS owner,
       pg_catalog.obj_description(e.oid, 'pg_event_trigger') AS comment
	FROM pg_catalog.pg_event_trigger e
			 INNER JOIN pg_catalog.pg_proc p ON p.oid = e.evtfoid
	WHERE e.evtname = $1
	  AND pg_catalog.pg_function_is_visible(p.oid);`

	existsEventTriggerQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_event_trigger
		WHERE evtname = $1
	);`

	alterEventTriggerStatusQuery = `ALTER EVENT TRIGGER %s %s;`

	alterEventTriggerRename = `ALTER EVENT TRIGGER %s RENAME TO %s;`
)

type EventTriggerRepo interface {
	Create(ctx context.Context, dbtx DBTX, params EventTriggerCreateParams) error
	Drop(ctx context.Context, dbtx DBTX, name string) error
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
	GetOne(ctx context.Context, dbtx DBTX, name string) (*EventTriggerModel, error)
	Update(ctx context.Context, dbtx DBTX, name string, params EventTriggerUpdateParams) error
}

func NewEventTriggerRepo() EventTriggerRepo {
	return &eventTriggerRepo{}
}

type (
	EventTriggerModel struct {
		Name     pgtype.Text               `db:"name"`
		Event    pgtype.Text               `db:"event"`
		Tags     pgtype.Array[pgtype.Text] `db:"tags"`
		ExecFunc pgtype.Text               `db:"exec_func"`
		Enabled  pgtype.Bool               `db:"enabled"`
		Database pgtype.Text               `db:"database"`
		Owner    pgtype.Text               `db:"owner"`
		Comment  pgtype.Text               `db:"comment"`
	}
	EventTriggerCreateParams struct {
		Name     string   `json:"name" validate:"required"`
		Event    string   `json:"event" validate:"required,oneof=ddl_command_start ddl_command_end sql_drop table_rewrite"`
		Tags     []string `json:"tags" validate:"required"` // Assuming tags are stored as a comma-separated string
		ExecFunc string   `json:"exec_func" validate:"required"`
		Enabled  bool     `json:"enabled" validate:"boolean"`
		Database string   `json:"database" validate:"required"`
		Owner    string   `json:"owner"`
		Comment  string   `json:"comment"`
	}
	EventTriggerUpdateParams struct {
		Name    *string `json:"name"`
		Enabled *bool   `json:"enabled"`
		Owner   *string `json:"owner"`
		Comment *string `json:"comment"`
	}
)

type eventTriggerRepo struct{}

func (e eventTriggerRepo) Create(ctx context.Context, dbtx DBTX, params EventTriggerCreateParams) error {
	var err error

	validate := helpers.GetSafeValidator()
	if err = validate.Struct(params); err != nil {
		return err
	}

	var name string
	if name, err = sanitizeInput(params.Name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid event trigger name %q. error: %w", params.Name, err)
	}

	var whenClause string
	if len(params.Tags) > 0 {
		whenClause = fmt.Sprintf("WHEN TAG IN (%s)", e.formatTagsAsString(params.Tags))
	}

	query := fmt.Sprintf(createEventTriggerQuery, name, params.Event, whenClause, params.ExecFunc)

	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create Event Trigger %s: %w", params.Name, err)
	}

	if !params.Enabled {
		if err = e.setEventTriggerStatus(ctx, dbtx, name, params.Enabled); err != nil {
			return fmt.Errorf("failed to set event trigger status: %w", err)
		}
	}

	if err = CreateComment(ctx, dbtx, PgObjEventTrigger, name, params.Comment); err != nil {
		return err
	}

	if err = AlterOwner(ctx, dbtx, PgObjEventTrigger, name, params.Owner); err != nil {
		return err
	}
	return nil
}

func (e eventTriggerRepo) Drop(ctx context.Context, dbtx DBTX, name string) error {
	err := DropObject(ctx, dbtx, PgObjEventTrigger, name)
	if err != nil {
		return fmt.Errorf("failed to drop event trigger with name %s: %w", name, err)
	}

	return nil
}

func (e eventTriggerRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return false, fmt.Errorf("invalid event trigger name %q. error: %w", name, err)
	}

	var exists bool
	err = dbtx.QueryRow(ctx, existsEventTriggerQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if event trigger %s exists: %w", name, err)
	}

	return exists, nil
}

func (e eventTriggerRepo) GetOne(ctx context.Context, dbtx DBTX, name string) (*EventTriggerModel, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return nil, fmt.Errorf("invalid event trigger name %q. error: %w", name, err)
	}

	var model EventTriggerModel
	var enabledRaw pgtype.Text
	row := dbtx.QueryRow(ctx, selectEventTriggerQuery, name)
	err = row.Scan(
		&model.Name,
		&model.Event,
		&model.Tags,
		&enabledRaw,
		&model.ExecFunc,
		&model.Database,
		&model.Owner,
		&model.Comment,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get event trigger %s: %w", name, err)
	}

	// enabledRaw value is expected to be one of these characters; it indicates in which session_replication_role modes the event trigger fires.
	// O = trigger fires in "origin" and "local" modes
	// D = trigger is disabled
	// R = trigger fires in "replica" mode
	// A = trigger fires always
	// any other value different from "D" means the event trigger is enabled
	model.Enabled = pgtype.Bool{
		Bool:  enabledRaw.String != "D",
		Valid: true,
	}

	return &model, nil
}

func (e eventTriggerRepo) Update(ctx context.Context, dbtx DBTX, name string, params EventTriggerUpdateParams) error {
	var newName string
	var err error

	if params.Enabled != nil {
		if err = e.setEventTriggerStatus(ctx, dbtx, name, *params.Enabled); err != nil {
			return fmt.Errorf("failed to set event trigger status: %w", err)
		}
	}

	if params.Owner != nil {
		if err = AlterOwner(ctx, dbtx, PgObjEventTrigger, name, *params.Owner); err != nil {
			return fmt.Errorf("failed to alter owner for event trigger %s: %w", name, err)
		}
	}

	if params.Comment != nil {
		if err = CreateComment(ctx, dbtx, PgObjEventTrigger, name, *params.Comment); err != nil {
			return fmt.Errorf("failed to set comment for event trigger %s: %w", name, err)
		}
	}

	if params.Name != nil {
		if newName, err = sanitizeInput(*params.Name, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid event trigger name %s. error: %w", *params.Name, err)
		}

		renamedQuery := fmt.Sprintf(alterEventTriggerRename, name, newName)
		_, err = dbtx.Exec(ctx, renamedQuery)
	}

	return err
}

func (e eventTriggerRepo) setEventTriggerStatus(ctx context.Context, dbtx DBTX, name string, enabled bool) error {
	status := "ENABLE"
	if !enabled {
		status = "DISABLE"
	}
	_, err := dbtx.Exec(ctx, fmt.Sprintf(alterEventTriggerStatusQuery, name, status))
	return err
}

func (e eventTriggerRepo) formatTagsAsString(tags []string) string {
	result := ""
	reduceFunc := func(acc, tag string) string {
		if acc == "" {
			return fmt.Sprintf(`'%s'`, tag)
		}
		return fmt.Sprintf(`%s, '%s'`, acc, tag)
	}

	result = helpers.SliceReduce(tags, reduceFunc, result)

	return result
}
