package core

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
)

// Config provides configuration for the deploy engine application.
// This parses configuratoin from the current environment.
type Config struct {
	// The version of the deploy engine API to use.
	// Defaults to "v1".
	APIVersion string `env:"BLUELINK_DEPLOY_ENGINE_API_VERSION" envDefault:"v1"`
	// The current version of the deploy engine software.
	// This will be set based on a value of a constant determined at build time.
	Version string
	// The current version of the plugin framework that is being used
	// by the deploy engine.
	// This will be set based on a value of a constant determined at build time.
	PluginFrameworkVersion string
	// The current version of the blueprint framework that is being used
	// by the deploy engine.
	// This will be set based on a value of a constant determined at build time.
	BlueprintFrameworkVersion string
	// The current version of the provider plugin protocol that is being used
	// by the deploy engine when acting as a plugin host.
	// This will be set at runtime based on the version of the plugin protocol
	// that the selected API version of the deploy engine uses.
	ProviderPluginProtocolVersion string
	// The current version of the transformer plugin protocol that is being used
	// by the deploy engine when acting as a plugin host.
	// This will be set at runtime based on the version of the plugin protocol
	// that the selected API version of the deploy engine uses.
	TransformerPluginProtocolVersion string
	// The TCP port to listen on for incoming connections.
	// This will be ignored if UseUnixSocket is set to true.
	// Defaults to "8325".
	Port int `env:"BLUELINK_DEPLOY_ENGINE_PORT" envDefault:"8325"`
	// Determines whether or not to use unix sockets for handling
	// incoming connections instead of TCP.
	// If set to true, the Port will be ignored and the UnixSocketPath
	// will be used instead.
	// Defaults to "false".
	UseUnixSocket bool `env:"BLUELINK_DEPLOY_ENGINE_USE_UNIX_SOCKET" envDefault:"false"`
	// The path to the unix socket to listen on for incoming connections.
	// This will be ignored if UseUnixSocket is set to false.
	// Defaults to "/tmp/bluelink.sock".
	UnixSocketPath string `env:"BLUELINK_DEPLOY_ENGINE_UNIX_SOCKET_PATH" envDefault:"/tmp/bluelink.sock"`
	// LoopbackOnly determines whether or not to restrict the server
	// to only accept connections from the loopback interface.
	// Defaults to "true" for a more secure default.
	// This should be intentionally set to false for deployments
	// of the deploy engine that are intended to be accessible
	// over a private network or the public internet.
	LoopbackOnly bool `env:"BLUELINK_DEPLOY_ENGINE_LOOPBACK_ONLY" envDefault:"true"`
	// Environment determines whether the deploy engine is running
	// in a production or development environment.
	// This is used to determine things like the formatting of logs,
	// in development mode, logs are formatted in a more human readable format,
	// while in production mode, logs are formatted purely in JSON for easier
	// parsing and processing by log management systems.
	// Defaults to "production".
	Environment string `env:"BLUELINK_DEPLOY_ENGINE_ENVIRONMENT" envDefault:"production"`
	// LogLevel determines the level of logging to use for the deploy engine.
	// Defaults to "info".
	// Can be set to any of the logging levels supported by zap:
	// debug, info, warn, error, dpanic, panic, fatal.
	// See: https://pkg.go.dev/go.uber.org/zap#Level
	LogLevel string `env:"BLUELINK_DEPLOY_ENGINE_LOG_LEVEL" envDefault:"info"`
	// Auth provides configuration for the way authentication
	// should be handled by the deploy engine.
	Auth AuthConfig `envPrefix:"BLUELINK_DEPLOY_ENGINE_AUTH_"`
	// PluginsV1 provides configuration for the v1 plugin system
	// implemented by the deploy engine.
	PluginsV1 PluginsV1Config
	// Blueprints provides configuration for the blueprint loader
	// used by the deploy engine.
	Blueprints BlueprintConfig `envPrefix:"BLUELINK_DEPLOY_ENGINE_BLUEPRINTS_"`
	// State provides configuration for the state management/persistence
	// layer used by the deploy engine.
	State StateConfig `envPrefix:"BLUELINK_DEPLOY_ENGINE_STATE_"`
	// Resolvers provides configuration for the child blueprint resolvers
	// used by the deploy engine.
	Resolvers ResolversConfig `envPrefix:"BLUELINK_DEPLOY_ENGINE_RESOLVERS_"`
	// Maintenance provides configuration for the maintenance
	// of short-lived resources in the deploy engine.
	// This is used for things like the retention periods for
	// blueprint validations and change sets.
	Maintenance MaintenanceConfig `envPrefix:"BLUELINK_DEPLOY_ENGINE_MAINTENANCE_"`
}

