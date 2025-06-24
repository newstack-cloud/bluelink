package engine

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/config"
	deployengine "github.com/newstack-cloud/bluelink/libs/deploy-engine-client"
	"go.uber.org/zap"
)

// Create a new deploy engine client based on how the CLI is configured.
func Create(confProvider *config.Provider, logger *zap.Logger) (DeployEngine, error) {
	return deployengine.NewClient(
		deployengine.WithClientAuthMethod(deployengine.AuthMethodAPIKey),
		deployengine.WithClientEndpoint("http://localhost:8325"),
		deployengine.WithClientConnectProtocol(deployengine.ConnectProtocolTCP),
		deployengine.WithClientAPIKey("test-api-key"),
	)
}
