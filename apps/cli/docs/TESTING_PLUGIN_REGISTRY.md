# Testing Plugin Registry Commands

This guide explains how to manually test the `bluelink plugins` commands using the plugin registry test server.

## Overview

The plugin registry test server (`tools/plugin-registry-test-server`) provides a local mock plugin registry for testing:

- **Login** - Authentication flows (API Key, OAuth2 Client Credentials, OAuth2 Auth Code)
- **Install** - Plugin discovery, download, and installation (including dependency resolution)
- **Uninstall** - Plugin removal from local machine
- **List** - Listing installed plugins with dependency trees, filtering, and search

## Prerequisites

- Go 1.24 or later
- Docker and Docker Compose (optional, for containerized testing)
- The `bluelink` CLI built locally

## Starting the Test Server

### Option 1: Run directly with Go

```bash
cd tools/plugin-registry-test-server
GOWORK=off go run .
```

### Option 2: Run with Docker Compose

```bash
cd tools/plugin-registry-test-server
docker compose up -d
```

The server will start on `http://localhost:8080`.

## Test Credentials

The test server accepts these default credentials:

| Type | Value |
|------|-------|
| API Key | `test-api-key-12345` |
| Client ID | `test-client-id` |
| Client Secret | `test-client-secret` |

---

## Testing `plugins login`

The login command authenticates with a plugin registry and stores credentials locally.

### API Key Authentication

API Key auth is the simplest flow - you enter your API key and it's stored for future use.

```bash
bluelink plugins login http://localhost:8080
```

When prompted:
1. Select "API Key" from the auth type menu
2. Enter: `test-api-key-12345`

**Expected behavior:**
- Service discovery finds the registry's auth configuration
- You're prompted to select an auth method (since the test server supports all three)
- After entering the API key, credentials are stored in `~/.bluelink/clients/plugins.auth.json`

**Verify the stored credentials:**
```bash
cat ~/.bluelink/clients/plugins.auth.json
```

### OAuth2 Client Credentials

Client credentials flow is typically used for machine-to-machine authentication.

```bash
bluelink plugins login http://localhost:8080
```

When prompted:
1. Select "OAuth2 Client Credentials" from the auth type menu
2. Enter Client ID: `test-client-id`
3. Enter Client Secret: `test-client-secret`

**Expected behavior:**
- CLI exchanges credentials for an access token
- Credentials (not tokens) are stored in `~/.bluelink/clients/plugins.auth.json`

### OAuth2 Authorization Code with PKCE

Authorization code flow opens a browser for interactive login - commonly used for user authentication.

```bash
bluelink plugins login http://localhost:8080
```

When prompted:
1. Select "OAuth2 Authorization Code" from the auth type menu
2. Browser opens to the authorization page
3. Click "Approve" to authorize the application
4. Browser shows success message and can be closed

**Expected behavior:**
- CLI generates PKCE code verifier and challenge
- Browser opens to `http://localhost:8080/oauth2/authorize?...`
- After approval, callback server receives the authorization code
- CLI exchanges code for tokens
- Tokens are stored in `~/.bluelink/clients/plugins.tokens.json`

**Verify the stored tokens:**
```bash
cat ~/.bluelink/clients/plugins.tokens.json
```

### Login Error Cases

**Invalid API Key:**
```bash
bluelink plugins login http://localhost:8080
# Select "API Key", enter: wrong-key
```
Expected: Error message about invalid credentials

**Invalid Client Credentials:**
```bash
bluelink plugins login http://localhost:8080
# Select "OAuth2 Client Credentials"
# Enter: wrong-client-id / wrong-secret
```
Expected: Error message about invalid client credentials

**Authorization Denied:**
```bash
bluelink plugins login http://localhost:8080
# Select "OAuth2 Authorization Code"
# In browser, click "Deny"
```
Expected: Error message about access denied

**Unreachable Registry:**
```bash
bluelink plugins login https://nonexistent.example.com
```
Expected: Error about failed service discovery

---

## Testing `plugins install`

The install command downloads, verifies (GPG signature + SHA256 checksum), and extracts plugins from a registry.

### Prerequisites

Before testing install, ensure you've logged in:
```bash
bluelink plugins login http://localhost:8080
```

### Test Plugins

The test server provides these test plugins:

