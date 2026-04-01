resource "postgresql_schema" "myschema" {
  name          = "myschema"
  owner         = "postgres"
  if_not_exists = false
  drop_cascade  = false
  policy        = "error_on_collision"
}
