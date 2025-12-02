package host

import (
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/utils"
	"github.com/spf13/afero"
	"go.uber.org/zap/zapcore"
)

type Container struct {
	Launcher        *plugin.Launcher
	Manager         pluginservicev1.Manager
	CloseHostServer func()
	Logger          core.Logger
}

// Setup initialises the plugin service (host) and a launcher that can be used
// to launch plugins.
func Setup(
	targetProviders map[string]provider.Provider,
	targetTransformers map[string]transform.SpecTransformer,
	executor plugin.PluginExecutor,
	instanceFactory pluginservicev1.PluginFactory,
	envConfig *env.Config,
	fs afero.Fs,
	listener net.Listener,
) (*Container, error) {
	zapLogger, err := env.CreateLogger(
		zapcore.Lock(os.Stdout),
		zapcore.Lock(os.Stderr),
		envConfig,
	)
	if err != nil {
		return nil, err
	}

	hostID := uuid.New().String()
	manager := pluginservicev1.NewManager(
		map[pluginservicev1.PluginType]string{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    providerserverv1.ProtocolVersion,
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: transformerserverv1.ProtocolVersion,
		},
		instanceFactory,
		hostID,
	)

	logger := core.NewLoggerFromZap(
		zapLogger,
	)
	launcher := plugin.NewLauncher(
		envConfig.PluginPath,
		manager,
		executor,
		logger,
		plugin.WithLauncherWaitTimeout(
			time.Duration(envConfig.LaunchWaitTimeoutMS)*time.Millisecond,
		),
		plugin.WithLauncherTransformerKeyType(
			plugin.TransformerKeyTypePluginName,
		),
		plugin.WithLauncherFS(fs),
	)

	// Create an empty set of providers and transformers to be populated after launching.
	// We need to instantiate the maps so they can be used to create the services
	// required by the plugin service.
	providers := map[string]provider.Provider{}
	transformers := map[string]transform.SpecTransformer{}
	functionRegistry := provider.NewFunctionRegistry(providers)
	resourceDeployService := resourcehelpers.NewRegistry(
		providers,
		transformers,
		// This shouldn't be used by the plugin doc generator.
		/* stabilisationPollingTimeout */
		30*time.Second,
		/* stateContainer */ nil,
		utils.CreateBlueprintParams(envConfig),
	)
	pluginService := pluginservicev1.NewServiceServer(
		manager,
		functionRegistry,
		resourceDeployService,
		hostID,
		pluginservicev1.WithPluginToPluginCallTimeout(30000),
	)

	pluginServiceOpts := []pluginservicev1.ServerOption{}
	if listener != nil {
		pluginServiceOpts = append(pluginServiceOpts, pluginservicev1.WithListener(listener))
	}

	pluginServiceServer := pluginservicev1.NewServer(
		pluginService,
		pluginServiceOpts...,
	)
	close, err := pluginServiceServer.Serve()
	if err != nil {
		return nil, err
	}

	return &Container{
		Launcher:        launcher,
		Manager:         manager,
		CloseHostServer: close,
		Logger:          logger,
	}, nil
}
