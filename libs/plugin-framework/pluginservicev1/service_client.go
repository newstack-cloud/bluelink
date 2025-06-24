package pluginservicev1

import (
	"fmt"
	"os"
	"strconv"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewEnvServiceClient creates a new plugin service client
// from the current environment.
func NewEnvServiceClient() (ServiceClient, func(), error) {
	servicePort := os.Getenv("BLUELINK_BUILD_ENGINE_PLUGIN_SERVICE_PORT")
	if servicePort == "" {
		servicePort = strconv.Itoa(DefaultPort)
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%s", servicePort),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	client := NewServiceClient(conn)
	close := func() {
		conn.Close()
	}
	return client, close, nil
}