| Plugin | Versions | Dependencies | Notes |
|--------|----------|--------------|-------|
| `bluelink/test-provider` | 1.0.0, 1.1.0, 2.0.0 | None | Base provider plugin |
| `bluelink/test-transformer` | 1.0.0 | None | Base transformer plugin |
| `bluelink/aws-link-provider` | 1.0.0 | `bluelink/test-provider@1.0.0` | Provider with single dependency |
| `bluelink/multi-cloud-transformer` | 1.0.0 | `bluelink/test-provider@1.0.0`, `bluelink/aws-link-provider@1.0.0` | Plugin with dependency chain |
| `bluelink/bad-signature` | 1.0.0 | None | Returns invalid GPG signature |
| `bluelink/unsigned` | 1.0.0 | None | No signature URLs (should fail) |

### Basic Install

Install a single plugin with version pinning:
```bash
bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
```

**Expected behavior:**
- Plugin metadata is fetched from the registry
- Archive is downloaded with progress reporting
- SHA256SUMS file is downloaded
- GPG signature is downloaded and verified
- Checksum is verified against SHA256SUMS
- Archive is extracted to plugin directory
- Manifest is updated

### Install Multiple Plugins

```bash
bluelink plugins install \
  localhost:8080/bluelink/test-provider@1.0.0 \
  localhost:8080/bluelink/test-transformer@1.0.0
```

### Install with Dependency Resolution

Install a plugin that has a dependency on another plugin. The CLI automatically resolves and installs dependencies first:
```bash
bluelink plugins install localhost:8080/bluelink/aws-link-provider@1.0.0
```

**Expected behavior:**
- CLI resolves that `aws-link-provider` depends on `bluelink/test-provider@1.0.0`
- Both plugins are installed in topological order (dependency first)
- Dependency plugins are labeled with `(dependency)` in the output
- Shows "Installed: 2" in the summary

### Install Deep Dependency Chain

Install a plugin with a multi-level dependency tree:
```bash
bluelink plugins install localhost:8080/bluelink/multi-cloud-transformer@1.0.0
```

**Expected behavior:**
- CLI resolves the full dependency tree:
  1. `bluelink/test-provider@1.0.0` (no dependencies)
  2. `bluelink/aws-link-provider@1.0.0` (depends on test-provider)
  3. `bluelink/multi-cloud-transformer@1.0.0` (depends on both)
- All three plugins are installed in topological order
- Dependency plugins are labeled with `(dependency)` in the output
- Shows "Installed: 3" in the summary

### Verify Dependencies in Manifest

After installing plugins with dependencies, inspect the manifest to see dependency metadata:
```bash
cat ${BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH:-~/.bluelink/engine/plugins}/manifest.json | jq '
  .plugins | to_entries[] | {
    plugin: .value.id,
    type: .value.type,
    dependencies: .value.dependencies
  }'
```

### Install Latest Version

Omit the version to install the latest:
```bash
bluelink plugins install localhost:8080/bluelink/test-provider
```

### Verify Installation

Check the manifest:
```bash
cat ${BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH:-~/.bluelink/engine/plugins/bin}/manifest.json
```

Check extracted files:
```bash
ls ${BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH:-~/.bluelink/engine/plugins/bin}/bluelink/test-provider/1.0.0/
```

### Custom Plugin Directory

Use the `BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH` environment variable:
```bash
BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH=/tmp/my-plugins \
  bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
```

### Install Error Cases

**Plugin not found:**
```bash
bluelink plugins install localhost:8080/bluelink/nonexistent@1.0.0
```
Expected: Error "plugin not found"

**Version not found:**
```bash
bluelink plugins install localhost:8080/bluelink/test-provider@99.99.99
```
Expected: Error "version not found"

**Not logged in:**
```bash
rm -f ~/.bluelink/clients/plugins.auth.json
bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
```
Expected: Error "no credentials found - run 'bluelink plugins login' first"

**Invalid GPG signature:**
```bash
bluelink plugins install localhost:8080/bluelink/bad-signature@1.0.0
```
Expected: Error "signature verification failed"

**Missing signature (unsigned plugin):**
```bash
bluelink plugins install localhost:8080/bluelink/unsigned@1.0.0
```
Expected: Error "signature required but not provided by registry"

### Plugin Already Installed

If a plugin with the same version is already installed, it will be skipped:
```bash
bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
# Run again - should skip
bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
```
Expected: Shows "Skipped: 1" on second run

---

## Testing `plugins uninstall`

