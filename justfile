version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_flags := "-s -w -X 'main.version=" + version + "'"
image := "sync"
name := "sync-dev"
dockerfile := "Dockerfile"
devcontainer_dockerfile := ".devcontainer/Dockerfile"
pwd := `pwd`

# List all available commands
@default:
    just --list

# Build production Docker image
build:
    docker build -f {{dockerfile}} --build-arg VERSION={{version}} -t {{image}}:{{version}} -t {{image}}:latest .
    @echo "Built {{image}}:{{version}} and tagged as {{image}}:latest"

# Build devcontainer image (normally handled by VS Code)
build-dev:
    docker build -f {{devcontainer_dockerfile}} -t {{image}}-devcontainer:latest .

# Run the production container
run *ARGS:
    docker run --rm {{image}}:latest {{ARGS}}

# Run the production container interactively
run-interactive:
    docker run --rm -it {{image}}:latest

# Start devcontainer manually (normally use VS Code's "Reopen in Container")
dev-start:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod up . --ide none
    else
        just build-dev
        docker run -d --name {{name}} -v "{{pwd}}:/workspace" -w /workspace {{image}}-devcontainer:latest sleep infinity
    fi

# SSH into devcontainer
dev-shell:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod ssh {{name}}
    else
        docker exec -it --user vscode {{name}} zsh
    fi

# Stop devcontainer
dev-stop:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod stop {{name}} || true
    else
        docker stop {{name}} || true
    fi

# Delete devcontainer
dev-delete:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod delete {{name}} || true
    else
        docker rm -f {{name}} || true
        docker image rm {{image}}-devcontainer:latest || true
    fi

# Clean up all Docker artifacts
clean:
    docker rm -f {{name}} || true
    docker image rm {{image}}:{{version}} || true
    docker image rm {{image}}:latest || true
    docker image rm {{image}}-devcontainer:latest || true

# Run tests with coverage
test:
    mockery && go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

# Run tests without mockery  
test-only:
    go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

# Format Go code
fmt:
    goimports -w .
    go fmt ./...

# Lint Go code
lint:
    staticcheck ./...
    go vet ./...

# Update Go dependencies
deps:
    go get -u ./...
    go mod tidy

# Download Go dependencies
deps-download:
    go mod download

# Build the sync binary locally
build-binary:
    go build -ldflags {{build_flags}} -o `go env GOPATH`/bin/sync

# Install the sync binary
install: build-binary
    @echo "sync installed to `go env GOPATH`/bin/sync"

# Run the sync binary locally (use after build-binary)
run-local *ARGS:
    go run -ldflags {{build_flags}} ./main.go {{ARGS}}

# Create and push a new git tag (usage: just tag v1.0.0)
tag VERSION:
    #!/usr/bin/env bash
    if [[ ! "{{VERSION}}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Error: Version must be in format v1.2.3"
        exit 1
    fi
    echo "Creating tag {{VERSION}}..."
    git tag -a {{VERSION}} -m "Release {{VERSION}}"
    git push origin {{VERSION}}
    echo "Tag {{VERSION}} created and pushed"

# Show current version
show-version:
    @echo "Current version: {{version}}"
