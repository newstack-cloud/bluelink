package testutils

import (
	"context"
	"log"
	"net"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func StartPluginServiceServer(
	hostID string,
	pluginManager pluginservicev1.Manager,
	functionRegistry provider.FunctionRegistry,
	resourceService provider.ResourceService,
) (pluginservicev1.ServiceClient, func()) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	serviceServer := pluginservicev1.NewServiceServer(
		pluginManager,
		functionRegistry,
		resourceService,
		hostID,
		// Plugin to plugin call timeout is set to 10 milliseconds.
		pluginservicev1.WithPluginToPluginCallTimeout(10),
	)

	server := pluginservicev1.NewServer(
		serviceServer,
		pluginservicev1.WithListener(listener),
	)
	close, err := server.Serve()
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

	client := pluginservicev1.NewServiceClient(conn)

	return client, close
}
