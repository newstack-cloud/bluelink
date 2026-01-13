# Plugin Registry Test Server

A combined plugin registry and OAuth2/OIDC test server for manual testing of bluelink plugin commands (`login`, `install`, `uninstall`).

## Features

- **Service Discovery**: Serves Bluelink service discovery document at `/.well-known/bluelink-services.json`
- **API Key Authentication**: Validates API keys via `/auth/verify`
- **OAuth2 Client Credentials**: Issues tokens via `/oauth2/token` with `grant_type=client_credentials`
- **OAuth2 Authorization Code + PKCE**: Full browser-based auth flow via `/oauth2/authorize`
- **Standard OIDC Discovery**: OpenID Connect configuration at `/.well-known/openid-configuration`

## Quick Start

### Using Docker Compose (Recommended)

```bash
cd tools/plugin-registry-test-server
docker compose up -d
```

### Using Go directly

```bash
cd tools/plugin-registry-test-server
GOWORK=off go run .
```

## Testing Plugin Login

Once the server is running, test the different authentication flows:

### API Key Authentication

```bash
# The server returns all auth types in service discovery
bluelink plugins login http://localhost:8080

# Select "API Key" when prompted, then enter:
# test-api-key-12345
```

### OAuth2 Client Credentials

```bash
# Select "OAuth2 Client Credentials" when prompted
bluelink plugins login http://localhost:8080

# Enter when prompted:
#   Client ID: test-client-id
#   Client Secret: test-client-secret
```

### OAuth2 Authorization Code

```bash
# Select "OAuth2 Authorization Code" when prompted
bluelink plugins login http://localhost:8080

# Browser will open - click "Approve" to complete the flow
```

### Testing Token Endpoint Directly

```bash
# Client Credentials flow
curl -X POST http://localhost:8080/oauth2/token \
  -u "test-client-id:test-client-secret" \
  -d "grant_type=client_credentials"

# Verify API key
curl http://localhost:8080/auth/verify \
  -H "X-API-Key: test-api-key-12345"
```

## Test Credentials

| Credential | Default Value |
|------------|---------------|
| Client ID | `test-client-id` |
| Client Secret | `test-client-secret` |
| API Key | `test-api-key-12345` |

These can be overridden via environment variables:
- `OAUTH2_CLIENT_ID`
- `OAUTH2_CLIENT_SECRET`
- `OAUTH2_API_KEY`

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `/.well-known/bluelink-services.json` | Bluelink service discovery |
| `/.well-known/openid-configuration` | OIDC discovery document |
| `/.well-known/jwks.json` | JSON Web Key Set |
| `/oauth2/authorize` | Authorization endpoint (browser) |
| `/oauth2/token` | Token endpoint |
| `/auth/verify` | API key verification |
| `/health` | Health check |

## Development

```bash
# Run directly (GOWORK=off needed due to go.work in parent)
GOWORK=off go run .

# Run with auto-reload (requires watchexec)
GOWORK=off watchexec -r -e go "go run ."

# Build Docker image
docker build -t plugin-registry-test-server .

# Run Docker container
docker run -p 8080:8080 plugin-registry-test-server
```
