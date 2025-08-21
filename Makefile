# Makefile for building the Go project for multiple platforms with optimized size and security
# Version number (update here to change version across all builds)
VERSION = 0.1.6

# Binary name
BINARY_NAME = sync

# Directories
BIN_DIR = bin

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOVET = $(GOCMD) vet
GOLINT = golangci-lint run
GOFLAGS = -ldflags="-s -w -X main.version=$(VERSION)" -gcflags="all=-N -l" -trimpath -buildmode=pie
GOFLAGS_FREEBSD = -ldflags="-s -w -X main.version=$(VERSION)" -gcflags="all=-N -l" -trimpath

# Default target
all: lint vet test build-all

# Lint the code
lint:
	$(GOLINT) ./...

# Run static analysis
vet:
	$(GOVET) ./...

# Run tests with race detector for leak detection
test:
	$(GOTEST) -v -race ./...

# Build for all platforms
build-all: build-linux build-windows build-freebsd

# Build for Linux (amd64) with PIE for security and size optimization
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux .

# Build for Windows (amd64) with PIE for security and size optimization
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows.exe .

# Build for FreeBSD (amd64) without PIE to avoid cgo requirement
build-freebsd:
	GOOS=freebsd GOARCH=amd64 $(GOBUILD) $(GOFLAGS_FREEBSD) -o $(BIN_DIR)/$(BINARY_NAME)-freebsd .

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)/*

# Create bin directory if not exists
$(shell mkdir -p $(BIN_DIR))

.PHONY: all lint vet test build-all build-linux build-windows build-freebsd clean