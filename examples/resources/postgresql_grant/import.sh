# Import format: object_type:schema:object_name:role
# For database-level objects, use an empty schema segment.

# Import a schema grant
terraform import postgresql_grant.schema_usage "schema::app_schema:app_user"

# Import a database grant (empty schema segment)
terraform import postgresql_grant.db_connect "database::mydb:readonly_user"

# Import a table grant with schema
terraform import postgresql_grant.table_select "table:public:users:readonly_user"
