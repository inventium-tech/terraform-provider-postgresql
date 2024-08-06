package client

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
)

type pgExecContextFunc func(ctx context.Context, query string, args ...any) (sql.Result, error)

func parseExecContextFunc[T *sql.DB | *sql.Tx](d T) pgExecContextFunc {
	var fn pgExecContextFunc

	switch dType := any(d).(type) {
	case *sql.DB:
		fn = dType.ExecContext
	case *sql.Tx:
		fn = dType.ExecContext
	}
	return fn
}

func CreateComment[T *sql.DB | *sql.Tx](ctx context.Context, d T, oType, oName, comment string) error {
	execContext := parseExecContextFunc(d)

	commentQuery := `COMMENT ON %s %s IS %s;`
	err := WithQueryExecHandler(execContext(ctx, fmt.Sprintf(commentQuery, oType, oName, pq.QuoteLiteral(comment))))
	if err != nil {
		return PgErrWithMetadata(err, "pg_cmd", opCreateComment)
	}
	return nil
}

func DropObject[T *sql.DB | *sql.Tx](ctx context.Context, d T, oType, oName string) error {
	execContext := parseExecContextFunc(d)

	dropQuery := `DROP %s IF EXISTS %s;`
	err := WithQueryExecHandler(execContext(ctx, fmt.Sprintf(dropQuery, oType, oName)))
	if err != nil {
		return PgErrWithMetadata(err, "pg_cmd", opDropObject)
	}
	return nil
}
