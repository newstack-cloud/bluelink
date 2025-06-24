package utils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
)

const (
	hostAppKey = "plugin-docgen"
)

// Produces default context variables for
// the current version of the plugin docgen tool.
func defaultContextVars(config *env.Config) map[string]*core.ScalarValue {
	return map[string]*core.ScalarValue{
		"hostApp":                          core.ScalarFromString(hostAppKey),
		"pluginDocgenVersion":              core.ScalarFromString(config.Version),
		"pluginFrameworkVersion":           core.ScalarFromString(config.PluginFrameworkVersion),
		"blueprintFrameworkVersion":        core.ScalarFromString(config.BlueprintFrameworkVersion),
		"providerPluginProtocolVersion":    core.ScalarFromString(config.ProviderPluginProtocolVersion),
		"transformerPluginProtocolVersion": core.ScalarFromString(config.TransformerPluginProtocolVersion),
	}
}

// CreateBlueprintParams creates an empty BlueprintParams object
// with all fields initialized to empty maps or nil values.
func CreateBlueprintParams(config *env.Config) core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		defaultContextVars(config),
		map[string]*core.ScalarValue{},
	)
}
