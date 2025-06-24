package testtransformer

import (
	"context"
	"log"
	"net"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// StartPluginServer starts the test transformer plugin server
// to run in the same process as the test suite.
func StartPluginServer(
	serviceClient pluginservicev1.ServiceClient,
	failingPlugin bool,
) (transformerserverv1.TransformerClient, func()) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	pluginHostInfoContainer := pluginutils.NewHostInfoContainer()
	transformerServer := createTransformerServer(
		failingPlugin,
		pluginHostInfoContainer,
		serviceClient,
	)
	id := createPluginID(failingPlugin)
	config := plugin.ServePluginConfiguration{
		ID: id,
		PluginMetadata: &pluginservicev1.PluginMetadata{
			PluginVersion:        "1.0.0",
			DisplayName:          "Celerity Application",
			FormattedDescription: "Celerity transformer plugin that powers **Celerity** applications.",
			RepositoryUrl:        "https://github.com/newstack-cloud/celerity-transformer-celerity-app",
			Author:               "Two Hundred",
		},
		ProtocolVersion: providerserverv1.ProtocolVersion,
		Listener:        listener,
	}

	close, err := plugin.ServeTransformerV1(
		context.Background(),
		transformerServer,
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

	client := transformerserverv1.NewTransformerClient(conn)

	return client, close
}

func createTransformerServer(
	failingPlugin bool,
	pluginHostInfoContainer pluginutils.HostInfoContainer,
	serviceClient pluginservicev1.ServiceClient,
) transformerserverv1.TransformerServer {
	if failingPlugin {
		return &failingTransformerServer{}
	}
	return transformerv1.NewTransformerPlugin(
		NewTransformer(),
		pluginHostInfoContainer,
		serviceClient,
	)
}

func createPluginID(failingPlugin bool) string {
	if failingPlugin {
		return "celerity-failing/transform2"
	}
	return "celerity/transform"
}
