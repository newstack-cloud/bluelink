// A minimal provider plugin used to exercise the deploy engine locally.
// It provides custom variable types with a fixed set of options along with a
// resource type that does not talk to an upstream system, which is enough to
// validate blueprints against a running deploy engine.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

// Set at build time, see scripts/build-test-plugins.sh.
var version = "0.0.0-dev"

func main() {
	serviceClient, closeService, err := pluginservicev1.NewEnvServiceClient()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer closeService()

	hostInfoContainer := pluginutils.NewHostInfoContainer()
	providerServer := providerv1.NewProviderPlugin(
		&providerv1.ProviderPluginDefinition{
			ProviderNamespace: "example",
			Resources: map[string]provider.Resource{
				"example/service": resourceService(),
			},
			CustomVariableTypes: map[string]provider.CustomVariableType{
				"example/region":       customVarTypeRegion(),
				"example/instanceSize": customVarTypeInstanceSize(),
				"example/environment":  customVarTypeEnvironment(),
			},
		},
		hostInfoContainer,
		serviceClient,
	)

	config := plugin.ServePluginConfiguration{
		ID: "newstack-cloud/example",
		PluginMetadata: &pluginservicev1.PluginMetadata{
			PluginVersion:        version,
			DisplayName:          "Example",
			FormattedDescription: "An example provider used for local testing.",
			Author:               "NewStack Cloud Limited",
		},
		ProtocolVersion: "1.0",
	}

	fmt.Println("Starting Bluelink Example Provider Plugin Server...")
	close, err := plugin.ServeProviderV1(
		context.Background(),
		providerServer,
		serviceClient,
		hostInfoContainer,
		config,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	pluginutils.WaitForShutdown(close)
}