The uninstall command removes locally installed plugins from the plugin directory and updates the manifest.

### Prerequisites

Before testing uninstall, install some plugins first:
```bash
bluelink plugins login http://localhost:8080
bluelink plugins install localhost:8080/bluelink/test-provider@1.0.0
bluelink plugins install localhost:8080/bluelink/test-transformer@1.0.0
```

### Basic Uninstall

Uninstall a single plugin:
```bash
bluelink plugins uninstall localhost:8080/bluelink/test-provider
```

**Expected behavior:**
- Plugin files are removed from the plugin directory
- Plugin entry is removed from manifest
- Empty parent directories are cleaned up
- Shows "Removed: 1" in the summary

### Uninstall Multiple Plugins

```bash
bluelink plugins uninstall \
  localhost:8080/bluelink/test-provider \
  localhost:8080/bluelink/test-transformer
```

**Expected behavior:**
- Both plugins are removed
- Shows "Removed: 2" in the summary

### Uninstall with Default Registry

For plugins installed from the default registry (`registry.bluelink.dev`), you can use the short form:
```bash
bluelink plugins uninstall bluelink/aws
```

This is equivalent to:
```bash
bluelink plugins uninstall registry.bluelink.dev/bluelink/aws
```

### Verify Uninstallation

Check that the plugin was removed from the manifest:
```bash
cat ${BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH:-~/.bluelink/engine/plugins}/manifest.json
```

Check that plugin files were removed:
```bash
ls ${BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH:-~/.bluelink/engine/plugins}/bin/bluelink/
```

### Uninstall Error Cases

**Plugin not installed (not found):**
```bash
bluelink plugins uninstall localhost:8080/bluelink/nonexistent
```
Expected: Shows "Not found: 1" in the summary (not an error, graceful handling)

**Missing plugin argument:**
```bash
bluelink plugins uninstall
```
Expected: Error "requires at least 1 arg"

**Invalid plugin ID format:**
```bash
bluelink plugins uninstall invalid-format
```
Expected: Error about invalid plugin ID

### Mixed Results

When uninstalling multiple plugins where some exist and some don't:
```bash
bluelink plugins uninstall \
  localhost:8080/bluelink/test-provider \
  localhost:8080/bluelink/nonexistent
```
Expected: Shows "Removed: 1, Not found: 1" (processes all plugins, reports combined results)

### Custom Plugin Directory

Use the `BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH` environment variable:
```bash
BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH=/tmp/my-plugins \
  bluelink plugins uninstall localhost:8080/bluelink/test-provider
```

---

## Testing `plugins list`

The list command shows all installed plugins with their dependency trees, supports type filtering and name search.

### Prerequisites

Before testing list, install some plugins including ones with dependencies:
```bash
bluelink plugins login http://localhost:8080
bluelink plugins install \
  localhost:8080/bluelink/test-provider@1.0.0 \
  localhost:8080/bluelink/test-transformer@1.0.0
bluelink plugins install localhost:8080/bluelink/aws-link-provider@1.0.0
```

### List All Plugins

```bash
bluelink plugins list
```

**Expected behavior (interactive mode):**
- Shows a styled list of all installed plugins grouped by type
- Each plugin shows version, install date, and dependency tree
- Footer shows total count and keybinding hints
- Press `↑/↓` or `j/k` to navigate, `/` to search, `q` to quit

### Filter by Type

```bash
bluelink plugins list --type provider
```

**Expected behavior:**
- Only plugins of type "provider" are shown
- Header includes "(type: provider)"
- `test-transformer` is excluded from the list

### Search by Name

```bash
bluelink plugins list --search "aws"
```

**Expected behavior:**
- Only plugins matching "aws" (case-insensitive) are shown
- `aws-link-provider` appears in results
- Other plugins are excluded
- Header includes `(search: "aws")`

### Verify Dependency Tree in Output

After installing `aws-link-provider` (which depends on `test-provider`):
```bash
bluelink plugins list | cat
```

**Expected behavior (headless output):**
- Output includes `bluelink/aws-link-provider@1.0.0 [provider]`
- Dependency section shows `Dependencies:` followed by `bluelink/test-provider@1.0.0`
- Total count reflects all installed plugins

### Headless Output

When piped or in a non-interactive terminal, output is plain text with `[plugins]` prefix:
```bash
bluelink plugins list | cat
```

