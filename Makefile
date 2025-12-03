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