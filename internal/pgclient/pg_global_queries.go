package pgclient

import (
	"context"
	"fmt"
)

const (
	createCommentQuery = `COMMENT ON %s %s IS '%s';`
	alterOwnerQuery    = `ALTER %s %s OWNER TO %s;`
)

// CreateComment adds a comment to a specified PostgreSQL object if a comment is provided. Returns an error on failure.
// Note: If the comment is an empty string, the function does nothing and returns nil.
func CreateComment[T DBTX](ctx context.Context, d T, objType PgObjectType, objName, comment string) error {
	if comment == "" {
		return nil
	}

	sqlQuery := fmt.Sprintf(createCommentQuery, objType.String(), objName, comment)
	_, err := d.Exec(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to create comment on object '%s %s': %v", objType.String(), objName, err)
	}
	return nil
}

// AlterOwner modifies the owner of a specified PostgreSQL object to the given new owner.
// Note: If the new owner is an empty string, the function does nothing and returns nil.
func AlterOwner[T DBTX](ctx context.Context, d T, objType PgObjectType, objName, newOwner string) error {
	if newOwner == "" {
		return nil
	}

	sqlQuery := fmt.Sprintf(alterOwnerQuery, objType.String(), objName, newOwner)
	_, err := d.Exec(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to alter owner of object '%s %s': %v", objType.String(), objName, err)
	}
	return nil
}

// DropObject removes a PostgreSQL object of the specified type and name if it exists in the connected database.
func DropObject[T DBTX](ctx context.Context, d T, objType PgObjectType, objId string) error {
	sqlQuery := fmt.Sprintf("DROP %s IF EXISTS %s;", objType.String(), objId)
	_, err := d.Exec(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to drop object '%s %s': %v", objType.String(), objId, err)
	}
	return nil
}

// RenameObject changes the name of a PostgreSQL object with the specified type and ID to the new name.
func RenameObject[T DBTX](ctx context.Context, d T, objType PgObjectType, objId, newName string) error {
	sqlQuery := fmt.Sprintf("ALTER %s %s RENAME TO %s;", objType.String(), objId, newName)
	_, err := d.Exec(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to rename object '%s %s' to '%s': %v", objType.String(), objId, newName, err)
	}
	return nil
}
