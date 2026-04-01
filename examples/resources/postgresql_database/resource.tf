resource "postgresql_database" "mydb" {
  name              = "mydb"
  owner             = "postgres"
  encoding          = "UTF8"
  collation         = "en_US.UTF-8"
  ctype             = "en_US.UTF-8"
  template          = "template0"
  connection_limit  = 100
  allow_connections = true
  is_template       = false
  tablespace        = "pg_default"
  comment           = "My application database"
  force_drop        = false
}
