package pgclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackc/pgx/v5"
	"regexp"
	"strings"
)

// SanitationType defines the type of sanitization to apply to input strings.
type SanitationType int

const (
	// SanitizeIdentifier indicates that the input should be sanitized as a PostgreSQL identifier
	// (e.g., table name, column name, function name).
	SanitizeIdentifier SanitationType = iota

	// SanitizeReturns indicates that the input should be sanitized as a PostgreSQL return type
	// (e.g., integer, text[], etc.).
	SanitizeReturns
)

// DeferredRollback provides a safe way to roll back a transaction in case of errors
func DeferredRollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		tflog.Error(ctx, "Error rolling back transaction", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// sanitizeInput sanitizes the input string based on the specified sanitation type.
// It delegates to the appropriate sanitization function based on the type.
func sanitizeInput(value string, sType SanitationType) (string, error) {
	switch sType {
	case SanitizeIdentifier:
		return sanitizeIdentifierInput(value)
	case SanitizeReturns:
		return sanitizeReturnsTypeInput(value)
	default:
		return "", fmt.Errorf("invalid sanitation type")
	}
}

// sanitizeIdentifierInput sanitizes a PostgreSQL identifier (like table names, column names, etc.)
// to prevent SQL injection. It checks that the identifier matches a safe pattern and
// sanitizes it if it contains multiple parts separated by dots.
func sanitizeIdentifierInput(value string) (string, error) {
	// make sure the string does not contain characters leading to SQL injection
	nameRE := regexp.MustCompile(`^[a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)*$`)
	if !nameRE.MatchString(value) {
		return "", fmt.Errorf("the value contains non-accepted characters")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		value = pgx.Identifier(parts).Sanitize()
	}

	return value, nil
}

// sanitizeReturnsTypeInput sanitizes a PostgreSQL return type (like 'integer', 'text[]', etc.)
// to prevent SQL injection. It handles array types (with '[]' suffix) and delegates to
// sanitizeIdentifierInput for the base type.
func sanitizeReturnsTypeInput(returns string) (string, error) {
	// make sure the string does not contain characters leading to SQL injection
	var returnsRE = regexp.MustCompile(`^[a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)*(?:\[])?$`)
	if !returnsRE.MatchString(returns) {
		return "", fmt.Errorf("the value contains non-accepted characters")
	}

	var err error
	var result, withSuffix string

	if strings.HasSuffix(returns, "[]") {
		withSuffix = "[]"
		returns = strings.TrimSuffix(returns, "[]")
	}

	result, err = sanitizeIdentifierInput(returns)
	if err != nil {
		return result, err
	}

	return result + withSuffix, nil
}
