.PHONY: build run clean test test-unit test-integration test-performance test-coverage test-all test-short test-verbose help

# Binary name
BINARY_NAME=terraform-backend-service
KEYGEN_BINARY=keygen

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Test parameters
TEST_TIMEOUT=10m
TEST_COVERAGE_FILE=coverage.out
TEST_COVERAGE_HTML=coverage.html

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server

build-keygen: ## Build the keygen binary
	$(GOBUILD) -o $(KEYGEN_BINARY) -v ./cmd/keygen

build-all: build build-keygen ## Build all binaries

run: build ## Build and run the service
	./$(BINARY_NAME)

clean: ## Remove binary and clean build cache
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(KEYGEN_BINARY)
	rm -f $(TEST_COVERAGE_FILE) $(TEST_COVERAGE_HTML)

# Test targets
test: ## Run all tests with default settings
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./...

test-short: ## Run tests in short mode (skip long-running tests)
	$(GOTEST) -v -short -timeout 2m ./...

test-verbose: ## Run tests with verbose output
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./... -count=1

test-unit: ## Run only unit tests (excluding integration and performance)
	$(GOTEST) -v -short -timeout 5m ./cmd/keygen/... ./internal/auth/... -run "^Test[^I][^n][^t]"

test-integration: ## Run integration tests
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/auth -run "TestEndToEnd|TestHotReload|TestIntegration|TestConcurrent"

test-performance: ## Run performance and load tests (takes longer)
	$(GOTEST) -v -timeout 15m ./internal/auth -run "^TestValidationPerformance|^TestValidationLatency|^TestConcurrent|^TestMemoryUsage|^TestLargeScale|^TestReloadPerformance"

test-edge-cases: ## Run edge case and error handling tests
	$(GOTEST) -v -timeout 5m ./internal/auth -run "^TestEdgeCase"

coverage: ## Generate coverage report for critical paths (auth, validation, storage)
	@echo "Generating coverage report for critical paths..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) \
		-coverprofile=$(TEST_COVERAGE_FILE) \
		-covermode=atomic \
		./internal/auth/... \
		./internal/validation/... \
		./internal/storage/... \
		./tests/unit-tests/... \
		./tests/integration-tests/...
	$(GOCMD) tool cover -html=$(TEST_COVERAGE_FILE) -o $(TEST_COVERAGE_HTML)
	@echo "Coverage report generated: $(TEST_COVERAGE_HTML)"
	@echo ""
	@echo "Coverage summary for critical paths:"
	$(GOCMD) tool cover -func=$(TEST_COVERAGE_FILE) | grep -E "auth|validation|storage|total"
	@echo ""
	@echo "Constitution requirement: >= 80% coverage for critical paths (auth, validation, storage)"

test-coverage: coverage ## Alias for coverage target (backward compatibility)

test-coverage-summary: ## Run tests and show coverage summary
	$(GOTEST) -timeout $(TEST_TIMEOUT) -coverprofile=$(TEST_COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -func=$(TEST_COVERAGE_FILE)

test-race: ## Run tests with race detector
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) ./...

test-bench: ## Run benchmark tests
	$(GOTEST) -bench=. -benchmem -timeout $(TEST_TIMEOUT) ./...

test-bench-auth: ## Run only auth benchmarks
	$(GOTEST) -bench=. -benchmem -timeout $(TEST_TIMEOUT) ./internal/auth/...

test-bench-keygen: ## Run only keygen benchmarks
	$(GOTEST) -bench=. -benchmem -timeout 5m ./cmd/keygen/...

test-all: ## Run all test categories (unit, integration, edge-case, performance) from tests/ directory
	@echo "Running all test categories..."
	@echo "1/4: Running unit tests..."
	$(GOTEST) -v -timeout 5m ./tests/unit-tests/...
	@echo "2/4: Running integration tests..."
	$(GOTEST) -v -timeout 10m ./tests/integration-tests/...
	@echo "3/4: Running edge case tests..."
	$(GOTEST) -v -timeout 5m ./tests/edge-case-tests/...
	@echo "4/4: Running performance tests..."
	$(GOTEST) -v -timeout 15m -bench=. ./tests/performance-tests/...
	@echo "All test categories completed successfully!"

test-ci: ## Run tests suitable for CI environment
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) -coverprofile=$(TEST_COVERAGE_FILE) -covermode=atomic ./...

test-watch: ## Watch for changes and run tests (requires entr: brew install entr)
	find . -name "*.go" | entr -c make test-short

# Dependency management
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-verify: ## Verify dependencies
	$(GOMOD) verify

deps-update: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Code quality
fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run

security-scan: ## Run security scan (requires gosec: go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -quiet ./...

# Development workflow
dev-setup: deps build-all ## Set up development environment
	@echo "Development environment ready!"

pre-commit: fmt vet test-short ## Run pre-commit checks (format, vet, quick tests)

ci-checks: fmt vet test-ci security-scan ## Run all CI checks

all: clean deps fmt vet build-all test-short ## Run full build pipeline

# Test data generation
generate-test-data: build-keygen ## Generate test authentication data
	./$(KEYGEN_BINARY) init-config.cfg auth.cfg

# Development utilities
clean-test-cache: ## Clean test cache
	$(GOCMD) clean -testcache

list-tests: ## List all test functions
	@grep -r "^func Test" . --include="*_test.go" | sed 's/.*func //' | sed 's/(.*)//' | sort

count-tests: ## Count total number of tests
	@echo "Total test functions: $$(grep -r "^func Test" . --include="*_test.go" | wc -l)"
	@echo "Total benchmark functions: $$(grep -r "^func Benchmark" . --include="*_test.go" | wc -l)"

# Documentation
test-docs: ## Generate test documentation
	@echo "Generating test documentation..."
	@echo "See TEST_STRATEGY.md for comprehensive test documentation"
