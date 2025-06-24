package plugin

import (
	context "context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginbase"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
)

var (
	// ErrUnsupportedProviderProtocolVersion is returned when the protocol version
	// is not supported.
	ErrUnsupportedProviderProtocolVersion = errors.New("unsupported provider protocol version")
)

// ServeProviderV1 handles serving the v1 provider plugin with the given providerServer and options.
// This will deal with registering the provider with the host service.
func ServeProviderV1(
	ctx context.Context,
	providerServer any,
	pluginServiceClient pluginservicev1.ServiceClient,
	hostInfoContainer pluginutils.HostInfoContainer,
	config ServePluginConfiguration,
) (func(), error) {
	if config.ID == "" {
		return nil, errors.New("ID is required for a provider plugin")
	}

	if config.PluginMetadata == nil {
		return nil, errors.New("PluginMetadata is required for a provider plugin")
	}

	if config.ProtocolVersion != providerserverv1.ProtocolVersion {
		return nil, ErrUnsupportedProviderProtocolVersion
	}

	provider, isv1Provider := providerServer.(providerserverv1.ProviderServer)
	if !isv1Provider {
		return nil, errors.New("unsupported provider server type")
	}

	opts := []pluginbase.ServerOption[providerserverv1.ProviderServer]{}
	if config.Debug {
		opts = append(opts, pluginbase.WithDebug[providerserverv1.ProviderServer]())
	}

	if config.TCPPort != 0 && config.UnixSocketPath != "" {
		return nil, errors.New("both TCPPort and UnixSocketPath cannot be set")
	}

	if config.UnixSocketPath != "" {
		opts = append(
			opts,
			pluginbase.WithUnixSocket[providerserverv1.ProviderServer](config.UnixSocketPath),
		)
	}

	if config.TCPPort != 0 {
		opts = append(
			opts,
			pluginbase.WithTCPPort[providerserverv1.ProviderServer](config.TCPPort),
		)
	}

	if config.Listener != nil {
		opts = append(
			opts,
			pluginbase.WithListener[providerserverv1.ProviderServer](config.Listener),
		)
	}

	server := providerserverv1.NewServer(
		config.ID,
		config.PluginMetadata,
		provider,
		pluginServiceClient,
		hostInfoContainer,
		opts...,
	)
	return server.Serve()
}
