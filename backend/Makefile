stage := $(or dev)
include  Makefile.$(stage)

sqlc:
	~/go/bin/sqlc generate

MIGRATION_NAME=$(or $(MIGRATION), init)
migrate-create:
	mkdir -p ./storage/schema
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) create $(MIGRATION_NAME) sql

migrate-up:
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) up
migrate-redo:
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) redo
migrate-down:
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) down
migrate-reset:
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) reset
migrate-status:
	~/go/bin/goose -dir ./storage/schema -table schema_migrations postgres $(POSTGRES_URI) status
