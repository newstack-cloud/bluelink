package core

import (
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"github.com/spf13/viper"
)

// Config provides configuration for the deploy engine application.
// This parses configuratoin from the current environment.
type Config struct {
	// The version of the deploy engine API to use.
	// Defaults to "v1".
	APIVersion string `mapstructure:"api_version"`
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
	Port int `mapstructure:"port"`
	// Determines whether or not to use unix sockets for handling
	// incoming connections instead of TCP.
	// If set to true, the Port will be ignored and the UnixSocketPath
	// will be used instead.
	// Defaults to "false".
	UseUnixSocket bool `mapstructure:"use_unix_socket"`
	// The path to the unix socket to listen on for incoming connections.
	// This will be ignored if UseUnixSocket is set to false.
	// Defaults to "/tmp/bluelink.sock".
	UnixSocketPath string `mapstructure:"unix_socket_path"`
	// LoopbackOnly determines whether or not to restrict the server
	// to only accept connections from the loopback interface.
	// Defaults to "true" for a more secure default.
	// This should be intentionally set to false for deployments
	// of the deploy engine that are intended to be accessible
	// over a private network or the public internet.
	LoopbackOnly bool `mapstructure:"loopback_only"`
	// Environment determines whether the deploy engine is running
	// in a production or development environment.
	// This is used to determine things like the formatting of logs,
	// in development mode, logs are formatted in a more human readable format,
	// while in production mode, logs are formatted purely in JSON for easier
	// parsing and processing by log management systems.
	// Defaults to "production".
	Environment string `mapstructure:"environment"`
	// LogLevel determines the level of logging to use for the deploy engine.
	// Defaults to "info".
	// Can be set to any of the logging levels supported by zap:
	// debug, info, warn, error, dpanic, panic, fatal.
	// See: https://pkg.go.dev/go.uber.org/zap#Level
	LogLevel string `mapstructure:"log_level"`
	// Auth provides configuration for the way authentication
	// should be handled by the deploy engine.
	Auth AuthConfig `mapstructure:"auth"`
	// PluginsV1 provides configuration for the v1 plugin system
	// implemented by the deploy engine.
	PluginsV1 PluginsV1Config `mapstructure:"plugins_v1"`
	// Blueprints provides configuration for the blueprint loader
	// used by the deploy engine.
	Blueprints BlueprintConfig `mapstructure:"blueprints"`
	// State provides configuration for the state management/persistence
	// layer used by the deploy engine.
	State StateConfig `mapstructure:"state"`
	// Resolvers provides configuration for the child blueprint resolvers
	// used by the deploy engine.
	Resolvers ResolversConfig `mapstructure:"resolvers"`
	// Maintenance provides configuration for the maintenance
	// of short-lived resources in the deploy engine.
	// This is used for things like the retention periods for
	// blueprint validations and change sets.
	Maintenance MaintenanceConfig `mapstructure:"maintenance"`
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
	// Defaults to $HOME/.bluelink/engine/plugins/bin on Linux and macOS,
	// where $HOME will be expanded to the current user's home directory.
	// Defaults to %LOCALAPPDATA%\Bluelink\engine\plugins on Windows.
	PluginPath string `mapstructure:"plugin_path"`
	// LogFileRootDir is the path to a single root directory used to store
	// logs for all plugins. stdout and stderr for each plugin
	// will be redirected to log files under this directory.
	// Defaults to $HOME/.bluelink/engine/plugins/logs on Linux and macOS,
	// where $HOME will be expanded to the current user's home directory.
	// Defaults to %LOCALAPPDATA%\Bluelink\engine\plugins\logs on Windows.
	LogFileRootDir string `mapstructure:"log_file_root_dir"`
	// LaunchWaitTimeoutMS is the timeout in milliseconds
	// to wait for a plugin to register with the host.
	// This is used when the plugin host is started and
	// a plugin is expected to register with the host.
	// Defaults to 15,000ms (15 seconds)
	LaunchWaitTimeoutMS int `mapstructure:"launch_wait_timeout_ms"`
	// TotalLaunchWaitTimeoutMS is the timeout in milliseconds
	// to wait for all plugins to register with the host.
	// This is used when the plugin host is started and
	// all plugins are expected to register with the host.
	// Defaults to 60,000ms (1 minute)
	TotalLaunchWaitTimeoutMS int `mapstructure:"total_launch_wait_timeout_ms"`
	// ResourceStabilisationPollingTimeoutMS is the timeout in milliseconds
	// to wait for a resource to stabilise when calls are made
	// into the resource registry through the plugin service.
	// This same timeout is used for configuring the blueprint loader and
	// plugin host.
	// Defaults to 3,600,000ms (1 hour)
	ResourceStabilisationPollingTimeoutMS int `mapstructure:"resource_stabilisation_polling_timeout_ms"`
	// PluginToPluginCallTimeoutMS is the timeout in milliseconds
	// to wait for a plugin to respond to a call initiated by another
	// or the same plugin through the plugin service.
	// The exception, where this timeout is not used, is when waiting for
	// a resource to stabilise when calls are made into the resource registry
	// through the plugin service.
	// Defaults to 120,000ms (2 minutes)
	PluginToPluginCallTimeoutMS int `mapstructure:"plugin_to_plugin_call_timeout_ms"`
}

