SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -c

# Variables
BINARY_NAME := ludustudii
FRONTEND_DIR := frontend
BACKEND_DIR := backend
GO_TOOL := go -C $(BACKEND_DIR) tool -modfile=tools/go.mod
BUILD_DIR := backend/bin
SPA_EMBED_DIR := backend/internal/infra/spa/dist
SCALAR_SRC := $(FRONTEND_DIR)/node_modules/@scalar/api-reference/dist/browser/standalone.js
SCALAR_EMBED := backend/internal/infra/apidocs/dist/assets/scalar.js.gz
SPEC_SRC := $(shell find backend/apispec -name '*.yml')
SPEC_BUNDLE := backend/internal/infra/apidocs/dist/openapi.json
MAIN_REL_DIR := ./cmd/server/api
# go -C runs migration commands from the backend module.
MIGRATIONS_DIR := migrations
# API_DOCS=1 builds the binary with the API reference; any other value omits it.
API_DOCS_TAGS := $(if $(filter 1,$(API_DOCS)),-tags=apidocs)
LOG_FILE ?= $(BACKEND_DIR)/tmp/app.log
PKG ?= ./...
RUN ?=
RUN_FLAG = $(if $(RUN),-run '$(RUN)')

.DEFAULT_GOAL := help
# Deletes a target whose recipe fails, such as a partly written Scalar bundle.
.DELETE_ON_ERROR:

.PHONY: help
## help: Lists available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'

$(SPA_EMBED_DIR)/index.html:
	@mkdir -p $(SPA_EMBED_DIR)
	@printf '<!doctype html><title>ludustudii (placeholder)</title>\n' > $@

# Gzips the Scalar bundle when frontend dependencies are installed. Without it
# builds with the apidocs tag still compile, and the server starts without the
# API reference. -n leaves the source name and mtime out of the header, so the
# bytes and their ETag only change with the bundle itself.
$(SCALAR_EMBED): $(wildcard $(SCALAR_SRC))
	@if [ -f $(SCALAR_SRC) ]; then gzip -9nc $(SCALAR_SRC) > $@; fi

dev test-unit test-integration: $(SPA_EMBED_DIR)/index.html
dev test-unit: $(SCALAR_EMBED)

.PHONY: spec-bundle

## spec-bundle: Regenerates the bundled OpenAPI spec the API reference serves, if a spec file changed
spec-bundle: $(SPEC_BUNDLE)

# bundlespec writes the file only once the spec bundles, so a failed run leaves
# it unchanged and .DELETE_ON_ERROR keeps it.
$(SPEC_BUNDLE): $(SPEC_SRC)
	cd $(BACKEND_DIR) && go generate ./internal/infra/apidocs

.PHONY: dev

## dev: Runs Vite dev server and Go backend (with Air live-reload) concurrently
## ---- Backend output is also appended to LOG_FILE (default: backend/tmp/app.log)
dev: export LOG_FORMAT := json
dev: export APP_API_DOCS ?= true
dev:
	@echo "==> Starting local development servers..."
	@mkdir -p $(dir $(LOG_FILE))
	@trap 'kill 0' INT TERM EXIT; \
		($(GO_TOOL) air) 2>&1 | (trap '' INT TERM; exec tee -a $(LOG_FILE)) & \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		wait

.PHONY: logs logs-errors logs-req logs-slow logs-stats

# Each target reads LOG_FILE line by line and skips lines that are not JSON
# (Air's own output) with 'fromjson?'.

## logs: Follows LOG_FILE and pretty-prints each JSON log record
logs:
	@tail -F $(LOG_FILE) | jq --unbuffered -R 'fromjson? | .'

## logs-errors: Prints ERROR and WARN records from LOG_FILE
logs-errors:
	@jq -cR 'fromjson? | select(.level == "ERROR" or .level == "WARN")' $(LOG_FILE)

## logs-req: Prints every record for the request with the given ID
## ---- Usage: make logs-req ID=<request id>
logs-req:
	@if [ -z "$(ID)" ]; then echo "Error: ID is required (e.g., make logs-req ID=abc123)"; exit 1; fi
	@jq -cR --arg id "$(ID)" 'fromjson? | select(.request_id == $$id)' $(LOG_FILE)

## logs-slow: Prints "request" records slower than MS milliseconds (default 200)
## ---- Usage: make logs-slow MS=500
logs-slow: MS ?= 200
logs-slow:
	@jq -cR --argjson ms "$(MS)" \
		'fromjson? | select(.msg == "request" and .latency > $$ms * 1000000)' $(LOG_FILE)

## logs-stats: Counts "request" records by status, method and route, most frequent first
logs-stats:
	@jq -nrR '[inputs | fromjson? | select(.msg == "request") | {status, method, route}] | group_by(.) | map({count: length} + .[0]) | sort_by(-.count) | .[] | "\(.count)\t\(.status)\t\(.method)\t\(.route)"' $(LOG_FILE)

.PHONY: test test-unit test-integration coverage

## test: Runs both unit and integration tests
test: test-unit test-integration

## test-unit: Runs unit tests with the 'unit' build tag
## ---- PKG selects packages; RUN selects test names (defaults: all packages and tests)
## ---- If selected, the apidocs package runs again with the 'apidocs' tag
test-unit:
	@echo "==> Running unit tests..."
	cd $(BACKEND_DIR) && go test -tags=unit -race -count=1 -shuffle=on $(RUN_FLAG) $(PKG) 2>&1 | { grep --line-buffered -v '^?' || [ $$? -eq 1 ]; }
	@cd $(BACKEND_DIR) && packages=$$(go list -f '{{.Dir}}' $(PKG)) && \
		if printf '%s\n' "$$packages" | grep -Fxq "$$PWD/internal/infra/apidocs"; then \
			go test -tags=unit,apidocs -race -count=1 -shuffle=on $(RUN_FLAG) ./internal/infra/apidocs/; \
		fi

