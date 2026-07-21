# Load variables from .env for the host-side targets (run/migrate-local).
ifneq (,$(wildcard .env))
include .env
export
endif

COMPOSE := docker compose
BIN_DIR := bin

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Local development (Go on the host, Postgres in Docker)
# ---------------------------------------------------------------------------

## tidy: sync go.mod / go.sum
.PHONY: tidy
tidy:
	go mod tidy

## sqlc: regenerate sqlc code from db/queries
.PHONY: sqlc
sqlc:
	sqlc generate

## build: compile bot and migrate binaries into ./bin
.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/bot ./cmd/bot
	go build -o $(BIN_DIR)/migrate ./cmd/migrate

## db-up: start only Postgres in the background
.PHONY: db-up
db-up:
	$(COMPOSE) up -d db

## migrate-up: apply migrations against the local DATABASE_URL
.PHONY: migrate-up
migrate-up: build
	./$(BIN_DIR)/migrate up

## migrate-down: roll back the most recent migration
.PHONY: migrate-down
migrate-down: build
	./$(BIN_DIR)/migrate down

## run: run the bot on the host (needs db-up + a filled .env)
.PHONY: run
run: build
	./$(BIN_DIR)/bot

## dev: start DB, apply migrations, then run the bot locally
.PHONY: dev
dev: db-up
	@echo "waiting for Postgres..."
	@until $(COMPOSE) exec -T db pg_isready -U $(POSTGRES_USER) >/dev/null 2>&1; do sleep 1; done
	$(MAKE) migrate-up
	$(MAKE) run

# ---------------------------------------------------------------------------
# Full Docker stack (db + migrations + bot, all in containers)
# ---------------------------------------------------------------------------

## docker-build: build the application image
.PHONY: docker-build
docker-build:
	$(COMPOSE) build

## up: build images and start the whole stack (db -> migrate -> bot)
.PHONY: up
up:
	$(COMPOSE) up --build -d
	$(COMPOSE) logs -f bot

## down: stop the stack (keeps the DB volume)
.PHONY: down
down:
	$(COMPOSE) down

## clean: stop the stack and delete the DB volume
.PHONY: clean
clean:
	$(COMPOSE) down -v
	rm -rf $(BIN_DIR)

## logs: follow the bot logs
.PHONY: logs
logs:
	$(COMPOSE) logs -f bot

## ps: show stack status
.PHONY: ps
ps:
	$(COMPOSE) ps
