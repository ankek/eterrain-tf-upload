.PHONY: build run clean test help

# Binary name
BINARY_NAME=terraform-backend-service

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server

run: build ## Build and run the service
	./$(BINARY_NAME)

clean: ## Remove binary and clean build cache
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: ## Run tests
	$(GOTEST) -v ./...

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run

all: clean deps fmt vet build ## Run all build steps
