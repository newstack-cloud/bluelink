// This file provides adaptors for the host service to interact
// with the plugin service manager using the blueprint framework
// interfaces for providers and transformer plugins.
package pluginservicev1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// GetProviderPluginAdaptors returns a map of provider adaptors
// that can be used by the host service to interact with
// registered provider plugins.
func GetProviderPluginAdaptors(manager Manager) map[string]provider.Provider {
	providerPlugins := manager.GetPlugins(PluginType_PLUGIN_TYPE_PROVIDER)
	adaptors := make(map[string]provider.Provider)

	for _, plugin := range providerPlugins {
		namespace := utils.ExtractPluginNamespace(plugin.Info.ID)
		// The factory used by the manager is expected to wrap the plugin clients
		// with adaptors that produce an implementation of the provider.Provider interface.
		providerPlugin, isProvider := plugin.Client.(provider.Provider)
		if isProvider {
			adaptors[namespace] = providerPlugin
		}
	}

	return adaptors
}
