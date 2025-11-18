.PHONY: help build test test-security test-coverage run clean docker-build docker-run docker-stop install lint fmt vet

# Variables
APP_NAME=address-parser
DOCKER_IMAGE=$(APP_NAME):latest
DOCKER_CONTAINER=$(APP_NAME)-server
GO=go
GOFLAGS=-v
PORT?=8080

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	$(GO) build $(GOFLAGS) -o bin/$(APP_NAME) cmd/server/main.go
	@echo "Build complete: bin/$(APP_NAME)"

install: ## Install dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

test: ## Run all tests
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

test-security: ## Run security-focused tests
	@echo "Running security tests..."
	$(GO) test $(GOFLAGS) -run Security ./pkg/parser

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

run: ## Run the application locally
	@echo "Starting $(APP_NAME) on port $(PORT)..."
	SERVER_PORT=$(PORT) $(GO) run cmd/server/main.go

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping..."; \
		echo "Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

# Docker targets

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-run: docker-build ## Run application in Docker
	@echo "Running Docker container $(DOCKER_CONTAINER)..."
	docker run -d \
		--name $(DOCKER_CONTAINER) \
		-p $(PORT):8080 \
		--env-file .env 2>/dev/null || true \
		$(DOCKER_IMAGE)
	@echo "Container running on http://localhost:$(PORT)"

docker-stop: ## Stop and remove Docker container
	@echo "Stopping Docker container..."
	@docker stop $(DOCKER_CONTAINER) 2>/dev/null || true
	@docker rm $(DOCKER_CONTAINER) 2>/dev/null || true
	@echo "Container stopped"

docker-logs: ## View Docker container logs
	docker logs -f $(DOCKER_CONTAINER)

docker-compose-up: ## Start with docker-compose
	docker-compose up -d
	@echo "Services started with docker-compose"

docker-compose-down: ## Stop docker-compose services
	docker-compose down
	@echo "Services stopped"

# Development helpers

dev: ## Run in development mode with auto-reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

all: clean install fmt vet test build ## Run full build pipeline
	@echo "Full build complete"

# Security checks

security-scan: ## Run security vulnerability scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

deps-check: ## Check for dependency vulnerabilities
	@echo "Checking dependencies for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi
