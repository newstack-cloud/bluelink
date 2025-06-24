package env

import (
	"github.com/caarlos0/env/v11"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
)

// Config holds the configuration for the github
// registry service.
type Config struct {
	// The current version of the plugin docgen software.
	// This will be set based on a value of a constant determined at build time.
	Version string
	// The current version of the plugin framework that is being used
	// by the plugin docgen tool.
	// This will be set based on a value of a constant determined at build time.
	PluginFrameworkVersion string
	// The current version of the blueprint framework that is being used
	// by the plugin docgen tool.
	// This will be set based on a value of a constant determined at build time.
	BlueprintFrameworkVersion string
	// The current version of the provider plugin protocol that is being used
	// by the plugin docgen tool when acting as a plugin host.
	// This will be set at runtime based on the version of the plugin protocol
	// that the current version of the tool uses.
	ProviderPluginProtocolVersion string
	// The current version of the transformer plugin protocol that is being used
	// by the plugin docgen tool when acting as a plugin host.
	// This will be set at runtime based on the version of the plugin protocol
	// that the current version of the tool uses.
	TransformerPluginProtocolVersion string
	PluginPath                       string `env:"BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH"`
	PluginLogFileRootDir             string `env:"BLUELINK_PLUGIN_DOCGEN_PLUGIN_LOG_FILE_ROOT_DIR"`
	LogLevel                         string `env:"BLUELINK_PLUGIN_DOCGEN_LOG_LEVEL" envDefault:"info"`
	LaunchWaitTimeoutMS              int    `env:"BLUELINK_PLUGIN_DOCGEN_PLUGIN_LAUNCH_WAIT_TIMEOUT_MS" envDefault:"10000"`
	GenerateTimeoutMS                int    `env:"BLUELINK_PLUGIN_DOCGEN_GENERATE_TIMEOUT_MS" envDefault:"30000"`
}

// LoadConfig loads environment configuration
// for the plugin JSON doc generator tool.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	// Set versions from generated constants.
	config.Version = pluginDocgenVersion
	config.PluginFrameworkVersion = pluginFrameworkVersion
	config.BlueprintFrameworkVersion = blueprintFrameworkVersion

	// Set plugin protocol versions used by the current version of the tool.
	config.ProviderPluginProtocolVersion = providerserverv1.ProtocolVersion
	config.TransformerPluginProtocolVersion = transformerserverv1.ProtocolVersion

	return config, nil
}
