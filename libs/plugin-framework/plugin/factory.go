package plugin

import (
	"errors"
	"fmt"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreatePluginInstance is a function that creates a new instance of a plugin.
// This implements the pluginservicev1.PluginFactory interface.
func CreatePluginInstance(info *pluginservicev1.PluginInstanceInfo, hostID string) (any, func(), error) {
	if info.PluginType == pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER &&
		slices.Contains(info.ProtocolVersions, "1.0") {
		return createV1ProviderPlugin(info, hostID)
	}

	if info.PluginType == pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER &&
		slices.Contains(info.ProtocolVersions, "1.0") {
		return createV1TransformerPlugin(info, hostID)
	}

	return nil, nil, errors.New("unsupported plugin type or protocol version")
}

func createV1ProviderPlugin(info *pluginservicev1.PluginInstanceInfo, hostID string) (any, func(), error) {

	conn, err := createGRPCConnection(info)
	closeConn := func() {
		conn.Close()
	}
	if err != nil {
		return nil, closeConn, err
	}

	client := providerserverv1.NewProviderClient(conn)
	// Give the deploy engine an instance of the provider.Provider
	// interface for the blueprint framework to interact with,
	// this plays into a more seamless integration with the deploy engine
	// and the blueprint framework, allowing for an instance of the deploy engine
	// to opt out of using the gRPC server plugin system.
	wrapped := providerserverv1.WrapProviderClient(client, hostID)
	return wrapped, closeConn, nil
}

func createV1TransformerPlugin(info *pluginservicev1.PluginInstanceInfo, hostID string) (any, func(), error) {

	conn, err := createGRPCConnection(info)
	closeConn := func() {
		conn.Close()
	}
	if err != nil {
		return nil, closeConn, err
	}

	client := transformerserverv1.NewTransformerClient(conn)
	// Give the deploy engine an instance of the transform.SpecTransformer
	// interface for the blueprint framework to interact with,
	// this plays into a more seamless integration with the deploy engine
	// and the blueprint framework, allowing for an instance of the deploy engine
	// to opt out of using the gRPC server plugin system.
	wrapped := transformerserverv1.WrapTransformerClient(client, hostID)
	return wrapped, closeConn, nil
}

func createGRPCConnection(info *pluginservicev1.PluginInstanceInfo) (*grpc.ClientConn, error) {
	if info.UnixSocketPath != "" {
		return grpc.NewClient("unix://"+info.UnixSocketPath, grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		))
	}
	return grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", info.TCPPort), grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
}
