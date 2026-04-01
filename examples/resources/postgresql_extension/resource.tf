resource "postgresql_extension" "uuid_ossp" {
  name = "uuid-ossp"
}

resource "postgresql_extension" "pg_trgm" {
  name   = "pg_trgm"
  schema = "public"
}

resource "postgresql_extension" "postgis" {
  name    = "postgis"
  version = "3.4.0"
  cascade = true
}
