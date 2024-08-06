package client

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
	"log/slog"
	"os"
	"slices"
	"strings"
)

const (
	msgErrorCommitingTransaction       = "error commiting transaction."
	msgErrorCreatingObject             = "error creating %s."
	msgErrorExecContextGetRowsAffected = "error retrieving rows affected by query execution."
	msgErrorStartingTransaction        = "error starting transaction."
	msgSuccessCreatingObject           = "%s created successfully."
)

const (
	opCommitTransaction   = "commit_transaction"
	opCreateEventTrigger  = "create_event_trigger"
	opCreateComment       = "create_comment"
	opCreateUserFunction  = "create_user_function"
	opDropObject          = "drop_object"
	opDropEventTrigger    = "drop_event_trigger"
	opExecute             = "execute"
	opExistsEventTrigger  = "exists_event_trigger"
	opGetEventTrigger     = "get_event_trigger"
	opQuery               = "query"
	opQueryRow            = "query_row"
	opRollbackTransaction = "rollback_transaction"
	opStartTransaction    = "start_transaction"
	opStructValidation    = "struct_validation"
	opScanRowResult       = "scan_row_result"
	opUpdateEventTrigger  = "update_event_trigger"
)

func pgQuoteListOfLiterals(list []string) string {
	quoted := make([]string, len(list))
	for i, item := range list {
		quoted[i] = pq.QuoteLiteral(item)
	}
	return strings.Join(quoted, ", ")
}

func pgMapToFuncArg(mapArg map[string]string) string {
	args := make([]string, 0, len(mapArg))
	for k, v := range mapArg {
		args = append(args, fmt.Sprintf("%s %s", k, v))
	}
	return strings.Join(args, ", ")
}

func GetValidatorFromCtx(ctx context.Context) *validator.Validate {
	if v, ok := ctx.Value("validator").(*validator.Validate); ok {
		return v
	}

	return validator.New(validator.WithRequiredStructEnabled())
}

func WithQueryExecHandler(result sql.Result, err error) error {
	if err != nil {
		return PgErrWithMetadata(err, "pg_cmd", opExecute)
	}

	rowsAffected, errRows := result.RowsAffected()
	if errRows != nil {
		return WrapPgError(errRows, msgErrorExecContextGetRowsAffected)
	}

	slog.Debug("query executed successfully", "rows_affected", rowsAffected)

	return nil
}

func DeferredRollback(txn *sql.Tx) {
	err := txn.Rollback()
	switch {
	case errors.Is(err, sql.ErrTxDone):
		tfLog := os.Getenv("TF_LOG")
		if slices.Contains([]string{"DEBUG", "TRACE"}, tfLog) {
			slog.Debug("transaction has already been committed or rolled back")
		}
	case err != nil:
		panic(PgErrWithMetadata(err, "operation", opRollbackTransaction))
	}

}
