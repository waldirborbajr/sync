# Configuration variables
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_flags := "-s -w -X 'main.version=" + version + "'"
image := "sync"
name := "sync-dev"
dockerfile := "Dockerfile"
devcontainer_dockerfile := ".devcontainer/Dockerfile"
pwd := `pwd`

# ──────────────────────────────────────────────────────────────────────────────
# General
# ──────────────────────────────────────────────────────────────────────────────

@default:
    just --list

[group: 'general']
show-version:
    @echo "Current version: {{version}}"

# ──────────────────────────────────────────────────────────────────────────────
# Docker / Build
# ──────────────────────────────────────────────────────────────────────────────

[group: 'docker-build']
build:
    docker build -f {{dockerfile}} --build-arg VERSION={{version}} -t {{image}}:{{version}} -t {{image}}:latest .
    @echo "Built {{image}}:{{version}} and tagged as {{image}}:latest"

[group: 'docker-build']
build-dev:
    docker build -f {{devcontainer_dockerfile}} -t {{image}}-devcontainer:latest .

[group: 'docker-build']
build-binary:
    go build -ldflags {{build_flags}} -o `go env GOPATH`/bin/sync

[group: 'docker-build']
build-optimized:
    @echo "Building optimized version..."
    @echo "Note: This uses the experimental optimized processor"
    @echo "To enable: modify main_helpers.go to use processor.ProcessRowsOptimized()"

# ──────────────────────────────────────────────────────────────────────────────
# Docker / Run
# ──────────────────────────────────────────────────────────────────────────────

[group: 'docker-run']
run *ARGS:
    docker run --rm {{image}}:latest {{ARGS}}

[group: 'docker-run']
run-interactive:
    docker run --rm -it {{image}}:latest

[group: 'docker-run']
run-local *ARGS:
    go run -ldflags {{build_flags}} ./main.go {{ARGS}}

# ──────────────────────────────────────────────────────────────────────────────
# Docker / Management
# ──────────────────────────────────────────────────────────────────────────────

[group: 'docker-management']
clean:
    docker rm -f {{name}} || true
    docker image rm {{image}}:{{version}} || true
    docker image rm {{image}}:latest || true
    docker image rm {{image}}-devcontainer:latest || true

# ──────────────────────────────────────────────────────────────────────────────
# Development / Environment
# ──────────────────────────────────────────────────────────────────────────────

[group: 'dev-env']
dev-start:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod up . --ide none
    else
        just build-dev
        docker run -d --name {{name}} -v "{{pwd}}:/workspace" -w /workspace {{image}}-devcontainer:latest sleep infinity
    fi

[group: 'dev-env']
dev-shell:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod ssh {{name}}
    else
        docker exec -it --user vscode {{name}} zsh
    fi

[group: 'dev-env']
dev-stop:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod stop {{name}} || true
    else
        docker stop {{name}} || true
    fi

[group: 'dev-env']
dev-delete:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod delete {{name}} || true
    else
        docker rm -f {{name}} || true
        docker image rm {{image}}-devcontainer:latest || true
    fi

# ──────────────────────────────────────────────────────────────────────────────
# Go / Quality
# ──────────────────────────────────────────────────────────────────────────────

[group: 'go-quality']
fmt:
    goimports -w .
    go fmt ./...

[group: 'go-quality']
lint:
    staticcheck ./...
    go vet ./...

# ──────────────────────────────────────────────────────────────────────────────
# Go / Testing
# ──────────────────────────────────────────────────────────────────────────────

[group: 'go-testing']
test:
    mockery && go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

[group: 'go-testing']
test-only:
    go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

# ──────────────────────────────────────────────────────────────────────────────
# Go / Dependencies
# ──────────────────────────────────────────────────────────────────────────────

[group: 'go-dependencies']
deps:
    go get -u ./...
    go mod tidy

[group: 'go-dependencies']
deps-download:
    go mod download

# ──────────────────────────────────────────────────────────────────────────────
# Go / Installation
# ──────────────────────────────────────────────────────────────────────────────

[group: 'go-install']
install: build-binary
    @echo "sync installed to `go env GOPATH`/bin/sync"

# ──────────────────────────────────────────────────────────────────────────────
# Release / Versioning
# ──────────────────────────────────────────────────────────────────────────────

[group: 'release']
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

# ──────────────────────────────────────────────────────────────────────────────
# Performance
# ──────────────────────────────────────────────────────────────────────────────

[group: 'performance']
benchmark:
    @echo "Running performance benchmark..."
    @echo "Note: Ensure you have a test dataset ready"
    @echo "\n=== Original Version ==="
    time ./sync-original || echo "Build sync-original first"
    @echo "\n=== Optimized Version ==="
    time ./sync-optimized || echo "Build sync-optimized first"
