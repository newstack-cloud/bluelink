package providerserverv1

import (
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginbase"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
)

const (
	// The protocol version that is used during the handshake to ensure the plugin
	// is compatible with the host service.
	ProtocolVersion = "1.0"
)

// NewServer creates a new plugin server for a provider plugin, taking
// care of registration and running the server.
func NewServer(
	pluginID string,
	pluginMetadata *pluginservicev1.PluginMetadata,
	provider ProviderServer,
	pluginServiceClient pluginservicev1.ServiceClient,
	hostInfoContainer pluginutils.HostInfoContainer,
	opts ...pluginbase.ServerOption[ProviderServer],
) *pluginbase.Server[ProviderServer] {
	return pluginbase.NewServer(
		&pluginbase.CorePluginConfig[ProviderServer]{
			PluginID:        pluginID,
			PluginType:      pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER,
			ProtocolVersion: ProtocolVersion,
			PluginServer:    provider,
		},
		RegisterProviderServer,
		pluginMetadata,
		pluginServiceClient,
		hostInfoContainer,
		opts...,
	)
}
