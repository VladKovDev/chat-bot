.PHONY: build run clean docker-build docker-run docker-rebuild docker-stop docker-logs

BINARY_NAME=web-adapter
BUILD_DIR=backend/.bin
BACKEND_DIR=backend
IMAGE_NAME=web-adapter
CONTAINER_NAME=chat-bot-web
PORT=8081

build:
	@echo "Building $(BINARY_NAME)"
	@mkdir -p $(BUILD_DIR)
	@cd $(BACKEND_DIR) && go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/app

run: build
	@echo "Running $(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning build artifacts"
	@rm -rf $(BUILD_DIR)

docker-build:
	@echo "Building Docker image $(IMAGE_NAME)"
	@docker build -t $(IMAGE_NAME) .

docker-run:
	@echo "Running Docker container $(CONTAINER_NAME)"
	@docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):8081 \
		--add-host=host.docker.internal:host-gateway \
		-e DECISION_ENGINE_URL=http://host.docker.internal:8080 \
		$(IMAGE_NAME)

docker-rebuild: clean docker-build
	@echo "Rebuilding and restarting Docker container"
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true
	@docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):8081 \
		--add-host=host.docker.internal:host-gateway \
		-e DECISION_ENGINE_URL=http://host.docker.internal:8080 \
		$(IMAGE_NAME)

docker-stop:
	@echo "Stopping Docker container $(CONTAINER_NAME)"
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true

docker-logs:
	@docker logs -f $(CONTAINER_NAME)