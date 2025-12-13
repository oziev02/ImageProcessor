.PHONY: help build run test clean install lint docker-up docker-down docker-restart migrate

# Variables
BINARY_NAME=imageprocessor
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/imageprocessor/main.go
DOCKER_COMPOSE=docker-compose.yml

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@rm -f $(BINARY_NAME)  # Remove binary from root if exists
	go build -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && go run $(MAIN_PATH); \
	else \
		go run $(MAIN_PATH); \
	fi

run-build: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && ./$(BINARY_PATH); \
	else \
		./$(BINARY_PATH); \
	fi

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f $(BINARY_NAME)  # Remove binary from root if exists
	rm -rf storage/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-up: ## Start Docker containers (PostgreSQL, Kafka)
	@echo "Starting Docker containers..."
	docker-compose -f $(DOCKER_COMPOSE) up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Docker containers started"

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker-compose -f $(DOCKER_COMPOSE) down
	@echo "Docker containers stopped"

docker-restart: docker-down docker-up ## Restart Docker containers

docker-logs: ## Show Docker container logs
	docker-compose -f $(DOCKER_COMPOSE) logs -f

docker-ps: ## Show Docker container status
	docker-compose -f $(DOCKER_COMPOSE) ps

migrate: ## Run database migrations (migrations run automatically on app startup)
	@echo "Note: Database migrations are handled automatically on application startup"
	@echo "Migrations are located in: internal/migrations/"
	@echo "To manually run migrations, use the migrate CLI tool:"
	@echo "  migrate -path internal/migrations -database 'postgres://user:pass@host:port/dbname?sslmode=disable' up"

dev: docker-up ## Start development environment (Docker + app)
	@echo "Starting development environment..."
	@echo "Docker containers are up. Starting application..."
	@make run

start: docker-up ## Start everything (Docker + app) - one command to rule them all
	@echo "=========================================="
	@echo "Starting ImageProcessor..."
	@echo "=========================================="
	@echo "1. Docker containers: UP"
	@echo "2. Waiting for services to be ready..."
	@sleep 3
	@echo "3. Starting application..."
	@echo "=========================================="
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && go run $(MAIN_PATH); \
	else \
		echo "Warning: .env file not found, using default values"; \
		go run $(MAIN_PATH); \
	fi

stop: docker-down ## Stop development environment

setup: install docker-up ## Initial setup (install deps + start Docker)
	@echo "Setup complete!"
	@echo "Run 'make run' to start the application"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

# Development helpers
watch: ## Watch for changes and rebuild (requires entr: apt-get install entr)
	@echo "Watching for changes..."
	@find . -name "*.go" -not -path "./vendor/*" | entr -r make run

# Production build
build-release: ## Build optimized release binary
	@echo "Building release binary..."
	@mkdir -p bin
	@rm -f $(BINARY_NAME)  # Remove binary from root if exists
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Release build complete: $(BINARY_PATH)"

# Database helpers
db-shell: ## Connect to PostgreSQL shell
	docker-compose -f $(DOCKER_COMPOSE) exec postgres psql -U postgres -d imageprocessor

kafka-topics: ## List Kafka topics
	docker-compose -f $(DOCKER_COMPOSE) exec kafka kafka-topics.sh --list --bootstrap-server localhost:9092

kafka-create-topic: ## Create Kafka topic (usage: make kafka-create-topic TOPIC=image-processing)
	docker-compose -f $(DOCKER_COMPOSE) exec kafka kafka-topics.sh --create --topic $(TOPIC) --bootstrap-server localhost:9092 --if-not-exists

