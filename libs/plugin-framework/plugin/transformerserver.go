package plugin

import (
	context "context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginbase"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
)

var (
	// ErrUnsupportedTransformerProtocolVersion is returned when the protocol version
	// is not supported.
	ErrUnsupportedTransformerProtocolVersion = errors.New("unsupported transformer protocol version")
)

// ServeTransformerV1 handles serving the v1 transformer plugin with the given transformerServer and options.
// This will deal with registering the transformer with the host service.
func ServeTransformerV1(
	ctx context.Context,
	transformerServer any,
	pluginServiceClient pluginservicev1.ServiceClient,
	hostInfoContainer pluginutils.HostInfoContainer,
	config ServePluginConfiguration,
) (func(), error) {
	if config.ID == "" {
		return nil, errors.New("ID is required for a transformer plugin")
	}

	if config.PluginMetadata == nil {
		return nil, errors.New("PluginMetadata is required for a transformer plugin")
	}

	if config.ProtocolVersion != transformerserverv1.ProtocolVersion {
		return nil, ErrUnsupportedProviderProtocolVersion
	}

	transformer, isv1Transformer := transformerServer.(transformerserverv1.TransformerServer)
	if !isv1Transformer {
		return nil, errors.New("unsupported transformer server type")
	}

	opts := []pluginbase.ServerOption[transformerserverv1.TransformerServer]{}
	if config.Debug {
		opts = append(opts, pluginbase.WithDebug[transformerserverv1.TransformerServer]())
	}

	if config.TCPPort != 0 && config.UnixSocketPath != "" {
		return nil, errors.New("both TCPPort and UnixSocketPath cannot be set")
	}

	if config.UnixSocketPath != "" {
		opts = append(
			opts,
			pluginbase.WithUnixSocket[transformerserverv1.TransformerServer](config.UnixSocketPath),
		)
	}

	if config.TCPPort != 0 {
		opts = append(
			opts,
			pluginbase.WithTCPPort[transformerserverv1.TransformerServer](config.TCPPort),
		)
	}

	if config.Listener != nil {
		opts = append(
			opts,
			pluginbase.WithListener[transformerserverv1.TransformerServer](config.Listener),
		)
	}

	server := transformerserverv1.NewServer(
		config.ID,
		config.PluginMetadata,
		transformer,
		pluginServiceClient,
		hostInfoContainer,
		opts...,
	)
	return server.Serve()
}