func (p *Config) GetPluginPath() string {
	return p.PluginsV1.PluginPath
}

func (p *Config) GetLaunchWaitTimeoutMS() int {
	return p.PluginsV1.LaunchWaitTimeoutMS
}

func (p *Config) GetTotalLaunchWaitTimeoutMS() int {
	return p.PluginsV1.TotalLaunchWaitTimeoutMS
}

func (p *Config) GetResourceStabilisationPollingTimeoutMS() int {
	return p.PluginsV1.ResourceStabilisationPollingTimeoutMS
}

func (p *Config) GetResourceStabilisationPollingIntervalMS() int {
	return p.Blueprints.ResourceStabilisationPollingIntervalMS
}

func (p *Config) GetPluginToPluginCallTimeoutMS() int {
	return p.PluginsV1.PluginToPluginCallTimeoutMS
}

// PluginsV1Config provides configuration for the v1 plugin system
// implemented by the deploy engine.
type PluginsV1Config struct {
	// PluginPath is the path to one or more plugin root directories
	// separated by colons.
	// Defaults to $HOME/.bluelink/deploy-engine/plugins/bin,
	// where $HOME will be expanded to the current user's home directory.
	PluginPath string `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH" envDefault:"$HOME/.bluelink/deploy-engine/plugins/bin"`
	// LogFileRootDir is the path to a single root directory used to store
	// logs for all plugins. stdout and stderr for each plugin
	// will be redirected to log files under this directory.
	// Defaults to $HOME/.bluelink/deploy-engine/plugins/logs,
	// where $HOME will be expanded to the current user's home directory.
	LogFileRootDir string `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_LOG_FILE_ROOT_DIR" envDefault:"$HOME/.bluelink/deploy-engine/plugins/logs"`
	// LaunchWaitTimeoutMS is the timeout in milliseconds
	// to wait for a plugin to register with the host.
	// This is used when the plugin host is started and
	// a plugin is expected to register with the host.
	// Defaults to 15,000ms (15 seconds)
	LaunchWaitTimeoutMS int `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_LAUNCH_WAIT_TIMEOUT_MS" envDefault:"15000"`
	// TotalLaunchWaitTimeoutMS is the timeout in milliseconds
	// to wait for all plugins to register with the host.
	// This is used when the plugin host is started and
	// all plugins are expected to register with the host.
	// Defaults to 60,000ms (1 minute)
	TotalLaunchWaitTimeoutMS int `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_TOTAL_LAUNCH_WAIT_TIMEOUT_MS" envDefault:"60000"`
	// ResourceStabilisationPollingTimeoutMS is the timeout in milliseconds
	// to wait for a resource to stabilise when calls are made
	// into the resource registry through the plugin service.
	// This same timeout is used for configuring the blueprint loader and
	// plugin host.
	// Defaults to 3,600,000ms (1 hour)
	ResourceStabilisationPollingTimeoutMS int `env:"BLUELINK_DEPLOY_ENGINE_RESOURCE_STABILISATION_POLLING_TIMEOUT_MS" envDefault:"36000000"`
	// PluginToPluginCallTimeoutMS is the timeout in milliseconds
	// to wait for a plugin to respond to a call initiated by another
	// or the same plugin through the plugin service.
	// The exception, where this timeout is not used, is when waiting for
	// a resource to stabilise when calls are made into the resource registry
	// through the plugin service.
	// Defaults to 120,000ms (2 minutes)
	PluginToPluginCallTimeoutMS int `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_TO_PLUGIN_CALL_TIMEOUT_MS" envDefault:"120000"`
}

