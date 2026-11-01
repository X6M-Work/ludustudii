# Database migration commands

Run these commands from the `backend` directory. Set `DB_CONNECTION` to your
PostgreSQL connection string before running commands that access the database.

```sh
# Create a new SQL migration file in migrations/
go tool goose -dir ./migrations create migration_name sql

# Apply all pending migrations
go tool goose -dir ./migrations postgres "$DB_CONNECTION" up

# Apply migrations up to the specified version
go tool goose -dir ./migrations postgres "$DB_CONNECTION" up-to 20170506082420

# Apply the next pending migration
go tool goose -dir ./migrations postgres "$DB_CONNECTION" up-by-one

# Roll back the most recently applied migration
go tool goose -dir ./migrations postgres "$DB_CONNECTION" down

# Roll back migrations newer than the specified version
go tool goose -dir ./migrations postgres "$DB_CONNECTION" down-to 20170506082420

# Roll back all migrations
go tool goose -dir ./migrations postgres "$DB_CONNECTION" down-to 0

# Show the current migration version
go tool goose -dir ./migrations postgres "$DB_CONNECTION" version

# Show the status of all migrations
go tool goose -dir ./migrations postgres "$DB_CONNECTION" status
```
