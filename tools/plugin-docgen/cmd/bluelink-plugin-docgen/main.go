package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/host"
	"github.com/spf13/afero"
)

func main() {
	var pluginID string
	flag.StringVar(&pluginID, "plugin", "", "The ID of the plugin to generate documentation for.")
	flag.Parse()

	if pluginID == "" {
		log.Fatalf(
			"plugin ID is required to generate documentation, " +
				"please specify a value using the -plugin flag",
		)
	}

	envConfig, err := env.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load environment config: %v", err)
	}

	fs := afero.NewOsFs()

	// Create an empty set of providers and transformers to be populated after launching.
	// We need to instantiate the maps so they can be used to create the services
	// required by the plugin service.
	providers := map[string]provider.Provider{}
	transformers := map[string]transform.SpecTransformer{}

	hostContainer, err := host.Setup(
		providers,
		transformers,
		plugin.NewOSCmdExecutor(
			envConfig.PluginLogFileRootDir,
			/* env */ map[string]string{},
		),
		plugin.CreatePluginInstance,
		&envConfig,
		fs,
		/* listener */ nil,
	)
	if err != nil {
		log.Fatalf("Failed to setup host: %v", err)
	}
	defer hostContainer.CloseHostServer()

	pluginInstance, err := host.LaunchAndResolvePlugin(
		pluginID,
		hostContainer.Launcher,
		providers,
		transformers,
		&envConfig,
	)
	if err != nil {
		log.Fatalf("Failed to launch and or resolve plugin: %v", err)
	}

	pluginDocs, err := docgen.GeneratePluginDocs(
		pluginID,
		pluginInstance,
		hostContainer.Manager,
		&envConfig,
	)
	if err != nil {
		log.Fatalf("Failed to generate plugin documentation: %v", err)
	}

	serialised, err := json.MarshalIndent(pluginDocs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialise plugin documentation: %v", err)
	}

	err = os.WriteFile(
		"docs.json",
		serialised,
		0644,
	)
	if err != nil {
		log.Fatalf("Failed to write plugin documentation to file: %v", err)
	}
}
