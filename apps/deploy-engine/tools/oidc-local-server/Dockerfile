###################
# BUILD APPLICATION BINARY
###################

# Please keep up to date with the new-version of Golang docker for builder
FROM golang:1.24.2-bookworm AS build

# Add non-root user
RUN useradd -u 1001 bluelink && \
  apt-get update && \
  apt-get upgrade -y && \
  apt-get install -y --no-install-recommends git build-essential curl && \
  apt-get install --no-install-recommends openssh-server -y

WORKDIR /app/oidc_local_server

# Copy go module files to load dependencies.
COPY go.mod ./go.mod
COPY go.sum ./go.sum

COPY main.go ./main.go

RUN go mod download && \
  go build \
  # Build with static linking to ensure everything is included in the binary
  # which allows us to run the binary with scratch.
  -ldflags="-linkmode external -extldflags -static" \
  -tags netgo \
  -o oidc_local_server \
  main.go

###################
# PRODUCTION IMAGE (lean image to run pre-built binary)
###################

FROM scratch

WORKDIR /

COPY --from=build /etc/passwd /etc/passwd

COPY --from=build /app/oidc_local_server/oidc_local_server /oidc_local_server

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY ./public /public
COPY ./private /private

# Use non-root user
USER bluelink

EXPOSE 80

CMD ["/oidc_local_server"]