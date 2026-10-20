package enginev1

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginhostv1"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/spf13/afero"
)

// What the plugin host needs to launch plugins and serve them.
type pluginHostParams struct {
	config                *core.Config
	stateContainer        state.Container
	pluginServiceListener net.Listener
	fileSystem            afero.Fs
	idGenerator           bpcore.IDGenerator
	clock                 bpcore.Clock
	logger                bpcore.Logger
}

// Starts the plugin host service and launches the configured plugins.
//
// Returns the service, which has to be closed at shutdown to stop the plugins it
// launched, and the providers and transformers they registered.
func startPluginHost(
	params *pluginHostParams,
) (pluginhostv1.Service, *plugin.PluginMaps, error) {
	warnAboutPluginServiceTransport(params.pluginServiceListener, params.logger)

	pluginExecutor := plugin.NewOSCmdExecutor(
		params.config.PluginsV1.LogFileRootDir,
		getPluginExecutorEnvVars(params.pluginServiceListener),
	)

	service, err := pluginhostv1.LoadDefaultService(
		&pluginhostv1.LoadDependencies{
			Executor:         pluginExecutor,
			InstanceFactory:  plugin.CreatePluginInstance,
			PluginHostConfig: params.config,
			StateContainer:   params.stateContainer,
		},
		pluginhostv1.WithServiceLogger(params.logger),
		pluginhostv1.WithServiceFS(params.fileSystem),
		pluginhostv1.WithIDGenerator(params.idGenerator),
		pluginhostv1.WithInitialProviders(initialProviders(params)),
		pluginhostv1.WithPluginServiceListener(params.pluginServiceListener),
	)
	if err != nil {
		return nil, nil, err
	}

	pluginMaps, err := service.LoadPlugins(context.Background())
	if err != nil {
		return nil, nil, err
	}

	return service, pluginMaps, nil
}

// The providers the host serves before any plugin has registered one, which is the core
// provider alone. It is built into the engine rather than launched as a plugin.
func initialProviders(params *pluginHostParams) map[string]provider.Provider {
	return map[string]provider.Provider{
		"core": providerhelpers.NewCoreProvider(
			params.stateContainer.Links(),
			bpcore.BlueprintInstanceIDFromContext,
			os.Getwd,
			provider.NewFileSourceRegistry(),
			params.clock,
		),
	}
}

// Warns that the plugin service's transport gives no separation between local users.
//
// A caller supplying its own unix-socket listener, which the tests and an embedding host can
// do, is not warned.
func warnAboutPluginServiceTransport(
	pluginServiceListener net.Listener,
	logger bpcore.Logger,
) {
	if pluginServiceListener != nil {
		if _, isTCP := pluginServiceListener.Addr().(*net.TCPAddr); !isTCP {
			return
		}
	}

	logger.Warn(
		"the plugin service is listening on loopback TCP, which any local process can " +
			"connect to regardless of the user it runs as, so a host shared with untrusted " +
			"local workloads exposes the channel carrying provider credentials",
	)
}

func getPluginExecutorEnvVars(
	pluginServiceListener net.Listener,
) map[string]string {
	envVars := map[string]string{}

	// Ensure that when a custom listener is provided for the plugin service,
	// that the port is set in the environment variables for each plugin
	// that is launched.
	// As long as plugins use the `pluginservicev1.NewEnvServiceClient` function
	// to create a client to interact with the plugin service, they will
	// use the custom port.
	if pluginServiceListener != nil {
		tcpAddr, isTCPAddr := pluginServiceListener.Addr().(*net.TCPAddr)
		if isTCPAddr {
			envVars["BLUELINK_BUILD_ENGINE_PLUGIN_SERVICE_PORT"] = fmt.Sprintf("%d", tcpAddr.Port)
		}
	}

	return envVars
}
