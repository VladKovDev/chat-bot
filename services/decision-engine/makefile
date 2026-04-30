.PHONY: build run migrate-up migrate-down

BINARY_NAME=chat-bot
BUILD_DIR=./.bin
MIGRATIONS_DIR := ./migrations

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m

DATABASE_URL ?= postgres://postgres:postgres@localhost:5442/chat-bot?sslmode=disable

export CONFIG_PATH ?= ./configs

build:
	@echo "Building $(BINARY_NAME)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/app

run: build
	@echo "Running $(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)

migrate-up: 
	@echo "$(COLOR_YELLOW)Running migrations...$(COLOR_RESET)"
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "ERROR: DATABASE_URL is not set"; \
		exit 1; \
	fi
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(DATABASE_URL) goose -dir $(MIGRATIONS_DIR) up
	@echo "$(COLOR_GREEN)Migrations applied!$(COLOR_RESET)"

migrate-down: 
	@echo "$(COLOR_YELLOW)Rolling back last migration...$(COLOR_RESET)"
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "ERROR: DATABASE_URL is not set"; \
		exit 1; \
	fi
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(DATABASE_URL) goose -dir $(MIGRATIONS_DIR) down
	@echo "$(COLOR_GREEN)Migration rolled back!$(COLOR_RESET)"