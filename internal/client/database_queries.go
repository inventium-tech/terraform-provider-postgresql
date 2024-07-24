package client

import (
	"fmt"
	"github.com/lib/pq"
)

type PGDatabaseQueries interface {
	CheckDatabaseExistsQuery(dbName string) string
	GetDatabaseQuery(dbName string) string
}

var _ PGDatabaseQueries = &pgClientModel{}

func (cli *pgClientModel) CheckDatabaseExistsQuery(dbName string) string {
	query := `-- Check if the database exists
		SELECT EXISTS (SELECT 1
		   FROM pg_catalog.pg_database d
		   WHERE d.datname = %s);`

	return fmt.Sprintf(query, pq.QuoteLiteral(dbName))
}

func (cli *pgClientModel) GetDatabaseQuery(dbName string) string {
	query := `-- Select Database Information
		SELECT pg_catalog.pg_get_userbyid(d.datdba)               as "owner",
			   pg_catalog.shobj_description(d.oid, 'pg_database') as "comment",
			   pg_catalog.pg_encoding_to_char(d.encoding)         as "encoding",
			   d.datcollate                                       as "lc_collate",
			   d.datctype                                         as "lc_ctype",
			   d.datconnlimit                                     as "conn_limit",
			   d.datallowconn                                     as "allow_conn"
		FROM pg_catalog.pg_database d
		WHERE d.datname = %s;`

	return fmt.Sprintf(query, pq.QuoteLiteral(dbName))
}
