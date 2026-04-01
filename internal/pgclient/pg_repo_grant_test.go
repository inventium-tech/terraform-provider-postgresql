package pgclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGrantQuery(t *testing.T) {
	tests := []struct {
		name     string
		params   GrantParams
		expected string
	}{
		{
			name: "Grant SELECT on table",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "users",
				Schema:     "public",
				Role:       "app_user",
				Privileges: []string{"SELECT"},
			},
			expected: `GRANT SELECT ON TABLE "public"."users" TO "app_user"`,
		},
		{
			name: "Grant ALL on database",
			params: GrantParams{
				ObjectType: "database",
				ObjectName: "mydb",
				Role:       "admin",
				Privileges: []string{"ALL"},
			},
			expected: `GRANT ALL PRIVILEGES ON DATABASE "mydb" TO "admin"`,
		},
		{
			name: "Grant with grant option",
			params: GrantParams{
				ObjectType:      "schema",
				ObjectName:      "myschema",
				Role:            "manager",
				Privileges:      []string{"USAGE"},
				WithGrantOption: true,
			},
			expected: `GRANT USAGE ON SCHEMA "myschema" TO "manager" WITH GRANT OPTION`,
		},
		{
			name: "Grant USAGE on sequence with schema",
			params: GrantParams{
				ObjectType: "sequence",
				ObjectName: "my_seq",
				Schema:     "app",
				Role:       "writer",
				Privileges: []string{"USAGE"},
			},
			expected: `GRANT USAGE ON SEQUENCE "app"."my_seq" TO "writer"`,
		},
		{
			name: "Lowercase privileges normalized to uppercase",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "orders",
				Schema:     "public",
				Role:       "reader",
				Privileges: []string{"select", "insert"},
			},
			expected: `GRANT SELECT, INSERT ON TABLE "public"."orders" TO "reader"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildGrantQuery(tt.params)
			assert.Equal(t, tt.expected, query)
		})
	}
}

func TestBuildRevokeQuery(t *testing.T) {
	tests := []struct {
		name     string
		params   RevokeParams
		expected string
	}{
		{
			name: "Revoke SELECT from table",
			params: RevokeParams{
				ObjectType: "table",
				ObjectName: "users",
				Schema:     "public",
				Role:       "app_user",
				Privileges: []string{"SELECT"},
			},
			expected: `REVOKE SELECT ON TABLE "public"."users" FROM "app_user"`,
		},
		{
			name: "Revoke with cascade",
			params: RevokeParams{
				ObjectType: "database",
				ObjectName: "mydb",
				Role:       "admin",
				Privileges: []string{"ALL"},
				Cascade:    true,
			},
			expected: `REVOKE ALL PRIVILEGES ON DATABASE "mydb" FROM "admin" CASCADE`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildRevokeQuery(tt.params)
			assert.Equal(t, tt.expected, query)
		})
	}
}

func TestValidateGrantParams(t *testing.T) {
	tests := []struct {
		name    string
		params  GrantParams
		wantErr bool
	}{
		{
			name: "Valid params",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "users",
				Role:       "app_user",
				Privileges: []string{"SELECT"},
			},
			wantErr: false,
		},
		{
			name: "Missing object type",
			params: GrantParams{
				ObjectName: "users",
				Role:       "app_user",
				Privileges: []string{"SELECT"},
			},
			wantErr: true,
		},
		{
			name: "Missing privileges",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "users",
				Role:       "app_user",
				Privileges: []string{},
			},
			wantErr: true,
		},
		{
			name: "Invalid privilege",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "users",
				Role:       "app_user",
				Privileges: []string{"INVALID_PRIV"},
			},
			wantErr: true,
		},
		{
			name: "SQL injection attempt in privilege",
			params: GrantParams{
				ObjectType: "table",
				ObjectName: "users",
				Role:       "app_user",
				Privileges: []string{"SELECT; DROP TABLE users;--"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGrantParams(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePrivileges is a pure unit test — no database required.
func TestValidatePrivileges(t *testing.T) {
	tests := []struct {
		name       string
		privileges []string
		wantErr    bool
	}{
		{"valid SELECT uppercase", []string{"SELECT"}, false},
		{"valid lowercase select", []string{"select"}, false},
		{"valid mixed case", []string{"Select"}, false},
		{"valid multiple", []string{"SELECT", "INSERT", "UPDATE", "DELETE"}, false},
		{"valid ALL", []string{"ALL"}, false},
		{"valid USAGE", []string{"USAGE"}, false},
		{"valid EXECUTE", []string{"EXECUTE"}, false},
		{"invalid privilege", []string{"INVALID_PRIV"}, true},
		{"SQL injection attempt", []string{"SELECT; DROP TABLE users;--"}, true},
		{"empty privilege", []string{""}, true},
		{"mix valid and invalid", []string{"SELECT", "INVALID"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrivileges(tt.privileges)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildObjectReference is a pure unit test — no database required.
func TestBuildObjectReference(t *testing.T) {
	tests := []struct {
		name       string
		objectType string
		objectName string
		schema     string
		expected   string
	}{
		{"database", "database", "mydb", "", `DATABASE "mydb"`},
		{"schema", "schema", "myschema", "", `SCHEMA "myschema"`},
		{"table without schema", "table", "users", "", `TABLE "users"`},
		{"table with schema", "table", "users", "public", `TABLE "public"."users"`},
		{"sequence without schema", "sequence", "my_seq", "", `SEQUENCE "my_seq"`},
		{"sequence with schema", "sequence", "my_seq", "app", `SEQUENCE "app"."my_seq"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildObjectReference(tt.objectType, tt.objectName, tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}
