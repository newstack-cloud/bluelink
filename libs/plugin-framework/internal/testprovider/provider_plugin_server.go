package testprovider

import (
	"context"
	"log"
	"net"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// StartPluginServer starts the test provider plugin server
// to run in the same process as the test suite.
func StartPluginServer(
	serviceClient pluginservicev1.ServiceClient,
	failingPlugin bool,
) (providerserverv1.ProviderClient, func()) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	pluginHostInfoContainer := pluginutils.NewHostInfoContainer()
	providerServer := createProviderServer(
		failingPlugin,
		pluginHostInfoContainer,
		serviceClient,
	)
	id := createPluginID(failingPlugin)
	config := plugin.ServePluginConfiguration{
		ID: id,
		PluginMetadata: &pluginservicev1.PluginMetadata{
			PluginVersion: "1.0.0",
			DisplayName:   "AWS",
			FormattedDescription: "AWS provider for the Deploy Engine including `resources`, `data sources`," +
				" `links` and `custom variable types` for interacting with AWS services.",
			RepositoryUrl: "https://github.com/newstack-cloud/bluelink-provider-aws",
			Author:        "NewStack Cloud",
		},
		ProtocolVersion: providerserverv1.ProtocolVersion,
		Listener:        listener,
	}

	close, err := plugin.ServeProviderV1(
		context.Background(),
		providerServer,
		serviceClient,
		pluginHostInfoContainer,
		config,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	client := providerserverv1.NewProviderClient(conn)

	return client, close
}

func createProviderServer(
	failingPlugin bool,
	pluginHostInfoContainer pluginutils.HostInfoContainer,
	serviceClient pluginservicev1.ServiceClient,
) providerserverv1.ProviderServer {
	if failingPlugin {
		return &failingProviderServer{}
	}
	return providerv1.NewProviderPlugin(
		NewProvider(),
		pluginHostInfoContainer,
		serviceClient,
	)
}

func createPluginID(failingPlugin bool) string {
	if failingPlugin {
		return "bluelink-failing/aws2"
	}
	return "bluelink/aws"
}
