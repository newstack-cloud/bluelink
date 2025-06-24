package plugin

const (
	// DeployEnginePluginPathEnvVar is the name of the environment variable
	// that should be set to a colon-separated list of directories to search
	// for plugins in.
	DeployEnginePluginPathEnvVar = "BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH"
	// DeployEnginePluginLaunchAttemptLimitEnvVar is the name of the environment
	// variable that should be set to the number of times to attempt launching
	// a plugin before giving up.
	DeployEnginePluginLaunchAttemptLimitEnvVar = "BLUELINK_DEPLOY_ENGINE_PLUGIN_LAUNCH_ATTEMPT_LIMIT"
	// DeployEnginePluginLogFileRootDirEnvVar is the name of the environment variable
	// that should be set to the root directory for plugin log files.
	DeployEnginePluginLogFileRootDirEnvVar = "BLUELINK_DEPLOY_ENGINE_PLUGIN_LOG_FILE_ROOT_DIR"
)
