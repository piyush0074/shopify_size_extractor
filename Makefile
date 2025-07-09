# Shopify Size Chart Extractor Makefile
# This Makefile provides convenient commands for building, testing, and running the application

# Variables
BINARY_NAME=shopify_extractor
API_BINARY_NAME=shopify_api
BUILD_DIR=bin
MAIN_PATH=cmd/main.go
API_PATH=cmd/api/main.go

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty)"

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Shopify Size Chart Extractor - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
.PHONY: build
build: ## Build all binaries
	@echo "Building all binaries..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(API_BINARY_NAME) $(API_PATH)
	@echo "Build complete! Binaries available in $(BUILD_DIR)/"

.PHONY: build-cli
build-cli: ## Build CLI binary only
	@echo "Building CLI binary..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "CLI binary built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-api
build-api: ## Build API binary only
	@echo "Building API binary..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(API_BINARY_NAME) $(API_PATH)
	@echo "API binary built: $(BUILD_DIR)/$(API_BINARY_NAME)"

# Run targets
.PHONY: run
run: ## Run CLI with default settings (all stores)
	@echo "Running CLI extractor for all stores..."
	go run $(MAIN_PATH)

.PHONY: run-westside
run-westside: ## Run CLI for Westside store only
	@echo "Running CLI extractor for Westside..."
	go run $(MAIN_PATH) westside

.PHONY: run-littleboxindia
run-littleboxindia: ## Run CLI for LittleBoxIndia store only
	@echo "Running CLI extractor for LittleBoxIndia..."
	go run $(MAIN_PATH) littleboxindia

.PHONY: run-suqah
run-suqah: ## Run CLI for Suqah store only
	@echo "Running CLI extractor for Suqah..."
	go run $(MAIN_PATH) suqah

.PHONY: run-api
run-api: ## Start the API server
	@echo "Starting API server on port 8080..."
	@echo "Use 'make stop-api' to stop the server"
	go run $(API_PATH)

.PHONY: run-api-background
run-api-background: ## Start the API server in background
	@echo "Starting API server in background..."
	@go run $(API_PATH) &
	@echo "API server started. PID: $$(lsof -ti:8080)"

# Test targets
.PHONY: test
test: ## Run all tests
	@echo "Running all tests..."
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	@echo "Running all tests with verbose output..."
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -cover ./...

.PHONY: test-adapters
test-adapters: ## Run adapter tests only
	@echo "Running adapter tests..."
	go test ./adapters -v

.PHONY: test-extractors
test-extractors: ## Run extractor tests only
	@echo "Running extractor tests..."
	go test ./extractor -v

# Development targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f results_*.json
	@echo "Clean complete!"

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated!"

.PHONY: fmt
fmt: ## Format all Go code
	@echo "Formatting Go code..."
	go fmt ./...

.PHONY: lint
lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# API testing targets
.PHONY: test-api
test-api: ## Test API endpoints
	@echo "Testing API endpoints..."
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq . || echo "Health check failed or jq not available"
	@echo ""
	@echo "Testing extraction endpoint..."
	@curl -s -X POST http://localhost:8080/extract \
		-H "Content-Type: application/json" \
		-d '{"stores": ["westside.com"]}' | jq . || echo "Extraction test failed or jq not available"

.PHONY: test-api-westside
test-api-westside: ## Test API with Westside store
	@echo "Testing API with Westside store..."
	@curl -s -X POST http://localhost:8080/extract \
		-H "Content-Type: application/json" \
		-d '{"stores": ["westside.com"]}' | jq .

.PHONY: test-api-all
test-api-all: ## Test API with all stores
	@echo "Testing API with all stores..."
	@curl -s -X POST http://localhost:8080/extract \
		-H "Content-Type: application/json" \
		-d '{"stores": ["westside.com", "littleboxindia.com", "suqah.com"]}' | jq .

# Utility targets
.PHONY: stop-api
stop-api: ## Stop the API server
	@echo "Stopping API server..."
	@pkill -f "go run cmd/api/main.go" || echo "No API server found running"

.PHONY: kill-port
kill-port: ## Kill process on port 8080
	@echo "Killing process on port 8080..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || echo "No process found on port 8080"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Tools installed!"

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@echo "Documentation is available in the docs/ directory"
	@echo "- README.md: Setup and usage instructions"
	@echo "- docs/ARCHITECTURE.md: Technical architecture details"
	@echo "- docs/DEVELOPMENT.md: Development guide"

# Quick start targets
.PHONY: quick-start
quick-start: ## Quick start guide - build and run API
	@echo "=== Quick Start Guide ==="
	@echo "1. Building binaries..."
	@make build
	@echo ""
	@echo "2. Starting API server..."
	@make run-api-background
	@echo ""
	@echo "3. Testing API..."
	@sleep 2
	@make test-api
	@echo ""
	@echo "4. Stopping API server..."
	@make stop-api
	@echo ""
	@echo "Quick start complete!"

.PHONY: dev-setup
dev-setup: ## Setup development environment
	@echo "=== Development Environment Setup ==="
	@echo "1. Installing dependencies..."
	@make deps
	@echo ""
	@echo "2. Installing development tools..."
	@make install-tools
	@echo ""
	@echo "3. Running tests..."
	@make test
	@echo ""
	@echo "4. Formatting code..."
	@make fmt
	@echo ""
	@echo "Development environment ready!"

# Performance targets
.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. ./...

.PHONY: profile
profile: ## Run with profiling
	@echo "Running with CPU profiling..."
	go test -cpuprofile=cpu.prof -bench=. ./...
	@echo "Profile saved to cpu.prof"
	@echo "Analyze with: go tool pprof cpu.prof"

# Release targets
.PHONY: release
release: ## Build release binaries
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Release binaries built in $(BUILD_DIR)/"

# Docker targets (if needed in future)
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t shopify-extractor .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 shopify-extractor

# Default target
.DEFAULT_GOAL := help 