## test-integration: Runs integration tests with the 'integration' build tag
## ---- PKG selects packages; RUN selects test names (defaults: all packages and tests)
test-integration:
	@echo "==> Running integration tests..."
	cd $(BACKEND_DIR) && go test -tags=integration -race -count=1 -shuffle=on $(RUN_FLAG) $(PKG) 2>&1 | { grep --line-buffered -v '^?' || [ $$? -eq 1 ]; }

## coverage: Runs all backend tests and writes coverage profile and HTML report to backend/coverage/ (requires DB_TEST_CONNECTION)
coverage: $(SPA_EMBED_DIR)/index.html $(SCALAR_EMBED)
	@mkdir -p $(BACKEND_DIR)/coverage
	go -C $(BACKEND_DIR) test -tags=unit,integration,apidocs -race -count=1 -coverpkg=./... -coverprofile=coverage/coverage.out ./...
	go -C $(BACKEND_DIR) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Serve from the repository root: python3 -m http.server 8000 --directory backend/coverage"

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
## ---- API_DOCS=1 also embeds the API reference (for dev, test and staging builds)
ifeq ($(API_DOCS),1)
build-backend: $(SCALAR_EMBED)
endif
build-backend:
	@! grep -qs 'ludustudii (placeholder)' $(SPA_EMBED_DIR)/index.html || \
		{ echo "SPA not built: run 'make build-frontend' first"; exit 1; }
ifeq ($(API_DOCS),1)
	@[ -s $(SCALAR_EMBED) ] || \
		{ echo "Scalar bundle missing: run 'make build-frontend' first"; exit 1; }
endif
	@echo "==> Building Go binary..."
	mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build $(API_DOCS_TAGS) -ldflags="-w -s" -o ../$(BUILD_DIR)/$(BINARY_NAME) $(MAIN_REL_DIR)
	@echo "==> Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Deletes build outputs and temporary embed files
clean:
	@echo "==> Cleaning artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(BACKEND_DIR)/tmp
	rm -rf $(SPA_EMBED_DIR)
	rm -f $(SCALAR_EMBED)
	@echo "==> Clean complete."

.PHONY: require-db migrate-create migrate-up migrate-down migrate-status

require-db:
	@[ -n "$(DB_CONNECTION)" ] || { echo "Error: DB_CONNECTION is not set"; exit 1; }

migrate-up migrate-down migrate-status psql: require-db

## migrate-create: Creates a new database migration file with the given NAME
## ---- Usage: make migrate-create NAME=migration_name
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g., make migrate-create NAME=add_users)"; exit 1; fi
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

## migrate-up: Applies all pending database migrations
migrate-up:
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" up
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" version

## migrate-down: Rolls back the most recently applied database migration (down by one)
migrate-down:
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" down
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" version

## migrate-status: Shows the status of all database migrations
migrate-status:
	@$(GO_TOOL) goose -dir $(MIGRATIONS_DIR) postgres "$(DB_CONNECTION)" status

.PHONY: psql

## psql: Connects to the PostgreSQL database using DB_CONNECTION environment variable
psql:
	@echo "==> Connecting to PostgreSQL..."
	@psql "$(DB_CONNECTION)"

.PHONY: git-diff

## git-diff: Copies the git diff to the system clipboard in a terminal-friendly format
git-diff:
	git diff HEAD | base64 | tr -d '\n' | awk '{print "\033]52;c;" $$0 "\007"}'

.PHONY: check

## check: Runs every gate a change must pass (fast to slow). Agents and CI run this.
# Separate recursive calls preserve order and stop on failure, including with -j.
check:
	$(MAKE) fmt-check
	$(MAKE) lint
	$(MAKE) tidy-check
	$(MAKE) generate-check
	$(MAKE) test-unit
	$(MAKE) test-integration

.PHONY: lint lint-fix fmt fmt-check lint-verify

## lint: Lints the backend and tagged tests with API docs enabled
lint: $(SPA_EMBED_DIR)/index.html $(SCALAR_EMBED)
	cd $(BACKEND_DIR) && golangci-lint run

## lint-fix: Applies backend lint fixes with API docs enabled
lint-fix: $(SPA_EMBED_DIR)/index.html $(SCALAR_EMBED)
	cd $(BACKEND_DIR) && golangci-lint run --fix

## fmt: Formats the backend with gofumpt and goimports
fmt:
	cd $(BACKEND_DIR) && golangci-lint fmt

## fmt-check: Fails if any file needs formatting
fmt-check:
	cd $(BACKEND_DIR) && golangci-lint fmt --diff

## lint-verify: Validates the backend golangci-lint configuration
lint-verify:
	cd $(BACKEND_DIR) && golangci-lint config verify

.PHONY: vuln

## vuln: Checks backend packages for known vulnerabilities
vuln: $(SPA_EMBED_DIR)/index.html
	$(GO_TOOL) govulncheck ./...

.PHONY: tidy-check generate-check

## tidy-check: Fails if app or tool go.mod/go.sum files are not tidy
tidy-check:
	go -C $(BACKEND_DIR) mod tidy -diff
	go -C $(BACKEND_DIR)/tools mod tidy -diff

## generate-check: Fails if generation changes tracked backend files
generate-check: $(SPA_EMBED_DIR)/index.html
	@cd $(BACKEND_DIR) && snap() { git ls-files -z | xargs -0 sha256sum; }; \
		before=$$(snap) && go generate ./... && [ "$$before" = "$$(snap)" ] || \
		{ echo "generate-check: generated files are stale; commit the regenerated output"; exit 1; }
