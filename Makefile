# Simple build/run helpers for Hauslet Services

APP_NAME := hauslet
GO       := go
BIN_DIR  := bin




.PHONY: help
help:
	@echo "Usage:"
	@echo "  make run-api       # Run API server (cmd/api)"
	@echo "  make run-worker    # Run worker (cmd/worker)"
	@echo "  make build-api     # Build API binary -> $(BIN_DIR)/api"
	@echo "  make build-worker  # Build worker binary -> $(BIN_DIR)/worker"
	@echo "  make test          # go test ./..."
	@echo "  make compose-up    # docker-compose up -d"
	@echo "  make compose-down  # docker-compose down"
	@echo "  make scaffold MODULE=name # Scaffold internal/MODULE structure"
	@echo "  make migrate       # Run database migrations (export DATABASE_URL first)"
	@echo "  make reset-db      # Reset public schema (drops all tables except spatial_ref_sys)"
	@echo "  make seed          # Seed database with dev data (db/seeds/dev.sql)"
	@echo "  make fresh         # Reset DB, migrate, and seed"


$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: run-api
run-api:
	$(GO) run ./cmd/api

.PHONY: run-worker
run-worker:
	$(GO) run ./cmd/worker

.PHONY: build-api
build-api: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/api ./cmd/api

.PHONY: build-worker
build-worker: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/worker ./cmd/worker

.PHONY: test
test:
	$(GO) test ./...

.PHONY: compose-up
compose-up:
	docker compose up -d

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

# Scaffold a new module under internal/
.PHONY: scaffold
scaffold:
	@if [ -z "$(MODULE)" ]; then echo "MODULE is required, e.g. make scaffold MODULE=booking"; exit 1; fi
	@mkdir -p internal/$(MODULE)/service internal/$(MODULE)/repository/schema internal/$(MODULE)/domain internal/$(MODULE)/docs internal/$(MODULE)/templates
	@echo "Scaffolded internal/$(MODULE) with service, repository/schema, domain, docs, templates"

.PHONY: gql-gen
gql-gen:
	$(GO) tool gqlgen generate

.PHONY: migrate
migrate :
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required, e.g. export DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable"; exit 1; fi
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

	
.PHONY: reset-db
reset-db:
	psql -U firstnuel -d hauslet -f db/utils/reset_public_schema.sql

.PHONY: seed
seed:
	psql -U firstnuel -d hauslet -f db/seeds/dev.sql

.PHONY: fresh
fresh: reset-db migrate seed