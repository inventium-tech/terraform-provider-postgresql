package pgclient

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"regexp"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
)

// queries
const (
	selectUserFunctionQuery = `
	SELECT p.proname                           AS name,
		pg_get_function_arguments(p.oid)       AS args,
       	pg_get_function_result(p.oid)          AS returns,
       	pg_get_functiondef(p.oid)              AS body,
       	l.lanname                              AS language,
       	pg_catalog.current_database()          AS database,
       	n.nspname                              AS schema,
       	pg_catalog.pg_get_userbyid(p.proowner) AS owner,
       	pg_catalog.obj_description(p.oid)      AS comment
	FROM pg_catalog.pg_proc p
    	INNER JOIN pg_catalog.pg_namespace n   ON n.oid = p.pronamespace
        INNER JOIN pg_catalog.pg_language l    ON p.prolang = l.oid
	WHERE p.proname = $1							  -- Function name
	  	AND n.nspname = $2							  -- Schema name
  		AND p.prokind = 'f'							  -- Only include functions
	  	AND pg_catalog.pg_function_is_visible(p.oid)  -- Only include functions visible to the current user
	  	AND pg_get_function_arguments(p.oid) = $3	  -- Filter by specific parameter types signature to ensure uniqueness
	;`

	existsUserFunctionQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_proc p
    		INNER JOIN pg_catalog.pg_namespace n  ON n.oid = p.pronamespace
        	INNER JOIN pg_catalog.pg_language l   ON p.prolang = l.oid
		WHERE p.proname = $1							  -- Function name
			AND n.nspname = $2							  -- Schema name
			AND p.prokind = 'f'							  -- Only include functions
			AND pg_catalog.pg_function_is_visible(p.oid)  -- Only include functions visible to the current user
			AND pg_get_function_arguments(p.oid) = $3	  -- Filter by specific parameter types signature to ensure uniqueness
	);`

	createUserFunctionQuery = `
	CREATE%s FUNCTION %s.%s(%s)
		RETURNS %s
		LANGUAGE %s
	AS
	$%s$
	%s
	$%s$;`
)

type UserFunctionRepo interface {
	Create(ctx context.Context, dbtx DBTX, params UserFunctionCreateParams) error
	Drop(ctx context.Context, dbtx DBTX, name string) error
	Exists(ctx context.Context, dbtx DBTX, funcId string) (bool, error)
	GetOne(ctx context.Context, dbtx DBTX, fullName string) (*UserFunctionModel, error)
	Update(ctx context.Context, dbtx DBTX, funcId string, params UserFunctionUpdateParams) error
}

func NewUserFunctionRepo() UserFunctionRepo {
	return &userFunctionRepo{}
}

type (
	UserFunctionCreateParams struct {
		Name         string              `json:"name" validate:"required"`
		Args         []PgFunctionArgType `json:"args" validate:"required"`
		Returns      string              `json:"returns" validate:"required"`
		Language     string              `json:"language" validate:"required,oneof=plpgsql sql"`
		Body         string              `json:"body" validate:"required"`
		AllowReplace bool                `json:"allow_replace" validate:"boolean"`
		Schema       string              `json:"schema" validate:"required"`
		Owner        string              `json:"owner"`
		Comment      string              `json:"comment"`
	}
	UserFunctionUpdateParams struct {
		Name    *string `json:"name"`
		Owner   *string `json:"owner"`
		Comment *string `json:"comment"`
	}
)

type userFunctionRepo struct{}

func (f userFunctionRepo) Create(ctx context.Context, dbtx DBTX, params UserFunctionCreateParams) error {
	var err error

	validate := helpers.GetSafeValidator()
	if err = validate.Struct(params); err != nil {
		return err
	}

	var schema, name, funcArgs, orReplace, returns string

	if schema, err = sanitizeInput(params.Schema, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid schema name %q. error: %w", params.Schema, err)
	}

	if name, err = sanitizeInput(params.Name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid function name %q. error: %w", params.Name, err)
	}

	if returns, err = sanitizeInput(params.Returns, SanitizeReturns); err != nil {
		return fmt.Errorf("invalid return type %q. error: %w", params.Returns, err)
	}

	funcArgs = helpers.SliceReduce(params.Args, func(result string, arg PgFunctionArgType) string {
		if result == "" {
			return arg.String()
		}
		return fmt.Sprintf("%s, %s", result, arg.String())
	})

	if params.AllowReplace {
		orReplace = " OR REPLACE"
	}

	// create a temporal unique tag for the function body
	tag := fmt.Sprintf("function_body_%s", helpers.RandomString(8))

	query := fmt.Sprintf(createUserFunctionQuery, orReplace, schema, name, funcArgs, returns, params.Language, tag, params.Body, tag)
	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create function %s: %w", params.Name, err)
	}

	// this is the way postgresql identify the Function signature
	// format: <schema_name>.<function_name>(<args_type>, <args_type>...)
	argTypes := helpers.SliceReduce(params.Args, func(result string, arg PgFunctionArgType) string {
		if result == "" {
			return arg.Type
		}
		return fmt.Sprintf("%s, %s", result, arg.Type)
	})

	functionSignature := fmt.Sprintf("%s.%s(%s)", schema, name, argTypes)

	err = CreateComment(ctx, dbtx, PgObjFunction, functionSignature, params.Comment)
	if err != nil {
		return err
	}

	err = AlterOwner(ctx, dbtx, PgObjFunction, functionSignature, params.Owner)
	if err != nil {
		return err
	}

	return nil
}

func (f userFunctionRepo) Exists(ctx context.Context, dbtx DBTX, funcId string) (bool, error) {
	var err error

	schemaName, funcNameArgs, foundSplit := strings.Cut(funcId, ".")
	if !foundSplit {
		return false, fmt.Errorf("invalid function id %q: expected format <schema>.<function_name>", funcId)
	}

	// extract function args if they are part of the function name
	var fnName, fnArgs string

	if fnName, err = extractFunctionName(funcNameArgs); err != nil {
		return false, err
	}
	if fnArgs, err = extractFunctionArgsFull(funcNameArgs); err != nil {
		return false, err
	}

	//fnName, fnArgs, err := extractFunctionNameAndArgsSignature(funcNameArgs)
	//if err != nil {
	//	return false, err
	//}

	var exists bool
	err = dbtx.QueryRow(ctx, existsUserFunctionQuery, fnName, schemaName, fnArgs).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if function %s exists: %w", funcId, err)
	}

	return exists, nil
}

func (f userFunctionRepo) GetOne(ctx context.Context, dbtx DBTX, funcId string) (*UserFunctionModel, error) {
	var err error

	schemaName, funcNameArgs, foundSplit := strings.Cut(funcId, ".")
	if !foundSplit {
		return nil, fmt.Errorf("invalid function id %q: expected format <schema>.<function_name>", funcId)
	}

	// extract function args if they are part of the function name
	var fnName, fnArgs string

	if fnName, err = extractFunctionName(funcNameArgs); err != nil {
		return nil, err
	}
	if fnArgs, err = extractFunctionArgsFull(funcNameArgs); err != nil {
		return nil, err
	}

	//fnName, fnArgs, err := extractFunctionNameAndArgsSignature(funcSignature)
	//if err != nil {
	//	return nil, err
	//}

	// Define a structure to hold the result
	var model UserFunctionModel
	var fullFunctionDef string

	row := dbtx.QueryRow(ctx, selectUserFunctionQuery, fnName, schemaName, fnArgs)
	err = row.Scan(
		&model.Name,
		&model.Args,
		&model.Returns,
		&fullFunctionDef,
		&model.Language,
		&model.Database,
		&model.Schema,
		&model.Owner,
		&model.Comment,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get function %s: %w", funcId, err)
	}

	hasOrReplace := strings.HasPrefix(fullFunctionDef, "CREATE OR REPLACE")
	model.AllowReplace = pgtype.Bool{Bool: hasOrReplace, Valid: true}

	// Extract just the function body from the complete definition
	// This is necessary because pg_get_functiondef returns the complete CREATE FUNCTION statement
	parsedBody, err := extractFunctionBody(fullFunctionDef)
	if err != nil {
		return nil, err
	}
	model.Body = pgtype.Text{String: parsedBody, Valid: true}

	return &model, nil
}

func (f userFunctionRepo) Drop(ctx context.Context, dbtx DBTX, funcId string) error {
	var err error

	schemaName, funcNameArgs, foundSplit := strings.Cut(funcId, ".")
	if !foundSplit {
		return fmt.Errorf("invalid function id %q: expected format <schema>.<function_name>", funcId)
	}

	// extract function args if they are part of the function name
	var fnName, fnArgsTypes string

	if fnName, err = extractFunctionName(funcNameArgs); err != nil {
		return err
	}
	if fnArgsTypes, err = extractFunctionArgsTypes(funcNameArgs); err != nil {
		return err
	}

	objId := fmt.Sprintf("%s.%s(%s)", schemaName, fnName, fnArgsTypes)

	err = DropObject(ctx, dbtx, PgObjFunction, objId)
	if err != nil {
		return fmt.Errorf("failed to drop function with Id %s: %w", funcId, err)
	}

	return nil
}

func (f userFunctionRepo) Update(ctx context.Context, dbtx DBTX, funcId string, params UserFunctionUpdateParams) error {
	var err error

	schemaName, funcNameArgs, foundSplit := strings.Cut(funcId, ".")
	if !foundSplit {
		return fmt.Errorf("invalid function id %q: expected format <schema>.<function_name>", funcId)
	}

	// extract function args if they are part of the function name
	var fnName, fnArgsTypes string

	if fnName, err = extractFunctionName(funcNameArgs); err != nil {
		return err
	}
	if fnArgsTypes, err = extractFunctionArgsTypes(funcNameArgs); err != nil {
		return err
	}

	objId := fmt.Sprintf("%s.%s(%s)", schemaName, fnName, fnArgsTypes)

	if params.Owner != nil {
		if err = AlterOwner(ctx, dbtx, PgObjFunction, objId, *params.Owner); err != nil {
			return err
		}
	}

	if params.Comment != nil {
		if err = CreateComment(ctx, dbtx, PgObjFunction, objId, *params.Comment); err != nil {
			return err
		}
	}

	if params.Name != nil {
		var newName string
		if newName, err = sanitizeInput(*params.Name, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid Postgresql Function name %s. error: %w", *params.Name, err)
		}

		err = RenameObject(ctx, dbtx, PgObjFunction, objId, newName)
	}

	return err
}

// extractFunctionBody extracts the function body from a raw function definition string.
func extractFunctionBody(functionDef string) (string, error) {
	// Find the body between $function$ tag used by default for PostgreSQL after creation
	tagRE := regexp.MustCompile(`(?s)\$function\$(.*?)\$function\$`)
	matches := tagRE.FindStringSubmatch(functionDef)
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract function body from definition")
	}
	return strings.TrimSpace(matches[1]), nil
}

// extractFunctionName extracts the function name from a raw function name string.
// It expects the raw name to be in the format "function_name(args)".
func extractFunctionName(rawName string) (string, error) {
	if rawName == "" {
		return "", fmt.Errorf("function name cannot be empty")
	}

	// find the index of the first opening parenthesis
	argsStartIndex := strings.Index(rawName, "(")
	if argsStartIndex == -1 {
		return strings.TrimSpace(rawName), nil // no arguments, return the raw name
	}

	// extract the function name before the opening parenthesis
	return strings.TrimSpace(rawName[:argsStartIndex]), nil
}

// extractFunctionArgsParsed extracts the raw function arguments from a raw function name string.
// It expects the raw name to be in the format "function_name(args)".
// It returns a slice of PgFunctionArgType, which contains the parsed arguments.
func extractFunctionArgsParsed(rawName string) ([]PgFunctionArgType, error) {
	result := make([]PgFunctionArgType, 0)
	if rawName == "" {
		return result, fmt.Errorf("function name cannot be empty")
	}

	// find the index of the first opening parenthesis
	argsStartIndex := strings.Index(rawName, "(")
	if argsStartIndex == -1 {
		return result, fmt.Errorf("function name does not contain arguments: %s", rawName)
	}

	// extract the arguments' part from the raw name
	argsPart := rawName[argsStartIndex+1 : len(rawName)-1] // exclude the parentheses
	result, err := ParseRawFunctionArgs(argsPart)
	return result, err
}

// extractFunctionArgsFull extracts the function arguments with full syntax from a raw function name string.
// It expects the raw name to be in the format "function_name(args)".
// It returns the arguments as a string, including their modes, name, types, and defaults.
func extractFunctionArgsFull(rawName string) (string, error) {
	parsedArgs, err := extractFunctionArgsParsed(rawName)
	if err != nil {
		return "", err
	}

	result := helpers.SliceReduce(parsedArgs, func(result string, arg PgFunctionArgType) string {
		if result == "" {
			return arg.String()
		}
		return fmt.Sprintf("%s, %s", result, arg.String())
	})

	return result, nil
}

// extractFunctionArgsTypes extracts the function argument types from a raw function name string.
// It expects the raw name to be in the format "function_name(args)".
// It returns a string containing the types of the arguments, separated by comma.
func extractFunctionArgsTypes(rawName string) (string, error) {
	parsedArgs, err := extractFunctionArgsParsed(rawName)
	if err != nil {
		return "", err
	}

	result := helpers.SliceReduce(parsedArgs, func(result string, arg PgFunctionArgType) string {
		if result == "" {
			return arg.Type
		}
		return fmt.Sprintf("%s, %s", result, arg.Type)
	})

	return result, nil
}
