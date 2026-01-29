# GAR Credential Provider Makefile

# Variables
APP_NAME := gar-credential-provider
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags
LDFLAGS := -ldflags "-s -w \
	-X github.com/mondu-ai/gar-credential-provider/internal/version.Version=$(VERSION) \
	-X github.com/mondu-ai/gar-credential-provider/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/mondu-ai/gar-credential-provider/internal/version.BuildTime=$(BUILD_TIME)"

# Test flags
TEST_FLAGS := -v -race -timeout=30s
COVERAGE_FILE := coverage.out

.PHONY: all build build-all test lint clean help deps fmt vet security ci check-updates update-deps install

# Default target
all: lint test

# Help target
help: ## Show this help message
	@echo "GAR Credential Provider - Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Install dependencies
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	go mod tidy

# Run tests
test: ## Run unit tests
	@echo "Running unit tests..."
	go test $(TEST_FLAGS) ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test $(TEST_FLAGS) -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint: ## Run linter
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run ./...

# Format the code
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	@command -v goimports >/dev/null 2>&1 || { \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	}
	goimports -w .

# Vet the code
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Security scan
security: ## Run security scan with gosec
	@echo "Running security scan..."
	@command -v gosec >/dev/null 2>&1 || { \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	}
	gosec -fmt sarif -out results.sarif ./... || true

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/ dist/ $(COVERAGE_FILE) coverage.html results.sarif

# Check for updates
check-updates: ## Check for dependency updates
	@echo "Checking for dependency updates..."
	go list -u -m all

# Update dependencies
update-deps: ## Update all dependencies to latest versions
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Dependencies updated!"

# Full CI pipeline
ci: deps fmt vet lint security test ## Run full CI pipeline

# Build for current platform
build: ## Build for current platform
	@echo "Building $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/$(APP_NAME)
	@echo "Binary: bin/$(APP_NAME)"

# Build for all target platforms
build-all: build-linux-amd64 build-linux-arm64 ## Build for all target platforms

build-linux-amd64:
	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 ./cmd/$(APP_NAME)

build-linux-arm64:
	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64 ./cmd/$(APP_NAME)

# Install locally
install: build ## Install locally
	@echo "Installing $(APP_NAME)..."
	sudo cp bin/$(APP_NAME) /usr/local/bin/
	@echo "Installed to /usr/local/bin/$(APP_NAME)"
