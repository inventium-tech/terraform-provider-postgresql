package pgclient

import (
	"database/sql/driver"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"regexp"
	"strings"
)

type (
	UserFunctionModel struct {
		Name         pgtype.Text    `json:"name" validate:"required"`
		Args         PgFunctionArgs `json:"args" validate:"required"`
		Returns      pgtype.Text    `json:"returns" validate:"required"`
		Language     pgtype.Text    `json:"language" validate:"required,oneof=plpgsql sql"`
		Body         pgtype.Text    `json:"body" validate:"required"`
		AllowReplace pgtype.Bool    `json:"allow_replace" validate:"boolean"`
		Database     pgtype.Text    `json:"database" validate:"required"`
		Schema       pgtype.Text    `json:"schema" validate:"required"`
		Owner        pgtype.Text    `json:"owner" validate:"required"`
		Comment      pgtype.Text    `json:"comment"`
	}
	PgFunctionArgs struct {
		value []PgFunctionArgType
		Valid bool
	}
	PgFunctionArgType struct {
		Name    string `json:"name" validate:"required"`
		Type    string `json:"type" validate:"required"`
		Mode    string `json:"mode,omitempty"`
		Default string `json:"default,omitempty"`
	}
)

func (f PgFunctionArgs) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return f.value, nil
}

func (f *PgFunctionArgs) Scan(src any) error {
	vStr, ok := src.(string)
	if !ok {
		return fmt.Errorf("failed type assertion to string, received: %T", src)
	}

	parsedArgs, err := ParseRawFunctionArgs(vStr)
	if err != nil {
		return fmt.Errorf("failed to parse function arguments from string %q: %w", vStr, err)
	}

	f.value = parsedArgs
	f.Valid = true

	return nil
}

func (f *PgFunctionArgs) Items() []PgFunctionArgType {
	if f == nil || !f.Valid {
		return nil
	}
	return f.value
}

func (arg *PgFunctionArgType) String() string {
	if arg == nil {
		return ""
	}
	var parts []string
	if arg.Mode != "" && arg.Mode != "IN" {
		parts = append(parts, arg.Mode)
	}
	if arg.Name != "" {
		parts = append(parts, strings.ToLower(arg.Name))
	}
	if arg.Type != "" {
		parts = append(parts, strings.ToLower(arg.Type))
	}
	if arg.Default != "" {
		parts = append(parts, "DEFAULT")
		parts = append(parts, strings.ToLower(arg.Default))
	}
	return strings.Join(parts, " ")
}

func (arg *PgFunctionArgType) Parse(argRaw string) error {
	parsedArg, err := parseFunctionArgument(argRaw)
	if err != nil {
		return fmt.Errorf("failed to parse function argument %q: %w", argRaw, err)
	}
	*arg = parsedArg
	return nil
}

func ParseRawFunctionArgs(rawArgs string) ([]PgFunctionArgType, error) {
	result := make([]PgFunctionArgType, 0)

	rawArgs = strings.TrimSpace(rawArgs)
	if rawArgs == "" {
		return result, nil
	}

	rawArgSlice := strings.Split(rawArgs, ",")
	for _, rawArg := range rawArgSlice {
		if rawArg = strings.TrimSpace(rawArg); rawArg == "" {
			continue
		}

		arg, err := parseFunctionArgument(rawArg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse raw function argument %q: %w", rawArg, err)
		}
		result = append(result, arg)
	}

	return result, nil
}

// parseFunctionArgument parses a single raw function argument string into a PgFunctionArgType.
// It handles the argument's mode, name, type, and optional default value.
//
// The expected format is expected to be one of the following:
// - "name type" (e.g. "myarg integer")
// - "mode name type" (e.g. "OUT myarg integer")
// - "name type DEFAULT value" (e.g. "myarg integer DEFAULT 42")
// - "mode name type DEFAULT value" (e.g. "IN myarg integer DEFAULT 42")
func parseFunctionArgument(arg string) (PgFunctionArgType, error) {
	argument := PgFunctionArgType{}

	// Check if the argument has a default value
	argParts := strings.Split(arg, "DEFAULT")
	argWithoutDefault := strings.TrimSpace(argParts[0])
	if len(argParts) > 1 {
		argument.Default = strings.TrimSpace(argParts[1])
	}

	// Split the argument into parts (mode, name, type)
	parts := strings.Fields(argWithoutDefault)
	if len(parts) < 2 {
		return argument, fmt.Errorf("invalid function argument format: %s", argWithoutDefault)
	}

	// Check if the first part is a mode (IN, OUT, INOUT, VARIADIC)
	checkModeRegex := regexp.MustCompile(`^(?i)(in|out|inout|variadic)$`)
	if len(parts) > 2 && checkModeRegex.MatchString(parts[0]) {
		argument.Mode = parts[0]
		argument.Name = parts[1]
		argument.Type = parts[2]
	} else {
		argument.Name = parts[0]
		argument.Type = parts[1]
	}

	return argument, nil
}
