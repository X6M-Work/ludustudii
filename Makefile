# Variables
BINARY_NAME := ludustudii 
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BUILD_DIR := backend/bin
SPA_EMBED_DIR := backend/internal/spa/dist
MAIN_REL_DIR := ./cmd/server/api
MIGRATIONS_DIR := backend/migrations

.PHONY: build build-frontend build-backend clean dev migrate-up migrate-down migrate-status migrate-create

## build: Compiles frontend, outputs assets, and compiles Go binary
build: build-frontend build-backend

## build-frontend: Installs dependencies and builds Vite static output
build-frontend:
	@echo "==> Building frontend bundle..."
	cd $(FRONTEND_DIR) && npm ci && npm run build
	@echo "==> Frontend bundle built successfully."

## build-backend: Compiles static Go binary with embedded SPA
build-backend:
	@echo "==> Building Go binary..."
	mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BUILD_DIR)/$(BINARY_NAME) $(MAIN_REL_DIR)
	@echo "==> Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## dev: Runs Vite dev server and Go backend (with Air live-reload) concurrently
dev:
	@echo "==> Starting local development servers..."
	@mkdir -p $(SPA_EMBED_DIR)
	@[ -f $(SPA_EMBED_DIR)/index.html ] || touch $(SPA_EMBED_DIR)/index.html
	@trap 'kill 0' INT TERM EXIT; \
		(cd $(BACKEND_DIR) && air) & \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		wait

## clean: Deletes build outputs and temporary embed files
clean:
	@echo "==> Cleaning artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(BACKEND_DIR)/tmp
	rm -rf $(SPA_EMBED_DIR)
	@echo "==> Clean complete."

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

## migrate-create: Creates a new database migration file with the given NAME
## Usage: make migrate-create NAME=migration_name
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g., make migrate-create NAME=add_users)"; exit 1; fi
	@goose -dir $(MIGRATIONS_DIR) create $(NAME) sql
