# Makefile for building the Go project for multiple platforms with optimized size and security
# Version number (update here to change version across all builds)
VERSION = 0.1.6

# Binary name
BINARY_NAME = sync

# Directories
BIN_DIR = bin
SRC_DIR = .

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOVET = $(GOCMD) vet
GOLINT = golangci-lint run
GOFLAGS = -ldflags="-s -w -X main.version=$(VERSION)" -gcflags="all=-N -l" -trimpath -buildmode=pie
GOFLAGS_FREEBSD = -ldflags="-s -w -X main.version=$(VERSION)" -gcflags="all=-N -l" -trimpath

# Docker parameters
DOCKER_REGISTRY = docker.io
DOCKER_USERNAME = yourusername # Replace with your Docker Hub username
DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(BINARY_NAME)
DOCKER_TAG = $(VERSION)

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m # No Color

# Default goal
.DEFAULT_GOAL := help

# Create bin directory if not exists
$(shell mkdir -p $(BIN_DIR))

.PHONY: help
help: ## Show this help message
	@echo "${BLUE}Makefile for $(BINARY_NAME) v$(VERSION)${NC}"
	@echo "${BLUE}Available targets:${NC}"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "${GREEN}%-20s${NC} %s\n", $$1, $$2}'

.PHONY: all
all: lint vet test build-all ## Run lint, vet, test, and build for all platforms
	@echo "${GREEN}All tasks completed successfully!${NC}"

.PHONY: lint
lint: check-linter ## Run linter on all source files
	@echo "${BLUE}Running golangci-lint...${NC}"
	@$(GOLINT) $(SRC_DIR)/...
	@echo "${GREEN}Linting completed successfully!${NC}"

.PHONY: vet
vet: ## Run Go vet static analysis
	@echo "${BLUE}Running go vet...${NC}"
	@$(GOVET) $(SRC_DIR)/...
	@echo "${GREEN}Vet analysis completed successfully!${NC}"

.PHONY: test
test: ## Run tests with race detector
	@echo "${BLUE}Running tests with race detector...${NC}"
	@$(GOTEST) -v -race $(SRC_DIR)/...
	@echo "${GREEN}Tests completed successfully!${NC}"

.PHONY: build-all
build-all: build-linux build-windows build-freebsd ## Build for all supported platforms
	@echo "${GREEN}Builds completed for all platforms!${NC}"

.PHONY: build-linux
build-linux: ## Build for Linux (amd64) with PIE
	@echo "${BLUE}Building for Linux (amd64)...${NC}"
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux $(SRC_DIR)
	@echo "${GREEN}Linux build completed: $(BIN_DIR)/$(BINARY_NAME)-linux${NC}"

.PHONY: build-windows
build-windows: ## Build for Windows (amd64) with PIE
	@echo "${BLUE}Building for Windows (amd64)...${NC}"
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows.exe $(SRC_DIR)
	@echo "${GREEN}Windows build completed: $(BIN_DIR)/$(BINARY_NAME)-windows.exe${NC}"

.PHONY: build-freebsd
build-freebsd: ## Build for FreeBSD (amd64) without PIE
	@echo "${BLUE}Building for FreeBSD (amd64)...${NC}"
	@GOOS=freebsd GOARCH=amd64 $(GOBUILD) $(GOFLAGS_FREEBSD) -o $(BIN_DIR)/$(BINARY_NAME)-freebsd $(SRC_DIR)
	@echo "${GREEN}FreeBSD build completed: $(BIN_DIR)/$(BINARY_NAME)-freebsd${NC}"

.PHONY: docker-build
docker-build: check-docker build-linux ## Build Docker image for Linux with optimized flags
	@echo "${BLUE}Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG) for Linux...${NC}"
	@docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "${GREEN}Docker image built successfully: $(DOCKER_IMAGE):$(DOCKER_TAG)${NC}"

.PHONY: docker-push
docker-push: check-docker docker-build ## Push Docker image to Docker Hub
	@echo "${BLUE}Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG) to Docker Hub...${NC}"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest
	@echo "${GREEN}Docker image pushed successfully to $(DOCKER_IMAGE):$(DOCKER_TAG) and latest${NC}"

.PHONY: check-docker
check-docker: ## Check if Docker is installed
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "${RED}Error: Docker is not installed. Please install Docker first.${NC}"; \
		exit 1; \
	fi

.PHONY: clean
clean: ## Clean build artifacts and remove bin directory
	@echo "${BLUE}Cleaning build artifacts...${NC}"
	@$(GOCLEAN)
	@rm -rf $(BIN_DIR)/*
	@echo "${GREEN}Cleanup completed successfully!${NC}"

.PHONY: check-linter
check-linter: ## Check if golangci-lint is installed
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "${RED}Error: golangci-lint is not installed. Please install it first.${NC}"; \
		echo "${BLUE}Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin${NC}"; \
		exit 1; \
	fi

.PHONY: check-go
check-go: ## Check if Go is installed
	@if ! command -v $(GOCMD) >/dev/null 2>&1; then \
		echo "${RED}Error: Go is not installed. Please install it first.${NC}"; \
		exit 1; \
	fi

.PHONY: format
format: ## Run gofmt on all source files
	@echo "${BLUE}Formatting Go code...${NC}"
	@$(GOCMD) fmt $(SRC_DIR)/...
	@echo "${GREEN}Code formatting completed!${NC}"

.PHONY: tidy
tidy: ## Run go mod tidy to clean up go.mod
	@echo "${BLUE}Running go mod tidy...${NC}"
	@$(GOCMD) mod tidy
	@echo "${GREEN}Go module cleanup completed!${NC}"

.PHONY: clear-cache
clear-cache: check-go ## Clear Go module cache
	@echo "${BLUE}Clearing Go module cache...${NC}"
	@$(GOCMD) clean -modcache
	@echo "${GREEN}Go module cache cleared successfully!${NC}"

.PHONY: clear-cache-tidy
clear-cache-tidy: clear-cache tidy ## Clear Go module cache and run go mod tidy
	@echo "${GREEN}Cache cleared and go mod tidy completed!${NC}"

.PHONY: verify
verify: lint vet test ## Run all verification steps (lint, vet, test)
	@echo "${GREEN}All verification steps completed successfully!${NC}"

# Ensure dependencies for all targets
lint vet test build-linux build-windows build-freebsd clear-cache clear-cache-tidy docker-build docker-push: check-go