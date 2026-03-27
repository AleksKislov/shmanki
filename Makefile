# Use `.env` if it exists locally, otherwise fall back to the committed example file.
ENV_FILE := $(if $(wildcard .env),.env,.env.example)
# Reuse one Docker Compose command with the chosen env file.
COMPOSE := docker compose -f compose.dev.yaml --env-file $(ENV_FILE)
# Use the official golang-migrate container instead of requiring a local install.
MIGRATE_IMAGE := migrate/migrate:v4.18.3

# Default database name for local development.
POSTGRES_DB ?= shmanki
# Default database user for local development.
POSTGRES_USER ?= postgres
# Default database password for local development.
POSTGRES_PASSWORD ?= postgres
# Default host port exposed by the Postgres container.
POSTGRES_PORT ?= 5432

# Only try to include env values if the selected env file actually exists.
ifneq ($(wildcard $(ENV_FILE)),)
# Load variables from `.env` or `.env.example` into Make.
include $(ENV_FILE)
# End the conditional include block.
endif

# Connection string used by the migration container to reach Postgres on the host.
MIGRATE_DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@host.docker.internal:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Folder that contains the Go backend module.
BACKEND_DIR := backend

# Name of the built backend binary.
APP_BIN := $(BACKEND_DIR)/bin/api

# Mark these names as command targets, not files.
.PHONY: deps run build test lint db-up db-down db-logs db-psql db-reset migrate-up migrate-down seed

# Download and lock backend Go dependencies inside `backend/go.sum`.
deps:
	cd $(BACKEND_DIR) && go mod tidy

# Run the backend API server from the main package.
run:
	cd $(BACKEND_DIR) && go run ./cmd/api

# Build the backend API binary into `bin/api`.
build:
	mkdir -p $(BACKEND_DIR)/bin
	cd $(BACKEND_DIR) && go build -o bin/api ./cmd/api

# Run all Go tests in the repository.
test:
	cd $(BACKEND_DIR) && go test ./...

# Run `go test` as a minimal lint/verification step until golangci-lint is added.
lint:
	cd $(BACKEND_DIR) && go test ./...

# Start the local Postgres container in the background.
db-up:
	$(COMPOSE) up -d db

# Stop and remove the local Compose services.
db-down:
	$(COMPOSE) down

# Follow live logs from the Postgres container.
db-logs:
	$(COMPOSE) logs -f db

# Open an interactive `psql` shell inside the running Postgres container.
db-psql:
	$(COMPOSE) exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# Remove the database container and volume, then start a fresh empty database.
db-reset:
	$(COMPOSE) down -v --remove-orphans
	$(COMPOSE) up -d db

# Ensure the database is running before applying all pending migrations.
migrate-up: db-up
	docker run --rm -v "$(PWD)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(MIGRATE_DATABASE_URL)" up

# Ensure the database is running before rolling back the most recent migration.
migrate-down: db-up
	docker run --rm -v "$(PWD)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(MIGRATE_DATABASE_URL)" down 1

# Ensure the database is running before loading local demo data.
seed: db-up
	docker exec -i "$$( $(COMPOSE) ps -q db )" psql -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" -v ON_ERROR_STOP=1 < seeds/dev.sql
