# Stage 1: Build the Go binary
FROM cgr.dev/chainguard/go:1.22 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimized flags for Linux, using VERSION from build arg
ARG VERSION
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -gcflags="all=-N -l" -trimpath -buildmode=pie -o sync ./main.go

# Stage 2: Create a minimal runtime image
FROM cgr.dev/chainguard/static:latest

# Copy the binary from the builder stage
COPY --from=builder /app/sync /usr/local/bin/sync

# Set the entrypoint to run the sync tool
ENTRYPOINT ["/usr/local/bin/sync"]