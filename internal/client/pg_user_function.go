package client

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

const functionObjectType = "function"

type userFunctionSQL struct {
	db *sql.DB
}

type UserFunctionRepository interface {
	Create(ctx context.Context, params UserFunctionCreateParams) error
}

type UserFunctionCreateParams struct {
	Name    string            `validate:"required"`
	Args    map[string]string `validate:"unique,dive,required"`
	Returns string            `validate:"required"`
	Lang    string            `validate:"required,oneof=plpgsql sql"`
	Body    string            `validate:"required"`
	Replace bool              `validate:"boolean"`
}

var _ UserFunctionRepository = &userFunctionSQL{}

func NewUserFunctionRepository(db *sql.DB) UserFunctionRepository {
	return &userFunctionSQL{
		db: db,
	}
}

func (f userFunctionSQL) Create(ctx context.Context, params UserFunctionCreateParams) error {
	validate := GetValidatorFromCtx(ctx)
	if err := validate.Struct(params); err != nil {
		return err
	}

	txn, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return WrapPgError(err, msgErrorStartingTransaction)
	}
	defer DeferredRollback(txn)

	var orReplace, funcArgs string

	if len(params.Args) > 0 {
		funcArgs = pgMapToFuncArg(params.Args)
	}

	if params.Replace {
		orReplace = "OR REPLACE"
	}

	createQuery := `
		CREATE %s FUNCTION %s(%s)
		RETURNS %s
		LANGUAGE %s
		AS $$
		%s
		$$;`

	result, err := txn.ExecContext(ctx, fmt.Sprintf(createQuery,
		orReplace,
		params.Name,
		funcArgs,
		params.Returns,
		params.Lang,
		params.Body,
	))
	if err != nil {
		return WrapPgError(err, fmt.Sprintf(msgErrorCreatingObject, functionObjectType))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return WrapPgError(err, msgErrorExecContextGetRowsAffected)
	}

	if err = txn.Commit(); err != nil {
		return WrapPgError(err, msgErrorCommittingTransaction)
	}

	slog.Info(fmt.Sprintf(msgSuccessCreatingObject, functionObjectType), "rows_affected", rowsAffected)
	return nil
}
