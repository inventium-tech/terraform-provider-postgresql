# Grant USAGE on a schema
resource "postgresql_grant" "schema_usage" {
  object_type = "schema"
  object_name = "app_schema"
  role        = "app_user"
  privileges  = ["USAGE", "CREATE"]
}

# Grant privileges on a database
resource "postgresql_grant" "db_connect" {
  object_type = "database"
  object_name = "mydb"
  role        = "readonly_user"
  privileges  = ["CONNECT"]
}

# Grant SELECT on a table
resource "postgresql_grant" "table_select" {
  object_type = "table"
  object_name = "users"
  schema      = "public"
  role        = "readonly_user"
  privileges  = ["SELECT"]
}

# Grant ALL on a sequence with grant option
resource "postgresql_grant" "sequence_all" {
  object_type       = "sequence"
  object_name       = "orders_id_seq"
  schema            = "public"
  role              = "app_user"
  privileges        = ["ALL"]
  with_grant_option = true
}
