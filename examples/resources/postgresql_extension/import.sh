# Import by extension name (uses the provider's configured database)
terraform import postgresql_extension.uuid_ossp "uuid-ossp"

# Import with explicit database scope
terraform import postgresql_extension.uuid_ossp "mydb:uuid-ossp"
