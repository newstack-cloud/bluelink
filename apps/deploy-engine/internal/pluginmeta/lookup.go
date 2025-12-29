package pluginmeta

import (
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// ProviderMetadata holds plugin identification for a provider.
type ProviderMetadata struct {
	PluginID      string
	PluginVersion string
}

// Lookup provides access to provider plugin metadata.
type Lookup interface {
	// GetProviderMetadata returns the plugin ID and version for a provider namespace.
	// The providerNamespace is the last segment of the plugin ID (e.g., "aws" from "newstack-cloud/aws").
	// Returns nil if the provider is not registered.
	GetProviderMetadata(providerNamespace string) *ProviderMetadata
}

type lookupImpl struct {
	pluginManager pluginservicev1.Manager
}

// NewLookup creates a new Lookup that retrieves provider metadata
// from the plugin manager.
func NewLookup(manager pluginservicev1.Manager) Lookup {
	return &lookupImpl{
		pluginManager: manager,
	}
}

func (l *lookupImpl) GetProviderMetadata(providerNamespace string) *ProviderMetadata {
	if l.pluginManager == nil {
		return nil
	}

	// Find the full plugin ID for this namespace by iterating over registered plugins.
	// Plugin IDs have the format "{hostname/}?{namespace}/{plugin}" (e.g., "newstack-cloud/aws")
	// and the provider namespace is extracted from the last segment (e.g., "aws").
	pluginID := l.findPluginIDByNamespace(providerNamespace)
	if pluginID == "" {
		return nil
	}

	metadata := l.pluginManager.GetPluginMetadata(
		pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
		pluginID,
	)
	if metadata == nil {
		return nil
	}

	return &ProviderMetadata{
		PluginID:      pluginID,
		PluginVersion: metadata.PluginVersion,
	}
}

// findPluginIDByNamespace searches for a plugin ID that matches the given namespace.
// The namespace is the last segment of the plugin ID (e.g., "aws" from "newstack-cloud/aws").
// Returns empty string if not found.
func (l *lookupImpl) findPluginIDByNamespace(namespace string) string {
	plugins := l.pluginManager.GetPlugins(pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER)
	for _, plugin := range plugins {
		if plugin.Info != nil && utils.ExtractPluginNamespace(plugin.Info.ID) == namespace {
			return plugin.Info.ID
		}
	}
	return ""
}

// LookupFunc is a function type that can be used as a provider metadata lookup.
// This is useful for passing to DeployContext where a simple function signature is preferred.
type LookupFunc func(providerNamespace string) (pluginID, pluginVersion string)

// ToLookupFunc converts a Lookup interface to a LookupFunc.
func ToLookupFunc(lookup Lookup) LookupFunc {
	if lookup == nil {
		return nil
	}

	return func(providerNamespace string) (pluginID, pluginVersion string) {
		metadata := lookup.GetProviderMetadata(providerNamespace)
		if metadata == nil {
			return "", ""
		}
		return metadata.PluginID, metadata.PluginVersion
	}
}