// BlueprintConfig provides configuration for the blueprint loader
// used by the deploy engine.
type BlueprintConfig struct {
	// ValidateAfterTransform determines whether or not the blueprint
	// loader should validate blueprints after applying transformations.
	// Defaults to "false".
	// This should only really be set to true when there is a need to debug
	// issues that may be due to transformer plugins producing invalid output.
	ValidateAfterTransform bool `env:"VALIDATE_AFTER_TRANSFORM" envDefault:"false"`
	// EnableDriftCheck determines whether or not the blueprint
	// loader should check for drift in the state of resources
	// when staging changes for a blueprint deployment.
	// Defaults to "true".
	EnableDriftCheck bool `env:"ENABLE_DRIFT_CHECK" envDefault:"true"`
	// ResourceStabilisationPollingIntervalMS is the interval in milliseconds
	// to wait between polling for a resource to stabilise
	// when calls are made to a provider to check if a resource has stabilised.
	// This is used in the plugin host for plugin to plugin calls
	// (i.e. links deploying intermediary resources)
	// and in the blueprint container that manages deployment of resources declared
	// in a blueprint.
	// Defaults to 5,000ms (5 seconds)
	ResourceStabilisationPollingIntervalMS int `env:"RESOURCE_STABILISATION_POLLING_INTERVAL_MS" envDefault:"5000"`
	// DefaultRetryPolicy is the default retry policy to use
	// when a provider returns a retryable error for actions that support retries.
	// This should be a serialised JSON string that matches the structure of the
	// `provider.RetryPolicy` struct.
	// The built-in default will be used if this is not set or the JSON is not
	// in the correct format.
	DefaultRetryPolicy string `env:"DEFAULT_RETRY_POLICY"`
	// DeploymentTimeout is the time in seconds to wait for a deployment
	// to complete before timing out.
	// This timeout is for the background process that runs the deployment
	// when the deployment endpoints are called.
	// Defaults to 10,800 seconds (3 hours).
	DeploymentTimeout int `env:"DEPLOYMENT_TIMEOUT" envDefault:"10800"`
}

