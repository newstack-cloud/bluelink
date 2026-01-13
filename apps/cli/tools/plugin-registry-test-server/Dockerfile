# Plugin Registry Test Server Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go.mod and go.sum, download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o plugin-registry-test-server .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/plugin-registry-test-server .

# Create keys directory (keys are generated at runtime if not provided)
RUN mkdir -p /app/keys && chown appuser:appuser /app/keys

USER appuser

EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./plugin-registry-test-server"]
