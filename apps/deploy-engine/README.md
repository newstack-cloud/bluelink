# Bluelink Deploy Engine

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine)

The engine that validates and deploys blueprints for infrastructure as code deployments.

The deploy engine bundles the plugin framework's gRPC-based plugin system that allows for the creation of custom plugins for providers and transformers that can be pulled in at runtime.
The deploy engine also bundles a limited set of state persistence implementations for blueprint instances, the persistence implementation can be chosen with configuration.

- [Installing](#installing)
- [Configuration](#configuration)
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

### State

Configuration for the state management/persistence layer used by the deploy engine.

#### Storage Engine

`BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE`

_Config field:_ `state.storage_engine`

_**optional**_

The storage engine to use for the state management/persistence layer.
This can be set to `memfile` for in-memory storage with file system persistence
or `postgres` for a PostgreSQL database.

Postgres should be used for deploy engine deployments that need to scale horizontally,
the in-memory storage with file system persistence engine should be used for local deployments,
CI environments and production use cases where the deploy engine is not expected to scale horizontally.

If opting for the in-memory storage with file system persistence engine,
it would be a good idea to backup the state files to a remote location
to avoid losing all state in the event of a failure or destruction of the host machine.

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

## API Documentation

The API documentation for the v1 of the Deploy Engine HTTP API is available at the following URL:

https://bluelink.dev/deploy-engine/docs/http-api-reference/v1/deploy-engine-api

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