// BlueprintConfig provides configuration for the blueprint loader
// used by the deploy engine.
type BlueprintConfig struct {
	// ValidateAfterTransform determines whether or not the blueprint
	// loader should validate blueprints after applying transformations.
	// Defaults to "false".
	// This should only really be set to true when there is a need to debug
	// issues that may be due to transformer plugins producing invalid output.
	ValidateAfterTransform bool `mapstructure:"validate_after_transform"`
	// EnableDriftCheck determines whether or not the blueprint
	// loader should check for drift in the state of resources
	// when staging changes for a blueprint deployment.
	// Defaults to "true".
	EnableDriftCheck bool `mapstructure:"enable_drift_check"`
	// ResourceStabilisationPollingIntervalMS is the interval in milliseconds
	// to wait between polling for a resource to stabilise
	// when calls are made to a provider to check if a resource has stabilised.
	// This is used in the plugin host for plugin to plugin calls
	// (i.e. links deploying intermediary resources)
	// and in the blueprint container that manages deployment of resources declared
	// in a blueprint.
	// Defaults to 5,000ms (5 seconds)
	ResourceStabilisationPollingIntervalMS int `mapstructure:"resource_stabilisation_polling_interval_ms"`
	// DefaultRetryPolicy is the default retry policy to use
	// when a provider returns a retryable error for actions that support retries.
	// This should be a serialised JSON string that matches the structure of the
	// `provider.RetryPolicy` struct.
	// The built-in default will be used if this is not set or the JSON is not
	// in the correct format.
	DefaultRetryPolicy string `mapstructure:"default_retry_policy"`
	// DeploymentTimeout is the time in seconds to wait for a deployment
	// to complete before timing out.
	// This timeout is for the background process that runs the deployment
	// when the deployment endpoints are called.
	// Defaults to 10,800 seconds (3 hours).
	DeploymentTimeout int `mapstructure:"deployment_timeout"`
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
	StorageEngine string `mapstructure:"storage_engine"`
	// The threshold in seconds for retrieving recently queued events
	// for a stream when a starting event ID is not provided.
	// Any events that are older than currentTime - threshold
	// will not be considered as recently queued events.
	// This applies to all storage engines.
	// Defaults to 300 seconds (5 minutes).
	RecentlyQueuedEventsThreshold int64 `mapstructure:"recently_queued_events_threshold"`
	// The directory to use for persisting state files
	// when using the in-memory storage with file system (memfile) persistence engine.
	MemFileStateDir string `mapstructure:"memfile_state_dir"`
	// Sets the guide for the maximum size of a state chunk file in bytes
	// when using the in-memory storage with file system (memfile) persistence engine.
	// If a single record (instance or resource drift entry) exceeds this size,
	// it will not be split into multiple files.
	// This is only a guide, the actual size of the files are often likely to be larger.
	// Defaults to "1048576" (1MB).
	MemFileMaxGuideFileSize int64 `mapstructure:"memfile_max_guide_file_size"`
	// Sets the maximum size of an event channel partition file in bytes
	// when using the in-memory storage with file system (memfile) persistence engine.
	// Each channel (e.g. deployment or change staging process) will have its own partition file
	// for events that are captured from the blueprint container.
	// This is a hard limit, if a new event is added to a partition file
	// that causes the file to exceed this size, an error will occur and the event
	// will not be persisted.
	// Defaults to "10485760" (10MB).
	MemFileMaxEventPartitionSize int64 `mapstructure:"memfile_max_event_partition_size"`
	// The user name to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	PostgresUser string `mapstructure:"postgres_user"`
	// The password for the user to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	PostgresPassword string `mapstructure:"postgres_password"`
	// The host to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// Defaults to "localhost".
	PostgresHost string `mapstructure:"postgres_host"`
	// The port to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// Defaults to "5432".
	PostgresPort int `mapstructure:"postgres_port"`
	// The name of the PostgreSQL database to connect to
	// when using the PostgreSQL storage engine.
	PostgresDatabase string `mapstructure:"postgres_database"`
	// The SSL mode to use for connecting to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// See: https://www.postgresql.org/docs/current/libpq-ssl.html
	// Defaults to "disable".
	PostgresSSLMode string `mapstructure:"postgres_ssl_mode"`
	// The maximum number of connections that can be open at once
	// in the pool when using the PostgreSQL storage engine.
	// Defaults to "100".
	PostgresPoolMaxConns int `mapstructure:"postgres_pool_max_conns"`
	// The maximum lifetime of a connection to the PostgreSQL database
	// when using the PostgreSQL storage engine.
	// This should be in a format that can be parsed as a time.Duration.
	// See: https://pkg.go.dev/time#ParseDuration
	// Defaults to "1h30m".
	PostgresPoolMaxConnLifetime string `mapstructure:"postgres_pool_max_conn_lifetime"`
}

