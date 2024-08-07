package client

import (
	"errors"
	"fmt"
	"github.com/lib/pq"
)

type pgError struct {
	err      error
	metadata []any
}

// Error returns the original error message,
// ensuring compatibility with the standard error interface.
func (e *pgError) Error() string {
	return e.err.Error()
}

// Unwrap allows errors wrapped by errMetadata to be compatible with
// standard error unwrapping mechanism.
func (e *pgError) Unwrap() error {
	return e.err
}

func PgErrWithMetadata(err error, pairs ...any) error {
	data := make([]any, 0)

	var e *pq.Error
	if errors.As(err, &e) {
		data = append(data, []string{"code", string(e.Code)})
		data = append(data, []string{"code_name", e.Code.Name()})
		data = append(data, []string{"message", e.Message})
	}

	pairs = append(data, pairs...)

	return &pgError{
		err:      err,
		metadata: pairs,
	}
}

func WrapPgError(err error, message string) error {
	data := make([]any, 0)

	var e *pq.Error
	if errors.As(err, &e) {
		data = append(data, []string{"code", string(e.Code)})
		data = append(data, []string{"code_name", e.Code.Name()})
		data = append(data, []string{"message", e.Message})
	}

	return &pgError{
		err:      fmt.Errorf(message),
		metadata: data,
	}
}
