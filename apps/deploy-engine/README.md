# Bluelink Deploy Engine

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)

The engine that validates and deploys blueprints for infrastructure as code deployments.

The deploy engine bundles the plugin framework's gRPC-based plugin system that allows for the creation of custom plugins for providers and transformers that can be pulled in at runtime.
The deploy engine also bundles a limited set of state persistence implementations for blueprint instances, the persistence implementation can be chosen with configuration.

- [Installing](#installing)
- [Configuration](#configuration)
- [Maintenance Commands](#maintenance-commands)
- [API Documentation](#api-documentation)
- [Additional documentation](#additional-documentation)

## Installing

The Deploy Engine is available for installation as a part of the standard Bluelink installation.
See the [installing Bluelink documentation](https://bluelink.dev/docs/intro/installing-bluelink) for more information.

### Docker

You can also run the Deploy Engine as a Docker container using the public Docker image.

```bash
docker pull ghcr.io/newstack-cloud/bluelink-deploy-engine:latest
```

## Configuration

The deploy engine can be configured through environment variables or configuration files. This section lists both required and optional configuration along with default values.

### Configuration Methods

The deploy engine supports two configuration methods:

1. **Environment Variables** - Set environment variables with the `BLUELINK_DEPLOY_ENGINE_` prefix
2. **Configuration Files** - Use JSON, YAML, or TOML configuration files

#### Configuration Priority

When both methods are used, the configuration is applied in the following order (highest to lowest priority):

1. Environment variables
2. Configuration file values
3. Default values

This means environment variables will always override values set in configuration files.

#### Configuration Files

The deploy engine will search for a configuration file named `config.json`, `config.yaml`, or `config.toml` in the following locations (in order):

1. Current working directory (`.`)
2. OS-specific default config directory:
   - **Linux/macOS:** `$HOME/.bluelink/engine`
   - **Windows:** `%LOCALAPPDATA%\NewStack\Bluelink\engine`
3. Custom path specified by the `BLUELINK_DEPLOY_ENGINE_CONFIG_PATH` environment variable

A [config.example.json](config.example.json) file is provided as a reference for the structure of the configuration file.

**Example config.json:**

```json
{
  "port": 8325,
  "log_level": "info",
  "state": {
    "storage_engine": "memfile"
  }
}
```

#### Environment Variables

All configuration options can be set via environment variables using the `BLUELINK_DEPLOY_ENGINE_` prefix. Nested configuration fields use underscores to separate levels.

For example, the `state.storage_engine` configuration field can be set with:

```bash
BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE=postgres
```

**Special Environment Variable Handling:**

- **String slices** - Use comma-separated values: `"key1,key2,key3"` becomes `["key1", "key2", "key3"]`
- **Maps** - Use comma-separated key:value pairs: `"key1:value1,key2:value2"` becomes `{"key1": "value1", "key2": "value2"}`
- **Path expansion** - Environment variables in path values (e.g., `$HOME`, `%HOME%`) are automatically expanded

### Configuration Options

- [Server](#server)
- [Authentication](#authentication)
- [Plugins](#plugins)
- [Blueprints](#blueprints)
- [State](#state)
- [Resolvers](#resolvers)
- [Maintenance](#maintenance)

### Server

Core configuration for the Deploy Engine server.

#### Version

`BLUELINK_DEPLOY_ENGINE_API_VERSION`

_Config field:_ `api_version`

_**optional**_

The version of the deploy engine API. This is used to identify the version of the deploy engine HTTP API to use.
This is not the same as the version of the deploy engine itself, but rather the version of the HTTP API that the deploy engine exposes.
For the current implementation of the deploy engine, only `v1` is supported.

**default value:** `v1`

#### Port

`BLUELINK_DEPLOY_ENGINE_PORT`

_Config field:_ `port`

_**optional**_

The port that the deploy engine will listen on for incoming requests. This is used to configure the HTTP server.

**default value:** `8325`

#### Use Unix Socket?

`BLUELINK_DEPLOY_ENGINE_USE_UNIX_SOCKET`

_Config field:_ `use_unix_socket`

_**optional**_

If set to `true`, the deploy engine will use a Unix socket instead of a TCP socket. This is used to configure the HTTP server and should only be used when the deploy engine is running on a local machine where deployments and Bluelink applications are managed with local state.

**default value:** `false`

#### Unix Socket Path

`BLUELINK_DEPLOY_ENGINE_UNIX_SOCKET_PATH`

_Config field:_ `unix_socket_path`

_**optional**_

The path to the Unix socket that the deploy engine will listen on for incoming requests. This is used to configure the HTTP server and should only be used when the deploy engine is running on a local machine where deployments and Bluelink applications are managed with local state.
This will only be used if `BLUELINK_DEPLOY_ENGINE_USE_UNIX_SOCKET` is set to `true`.

**default value:** `/tmp/bluelink.sock`

#### Loopback Interface Only

`BLUELINK_DEPLOY_ENGINE_LOOPBACK_ONLY`

_Config field:_ `loopback_only`

_**optional**_

If set to `false`, the deploy engine will be accessible over the wider private network or public internet.
The default behaviour is to only allow access from the loopback interface (127.0.0.1) where connections
are only accepted from the same machine the deploy engine is running on.
This needs to be intentionally set to `false` to allow access over a wider private network or the public internet.

**default value:** `true`

#### IP Extraction Source

`BLUELINK_DEPLOY_ENGINE_IP_EXTRACTION_SOURCE`

_Config field:_ `ip_extraction_source`

_**optional**_

How the source IP of a request is determined. This is the address that actions
are attributed to in the audit trail, so it should be set to match what sits in
front of the deploy engine.

Supported values:

| Value | Where the address is read from |
| --- | --- |
| `direct` | The address the connection came from. Every header is ignored. |
| `aws-alb` | The entry an AWS application load balancer appended to `X-Forwarded-For`. A port is included when `routing.http.xff_client_port.enabled` is set on the load balancer, and is stripped. |
| `gcp-lb` | The client entry a Google Cloud external HTTP(S) load balancer appended to `X-Forwarded-For`. That balancer appends the client address followed by its own, so the client is the second entry from the right. |
| `azure-app-gateway` | The entry an Azure Application Gateway appended to `X-Forwarded-For`, which carries a port. |
| `azure-front-door` | The `X-Azure-ClientIP` header, which Azure Front Door sets. |
| `cloudflare` | The `CF-Connecting-IP` header, which Cloudflare sets. |

The default is `direct` because a deploy engine reached directly is the case
where a forwarded header is purely the caller's own claim. Trusting one by
default would let any caller choose the address recorded against its actions.
Only set this to match a proxy that is actually in the request path, since that
is what makes its header trustworthy.

For the values that read `X-Forwarded-For`, the entry is counted from the right
rather than the left. A caller can always send a header of its own, and these
proxies append after whatever it sent, so the right-hand end is the part the
caller cannot reach. A request carrying fewer entries than the configured proxy
appends did not come through that proxy, and is recorded with no source IP.

A value that is not an address is recorded as no source IP rather than as
itself. Starting the engine with a value that is not in the table above fails
rather than silently recording no address at all.

**default value:** `direct`

#### Environment

`BLUELINK_DEPLOY_ENGINE_ENVIRONMENT`

_Config field:_ `environment`

_**optional**_

The environment that the deploy engine is running in. This is used to determine things like the formatting of logs, in development mode, logs are formatted in a more human readable format,
while in production mode, logs are formatted purely in JSON for easier parsing and processing by log management systems.
This can be set to `development` or `production`.

**default value:** `production`

#### Log Level

`BLUELINK_DEPLOY_ENGINE_LOG_LEVEL`

_Config field:_ `log_level`

_**optional**_

The log level that the deploy engine will use. This is used to determine the verbosity of the logs that are generated by the deploy engine.
This can be set to any of the logging levels supported by zap:
`debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.
See: [Zap Logging Levels](https://pkg.go.dev/go.uber.org/zap#section-documentation)

**default value:** `info`

### Authentication

Configuration for the authentication methods that the deploy engine will use to authenticate requests.

#### OAuth2/OIDC JWT Issuer

`BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_ISSUER`

_Config field:_ `auth.oauth2_oidc_jwt_issuer`

_**required, if JWT authentication will be used**_

The issuer URL of an OAuth2/OIDC JWT token that can be used to authenticate with the deploy engine. This is used to verify the JWT token provided with the bearer scheme in the `Authorization` header of all requests.
There can only be one issuer configured for an instance of the deploy engine.
This will be checked before the Bluelink signature and API key authentication methods.

See the [JWTs](https://bluelink.dev/docs/auth/jwts) documentation for more information on the requirements for the issuer.

**Example:**

```
BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_ISSUER=oauth.example.com
```

#### OAuth2/OIDC JWT Issuer Secure

`BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_ISSUER_SECURE`

_Config field:_ `auth.oauth2_oidc_jwt_issuer_secure`

_**optional**_

If set to `true`, the deploy engine will make requests to get metadata and the JSON Web Key Set (JWKS) from the issuer URL over HTTPS.
If set to `false`, the deploy engine will make requests to get metadata and the JSON Web Key Set (JWKS) from the issuer URL over HTTP.
This should be set to `true` for production deployments and `false` for local development and testing.

**default value:** `true`

#### OAuth2/OIDC JWT Audience

`BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_AUDIENCE`

_Config field:_ `auth.oauth2_oidc_jwt_audience`

_**required, if JWT authentication will be used**_

The audience of the OAuth2/OIDC JWT token that can be used to authenticate with the deploy engine. This is used to verify the JWT token provided with the bearer scheme in the `Authorization` header of all requests.

See the [JWTs](https://bluelink.dev/docs/auth/jwts) documentation for more information on the requirements for the audience.

**Example:**

```
BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_AUDIENCE=deploy-engine-app-client-id
```

#### OAuth2/OIDC JWT Signing Algorithm

`BLUELINK_DEPLOY_ENGINE_AUTH_OAUTH2_OIDC_JWT_SIGNATURE_ALGORITHM`

_Config field:_ `auth.oauth2_oidc_jwt_signature_algorithm`

_**optional**_

The signing algorithm to use when verifying the JWT token provided with the bearer scheme in the `Authorization` header of all requests.

This can be set to any of the following signing algorithms:

- `EdDSA` - Edwards-curve Digital Signature Algorithm
- `HS256` - HMAC using SHA-256
- `HS384` - HMAC using SHA-384
- `HS512` - HMAC using SHA-512
- `RS256` - RSASSA-PKCS-v1.5 using SHA-256
- `RS384` - RSASSA-PKCS-v1.5 using SHA-384
- `RS512` - RSASSA-PKCS-v1.5 using SHA-512
- `ES256` - ECDSA using P-256 and SHA-256
- `ES384` - ECDSA using P-384 and SHA-384
- `ES512` - ECDSA using P-521 and SHA-512
- `PS256` - RSASSA-PSS using SHA256 and MGF1-SHA256
- `PS384` - RSASSA-PSS using SHA384 and MGF1-SHA384
- `PS512` - RSASSA-PSS using SHA512 and MGF1-SHA512

**default value:** `HS256`

#### Bluelink Signature v1 Key Pairs

`BLUELINK_DEPLOY_ENGINE_AUTH_BLUELINK_SIGNATURE_V1_KEY_PAIRS`

_Config field:_ `auth.bluelink_signature_v1_key_pairs`

_**required, if Bluelink Signature v1 authentication will be used**_

A comma-separated list of Bluelink signature key pairs that can be used to authenticate with the deploy engine. This is used to verify the Bluelink signature provided in the `Bluelink-Signature-V1` header of all requests.

The key pairs are in the format `keyId:secretKey`, where the public key is used to verify the signature and the private key is used to sign the request.

The deploy engine will check the `Bluelink-Signature-V1` header of all requests.
This will be checked after the OAuth2/OIDC JWT bearer token authentication method and before the API key authentication method.

**Example:**

```
BLUELINK_DEPLOY_ENGINE_AUTH_BLUELINK_SIGNATURE_V1_KEY_PAIRS=keyId1:secretKey1,keyId2:secretKey2
```

#### API Keys

`BLUELINK_DEPLOY_ENGINE_AUTH_BLUELINK_API_KEYS`

_Config field:_ `auth.bluelink_api_keys`

_**required, if API key authentication will be used**_

A comma-separated list of API keys that are allowed to access the deploy engine. This is used to authenticate all requests to the deploy engine.
The deploy engine will check the `Bluelink-Api-Key` header of all requests.
This will be checked after the JWT bearer token and Bluelink signature authentication methods.

**Example:**

```
BLUELINK_DEPLOY_ENGINE_AUTH_API_KEYS=key1,key2,key3
```

### Plugins

Configuration for the plugin host used to manage and interact with plugins for providers and transformers.

#### Plugin Path

`BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH`

_Config field:_ `plugins_v1.plugin_path`

_**optional**_

The path to one or more plugin root directories separated by an OS path list separator (colon on Unix, semicolon on Windows).
This environment variable, generally should be set globally when installed on developer machine as is used by multiple components of the Bluelink framework.

**default value:** `$HOME/.bluelink/engine/plugins/bin`

#### Plugin Log File Root Directory

`BLUELINK_DEPLOY_ENGINE_PLUGIN_LOG_FILE_ROOT_DIR`

_Config field:_ `plugins_v1.log_file_root_dir`

_**optional**_

The path to the root directory where plugin log files will be stored.
stdout and stderr for each plugin will be redirected to log files under this directory.

**default value:** `$HOME/.bluelink/engine/plugins/logs`

#### Plugin Launch Timeout in Milliseconds

`BLUELINK_DEPLOY_ENGINE_PLUGINS_V1_LAUNCH_WAIT_TIMEOUT_MS`
_Config field:_ `plugins_v1.launch_wait_timeout_ms`

_**optional**_

The timeout in milliseconds to wait for a plugin to launch before giving up and returning an error.
This is used when the plugin host is started and a plugin is expected to register with the host.

**default value:** `15000` (15 seconds)

#### Total Plugin Launch Timeout in Milliseconds

`BLUELINK_DEPLOY_ENGINE_PLUGINS_V1_TOTAL_LAUNCH_WAIT_TIMEOUT_MS`
_Config field:_ `plugins_v1.total_launch_wait_timeout_ms`

_**optional**_

The total timeout in milliseconds to wait for all plugins to launch before giving up and returning an error.
This is used when the plugin host is started and all plugins are expected to register with the host.

**default value:** `60000` (1 minute)

#### Resource Stabilisation Polling Timeout in Milliseconds

`BLUELINK_DEPLOY_ENGINE_PLUGINS_V1_RESOURCE_STABILISATION_POLLING_TIMEOUT_MS`
_Config field:_ `plugins_v1.resource_stabilisation_polling_timeout_ms`

_**optional**_

The timeout in milliseconds to wait for a resource to stabilise before giving up and returning an error.
This is used both in the context of plugins and the deploy engine itself.

The purpose of this for plugins is in link plugins that are given access to a helper interface
to deploy resources managed by the host, the link plugin will call into the resource registry of the host
to deploy resources and wait for them to stabilise before returning.

In the deploy engine, this will be used to wait for resources to stabilise before continuing to deploy the next elements of the blueprint that can only be deployed once the current resource is stable.

**default value:** `3600000` (1 hour)

#### Plugin to Plugin Call Timeout in Milliseconds

`BLUELINK_DEPLOY_ENGINE_PLUGINS_V1_PLUGIN_TO_PLUGIN_CALL_TIMEOUT_MS`
_Config field:_ `plugins_v1.plugin_to_plugin_call_timeout_ms`

_**optional**_

The timeout in milliseconds to wait for a plugin to respond to a call initiated from within a plugin before giving up and returning an error.
This helps eliminate infinite waits in the case of a plugin that is not responding or has crashed or where an erroneous recursive call could lead to an infinite loop.
The exception, where this timeout is not used, is when a link plugin is waiting for a resource to stabilise,
in which case the resource stabilisation polling timeout will be used.

**default value:** `120000` (2 minutes)

### Blueprints

Configuration for the blueprint loader/container used to load and manage blueprint instances along with validating source blueprint files.

#### Validate After Transform

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_VALIDATE_AFTER_TRANSFORM`

_Config field:_ `blueprints.validate_after_transform`

_**optional**_

Determines whether or not the blueprint loader should validate blueprints after applying transformations.
This should only really be set to true when there is a need to debug issues that may be due to transformer
plugins producing invalid output.

**default value:** `false`

#### Enable Drift Checks

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_ENABLE_DRIFT_CHECK`

_Config field:_ `blueprints.enable_drift_check`

_**optional**_

Determines whether or not the deploy engine should check for drift in the state of resources
when staging changes for a blueprint deployment.
Drift checks use the `GetExternalState` method of a resource implementation to check the state of the resource against the upstream provider.

**default value:** `true`

#### Resource Stabilisation Polling Interval in Milliseconds

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_RESOURCE_STABILISATION_POLLING_INTERVAL_MS`

_Config field:_ `blueprints.resource_stabilisation_polling_interval_ms`

_**optional**_

The interval in milliseconds to wait between polling for a resource to stabilise.
This is used both in the context of plugins and the deploy engine itself.
The purpose of this for plugins is in link plugins that are given access to a helper interface
to deploy resources managed by the host, the link plugin will call into the resource registry of the host
to deploy resources and wait for them to stabilise before returning.
In the deploy engine, this will be used to wait for resources to stabilise before continuing to deploy the next elements of the blueprint that can only be deployed once the current resource is stable.

**default value:** `5000` (5 seconds)

#### Default Retry Policy

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_DEFAULT_RETRY_POLICY`

_Config field:_ `blueprints.default_retry_policy`

_**optional**_

The default retry policy to use when a provider returns a retryable error for actions
that support retries.

**important:** This must be a **serialised JSON string** regardless of the configuration format used (environment variable, JSON config file, YAML config file, etc.). The value should always be a JSON string, not a native object/map structure.

The built-in default will be used if this is not set or the JSON is not in the correct format.
When a provider plugin has its own retry policy, that will always be used instead of the default.

The JSON string should match the structure of the `provider.RetryPolicy` struct:

**Example JSON structure:**

```json
{
  "maxRetries": 5,
  "firstRetryDelay": 2,
  "maxDelay": 300,
  "backofFactor": 2,
  "jitter": true
}
```

**Example in environment variable:**
The value must be in a single line and escaped appropriately:

```bash
BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_DEFAULT_RETRY_POLICY='{"maxRetries":5,"firstRetryDelay":2,"maxDelay":300,"backofFactor":2,"jitter":true}'
```

**Example in config.json:**

```json
{
  "blueprints": {
    "default_retry_policy": "{\"maxRetries\":5,\"firstRetryDelay\":2,\"maxDelay\":300,\"backofFactor\":2,\"jitter\":true}"
  }
}
```

**Example in config.yaml:**

```yaml
blueprints:
  default_retry_policy: '{"maxRetries":5,"firstRetryDelay":2,"maxDelay":300,"backofFactor":2,"jitter":true}'
```

#### Deployment Timeout in Seconds

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_DEPLOYMENT_TIMEOUT`

_Config field:_ `blueprints.deployment_timeout`

_**optional**_

The timeout in seconds to wait for a deployment to complete before giving up and returning an error.
This timeout is for the background process that runs the deployment when the deployment endpoints are called.

**default value:** `10800` (3 hours)

#### Drain Timeout in Seconds

`BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_DRAIN_TIMEOUT`

_Config field:_ `blueprints.drain_timeout`

_**optional**_

The time in seconds to wait for in-flight operations to complete after a terminal failure
before marking them as interrupted. When a resource fails during deployment, other resources
that are currently deploying are given this amount of time to complete before being marked
as interrupted. Resources in the CONFIG_COMPLETE stage (stabilization polling) benefit from
longer drain times as they have a higher chance of reaching the final CREATED status.

Increasing this value reduces the number of interrupted resources after a failure, making
subsequent re-deploy or destroy operations easier to manage.

**default value:** `120` (2 minutes)

### State

Configuration for the state management/persistence layer used by the deploy engine.

#### Storage Engine

`BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE`

_Config field:_ `state.storage_engine`

_**optional**_

The storage engine to use for the state management/persistence layer.
Valid values are `memfile`, `postgres`, and `objectstore`.

Postgres should be used for deploy engine deployments that need to scale horizontally,
the in-memory storage with file system persistence engine should be used for local deployments,
CI environments and production use cases where the deploy engine is not expected to scale horizontally.

Objectstore (S3, GCS or Azure Blob Storage) should be used when many CI/CD pipelines or
deploy engine instances need to share state safely without a managed database — per-entity
ETag / generation CAS serialises concurrent writers against a shared bucket/container.

If opting for the in-memory storage with file system persistence engine,
it would be a good idea to backup the state files to a remote location
to avoid losing all state in the event of a failure or destruction of the host machine.

State includes resolved secrets, so a `memfile` deployment should either encrypt the state
itself with `state.encryption.enabled`, documented below, or hold the state directory and its
backups on encrypted storage. The `postgres` and `objectstore` engines rely on encryption at
rest in the database or bucket they persist to. See
[secure deployment](docs/SECURE_DEPLOYMENT.md) for the whole picture.

**default value:** `memfile`

#### Recently Queued Events Threshold

`BLUELINK_DEPLOY_ENGINE_STATE_RECENTLY_QUEUED_EVENTS_THRESHOLD`

_Config field:_ `state.recently_queued_events_threshold`

_**optional**_

The threshold in seconds for retrieving recently queued events for a stream when a starting event ID is not provided.

Any events that are older than currentTime - threshold will not be considered as recently queued events.

This applies to all storage engines.

**default value:** `300` (5 minutes)

#### `memfile` Storage Engine State Directory

`BLUELINK_DEPLOY_ENGINE_STATE_MEMFILE_STATE_DIR`

_Config field:_ `state.memfile_state_dir`

_**optional**_

The directory to use for persisting state files when using the in-memory storage with file system
(memfile) persistence engine.

**default value:** `$HOME/.bluelink/engine/state`

#### `memfile` Storage Engine Max Guide File Size

`BLUELINK_DEPLOY_ENGINE_STATE_MEMFILE_MAX_GUIDE_FILE_SIZE`

_Config field:_ `state.memfile_max_guide_file_size`

_**optional**_

This sets the guide for the maximum size of a state chunk file in bytes
when using the in-memory storage with file system (memfile) persistence engine.
If a single record (instance or resource drift entry) exceeds this size,
it will not be split into multiple files.
This is only a guide, the actual size of the files are often likely to be larger.

**default value:** `1048576` (1MB)

#### `memfile` Storage Engine Max Event Partition File Size

`BLUELINK_DEPLOY_ENGINE_STATE_MEMFILE_MAX_EVENT_PARTITION_SIZE`

_Config field:_ `state.memfile_max_event_partition_size`

_**optional**_

This sets the maximum size of an event channel partition file in bytes
when using the in-memory storage with file system (memfile) persistence engine.
Each channel (e.g. deployment or change staging process) will have its own partition file
for events that are captured from the blueprint container.
This is a hard limit, if a new event is added to a partition file
that causes the file to exceed this size, an error will occur and the event
will not be persisted.

**default value:** `10485760` (10MB)

#### `memfile` Storage Engine State Encryption Enabled

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_ENABLED`

_Config field:_ `state.encryption.enabled`

_**optional**_

Whether the deploy engine encrypts state before writing it to disk when using the in-memory
storage with file system (memfile) persistence engine. State holds resolved secrets, such as a
connection string in a resource spec or a token in a change set, so encrypting it means the
host's disk, and any backup taken from it, no longer carries readable credentials.

Only supported for the `memfile` storage engine. The `postgres` and `objectstore` engines rely
on encryption at rest in the database or bucket they persist to, and the deploy engine will
refuse to start if this is enabled for either.

The key is for the life of the state it protects, and losing it loses the state. Changing it is
a two-step operation: list the old key under `state.encryption.previous_keys` so existing state
stays readable, then run the [`reencrypt-state`](#reencrypt-state) command to move the state
onto the new key, after which the old key can be dropped from the configuration.

Enabling this over a directory of existing plaintext state will fail at startup rather than
read it. Encrypt what is already there with the
[`reencrypt-state`](#reencrypt-state) command, with the deploy engine stopped.

**default value:** `false`

#### `memfile` Storage Engine State Encryption Key Provider

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_PROVIDER`

_Config field:_ `state.encryption.provider`

_**optional**_

Selects where the key that protects the state comes from.

Valid values are `passphrase`, `localkey`, `awskms`, `gcpkms` and `azurekeyvault`.

`passphrase` derives a key from a passphrase with Argon2id, the salt being generated once and
persisted alongside the state. `localkey` reads a raw 32-byte key from a file. The remaining
three wrap each object's data key with a key held by a cloud key management service, which
never releases the key itself. These are the stronger choice where one is available, the key
being held by a service with its own access control and audit trail rather than by the engine's
host, at the cost of a wrap call per object written.

**default value:** `passphrase`

#### `memfile` Storage Engine State Encryption Previous Keys

_Config field:_ `state.encryption.previous_keys`

_**optional**_

Keys that are no longer used to encrypt state, but are still needed to decrypt state written
before the current key was configured. Each entry takes the same fields as the current key, so
a deployment can move between provider types as well as between keys of one type.

This is what makes changing the key survivable. Without the old key listed here, state written
under it cannot be read, and the deploy engine refuses to start rather than write over it.

Keep an old key here only until the state has been moved onto the current key with the
[`reencrypt-state`](#reencrypt-state) command. Until that pass has run, the old key is still
the only thing protecting most of the state, so retiring it achieves nothing on its own.

There is no environment variable equivalent, a list of keys can only be given in a
configuration file:

```json
{
  "state": {
    "encryption": {
      "enabled": true,
      "provider": "awskms",
      "awskms": { "key_id": "alias/bluelink-state" },
      "previous_keys": [
        { "provider": "passphrase", "passphrase_file": "/run/secrets/old-state-passphrase" }
      ]
    }
  }
}
```

#### `memfile` Storage Engine State Encryption Passphrase / Passphrase File

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_PASSPHRASE`
`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_PASSPHRASE_FILE`

_Config fields:_ `state.encryption.passphrase`, `state.encryption.passphrase_file`

_**one of these is required, if the passphrase key provider is used**_

The passphrase a `passphrase` key provider derives its key from, either directly or as the path
to a file holding it. The file is preferred where a deployment mounts secrets as files, since
the value then stays out of the process environment, and takes precedence when both are set.
Trailing whitespace in the file is ignored.

#### `memfile` Storage Engine State Encryption Local Key File

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_LOCAL_KEY_FILE`

_Config field:_ `state.encryption.local_key_file`

_**required, if the localkey key provider is used**_

The path to a file holding a raw 32-byte key, 32 random bytes from a key management tool or
`head -c 32 /dev/urandom`.

The file must not be readable by users other than its owner, mode `0600`, and the deploy engine
will refuse to start otherwise.

#### `memfile` Storage Engine State Encryption AWS KMS Key ID

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AWSKMS_KEY_ID`

_Config field:_ `state.encryption.awskms.key_id`

_**required, if the awskms key provider is used**_

Identifies the AWS KMS key, as a key ID, ARN, alias name or alias ARN. The identity the deploy
engine runs as needs `kms:GenerateDataKey` and `kms:Decrypt` on the key.

An alias is the better choice where a deployment expects to move to a new key: the configured
value is recorded in the state the deploy engine writes, and KMS resolves an alias at the time
of each call.

#### `memfile` Storage Engine State Encryption AWS KMS Region

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AWSKMS_REGION`

_Config field:_ `state.encryption.awskms.region`

_**optional**_

The AWS region the KMS key lives in. When empty, the region resolved by the default credential
chain is used.

#### `memfile` Storage Engine State Encryption AWS KMS Endpoint

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AWSKMS_ENDPOINT`

_Config field:_ `state.encryption.awskms.endpoint`

_**optional**_

Overrides the KMS endpoint, for a VPC endpoint or a local stand-in such as LocalStack. When
empty, the default endpoint for the region is used.

#### `memfile` Storage Engine State Encryption AWS KMS Access Key ID / Secret Access Key

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AWSKMS_ACCESS_KEY_ID`
`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AWSKMS_SECRET_ACCESS_KEY`

_Config fields:_ `state.encryption.awskms.access_key_id`, `state.encryption.awskms.secret_access_key`

_**optional**_

Static credentials used in place of the default AWS SDK credential chain. Both must be set
together. Leaving them empty is usually right, the default credential chain picks up an
instance role, a container role or a profile without the deploy engine holding long-lived keys.

#### `memfile` Storage Engine State Encryption Google Cloud KMS Key Name

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_GCPKMS_KEY_NAME`

_Config field:_ `state.encryption.gcpkms.key_name`

_**required, if the gcpkms key provider is used**_

The key's resource name, in the form
`projects/{project}/locations/{location}/keyRings/{ring}/cryptoKeys/{key}`. The identity the
deploy engine runs as needs `cloudkms.cryptoKeyVersions.useToEncrypt` and `useToDecrypt` on the
key.

Naming the key rather than one of its versions lets Cloud KMS rotate versions underneath
without state sealed under an earlier version becoming unreadable.

#### `memfile` Storage Engine State Encryption Google Cloud KMS Credentials File

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_GCPKMS_CREDENTIALS_FILE`

_Config field:_ `state.encryption.gcpkms.credentials_file`

_**optional**_

The path to a service account key file, and only that: a file holding another kind of
credential configuration is refused rather than loaded, since loading an unexpected credential
type from a file is itself a risk.

When empty, Application Default Credentials are used, which is usually right and is the route
for workload identity federation as well: an attached service account or a federated identity
avoids the deploy engine holding a long-lived key at all.

#### `memfile` Storage Engine State Encryption Azure Key Vault URL / Key Name

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AZUREKEYVAULT_VAULT_URL`
`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AZUREKEYVAULT_KEY_NAME`

_Config fields:_ `state.encryption.azurekeyvault.vault_url`, `state.encryption.azurekeyvault.key_name`

_**required, if the azurekeyvault key provider is used**_

The vault's URL, `https://{vault}.vault.azure.net/` for a standard vault, and the name of the
key within it. The identity the deploy engine runs as needs the `wrapKey` and `unwrapKey`
permissions on the key. Credentials come from the default Azure credential chain, a managed
identity or the standard Azure environment variables.

#### `memfile` Storage Engine State Encryption Azure Key Vault Key Version

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AZUREKEYVAULT_KEY_VERSION`

_Config field:_ `state.encryption.azurekeyvault.key_version`

_**optional**_

Pins the key version used for wrapping, and is worth setting.

A wrapped key does not name the version that produced it, so the version has to come from
somewhere. When pinned, it is recorded in the state the deploy engine writes and used again to
unwrap, so state sealed before a rotation stays readable. When empty, the vault's current
version is used for both, and rotating the key leaves everything sealed under the previous
version unreadable.

#### `memfile` Storage Engine State Encryption Azure Key Vault Algorithm

`BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_AZUREKEYVAULT_ALGORITHM`

_Config field:_ `state.encryption.azurekeyvault.algorithm`

_**optional**_

Overrides the algorithm used to wrap data keys. Suits the RSA keys a standard vault holds by
default. A Managed HSM deployment wrapping with a symmetric key needs `A256KW`.

**default value:** `RSA-OAEP-256`

#### `postgres` Storage Engine User

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_USER`

_Config field:_ `state.postgres_user`

_**required, if postgres storage engine is used**_

The user to use to connect to the PostgreSQL database
when using the postgres storage engine.

#### `postgres` Storage Engine Password

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_PASSWORD`

_Config field:_ `state.postgres_password`

_**required, if postgres storage engine is used**_

The password to for the user used to connect to the PostgreSQL database
when using the postgres storage engine.

#### `postgres` Storage Engine Host

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_HOST`

_Config field:_ `state.postgres_host`

_**optional**_

The host to use to connect to the PostgreSQL database
when using the postgres storage engine.

**default value:** `localhost`

#### `postgres` Storage Engine Port

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_PORT`

_Config field:_ `state.postgres_port`

_**optional**_

The port to use to connect to the PostgreSQL database
when using the postgres storage engine.

**default value:** `5432`

#### `postgres` Storage Engine Database

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_DATABASE`

_Config field:_ `state.postgres_database`

_**required, if postgres storage engine is used**_

The name of the PostgreSQL database to connect to
when using the postgres storage engine.

#### `postgres` Storage Engine SSL Mode

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_SSL_MODE`

_Config field:_ `state.postgres_ssl_mode`

_**optional**_

The SSL mode to use to connect to the PostgreSQL database
when using the postgres storage engine.

This can be set to `disable`, `require`, `verify-ca` or `verify-full`.

State and the credentials used to reach it cross this connection, so the deploy engine warns at
startup when the mode is `disable` or `prefer` and the host is not the local machine. `prefer`
is included because it falls back to an unencrypted connection, silently, when the server will
not negotiate TLS. Note that `verify-full` needs a root certificate, for which there is no
configuration key yet, so it is usable only where the certificate is already trusted at the OS
level.

**default value:** `disable`

#### `postgres` Storage Engine Pool Max Connections

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_POOL_MAX_CONNS`

_Config field:_ `state.postgres_pool_max_conns`

_**optional**_

The maximum number of connections to the PostgreSQL database
in the client pool when using the postgres storage engine.

**default value:** `100`

#### `postgres` Storage Engine Pool Max Connection Lifetime

`BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_POOL_MAX_CONN_LIFETIME`

_Config field:_ `state.postgres_pool_max_conn_lifetime`

_**optional**_

The maximum lifetime of a connection in the PostgreSQL database
when using the postgres storage engine.
This should be in a format that can be parsed as a Go `time.Duration` value.
See: [time.Duration](https://pkg.go.dev/time#ParseDuration) for more information.

**default value:** `1h30m`

#### `objectstore` Storage Engine Provider

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_PROVIDER`

_Config field:_ `state.objectstore.provider`

_**required, if objectstore storage engine is used**_

The object store backend to use. Valid values are `s3`, `gcs`, and `azureblob`.
Exactly one provider is active per deploy-engine process; the corresponding
provider sub-config is used and the others are ignored.

#### `objectstore` Storage Engine Key Prefix

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_PREFIX`

_Config field:_ `state.objectstore.prefix`

_**optional**_

Prepended to every storage key, scoping state under a subpath of the
bucket/container. Useful when sharing a bucket across deploy-engine
deployments or with other workloads.

**default value:** `bluelink-state/`

#### `objectstore` Cache Max Age

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_CACHE_MAX_AGE_MS`

_Config field:_ `state.objectstore.cache_max_age_ms`

_**optional**_

How long, in milliseconds, the engine uses a cached blueprint instance before checking it
against the bucket again with a HEAD request. This is how a long-running engine sees changes
that other engines sharing the bucket made since it cached the instance, such as a CI run
deploying it or a retry of a failed deployment that ran elsewhere. A changed instance is
reloaded and a removed one dropped. `0` checks on every use.

Writes do not depend on this setting. Every instance write is conditional on the version the
engine last read, so a write based on an outdated copy is rejected and applied again to the
current one.

**default value:** `30000` (30 seconds)

#### `objectstore` Disable Cache Revalidation

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_DISABLE_CACHE_REVALIDATION`

_Config field:_ `state.objectstore.disable_cache_revalidation`

_**optional**_

Turns off checking cached instances against the bucket, so the engine makes no HEAD requests
to revalidate them. Suited to short-lived engines, such as one started for a CI run, which
load state fresh and finish before another engine's change would matter, and to an engine
that has the bucket to itself.

**default value:** `false`

#### `objectstore` S3 Bucket

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_BUCKET`

_Config field:_ `state.objectstore.s3.bucket`

_**required, if s3 objectstore provider is used**_

The S3 bucket name that holds state objects.

#### `objectstore` S3 Region

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_REGION`

_Config field:_ `state.objectstore.s3.region`

_**optional**_

The AWS region the bucket lives in. When empty, the default AWS SDK
region resolution (env var, shared config) is used.

#### `objectstore` S3 Endpoint

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_ENDPOINT`

_Config field:_ `state.objectstore.s3.endpoint`

_**optional**_

Overrides the default S3 endpoint. Required for S3-compatible gateways
(LocalStack, MinIO etc.).

#### `objectstore` S3 Use Path Style

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_USE_PATH_STYLE`

_Config field:_ `state.objectstore.s3.use_path_style`

_**optional**_

Switches addressing from virtual-hosted (default) to path-style.
Required for LocalStack and many S3-compatible gateways.

**default value:** `false`

#### `objectstore` S3 Access Key ID / Secret Access Key

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_ACCESS_KEY_ID`
`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_SECRET_ACCESS_KEY`

_Config fields:_ `state.objectstore.s3.access_key_id`, `state.objectstore.s3.secret_access_key`

_**optional**_

Static credentials used in place of the default AWS SDK credential
chain. Both must be set together. When either is empty, the default
credential chain (env vars, shared config, IAM role) is used.

#### `objectstore` GCS Bucket

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_GCS_BUCKET`

_Config field:_ `state.objectstore.gcs.bucket`

_**required, if gcs objectstore provider is used**_

The Google Cloud Storage bucket name that holds state objects.

#### `objectstore` GCS Endpoint

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_GCS_ENDPOINT`

_Config field:_ `state.objectstore.gcs.endpoint`

_**optional**_

Overrides the default GCS endpoint. Required for fake-gcs-server and
other GCS-compatible gateways.

#### `objectstore` GCS Without Authentication

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_GCS_WITHOUT_AUTHENTICATION`

_Config field:_ `state.objectstore.gcs.without_authentication`

_**optional**_

Disables Application Default Credentials. Set this when targeting
fake-gcs-server or another emulator that does not expect credentials.

**default value:** `false`

#### `objectstore` Azure Blob Service URL

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_AZUREBLOB_SERVICE_URL`

_Config field:_ `state.objectstore.azureblob.service_url`

_**required, if azureblob objectstore provider is used**_

The account-qualified blob service URL, e.g.
`https://<account>.blob.core.windows.net` against real Azure or
`http://localhost:10000/devstoreaccount1` against Azurite.

#### `objectstore` Azure Blob Account Name / Account Key

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_AZUREBLOB_ACCOUNT_NAME`
`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_AZUREBLOB_ACCOUNT_KEY`

_Config fields:_ `state.objectstore.azureblob.account_name`, `state.objectstore.azureblob.account_key`

_**required, if azureblob objectstore provider is used**_

The shared-key credential pair used to sign requests against
ServiceURL. AccountKey is the base64-encoded shared key.

#### `objectstore` Azure Blob Container

`BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_AZUREBLOB_CONTAINER`

_Config field:_ `state.objectstore.azureblob.container`

_**required, if azureblob objectstore provider is used**_

The blob container name that holds state objects.

### Audit

Configuration for the trail of actions taken through the deploy engine.

Every request that can change something is recorded with who made it, what it
was made against, and whether it worked. Requests that only read are not
recorded, since a record of every read would bury the actions.

#### Audit Enabled

`BLUELINK_DEPLOY_ENGINE_AUDIT_ENABLED`

_Config field:_ `audit.enabled`

_**optional**_

Whether a trail of actions is recorded at all.

A deploy engine recording nothing looks the same from the outside as one
recording everything, so turning this off is something to state deliberately.
A warning is logged at startup when it is off.

**default value:** `true`

#### Audit Records To Process Output

`BLUELINK_DEPLOY_ENGINE_AUDIT_STDOUT`

_Config field:_ `audit.stdout`

_**optional**_

Whether audit records are also written to the process output, as one JSON
object per line, alongside being written to state.

Records always go to state, which is what makes the trail queryable through
the `/audit/records` endpoints and extractable from the state backend
directly.

This is off by default because the process output is where the deploy engine's
own logs go, under a schema these records do not share. A log query or
dashboard that selects the engine's lines by level would pass over every audit
record in the stream. The records are written as raw JSON lines rather than
through the logger, deliberately, so that no log level can discard them.

Turn it on where the trail has to live outside the deploy engine's own
storage backend, so that the engine cannot rewrite its own history, or where state is
disposable and the process output is the durable copy. A record that fails to
persist is reported with the request that produced it, so losing one is not
silent either way.

**default value:** `false`

### Tracing

Configuration for the distributed traces the deploy engine exports.

Traces are exported over OTLP to an [OpenTelemetry
collector](https://opentelemetry.io/docs/collector/) rather than to a
particular backend, so which backend they end up in is the collector's
configuration and not the deploy engine's.

A trace covers handling a request, the cloud service calls the deploy engine
makes to object storage and key management within it, and each pass of the
scheduled state cleanup. Where a trace is being exported, the identity of the
span a request's log entries were written under is added to those entries, so
an entry leads to the trace it belongs to and back.

Resource operations carried out by provider and transformer plugins are not yet
traced. Plugins run out of process and the deploy engine does not currently
propagate a trace to them, so the time a deployment spends in them appears as a
gap in its trace. Child blueprints fetched from object storage are not traced
either, since each resolver builds its own client with no way to pass
instrumentation through.

An audit record carries the trace and span identifiers of the trace its action
was carried out under, where tracing was on and the trace was sampled, so an
audited action leads to the trace of the work it did. See
[Audit](#audit).

#### Tracing Enabled

`BLUELINK_DEPLOY_ENGINE_TRACING_ENABLED`

_Config field:_ `tracing.enabled`

_**optional**_

Whether traces are exported at all.

Off by default, because tracing needs a collector to export to and a deploy
engine that exported by default would spend every span on a connection that is
not there. When off, no request instrumentation is registered and the deploy
engine traces through an implementation that does nothing.

**default value:** `false`

#### Tracing OTLP Endpoint

`BLUELINK_DEPLOY_ENGINE_TRACING_OTLP_ENDPOINT` or
`OTEL_EXPORTER_OTLP_ENDPOINT`

_Config field:_ `tracing.otlp_endpoint`

_**optional**_

The OTLP gRPC endpoint of the collector to export traces to, including the
scheme, as in `http://collector:4317`.

OpenTelemetry's own variable name is accepted as a fallback, so a host already
set up to export telemetry needs nothing specific to the deploy engine. The
deploy engine's own variable is read first, so it can override what the host
already says.

Only OTLP over gRPC is supported. A collector endpoint serving OTLP over HTTP,
conventionally port 4318, will not work, and
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` and `OTEL_EXPORTER_OTLP_PROTOCOL` are not
read.

The scheme and host are checked at startup, because the exporter keeps its own
default endpoint when given a URL it cannot parse. Without that check a value
such as `collector:4317`, with no scheme, would have the deploy engine report
success at startup and export nothing. A host that does not resolve cannot be
caught this way, since the connection is made lazily and nothing dials it at
startup.

**The scheme decides whether the connection is encrypted.** `http://` exports
over plaintext gRPC and `https://` over TLS, which is OpenTelemetry's own
convention rather than anything specific to the deploy engine. The default is
plaintext to a collector on the same host, which is the usual arrangement, as a
sidecar container or an agent on the loopback interface. A collector reached
across a network needs `https://`, otherwise the span names, attributes and
trace identifiers described above cross that network in the clear. Nothing
stops the deploy engine exporting in plaintext to a remote address, so this is
the operator's call to make.

**Credentials for a remote collector come from the environment,** not from
`tracing.*`. The exporter reads OpenTelemetry's own variables and applies them
before the deploy engine's options, so the endpoint comes from the
configuration above while the credential comes from
`OTEL_EXPORTER_OTLP_HEADERS`, which is where an API key or bearer token goes.
`OTEL_EXPORTER_OTLP_CERTIFICATE` supplies a CA for a private certificate
authority, and `OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE` with
`OTEL_EXPORTER_OTLP_CLIENT_KEY` supplies a client certificate for a collector
requiring mTLS. There is deliberately no `tracing.headers` equivalent, a
collector credential belongs in the environment rather than in a configuration
file as is a concern of the OpenTelemetry library and not the engine.

**default value:** `http://localhost:4317`

#### Tracing Service Name

`BLUELINK_DEPLOY_ENGINE_TRACING_SERVICE_NAME` or `OTEL_SERVICE_NAME`

_Config field:_ `tracing.service_name`

_**optional**_

The name the deploy engine is given in the traces it produces, which is how a
backend tells its spans from those of everything else reporting to the same
destination.

Worth setting where several deploy engines report to one destination.

**default value:** `bluelink-deploy-engine`

#### Tracing Sampling Rate

`BLUELINK_DEPLOY_ENGINE_TRACING_SAMPLING_RATE`

_Config field:_ `tracing.sampling_rate`

_**optional**_

The proportion of traces to export, between 0 and 1.

A trace that arrived from a caller is exported if and only if the caller that
started it sampled it, so the decision is made once per trace and every
component it passes through agrees. This rate applies only to the traces the
deploy engine starts itself, which means a rate of 0 still exports a trace a
caller sampled, and a rate of 1 does not override a caller that declined one.

A tenth of traces by default. A managed deployment serving many tenants handles
enough traffic for exporting all of it to be a real cost, and raising the rate
should be the deliberate act rather than paying for every trace by default.

Set it to `1` where every operation matters more than the volume, which is the
usual case for a single-tenant or self-hosted deploy engine. Note that the rate
is applied per trace rather than per span, so a lower rate drops whole
operations rather than thinning the spans within one; an exported trace is
always complete.

Because a caller's decision is honoured, a client that needs a trace for one
particular operation can force it by sampling that trace upstream, whatever
this is set to.

**default value:** `0.1`

#### Tracing Platform

`BLUELINK_DEPLOY_ENGINE_TRACING_PLATFORM`

_Config field:_ `tracing.platform`

_**optional**_

The platform the deploy engine runs on, where it has a tracing identifier
format of its own that the engine has to produce for a trace to be readable
there.

Set this to `aws` where traces reach AWS X-Ray. X-Ray only accepts trace IDs in
a shape it recognises, which carries the time the trace started, so a trace the
deploy engine starts has to be given one rather than the random identifier that
is otherwise generated. In this mode the engine also reads and writes the
`X-Amzn-Trace-Id` header alongside W3C trace context, so a trace does not break
at an AWS load balancer in front of it, and adds the X-Ray form of the trace ID
to each log entry as `xrayTraceId`, which is the only form the X-Ray console
will find a trace by.

W3C trace context is read and written on every platform, including this one, and
takes precedence where a request carries both headers, since such a request came
from a caller that started a W3C trace and then passed through AWS.

Leave it unset for a collector exporting to a backend that reads W3C trace
context, which is every other backend. The only supported value is `aws`.

**default value:** `""`

### Resolvers

Configuration for child blueprint resolvers used by the deploy engine.

#### Resolver S3 Endpoint

`BLUELINK_DEPLOY_ENGINE_RESOLVERS_S3_ENDPOINT`
_Config field:_ `resolvers.s3_endpoint`

_**optional**_

A custom endpoint to use to connect to an S3-compatible object storage service
when resolving the source files for child blueprints.
When empty, the default AWS S3 endpoint will be used.

#### Resolver S3 Use Path Style

`BLUELINK_DEPLOY_ENGINE_RESOLVERS_S3_USE_PATH_STYLE`

_Config field:_ `resolvers.s3_use_path_style`

_**optional**_

Whether to use path-style addressing for S3 requests.
When true, requests will be made to `{endpoint}/{bucket}/{key}` instead of
`{bucket}.{endpoint}/{key}`. This is required for S3-compatible services
like MinIO that don't support virtual-hosted-style addressing.
Defaults to `false`.

#### Resolver Google Cloud Storage Endpoint

`BLUELINK_DEPLOY_ENGINE_RESOLVERS_GCS_ENDPOINT`

_Config field:_ `resolvers.gcs_endpoint`

_**optional**_

A custom endpoint to use to connect to a Google Cloud Storage-compatible object storage service
when resolving the source files for child blueprints.
When empty, the default Google Cloud Storage endpoint will be used.

#### Resolver HTTPS Client Timeout

`BLUELINK_DEPLOY_ENGINE_RESOLVERS_HTTPS_CLIENT_TIMEOUT`

_Config field:_ `resolvers.https_client_timeout`

_**optional**_

The timeout in seconds to use for the HTTPS client used to resolve blueprints
that use the the `https` file source scheme or child blueprint includes
that use the `https` source type.

**default value:** `30`

### Maintenance

Configuration for the maintenance of short-lived resources in the deploy engine.
This is used for things like the retention periods for blueprint validations and change sets.

#### Blueprint Validation Retention Period

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_BLUEPRINT_VALIDATION_RETENTION_PERIOD`

_Config field:_ `maintenance.blueprint_validation_retention_period`

_**optional**_

The retention period in seconds for blueprint validations.
This is used to determine how long to keep the results of blueprint validation
before deleting them.
When the clean up process runs for blueprint validations,
it will delete all validation results that are older than this period.

**default value:** `604800` (7 days)

#### Change Set Retention Period

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_CHANGESET_RETENTION_PERIOD`

_Config field:_ `maintenance.changeset_retention_period`

_**optional**_

The retention period in seconds for change sets.
This is used to determine how long to keep the results of change sets
before deleting them.
When the clean up process runs for change sets,
it will delete all change sets that are older than this period.

**default value:** `604800` (7 days)

#### Events Retention Period

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_EVENTS_RETENTION_PERIOD`

_Config field:_ `maintenance.events_retention_period`

_**optional**_

The retention period in seconds for events.
This is used to determine how long to keep the results of events
before deleting them.
When the clean up process runs for events,
it will delete all events that are older than this period.

**default value:** `604800` (7 days)

#### Reconciliation Results Retention Period

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_RECONCILIATION_RESULTS_RETENTION_PERIOD`

_Config field:_ `maintenance.reconciliation_results_retention_period`

_**optional**_

The retention period in seconds for reconciliation results.
This is used to determine how long to keep the results of drift reconciliation checks
before deleting them.
When the clean up process runs for reconciliation results,
it will delete all reconciliation results that are older than this period.

**default value:** `604800` (7 days)

#### Audit Record Retention Period

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_AUDIT_RECORD_RETENTION_PERIOD`

_Config field:_ `maintenance.audit_record_retention_period`

_**optional**_

The retention period in seconds for audit records.
This is used to determine how long to keep the trail of actions taken through
the deploy engine before deleting them.
When the clean up process runs for audit records,
it will delete all audit records that are older than this period.

The default is considerably longer than the other categories. The rest are
working data that is only of interest while an operation is recent; an audit
trail is of interest precisely when something is being looked into after the
fact. Deployments with a compliance requirement to retain a trail for a fixed
period should set this to match it.

**default value:** `7776000` (90 days)

#### Scheduled Cleanup Enabled

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_SCHEDULED_CLEANUP_ENABLED`

_Config field:_ `maintenance.scheduled_cleanup.enabled`

_**optional**_

Whether the deploy engine applies the retention periods above itself, on an
interval, rather than only when the cleanup endpoints are called.

The retention periods are otherwise used only to work out a threshold inside a
cleanup request, so a deployment that never sends one keeps every record for
ever. Enabling this has the engine delete state that has passed its retention
period without being asked each time.

Left off by default, because deleting state is something to opt into rather
than inherit on deployment of the engine. A deployment that treats its state as disposable,
such as a CI run working against a restored backup or object storage, is not a long running process that can effectively make use of cleanup on an interval.

A scheduled run and a request to a cleanup endpoint will not run over the same
category at the same time within one deploy engine, whichever asks second is
refused, and a cleanup endpoint answers `409 Conflict`. Across several deploy
engines sharing a state backend this is best effort, since the only evidence
another engine is mid-run is the cleanup operation it recorded.

**default value:** `false`

#### Scheduled Cleanup Interval

`BLUELINK_DEPLOY_ENGINE_MAINTENANCE_SCHEDULED_CLEANUP_INTERVAL`

_Config field:_ `maintenance.scheduled_cleanup.interval`

_**optional**_

How often, in seconds, the deploy engine cleans up each category when
`maintenance.scheduled_cleanup.enabled` is `true`.

**default value:** `86400` (24 hours)

## Maintenance Commands

Running the deploy engine with no arguments serves the API. Passing a command
runs that command instead and exits.

### `reencrypt-state`

```bash
bluelink-deploy-engine reencrypt-state [--dry-run] [--decrypt] [--backup-dir=DIR] [--no-backup] [--force]
```

Re-seals every object in a `memfile` state directory under the key in the deploy engine's
configuration. It reads the same configuration the server does, so what it does follows from
that configuration rather than from flags:

| Configuration                                                              | What the command does                           |
| -------------------------------------------------------------------------- | ----------------------------------------------- |
| `state.encryption.enabled` set, state currently plaintext                   | Encrypts the existing state for the first time   |
| `state.encryption` naming a new key, the old one under `previous_keys`      | Moves the state onto the new key                 |
| `state.encryption` naming the key the state is sealed with, with `--decrypt` | Writes the state back as plaintext               |

**Run it with the deploy engine stopped.** The deploy engine holds state in memory and persists
it on write, so it would write its own copy back over the pass. The command refuses to start if
something is listening on the configured address, which `--force` overrides.

Each object is dealt with independently and objects already sealed under the target key are
recognised and left alone, so an interrupted pass can simply be run again. What the command
cannot do is make the pass atomic: objects are rewritten in place, so a failure part-way through
leaves a directory holding a mix of keys, readable only by a configuration that still lists
both. For that reason the state directory is copied before anything is rewritten, to
`DIR` when `--backup-dir` is given and alongside the state directory otherwise.
`--no-backup` skips the copy, and `--dry-run` reports what would change while writing nothing
at all.

The encryption salt is never encrypted or replaced, since the current key may have been derived
from it.

## API Documentation

The API documentation for the v1 of the Deploy Engine HTTP API is available at the following URL:

https://bluelink.dev/deploy-engine/docs/http-api-reference/v1/deploy-engine-api

## Additional documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Secure deployment](docs/SECURE_DEPLOYMENT.md)
- [Contributing](docs/CONTRIBUTING.md)