// ResolversConfig provides configuration for the child blueprint resolvers
// used by the deploy engine.
type ResolversConfig struct {
	// A custom endpoint to use to make calls to Amazon S3
	// to retrieve the contents of child blueprint files.
	S3Endpoint string `mapstructure:"s3_endpoint"`
	// Whether to use path-style addressing for S3 requests.
	// When true, requests will be made to {endpoint}/{bucket}/{key} instead of
	// {bucket}.{endpoint}/{key}. This is required for S3-compatible services
	// like MinIO that don't support virtual-hosted-style addressing.
	// Defaults to false.
	S3UsePathStyle bool `mapstructure:"s3_use_path_style"`
	// A custom endpoint to use to make calls to Google Cloud Storage
	// to retrieve the contents of child blueprint files.
	GCSEndpoint string `mapstructure:"gcs_endpoint"`
	// A timeout in seconds to use for HTTP requests made for the "https"
	// blueprint file source scheme or for child blueprint includes
	// that use the "https"	source type.
	// Defaults to 30 seconds.
	HTTPSClientTimeout int `mapstructure:"https_client_timeout"`
}

// AuthConfig provides configuration for the way authentication
// should be handled by the deploy engine.
type AuthConfig struct {
	// The issuer URL of an OAuth2/OIDC JWT token that can be used
	// to authenticate with the deploy engine.
	// This is checked first before any other authentication methods.
	JWTIssuer string `mapstructure:"oauth2_oidc_jwt_issuer"`
	// Determines whether or not to use HTTPS when making requests
	// to the issuer URL to retrieve metadata and the JSON Web Key Set.
	// This should only be set to false when running the deploy engine
	// with a local OAuth2/OIDC provider running on the same machine.
	//
	// Defaults to "true".
	JWTIssuerSecure bool `mapstructure:"oauth2_oidc_jwt_issuer_secure"`
	// The audience of an OAuth2/OIDC JWT token that can be used
	// to authenticate with the deploy engine.
	// The deploy engine will check the audience of the token
	// against this value to ensure that the token is intended
	// for the deploy engine.
	JWTAudience string `mapstructure:"oauth2_oidc_jwt_audience"`
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
	JWTSignatureAlgorithm string `mapstructure:"oauth2_oidc_jwt_signature_algorithm"`
	// A map of key pairs to be used to verify (public key id -> secret key)
	// the contents of the Bluelink-Signature-V1 header.
	// This is checked after the JWT token but before the API key
	// authentication method.
	BluelinkSigV1KeyPairs map[string]string `mapstructure:"bluelink_signature_v1_key_pairs"`
	// A list of API keys to be used to authenticate with the deploy engine.
	// This is checked last and will be used if the `Authorization` and
	// `Bluelink-Signature-V1` headers are not present.
	APIKeys []string `mapstructure:"bluelink_api_keys"`
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
	BlueprintValidationRetentionPeriod int `mapstructure:"blueprint_validation_retention_period"`
	// The retention period in seconds for change sets.
	// Whenever the clean up process runs,
	// it will delete all change sets that are older
	// than this retention period.
	//
	// Defaults to 604,800 seconds (7 days).
	ChangesetRetentionPeriod int `mapstructure:"changeset_retention_period"`
	// The retention period in seconds for events.
	// Whenever the clean up process runs,
	// it will delete all events that are older
	// than this retention period.
	//
	// Defaults to 604,800 seconds (7 days).
	EventsRetentionPeriod int `mapstructure:"events_retention_period"`
}

