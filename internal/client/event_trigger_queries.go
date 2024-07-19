package client

import (
	"fmt"
	"github.com/lib/pq"
)

type PGEventTriggerQueries interface {
	CheckEventTriggerExistsQuery(evtTrigName string) string
	CreateEventTriggerQuery(name, event, execFn string, tags []string) string
	GetEventTriggerQuery(evtTrigName string) string
	UpdateEventTriggerEnableQuery(name string, enable bool) string
}

// Ensure the implementation satisfies the expected interfaces.
var _ PGEventTriggerQueries = &pgClientModel{}

func (cli *pgClientModel) CheckEventTriggerExistsQuery(evtTrigName string) string {
	query := `-- Check if the database exists
		SELECT EXISTS (SELECT 1
		   FROM pg_catalog.pg_event_trigger e
		   WHERE e.evtname = %s);`

	return fmt.Sprintf(query, pq.QuoteLiteral(evtTrigName))
}

func (cli *pgClientModel) CreateEventTriggerQuery(name, event, execFn string, tags []string) string {
	whenClause := ""
	if len(tags) > 0 {
		whenClause = fmt.Sprintf("WHEN TAG IN (%s)", pgQuoteListOfLiterals(tags))
		tags = append(tags, "ALL")
	}

	query := `-- Create Event Trigger
		CREATE EVENT TRIGGER %s
			ON %s
			%s
		EXECUTE FUNCTION %s();`

	return fmt.Sprintf(query, name, event, whenClause, execFn)
}

func (cli *pgClientModel) GetEventTriggerQuery(evtTrigName string) string {
	rawQuery := `
		SELECT pg_catalog.pg_get_userbyid(e.evtowner)                as "owner",
			   pg_catalog.obj_description(e.oid, 'pg_event_trigger') as "comment",
			   e.evtevent                                            as "event",
			   e.evttags                                             as "tags",
			   e.evtenabled                                          as "evtEnabled",
			   p.proname                                             as "exec_func"
		FROM pg_catalog.pg_event_trigger e
				 LEFT JOIN pg_catalog.pg_proc p ON p.oid = e.evtfoid
				 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE e.evtname = %s;`

	return fmt.Sprintf(rawQuery, pq.QuoteLiteral(evtTrigName))
}

func (cli *pgClientModel) UpdateEventTriggerEnableQuery(name string, enable bool) string {
	enabledText := "ENABLE ALWAYS"
	if !enable {
		enabledText = "DISABLE"
	}

	return fmt.Sprintf("ALTER EVENT TRIGGER %s %s;", name, enabledText)
}
