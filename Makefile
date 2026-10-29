SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -c

# Variables
BINARY_NAME := ludustudii
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BUILD_DIR := backend/bin
SPA_EMBED_DIR := backend/internal/spa/dist
MAIN_REL_DIR := ./cmd/server/api
MIGRATIONS_DIR := backend/migrations

.DEFAULT_GOAL := help

.PHONY: help
## help: Lists available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'

$(SPA_EMBED_DIR)/index.html:
	@mkdir -p $(SPA_EMBED_DIR)
	@printf '<!doctype html><title>ludustudii (placeholder)</title>\n' > $@

dev test-unit test-integration: $(SPA_EMBED_DIR)/index.html

.PHONY: dev

## dev: Runs Vite dev server and Go backend (with Air live-reload) concurrently
dev:
	@echo "==> Starting local development servers..."
	@trap 'kill 0' INT TERM EXIT; \
		(cd $(BACKEND_DIR) && air) & \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		wait

.PHONY: test test-unit test-integration

## test: Runs both unit and integration tests
test: test-unit test-integration

## test-unit: Runs unit tests with the 'unit' build tag
test-unit:
	@echo "==> Running unit tests..."
	cd $(BACKEND_DIR) && go test -tags=unit ./... | grep --line-buffered -v '^?'

## test-integration: Runs integration tests with the 'integration' build tag
test-integration:
	@echo "==> Running integration tests..."
	cd $(BACKEND_DIR) && go test -tags=integration -race -count=1 ./... | grep --line-buffered -v '^?'

.PHONY: build build-frontend build-backend clean

## build: Compiles frontend, outputs assets, and compiles Go binary
build:
	$(MAKE) build-frontend
	$(MAKE) build-backend

## build-frontend: Installs dependencies and builds Vite static output
build-frontend:
	@echo "==> Building frontend bundle..."
	cd $(FRONTEND_DIR) && npm ci && npm run build
	@echo "==> Frontend bundle built successfully."

## build-backend: Compiles static Go binary with embedded SPA
build-backend:
	@! grep -qs 'ludustudii (placeholder)' $(SPA_EMBED_DIR)/index.html || \
		{ echo "SPA not built: run 'make build-frontend' first"; exit 1; }
	@echo "==> Building Go binary..."
	mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BUILD_DIR)/$(BINARY_NAME) $(MAIN_REL_DIR)
	@echo "==> Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Deletes build outputs and temporary embed files
clean:
	@echo "==> Cleaning artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(BACKEND_DIR)/tmp
	rm -rf $(SPA_EMBED_DIR)
	@echo "==> Clean complete."

.PHONY: migrate-create migrate-up migrate-down migrate-status

## migrate-create: Creates a new database migration file with the given NAME
## ---- Usage: make migrate-create NAME=migration_name
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g., make migrate-create NAME=add_users)"; exit 1; fi
	@goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

## migrate-up: Applies all pending database migrations
migrate-up:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" up
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" version

## migrate-down: Rolls back the most recently applied database migration (down by one)
migrate-down:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" down
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" version

## migrate-status: Shows the status of all database migrations
migrate-status:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" status

.PHONY: psql

## psql: Connects to the PostgreSQL database using DB_CONNECTION environment variable
psql:
	@echo "==> Connecting to PostgreSQL..."
	@psql "$(DB_CONNECTION)"

.PHONY: git-diff

## git-diff: Copies the git diff to the system clipboard in a terminal-friendly format
git-diff:
	git diff HEAD | base64 | tr -d '\n' | awk '{print "\033]52;c;" $$0 "\007"}'