// LoadConfig loads the deploy engine configuration
// from environment variables or a config file or a combination of both,
// falling back to reasonable defaults for optional configuration values.
func LoadConfig() (Config, error) {
	viperInstance := viper.New()

	viperInstance.SetConfigName("config")
	addConfigPaths(viperInstance)
	bindEnvVars(viperInstance)
	setDefaults(viperInstance)

	err := viperInstance.ReadInConfig()
	if err != nil {
		// Config is created before the logger,
		// so we'll use standard library logging to output this message.
		log.Printf(
			"failed to read config file: %s, "+
				"will try to use environment variables and fall back to defaults",
			err,
		)
	} else {
		log.Printf(
			"config file read successfully, using %s", viperInstance.ConfigFileUsed(),
		)
	}

	var config Config
	err = viperInstance.Unmarshal(&config, viper.DecodeHook(configHook()))
	if err != nil {
		return Config{}, err
	}

	// Ensure the environment variables in the plugin path are expanded
	// as the plugin launcher only works with absolute paths.
	if config.PluginsV1.PluginPath != "" {
		config.PluginsV1.PluginPath = utils.ExpandEnv(config.PluginsV1.PluginPath)
	}

	// Ensure the environment variables in the state directory are expanded
	// as the state container only works with absolute paths.
	if config.State.MemFileStateDir != "" {
		config.State.MemFileStateDir = utils.ExpandEnv(config.State.MemFileStateDir)
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

func addConfigPaths(viperInstance *viper.Viper) {
	viperInstance.AddConfigPath(".")
	viperInstance.AddConfigPath(getOSDefaultConfigDirPath())
	customPath, customPathExists := os.LookupEnv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH")
	if customPathExists {
		viperInstance.AddConfigPath(customPath)
	}
}

func getOSDefaultConfigDirPath() string {
	if runtime.GOOS == "windows" {
		return utils.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine")
}

func bindEnvVars(viperInstance *viper.Viper) {
	viperInstance.SetEnvPrefix("bluelink_deploy_engine")
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viperInstance.BindEnv("api_version")
	viperInstance.BindEnv("port")
	viperInstance.BindEnv("use_unix_socket")
	viperInstance.BindEnv("unix_socket_path")
	viperInstance.BindEnv("loopback_only")
	viperInstance.BindEnv("environment")
	viperInstance.BindEnv("log_level")

	viperInstance.BindEnv("auth.oauth2_oidc_jwt_issuer")
	viperInstance.BindEnv("auth.oauth2_oidc_jwt_issuer_secure")
	viperInstance.BindEnv("auth.oauth2_oidc_jwt_audience")
	viperInstance.BindEnv("auth.oauth2_oidc_jwt_signature_algorithm")
	viperInstance.BindEnv("auth.bluelink_signature_v1_key_pairs")
	viperInstance.BindEnv("auth.bluelink_api_keys")

	// Shared environment variables that are used by multiple plugin hosts on the same machine.
	// For this reason, we won't use the standard viper naming convention based on the config structure.
	viperInstance.BindEnv("plugins_v1.plugin_path", "BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH")
	viperInstance.BindEnv("plugins_v1.log_file_root_dir", "BLUELINK_DEPLOY_ENGINE_PLUGIN_LOG_FILE_ROOT_DIR")

	viperInstance.BindEnv("plugins_v1.launch_wait_timeout_ms")
	viperInstance.BindEnv("plugins_v1.total_launch_wait_timeout_ms")
	viperInstance.BindEnv("plugins_v1.resource_stabilisation_polling_timeout_ms")
	viperInstance.BindEnv("plugins_v1.plugin_to_plugin_call_timeout_ms")

	viperInstance.BindEnv("blueprints.validate_after_transform")
	viperInstance.BindEnv("blueprints.enable_drift_check")
	viperInstance.BindEnv("blueprints.resource_stabilisation_polling_interval_ms")
	viperInstance.BindEnv("blueprints.default_retry_policy")
	viperInstance.BindEnv("blueprints.deployment_timeout")

	viperInstance.BindEnv("state.storage_engine")
	viperInstance.BindEnv("state.recently_queued_events_threshold")
	viperInstance.BindEnv("state.memfile_state_dir")
	viperInstance.BindEnv("state.memfile_max_guide_file_size")
	viperInstance.BindEnv("state.memfile_max_event_partition_size")
	viperInstance.BindEnv("state.postgres_user")
	viperInstance.BindEnv("state.postgres_password")
	viperInstance.BindEnv("state.postgres_host")
	viperInstance.BindEnv("state.postgres_port")
	viperInstance.BindEnv("state.postgres_database")
	viperInstance.BindEnv("state.postgres_ssl_mode")
	viperInstance.BindEnv("state.postgres_pool_max_conns")
	viperInstance.BindEnv("state.postgres_pool_max_conn_lifetime")

	viperInstance.BindEnv("resolvers.s3_endpoint")
	viperInstance.BindEnv("resolvers.s3_use_path_style")
	viperInstance.BindEnv("resolvers.gcs_endpoint")
	viperInstance.BindEnv("resolvers.https_client_timeout")

	viperInstance.BindEnv("maintenance.blueprint_validation_retention_period")
	viperInstance.BindEnv("maintenance.changeset_retention_period")
	viperInstance.BindEnv("maintenance.events_retention_period")
}

const (
	oneSecondMillis = 1000
	oneMinuteMillis = 60 * oneSecondMillis
	oneHourMillis   = 60 * oneMinuteMillis

	oneMinuteSeconds = 60
	oneHourSeconds   = 60 * oneMinuteSeconds
	oneDaySeconds    = 24 * oneHourSeconds

	oneMBInBytes = 1048576
)

func setDefaults(viperInstance *viper.Viper) {
	viperInstance.SetDefault("api_version", "v1")
	viperInstance.SetDefault("port", 8325)
	viperInstance.SetDefault("use_unix_socket", false)
	viperInstance.SetDefault("unix_socket_path", "/tmp/bluelink.sock")
	viperInstance.SetDefault("loopback_only", true)
	viperInstance.SetDefault("environment", "production")
	viperInstance.SetDefault("log_level", "info")

	viperInstance.SetDefault("auth.oauth2_oidc_jwt_issuer_secure", true)
	viperInstance.SetDefault("auth.oauth2_oidc_jwt_signature_algorithm", "HS256")

	viperInstance.SetDefault("plugins_v1.plugin_path", getOSDefaultPluginPath())
	viperInstance.SetDefault("plugins_v1.log_file_root_dir", getOSDefaultPluginLogFileRootDir())
	viperInstance.SetDefault("plugins_v1.launch_wait_timeout_ms", 15*oneSecondMillis)
	viperInstance.SetDefault("plugins_v1.total_launch_wait_timeout_ms", oneMinuteMillis)
	viperInstance.SetDefault("plugins_v1.resource_stabilisation_polling_timeout_ms", oneHourMillis)
	viperInstance.SetDefault("plugins_v1.plugin_to_plugin_call_timeout_ms", 2*oneMinuteMillis)

	viperInstance.SetDefault("blueprints.validate_after_transform", false)
	viperInstance.SetDefault("blueprints.enable_drift_check", true)
	viperInstance.SetDefault("blueprints.resource_stabilisation_polling_interval_ms", 5*oneSecondMillis)
	viperInstance.SetDefault("blueprints.deployment_timeout", 3*oneHourSeconds)

	viperInstance.SetDefault("state.storage_engine", "memfile")
	viperInstance.SetDefault("state.recently_queued_events_threshold", 5*oneMinuteSeconds)
	viperInstance.SetDefault("state.memfile_state_dir", getOSDefaultMemFileStateDir())
	viperInstance.SetDefault("state.memfile_max_guide_file_size", oneMBInBytes)
	viperInstance.SetDefault("state.memfile_max_event_partition_size", 10*oneMBInBytes)
	viperInstance.SetDefault("state.postgres_host", "localhost")
	viperInstance.SetDefault("state.postgres_port", 5432)
	viperInstance.SetDefault("state.postgres_ssl_mode", "disable")
	viperInstance.SetDefault("state.postgres_pool_max_conns", 100)
	viperInstance.SetDefault("state.postgres_pool_max_conn_lifetime", "1h30m")

	viperInstance.SetDefault("resolvers.https_client_timeout", 30)

	viperInstance.SetDefault("maintenance.blueprint_validation_retention_period", 7*oneDaySeconds)
	viperInstance.SetDefault("maintenance.changeset_retention_period", 7*oneDaySeconds)
	viperInstance.SetDefault("maintenance.events_retention_period", 7*oneDaySeconds)
}

func getOSDefaultPluginPath() string {
	if runtime.GOOS == "windows" {
		return utils.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\plugins")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/plugins/bin")
}

func getOSDefaultPluginLogFileRootDir() string {
	if runtime.GOOS == "windows" {
		return utils.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\plugins\\logs")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/plugins/logs")
}

func getOSDefaultMemFileStateDir() string {
	if runtime.GOOS == "windows" {
		return utils.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\state")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/state")
}

func configHook() mapstructure.DecodeHookFuncType {
	// Wrapped in a function call to add optional input parameters (eg. separator)
	return func(
		dataType reflect.Type,
		targetType reflect.Type,
		data any,
	) (any, error) {
		if dataType.Kind() == reflect.String && targetType.Kind() == reflect.Map {
			// When in environment variables, the data is a comma-separated list of key-value pairs.
			// (e.g. "key1:value1,key2:value2")
			keys := strings.Split(data.(string), ",")
			mapValue := make(map[string]string)
			for _, key := range keys {
				parts := strings.Split(key, ":")
				mapValue[parts[0]] = parts[1]
			}
			return mapValue, nil
		}

		if dataType.Kind() == reflect.String && targetType.Kind() == reflect.Slice {
			// When in environment variables, the data is a comma-separated list of values.
			keys := strings.Split(data.(string), ",")
			sliceValue := make([]string, len(keys))
			copy(sliceValue, keys)
			return sliceValue, nil
		}

		return data, nil
	}
}
