package client

import (
	"fmt"
	"github.com/lib/pq"
)

type PGRoleQueries interface {
	CheckRoleIsSuperuserQuery(role string) string
	CheckCurrenRoleIsSuperuserQuery() string
}

var _ PGRoleQueries = &pgClientModel{}

func (cli *pgClientModel) CheckRoleIsSuperuserQuery(role string) string {
	query := `-- Check if the role is a superuser
		SELECT r.rolsuper
		FROM pg_catalog.pg_roles r
		WHERE r.rolname = %s;`

	return fmt.Sprintf(query, pq.QuoteLiteral(role))
}

func (cli *pgClientModel) CheckCurrenRoleIsSuperuserQuery() string {
	query := `-- Check if the current role is a superuser
		SELECT r.rolsuper
		FROM pg_catalog.pg_roles r
		WHERE r.rolname = pg_catalog.current_user();`

	return query
}
