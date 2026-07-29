# justfile
set windows-shell := ["C:\\Program Files\\Git\\bin\\sh.exe","-c"]
# Set the database connection string as a variable
DB_URL := "postgres://parser:parser_password@localhost:5432/parser_db?sslmode=disable"

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
    PGPASSWORD="parser_password" pg_dump -U parser -h localhost -F p parser_db > dump.sql

# simple go run parser
run:
    go run cmd/parser/main.go

# docker build, docker up
docker-run:
    sudo docker compose build
    sudo docker compose up
