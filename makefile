.PHONY: build run

BINARY_NAME=chat-bot
BUILD_DIR=./.bin

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m

export CONFIG_PATH ?= ./configs

build:
	@echo "Building $(BINARY_NAME)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/app

run: build
	@echo "Running $(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)
	