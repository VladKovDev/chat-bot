.PHONY: help install update run dev test lint lint-fix format type-check clean docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies with poetry
	poetry install

update: ## Update dependencies
	poetry update

run: ## Run the application
	poetry run uvicorn app.main:app --host $(shell grep SERVER_HOST .env | cut -d '=' -f2) --port $(shell grep SERVER_PORT .env | cut -d '=' -f2)

dev: ## Run the application with auto-reload
	poetry run uvicorn app.main:app --host 0.0.0.0 --port 8001 --reload

test: ## Run tests
	poetry run pytest

test-cov: ## Run tests with coverage
	poetry run pytest --cov=app --cov-report=html --cov-report=term

lint: ## Run linter (ruff)
	poetry run ruff check .

lint-fix: ## Fix linting issues automatically
	poetry run ruff check . --fix

format: ## Format code with ruff
	poetry run ruff format .

type-check: ## Run type checker (mypy)
	poetry run mypy app

check: lint type-check ## Run all checks (lint and type-check)

clean: ## Clean cache files
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type d -name ".pytest_cache" -exec rm -rf {} +
	find . -type d -name ".ruff_cache" -exec rm -rf {} +
	find . -type d -name ".mypy_cache" -exec rm -rf {} +
	rm -f .coverage
	rm -rf htmlcov/

docker-build: ## Build Docker image
	docker build -t llm-service .

docker-run: ## Run Docker container
	docker run --name chat-bot-llm -p 8001:8001 --add-host=host.docker.internal:host-gateway --env-file .env llm-service

all: install lint type-check test ## Install dependencies and run all checks
