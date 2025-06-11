package pgclient

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type PgObjectType int

const (
	PgObjFunction PgObjectType = iota
	PgObjEventTrigger
	PgObjDatabase
	PgObjRole
)

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type PgSchemaObject struct {
	Database pgtype.Text `json:"database" validate:"required"`
	Schema   pgtype.Text `json:"schema" validate:"required"`
	Owner    pgtype.Text `json:"owner" validate:"required"`
	Comment  pgtype.Text `json:"comment"`
}

func (p PgObjectType) String() string {
	switch p {
	case PgObjFunction:
		return "FUNCTION"
	case PgObjEventTrigger:
		return "EVENT TRIGGER"
	case PgObjDatabase:
		return "DATABASE"
	case PgObjRole:
		return "ROLE"
	default:
		return "UNKNOWN"
	}
}
