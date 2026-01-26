# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.25.6-trixie AS builder

WORKDIR /build

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
COPY tools/genversions/go.mod tools/genversions/go.sum ./tools/genversions/
RUN go mod download

WORKDIR /build/tools/genversions
RUN go mod download
WORKDIR /build

# Copy source code
COPY . .

# Build argument for version injection
ARG DEPLOY_ENGINE_APPLICATION_VERSION=dev

# Target platform args (automatically set by BuildKit when using --platform)
ARG TARGETOS
ARG TARGETARCH

# Generate version file using the tool's own module (has golang.org/x/mod dependency)
WORKDIR /build/tools/genversions
# hadolint ignore=DL3062
RUN DEPLOY_ENGINE_APPLICATION_VERSION=${DEPLOY_ENGINE_APPLICATION_VERSION} go run main.go
WORKDIR /build

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/deploy-engine \
    ./cmd/main.go

# =============================================================================
# Stage 2: Production Runtime
# =============================================================================
FROM debian:trixie-slim

# Install CA certificates and curl for health checks
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    curl && \
    rm -rf /var/lib/apt/lists/* && \
    update-ca-certificates

# Create non-root user
RUN groupadd --gid 1000 bluelink && \
    useradd --uid 1000 --gid 1000 --shell /bin/false --create-home bluelink

# Create required directories
RUN mkdir -p /app && \
    mkdir -p /home/bluelink/.bluelink/engine/plugins/bin && \
    mkdir -p /home/bluelink/.bluelink/engine/plugins/logs && \
    mkdir -p /home/bluelink/.bluelink/engine/state && \
    chown -R bluelink:bluelink /app /home/bluelink/.bluelink

# Copy binary from builder
COPY --from=builder --chown=bluelink:bluelink /app/deploy-engine /app/deploy-engine

# Switch to non-root user
USER bluelink

# Set working directory
WORKDIR /app

# Expose the default port
EXPOSE 8325

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8325/v1/health || exit 1

# Default environment variables
ENV BLUELINK_DEPLOY_ENGINE_PORT=8325 \
    BLUELINK_DEPLOY_ENGINE_LOOPBACK_ONLY=false \
    BLUELINK_DEPLOY_ENGINE_ENVIRONMENT=production

# Run the application
ENTRYPOINT ["/app/deploy-engine"]
