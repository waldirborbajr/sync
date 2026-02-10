build_flags := "-s -w -X 'main.version=`git describe --tags --abbrev=0`'"
image := "gocontainer:latest"
name := "gocontainer"
dockerfile := ".devcontainer/Dockerfile"
pwd := `pwd`

# Build Docker image (skips if devpod is detected)
build:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        echo "devpod detected; skip docker build (use 'just on' to create the pod)"
    else
        docker build -f {{dockerfile}} -t {{image}} .
    fi

# Start container/pod
on:
    #!/usr/bin/env bash
    if command -v devpod &>/dev/null; then
        devpod up . --ide none
    else
        just build
        docker run -d --name {{name}} -v "{{pwd}}:/workspace" -w /workspace {{image}} tail -f /dev/null
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
        docker image rm {{image}} || true
    fi

# Run tests with coverage
test:
    mockery && go test -cover -bench=. -benchmem -race ./... -coverprofile=coverage.out

# Build the sync binary
build-binary:
    go build -ldflags {{build_flags}} -o `go env GOPATH`/bin/sync