**Expected output format:**
```
[plugins]
[plugins] Installed Plugins
[plugins] ════════════════════════════════════════════════════════════
[plugins]   bluelink/test-provider@1.0.0 [provider]
[plugins]     Version: 1.0.0
[plugins]     Registry: localhost:8080
[plugins]     Installed: 2025-01-30 10:00:00
[plugins]   bluelink/aws-link-provider@1.0.0 [provider]
[plugins]     Version: 1.0.0
[plugins]     Registry: localhost:8080
[plugins]     Installed: 2025-01-30 10:00:01
[plugins]     Dependencies:
[plugins]       - bluelink/test-provider@1.0.0
[plugins]
[plugins] ════════════════════════════════════════════════════════════
[plugins] Total: 2 plugin(s)
```

### Interactive Search

In interactive mode (not piped), you can search dynamically:
1. Run `bluelink plugins list`
2. Press `/` to enter search mode
3. Type a search term (e.g., "provider")
4. Press `Enter` to apply the filter
5. Press `Esc` to clear the search

### List with No Plugins

If no plugins are installed:
```bash
BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH=/tmp/empty-plugins \
  bluelink plugins list | cat
```

**Expected behavior:**
- Shows "No plugins found."
- Shows "Total: 0 plugin(s)"

---

## Verifying Endpoints Directly

You can test the OAuth2 and registry endpoints directly with curl:

### Service Discovery
```bash
curl http://localhost:8080/.well-known/bluelink-services.json | jq .
```

### OIDC Configuration
```bash
curl http://localhost:8080/.well-known/openid-configuration | jq .
```

### Client Credentials Token
```bash
curl -X POST http://localhost:8080/oauth2/token \
  -u "test-client-id:test-client-secret" \
  -d "grant_type=client_credentials"
```

### API Key Verification
```bash
curl http://localhost:8080/auth/verify \
  -H "X-API-Key: test-api-key-12345"
```

### List Plugin Versions
```bash
curl http://localhost:8080/v1/plugins/bluelink/test-provider/versions \
  -H "Authorization: Bearer test-api-key-12345" | jq .
```

### Get Package Metadata
```bash
curl http://localhost:8080/v1/plugins/bluelink/test-provider/1.0.0/package/darwin/arm64 \
  -H "Authorization: Bearer test-api-key-12345" | jq .
```

### Get Package Metadata with Dependencies
```bash
curl http://localhost:8080/v1/plugins/bluelink/multi-cloud-transformer/1.0.0/package/darwin/arm64 \
  -H "Authorization: Bearer test-api-key-12345" | jq .dependencies
```

Expected output:
```json
{
  "bluelink/test-provider": "1.0.0",
  "bluelink/aws-link-provider": "1.0.0"
}
```

### Download Plugin Archive
```bash
curl -O http://localhost:8080/download/bluelink/test-provider/1.0.0/test-provider_1.0.0_darwin_arm64.tar.gz
```

### Download SHA256SUMS
```bash
curl http://localhost:8080/download/bluelink/test-provider/1.0.0/SHA256SUMS
```

### Download GPG Signature
```bash
curl http://localhost:8080/download/bluelink/test-provider/1.0.0/SHA256SUMS.sig
```

---

## Cleaning Up

### Remove stored credentials
```bash
rm -f ~/.bluelink/clients/plugins.auth.json
rm -f ~/.bluelink/clients/plugins.tokens.json
```

### Stop the test server
```bash
# If running with Docker Compose
cd tools/plugin-registry-test-server
docker compose down

# If running with Go, just Ctrl+C
```

---

## Troubleshooting

### "API key authentication requires an interactive terminal"
This error appears when running in headless mode (CI, piped output). The login command requires an interactive terminal for user input.

### "multiple authentication methods available; interactive terminal required"
Similar to above - when a registry supports multiple auth types, user selection is needed.

### Browser doesn't open
If the browser doesn't open automatically for authorization code flow:
1. Check if `BROWSER` or `DISPLAY` environment variables are set
2. Manually open the URL shown in the terminal

### Callback server port conflict
The authorization code flow starts a local callback server. If port 8000-8100 range is busy, the flow may fail. Check for processes using those ports.

### Server not responding
Ensure the test server is running:
```bash
curl http://localhost:8080/health
```

If using Docker Compose:
```bash
docker compose -f tools/plugin-registry-test-server/docker-compose.yaml logs
```
