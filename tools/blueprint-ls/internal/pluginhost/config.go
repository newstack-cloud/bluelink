package pluginhost

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
)

const (
	envPluginPath             = "BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH"
	envPluginLogFileRootDir   = "BLUELINK_DEPLOY_ENGINE_PLUGIN_LOG_FILE_ROOT_DIR"
	envPluginsEnabled         = "BLUELINK_LS_PLUGINS_ENABLED"
	envLaunchWaitTimeoutMS    = "BLUELINK_LS_PLUGIN_LAUNCH_TIMEOUT_MS"
	envTotalLaunchWaitTimeout = "BLUELINK_LS_PLUGIN_TOTAL_LAUNCH_TIMEOUT_MS"

	defaultLaunchWaitTimeoutMS      = 5000
	defaultTotalLaunchWaitTimeoutMS = 60000
)

// Config defines configuration for the plugin host.
type Config interface {
	GetPluginPath() string
	GetLogFileRootDir() string
	GetLaunchWaitTimeoutMS() int
	GetTotalLaunchWaitTimeoutMS() int
	IsEnabled() bool
}

// DefaultConfig provides environment-based configuration with LSP client overrides.
type DefaultConfig struct {
	pluginsEnabled         *bool
	pluginPathOverride     *string
	logFileRootDirOverride *string
}

// NewDefaultConfig creates a new DefaultConfig instance.
func NewDefaultConfig() *DefaultConfig {
	return &DefaultConfig{}
}

// WithInitOptions applies LSP client configuration from initializationOptions.
func (c *DefaultConfig) WithInitOptions(opts *InitializationOptions) *DefaultConfig {
	if opts == nil || opts.Plugins == nil {
		return c
	}
	c.pluginsEnabled = opts.Plugins.Enabled
	c.pluginPathOverride = opts.Plugins.PluginPath
	c.logFileRootDirOverride = opts.Plugins.LogFileRootDir
	return c
}

// GetPluginPath returns the plugin path, with LSP client override taking precedence
// over the shared environment variable used by deploy engine and plugin-docgen.
func (c *DefaultConfig) GetPluginPath() string {
	if c.pluginPathOverride != nil && *c.pluginPathOverride != "" {
		return expandEnv(*c.pluginPathOverride)
	}
	if val := os.Getenv(envPluginPath); val != "" {
		return expandEnv(val)
	}
	return ""
}

// GetLogFileRootDir returns the root directory for plugin log files,
// with LSP client override taking precedence over the shared environment variable
// used by the deploy engine, falling back to a sensible OS-specific default.
func (c *DefaultConfig) GetLogFileRootDir() string {
	if c.logFileRootDirOverride != nil && *c.logFileRootDirOverride != "" {
		return expandEnv(*c.logFileRootDirOverride)
	}
	if val := os.Getenv(envPluginLogFileRootDir); val != "" {
		return expandEnv(val)
	}
	return getOSDefaultPluginLogFileRootDir()
}

// IsEnabled returns whether plugin loading is enabled, with LSP client override
// taking precedence over the environment variable.
func (c *DefaultConfig) IsEnabled() bool {
	if c.pluginsEnabled != nil {
		return *c.pluginsEnabled
	}
	return os.Getenv(envPluginsEnabled) == "true"
}

// GetLaunchWaitTimeoutMS returns the timeout in milliseconds for waiting
// for a single plugin to register with the host.
func (c *DefaultConfig) GetLaunchWaitTimeoutMS() int {
	return getEnvInt(envLaunchWaitTimeoutMS, defaultLaunchWaitTimeoutMS)
}

// GetTotalLaunchWaitTimeoutMS returns the timeout in milliseconds for waiting
// for all plugins to register with the host.
func (c *DefaultConfig) GetTotalLaunchWaitTimeoutMS() int {
	return getEnvInt(envTotalLaunchWaitTimeout, defaultTotalLaunchWaitTimeoutMS)
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

func getOSDefaultPluginLogFileRootDir() string {
	if runtime.GOOS == "windows" {
		return expandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\plugins\\logs")
	}
	return expandEnv("$HOME/.bluelink/engine/plugins/logs")
}

// expandEnv expands environment variables in the given string,
// supporting both Unix ($VAR) and Windows (%VAR%) style syntax.
func expandEnv(input string) string {
	// Convert Windows %VAR% to Unix ${VAR} syntax for os.ExpandEnv
	converted := regexp.MustCompile(`%[^%]+%`).ReplaceAllStringFunc(
		input,
		func(match string) string {
			return fmt.Sprintf("${%s}", match[1:len(match)-1])
		},
	)
	return os.ExpandEnv(converted)
}

// InitializationOptions represents the LSP initializationOptions from the client.
type InitializationOptions struct {
	Plugins *PluginInitOptions `json:"plugins,omitempty"`
}

// PluginInitOptions holds plugin-specific initialization options.
type PluginInitOptions struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	PluginPath     *string `json:"pluginPath,omitempty"`
	LogFileRootDir *string `json:"logFileRootDir,omitempty"`
}
