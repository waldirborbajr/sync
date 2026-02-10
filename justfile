version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_flags := "-s -w -X 'main.version=" + version + "'"
image := "fb2mysql"
name := "fb2mysql"
dockerfile := ".devcontainer/Dockerfile"
pwd := `pwd`

# Build Docker image (skips if devpod is detected)
build:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        echo "devpod detected; skip docker build (use 'just on' to create the pod)"
    else
        docker build -f {{dockerfile}} -t {{image}}:{{version}} -t {{image}}:latest .
        echo "Built {{image}}:{{version}} and tagged as {{image}}:latest"
    fi

# Start container/pod
on:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod up . --ide none
    else
        just build
        docker run -d --name {{name}} -v "{{pwd}}:/workspace" -w /workspace {{image}}:latest tail -f /dev/null
    fi

# SSH into container/pod
ssh:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod ssh {{name}}
    else
        docker exec -it --user vscode {{name}} zsh
    fi

# Stop container/pod
off:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod stop {{name}} || true
    else
        docker stop {{name}} || true
    fi

# Delete container/pod and image
delete:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod delete {{name}} || true
    else
        docker rm -f {{name}} || true
        docker image rm {{image}}:{{version}} || true
        docker image rm {{image}}:latest || true
    fi

# Run tests with coverage
test:
    mockery && go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

# Build the sync binary
build-binary:
    go build -ldflags {{build_flags}} -o `go env GOPATH`/bin/sync
