.PHONY: build run clean

BINARY_NAME=web-adapter
BUILD_DIR=.bin
CONFIG_PATH=./configs

build:
	@echo "Building $(BINARY_NAME)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/app

run: build
	@echo "Running $(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning build artifacts"
	@rm -rf $(BUILD_DIR)