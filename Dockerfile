# Stage 1: Build the Go binary
FROM golang:1.22 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimized flags for Linux, matching Makefile
ARG VERSION=0.1.7
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -gcflags="all=-N -l" -trimpath -buildmode=pie -o sync ./main.go

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Install ca-certificates for secure connections
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/sync /usr/local/bin/sync

# Set the entrypoint to run the sync tool
ENTRYPOINT ["sync"]