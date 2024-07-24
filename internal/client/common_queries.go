package client

import (
	"fmt"
	"github.com/lib/pq"
)

type PGCommonQueries interface {
	AlterObjectNameQuery(oType, oOldName, oNewName string) string
	AlterObjectOwnerQuery(oType, oName, oOwner string) string
	CreateCommentQuery(oType, oName, comment string) string
	DropObjectQuery(oType, oName string) string
	GetObjectOwnerQuery(oDb, oOwnerCol, oFilterCol, oFilterVal string) string
}

var _ PGCommonQueries = &pgClientModel{}

func (cli *pgClientModel) AlterObjectNameQuery(oType, oOldName, oNewName string) string {
	rawQuery := `-- Update Object Name
		ALTER %s %s RENAME TO %s;`

	return fmt.Sprintf(rawQuery, oType, oOldName, oNewName)
}

func (cli *pgClientModel) AlterObjectOwnerQuery(oType, oName, oOwner string) string {
	rawQuery := `-- Update Object Owner
		ALTER %s %s OWNER TO %s;`

	return fmt.Sprintf(rawQuery, oType, oName, oOwner)
}

func (cli *pgClientModel) CreateCommentQuery(oType, oName, comment string) string {
	query := `-- Comment on Object
		COMMENT ON %s %s IS %s;`

	return fmt.Sprintf(query, oType, oName, pq.QuoteLiteral(comment))
}

func (cli *pgClientModel) DropObjectQuery(oType, oName string) string {
	rawQuery := `-- Drop Object
		DROP %s IF EXISTS %s;`

	return fmt.Sprintf(rawQuery, oType, oName)
}

func (cli *pgClientModel) GetObjectOwnerQuery(oDb, oOwnerCol, oFilterCol, oFilterVal string) string {
	query := `-- Get Object Owner
		SELECT pg_catalog.pg_get_userbyid(o.%s) as "owner"
		FROM pg_catalog.%s o
		WHERE o.%s = %s;`

	return fmt.Sprintf(query, oOwnerCol, oDb, oFilterCol, pq.QuoteLiteral(oFilterVal))
}
