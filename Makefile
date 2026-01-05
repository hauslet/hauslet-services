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
	@echo ""
	@echo "Database:"
	@echo "  make migrate       # Run database migrations (export DATABASE_URL first)"
	@echo "  make auto-migrate  # Run GORM auto-migrations (cmd/migrate)"
	@echo "  make reset-db      # Reset public schema (export DATABASE_URL first)"
	@echo "  make seed          # Seed database with realistic test data"
	@echo "  make seed-test     # Seed minimal test data (fast)"
	@echo "  make seed-clear    # Clear all data and reseed"
	@echo "  make fresh         # Reset DB, migrate, and seed (full reset)"
	@echo "  make start-proxy    # Start Cloud SQL Proxy (export SQL_INSTANCE first)"


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

# Scaffold a new module under internal/modules/<MODULE>
.PHONY: scaffold
scaffold:
	@if [ -z "$(MODULE)" ]; then echo "MODULE is required, e.g. make scaffold MODULE=booking"; exit 1; fi
	@mkdir -p internal/modules/$(MODULE)/service internal/modules/$(MODULE)/repository/schema internal/modules/$(MODULE)/domain internal/modules/$(MODULE)/docs internal/modules/$(MODULE)/templates
	@echo "Scaffolded internal/modules/$(MODULE) with service, repository/schema, domain, docs, templates"

.PHONY: gql-gen
gql-gen:
	$(GO) tool gqlgen generate

.PHONY: migrate
migrate :
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required, e.g. export DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable"; exit 1; fi
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

.PHONY: auto-migrate 
migrate-auto:
	$(GO) run ./cmd/migrate

	
.PHONY: reset-db
reset-db:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is required. Example:"; \
		echo "  export DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable"; \
		echo "  make reset-db"; \
		exit 1; \
	fi
	psql "$(DATABASE_URL)" -f db/utils/reset_public_schema.sql

.PHONY: seed
seed:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is required. Example:"; \
		echo "  export DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable"; \
		echo "  make seed"; \
		exit 1; \
	fi
	$(GO) run ./db/seeds

.PHONY: seed-test
seed-test:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is required"; \
		exit 1; \
	fi
	$(GO) run ./db/seeds -config=test

.PHONY: seed-clear
seed-clear:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is required"; \
		exit 1; \
	fi
	$(GO) run ./db/seeds -clear

.PHONY: fresh
fresh: reset-db migrate seed


.PHONY: start-proxy
start-proxy:
	@if [ -z "$(SQL_INSTANCE)" ]; then \
		echo "SQL_INSTANCE is required, e.g. export SQL_INSTANCE=project:region:instance"; \
		exit 1; \
	fi
	cloud-sql-proxy $(SQL_INSTANCE) --address 127.0.0.1 --port 5432
