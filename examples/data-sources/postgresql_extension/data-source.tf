data "postgresql_extension" "uuid_ossp" {
  name = "uuid-ossp"
}

output "extension_version" {
  value = data.postgresql_extension.uuid_ossp.version
}
