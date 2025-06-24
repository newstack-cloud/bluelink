package params

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

const (
	hostAppKey = "deploy-engine"
)

// DefaultContextVars produces default context variables for
// the current version of the deploy engine.
func DefaultContextVars(config *core.Config) map[string]*bpcore.ScalarValue {
	return map[string]*bpcore.ScalarValue{
		"hostApp":                          bpcore.ScalarFromString(hostAppKey),
		"engineApiVersion":                 bpcore.ScalarFromString(config.APIVersion),
		"engineVersion":                    bpcore.ScalarFromString(config.Version),
		"pluginFrameworkVersion":           bpcore.ScalarFromString(config.PluginFrameworkVersion),
		"blueprintFrameworkVersion":        bpcore.ScalarFromString(config.BlueprintFrameworkVersion),
		"providerPluginProtocolVersion":    bpcore.ScalarFromString(config.ProviderPluginProtocolVersion),
		"transformerPluginProtocolVersion": bpcore.ScalarFromString(config.TransformerPluginProtocolVersion),
	}
}
