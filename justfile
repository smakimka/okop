# justfile
set windows-shell := ["C:\\Program Files\\Git\\bin\\sh.exe","-c"]
# Postgres connection settings. Names and defaults match internal/config/postgres.go,
# so overriding the environment affects just recipes and the Go binaries alike.
PG_HOST := env_var_or_default("POSTGRES_HOST", "localhost")
PG_PORT := env_var_or_default("POSTGRES_PORT", "5432")
PG_DB := env_var_or_default("POSTGRES_DB", "games_db")
PG_USER := env_var_or_default("POSTGRES_USER", "parser")
PG_PASSWORD := env_var_or_default("POSTGRES_PASSWORD", "parser_password")
PG_SSLMODE := env_var_or_default("POSTGRES_SSLMODE", "disable")

# Set the database connection string as a variable
DB_URL := "postgres://" + PG_USER + ":" + PG_PASSWORD + "@" + PG_HOST + ":" + PG_PORT + "/" + PG_DB + "?sslmode=" + PG_SSLMODE

# Task to create a new migration
new-migration migration_name:
    goose -dir migrations create {{migration_name}} sql

# Task to apply all pending migrations
migrate:
    goose postgres {{DB_URL}} -dir migrations up

# Task to rollback the last migration
rollback:
    goose postgres {{DB_URL}}  -dir migrations down

# Task to show the current migration status
migration-status:
    goose postgres {{DB_URL}}  -dir migrations status

#dump db into dump.sql
dump:
    PGPASSWORD="{{PG_PASSWORD}}" pg_dump -U {{PG_USER}} -h {{PG_HOST}} -p {{PG_PORT}} -F p {{PG_DB}} > dump.sql

# simple go run parser
run:
    go run cmd/parser/main.go

# docker build, docker up
docker-run:
    sudo docker compose build
    sudo docker compose up