// StateConfig provides configuration for the state management/persistence
// layer used by the deploy engine.
type StateConfig struct {
	// The storage engine to use for the state management/persistence layer.
	// This can be set to "memfile" for in-memory storage with file system persistence
	// or "postgres" for a PostgreSQL database.
	// Postgres should be used for deploy engine deployments that need to scale
	// horizontally, the in-memory storage with file system persistence
	// engine should be used for local deployments, CI environments and production
	// use cases where the deploy engine is not expected to scale horizontally.
	// If opting for the in-memory storage with file system persistence engine,
	// it would be a good idea to backup the state files to a remote location
	// to avoid losing all state in the event of a failure or destruction of the host machine.
	// Defaults to "memfile".
	StorageEngine string `env:"STORAGE_ENGINE" envDefault:"memfile"`
	// The threshold in seconds for retrieving recently queued events
	// for a stream when a starting event ID is not provided.
	// Any events that are older than currentTime - threshold
	// will not be considered as recently queued events.
	// This applies to all storage engines.
	// Defaults to 300 seconds (5 minutes).
	RecentlyQueuedEventsThreshold int64 `env:"RECENTLY_QUEUED_EVENTS_THRESHOLD" envDefault:"300"`
	// The directory to use for persisting state files
	// when using the in-memory storage with file system (memfile) persistence engine.
	MemFileStateDir string `env:"MEMFILE_STATE_DIR" envDefault:"$HOME/.bluelink/deploy-engine/state"`
	// Sets the guide for the maximum size of a state chunk file in bytes
	// when using the in-memory storage with file system (memfile) persistence engine.
	// If a single record (instance or resource drift entry) exceeds this size,
	// it will not be split into multiple files.
	// This is only a guide, the actual size of the files are often likely to be larger.
	// Defaults to "1048576" (1MB).
	MemFileMaxGuideFileSize int64 `env:"MEMFILE_MAX_GUIDE_FILE_SIZE" envDefault:"1048576"`
	// Sets the maximum size of an event channel partition file in bytes
	// when using the in-memory storage with file system (memfile) persistence engine.
	// Each channel (e.g. deployment or change staging process) will have its own partition file
	// for events that are captured from the blueprint container.
	// This is a hard limit, if a new event is added to a partition file
	// that causes the file to exceed this size, an error will occur and the event
	// will not be persisted.
	// Defaults to "10485760" (10MB).
	MemFileMaxEventPartitionSize int64 `env:"MEMFILE_MAX_EVENT_PARTITION_SIZE" envDefault:"10485760"`
	// The user name to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	PostgresUser string `env:"POSTGRES_USER"`
	// The password for the user to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	PostgresPassword string `env:"POSTGRES_PASSWORD"`
	// The host to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// Defaults to "localhost".
	PostgresHost string `env:"POSTGRES_HOST" envDefault:"localhost"`
	// The port to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// Defaults to "5432".
	PostgresPort int `env:"POSTGRES_PORT" envDefault:"5432"`
	// The name of the PostgreSQL database to connect to
	// when using the PostgreSQL storage engine.
	PostgresDatabase string `env:"POSTGRES_DATABASE"`
	// The SSL mode to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// See: https://www.postgresql.org/docs/current/libpq-ssl.html
	// Defaults to "disable".
	PostgresSSLMode string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`
	// The maximum number of connections that can be open at once
	// in the pool when using the PostgreSQL storage engine.
	// Defaults to "100".
	PostgresPoolMaxConns int `env:"POSTGRES_POOL_MAX_CONNS" envDefault:"100"`
	// The maximum lifetime of a connection to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// This should be in a format that can be parsed as a time.Duration.
	// See: https://pkg.go.dev/time#ParseDuration
	// Defaults to "1h30m".
	PostgresPoolMaxConnLifetime string `env:"POSTGRES_POOL_MAX_CONN_LIFETIME" envDefault:"1h30m"`
}

// ResolversConfig provides configuration for the child blueprint resolvers
// used by the deploy engine.
type ResolversConfig struct {
	// A custom endpoint to use to make calls to Amazon S3
	// to retrieve the contents of child blueprint files.
	S3Endpoint string `env:"S3_ENDPOINT"`
	// A custom endpoint to use to make calls to Google Cloud Storage
	// to retrieve the contents of child blueprint files.
	GCSEndpoint string `env:"GCS_ENDPOINT"`
	// A timeout in seconds to use for HTTP requests made for the "https"
	// blueprint file source scheme or for child blueprint includes
	// that use the "https"	source type.
	// Defaults to 30 seconds.
	HTTPSClientTimeout int `env:"HTTPS_CLIENT_TIMEOUT" envDefault:"30"`
}

// AuthConfig provides configuration for the way authentication
// should be handled by the deploy engine.
type AuthConfig struct {
	// The issuer URL of an OAuth2/OIDC JWT token that can be used
	// to authenticate with the deploy engine.
	// This is checked first before any other authentication methods.
	JWTIssuer string `env:"OAUTH2_OIDC_JWT_ISSUER"`
	// Determines whether or not to use HTTPS when making requests
	// to the issuer URL to retrieve metadata and the JSON Web Key Set.
	// This should only be set to false when running the deploy engine
	// with a local OAuth2/OIDC provider running on the same machine.
	//
	// Defaults to "true".
	JWTIssuerSecure bool `env:"OAUTH2_OIDC_JWT_ISSUER_SECURE" envDefault:"true"`
	// The audience of an OAuth2/OIDC JWT token that can be used
	// to authenticate with the deploy engine.
	// The deploy engine will check the audience of the token
	// against this value to ensure that the token is intended
	// for the deploy engine.
	JWTAudience string `env:"OAUTH2_OIDC_JWT_AUDIENCE"`
	// The signature algorithm that was used to create the JWT token
	// and should be used to verify the signature of the token.
	// Supported algorithms are:
	//
	// - "EdDSA" - Edwards-curve Digital Signature Algorithm
	// - "HS256" - HMAC using SHA-256
	// - "HS384" - HMAC using SHA-384
	// - "HS512" - HMAC using SHA-512
	// - "RS256" - RSASSA-PKCS-v1.5 using SHA-256
	// - "RS384" - RSASSA-PKCS-v1.5 using SHA-384
	// - "RS512" - RSASSA-PKCS-v1.5 using SHA-512
	// - "ES256" - ECDSA using P-256 and SHA-256
	// - "ES384" - ECDSA using P-384 and SHA-384
	// - "ES512" - ECDSA using P-521 and SHA-512
	// - "PS256" - RSASSA-PSS using SHA256 and MGF1-SHA256
	// - "PS384" - RSASSA-PSS using SHA384 and MGF1-SHA384
	// - "PS512" - RSASSA-PSS using SHA512 and MGF1-SHA512
	//
	// Defaults to "HS256".
	JWTSignatureAlgorithm string `env:"OAUTH2_OIDC_JWT_SIGNATURE_ALGORITHM" envDefault:"HS256"`
	// A map of key pairs to be used to verify (public key id -> secret key)
	// the contents of the Bluelink-Signature-V1 header.
	// This is checked after the JWT token but before the API key
	// authentication method.
	BluelinkSigV1KeyPairs map[string]string `env:"BLUELINK_SIGNATURE_V1_KEY_PAIRS"`
	// A list of API keys to be used to authenticate with the deploy engine.
	// This is checked last and will be used if the `Authorization` and
	// `Bluelink-Signature-V1` headers are not present.
	APIKeys []string `env:"BLUELINK_API_KEYS"`
}

// MaintenanceConfig provides configuration for the maintenance
// of short-lived resources in the deploy engine.
// This is used for things like the retention periods for
// blueprint validations and change sets.
type MaintenanceConfig struct {
	// The retention period in seconds for blueprint validations.
	// Whenever the clean up process runs,
	// it will delete all blueprint validations that are older
	// than this retention period.
	//
	// Defaults to 604,800 seconds (7 days).
	BlueprintValidationRetentionPeriod int `env:"BLUEPRINT_VALIDATION_RETENTION_PERIOD" envDefault:"604800"`
	// The retention period in seconds for change sets.
	// Whenever the clean up process runs,
	// it will delete all change sets that are older
	// than this retention period.
	//
	// Defaults to 604,800 seconds (7 days).
	ChangesetRetentionPeriod int `env:"CHANGESET_RETENTION_PERIOD" envDefault:"604800"`
	// The retention period in seconds for events.
	// Whenever the clean up process runs,
	// it will delete all events that are older
	// than this retention period.
	//
	// Defaults to 604,800 seconds (7 days).
	EventsRetentionPeriod int `env:"EVENTS_RETENTION_PERIOD" envDefault:"604800"`
}

// LoadConfigFromEnv loads the deploy engine configuration
// from environment variables.
func LoadConfigFromEnv() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	// Ensure the environment variables in the plugin path are expanded
	// as the plugin launcher only works with absolute paths.
	if config.PluginsV1.PluginPath != "" {
		config.PluginsV1.PluginPath = os.ExpandEnv(config.PluginsV1.PluginPath)
	}

	// Ensure the environment variables in the state directory are expanded
	// as the state container only works with absolute paths.
	if config.State.MemFileStateDir != "" {
		config.State.MemFileStateDir = os.ExpandEnv(config.State.MemFileStateDir)
	}

	// Set versions from generated constants.
	config.Version = deployEngineVersion
	config.PluginFrameworkVersion = pluginFrameworkVersion
	config.BlueprintFrameworkVersion = blueprintFrameworkVersion

	// Set plugin protocol versions based on the selected API version,
	// See the `internal/pluginhostv{N}` packages for the protocol versions
	// for each API version.
	switch config.APIVersion {
	case "v1":
		config.ProviderPluginProtocolVersion = providerserverv1.ProtocolVersion
		config.TransformerPluginProtocolVersion = transformerserverv1.ProtocolVersion
	}

	return config, nil
}
