package client

import (
	"fmt"
	"github.com/lib/pq"
)

type PGFunctionQueries interface {
	CheckFunctionExistsQuery(execFuncName string) string
	CreateFunctionQuery(fName, fReturns, fBody, fLang string, replaceIfExists bool) string
	DropFunctionQuery(fName string) string
}

// Ensure the implementation satisfies the expected interfaces.
var _ PGFunctionQueries = &pgClientModel{}

func (cli *pgClientModel) CheckFunctionExistsQuery(execFuncName string) string {
	query := `-- Check if the function exists
		SELECT EXISTS (SELECT 1
		   FROM pg_catalog.pg_proc f
		   WHERE f.proname = %s);`

	return fmt.Sprintf(query, pq.QuoteLiteral(execFuncName))
}

func (cli *pgClientModel) CreateFunctionQuery(fName, fReturns, fBody, fLang string, replaceIfExists bool) string {
	query := `-- Create Function
		CREATE %s FUNCTION %s() RETURNS %s AS
		$$
		%s
		$$ LANGUAGE %s;`

	orReplace := ""
	if replaceIfExists {
		orReplace = "OR REPLACE"
	}

	return fmt.Sprintf(query, orReplace, fName, fReturns, fBody, fLang)
}
func (cli *pgClientModel) DropFunctionQuery(fName string) string {
	query := `-- Drop Function
		DROP FUNCTION IF EXISTS %s();`

	return fmt.Sprintf(query, fName)
}
