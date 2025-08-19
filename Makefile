# Makefile for building the Go project for multiple platforms
# Binary name
BINARY_NAME = sync

# Directories
BIN_DIR = bin

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOLINT = golangci-lint run
GOFLAGS = -ldflags="-s -w" -trimpath  # Security: strip debug info and paths; Performance: default optimizations

# Default target
all: lint test build-linux build-windows build-freebsd

# Lint the code
lint:
	$(GOLINT) ./...

# Run tests with race detector for leak detection
test:
	$(GOTEST) -v -race ./...

# Build for Linux (amd64)
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux .

# Build for Windows (amd64)
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows.exe .

# Build for FreeBSD (amd64)
build-freebsd:
	GOOS=freebsd GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-freebsd .

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)/*

upload:
	@echo "Enviando arquivo via SFTP..."
	@sshpass -p '*senha@' sftp -oBatchMode=no -b - josemario@192.168.0.46 << !
	cd syn
	put bin/sync-freebsd
	bye
!

# Create bin directory if not exists
$(shell mkdir -p $(BIN_DIR))

.PHONY: all lint test build-linux build-windows build-freebsd clean