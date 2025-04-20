package pluginhostv1

import (
	"context"
	"net"
	"time"

	"github.com/spf13/afero"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/resourcehelpers"
	"github.com/two-hundred/celerity/libs/blueprint/transform"
	"github.com/two-hundred/celerity/libs/plugin-framework/plugin"
	"github.com/two-hundred/celerity/libs/plugin-framework/pluginservicev1"
	"github.com/two-hundred/celerity/libs/plugin-framework/providerserverv1"
	"github.com/two-hundred/celerity/libs/plugin-framework/transformerserverv1"
)

// Service provides an interface for a v1 plugin host.
type Service interface {
	// LoadPlugins loads plugins and returns a map of plugin names to their implementations
	// that can be used with the blueprint framework.
	LoadPlugins(ctx context.Context) (*plugin.PluginMaps, error)
	// Close the plugin host service and cleans up resources
	// used by the plugin host.
	// This will usually close the server backing the plugin host instance.
	Close()
}

// LoadDependencies is a struct that holds the required dependencies
// to set up the plugin host and load plugins.
// You can use options to override defaults for other dependencies.
type LoadDependencies struct {
	Executor         plugin.PluginExecutor
	InstanceFactory  pluginservicev1.PluginFactory
	PluginHostConfig Config
}

type serviceImpl struct {
	executor              plugin.PluginExecutor
	instanceFactory       pluginservicev1.PluginFactory
	launcher              *plugin.Launcher
	providers             map[string]provider.Provider
	transformers          map[string]transform.SpecTransformer
	fs                    afero.Fs
	logger                core.Logger
	pluginServiceListener net.Listener
	idGenerator           core.IDGenerator
	config                Config
	closePluginService    func()
}

// ServiceOption is a function that configures the plugin host service
// with optional dependencies.
type ServiceOption func(*serviceImpl)

// WithServiceFS sets the file system to be used by the plugin host service.
// The default is to use the current OS file system.
func WithServiceFS(fs afero.Fs) ServiceOption {
	return func(s *serviceImpl) {
		s.fs = fs
	}
}

// WithServiceLogger sets the logger to be used by the plugin host service.
// The default is to use a no-op logger.
// This should always be provided in production environments.
func WithServiceLogger(logger core.Logger) ServiceOption {
	return func(s *serviceImpl) {
		s.logger = logger
	}
}

// WithPluginServiceListener sets the network listener to be used to run the
// gRPC plugin service that manages plugins.
// The default is to use a new listener on the default plugin service port.
func WithPluginServiceListener(listener net.Listener) ServiceOption {
	return func(s *serviceImpl) {
		s.pluginServiceListener = listener
	}
}

// WithIDGenerator sets the ID generator to generate the ID for the host.
// The default is to use a UUID generator.
func WithIDGenerator(idGenerator core.IDGenerator) ServiceOption {
	return func(s *serviceImpl) {
		s.idGenerator = idGenerator
	}
}

// WithInitialProviders sets the initial providers to be used by the plugin host service.
// This is particularly useful for ensuring the provider for core blueprint specification
// functions is available to plugins via the function registry.
func WithInitialProviders(providers map[string]provider.Provider) ServiceOption {
	return func(s *serviceImpl) {
		// Make a copy of the initial providers map to avoid
		// modifying the original map.
		s.providers = make(map[string]provider.Provider)
		for namespace, provider := range providers {
			s.providers[namespace] = provider
		}
	}
}

// LoadDefaultService creates a new instance of the
// default implementation of a plugin host service
// that uses the gRPC plugin framework.
// This will set up the plugin host in preparation for loading plugins.
func LoadDefaultService(
	dependencies *LoadDependencies,
	opts ...ServiceOption,
) (Service, error) {
	service := &serviceImpl{
		executor:        dependencies.Executor,
		instanceFactory: dependencies.InstanceFactory,
		providers:       make(map[string]provider.Provider),
		transformers:    make(map[string]transform.SpecTransformer),
		fs:              afero.NewOsFs(),
		logger:          core.NewNopLogger(),
		idGenerator:     core.NewUUIDGenerator(),
		config:          dependencies.PluginHostConfig,
	}

	for _, opt := range opts {
		opt(service)
	}

	err := service.Initialise()
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *serviceImpl) Initialise() error {
	hostID, err := s.idGenerator.GenerateID()
	if err != nil {
		return err
	}

	manager := pluginservicev1.NewManager(
		map[pluginservicev1.PluginType]string{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    providerserverv1.ProtocolVersion,
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: transformerserverv1.ProtocolVersion,
		},
		s.instanceFactory,
		hostID,
	)

	s.launcher = plugin.NewLauncher(
		s.config.GetPluginPath(),
		manager,
		s.executor,
		s.logger,
		plugin.WithLauncherWaitTimeout(
			time.Duration(s.config.GetLaunchWaitTimeoutMS())*time.Millisecond,
		),
		plugin.WithLauncherFS(s.fs),
	)

	functionRegistry := provider.NewFunctionRegistry(s.providers)
	stabilisationPollingIntervalMS := s.config.GetResourceStabilisationPollingIntervalMS()
	resourceDeployService := resourcehelpers.NewRegistry(
		s.providers,
		s.transformers,
		time.Duration(stabilisationPollingIntervalMS)*time.Millisecond,
		// At this point, there aren't any params to pass to the registry.
		// However, on every request to the deploy engine, the request-specific
		// params need to be attached to a copy of the registry using
		// the `WithParams` method.
		/* params */
		nil,
	)

	pluginService := pluginservicev1.NewServiceServer(
		manager,
		functionRegistry,
		resourceDeployService,
		hostID,
		pluginservicev1.WithPluginToPluginCallTimeout(
			s.config.GetPluginToPluginCallTimeoutMS(),
		),
		pluginservicev1.WithResourceStabilisationTimeout(
			s.config.GetResourceStabilisationPollingTimeoutMS(),
		),
	)

	pluginServiceOpts := []pluginservicev1.ServerOption{}
	if s.pluginServiceListener != nil {
		pluginServiceOpts = append(
			pluginServiceOpts,
			pluginservicev1.WithListener(s.pluginServiceListener),
		)
	}

	pluginServiceServer := pluginservicev1.NewServer(
		pluginService,
		pluginServiceOpts...,
	)
	close, err := pluginServiceServer.Serve()
	s.closePluginService = close
	return err
}

func (s *serviceImpl) LoadPlugins(ctx context.Context) (*plugin.PluginMaps, error) {
	ctxWithTimeout, cancel := context.WithTimeout(
		ctx,
		time.Duration(s.config.GetTotalLaunchWaitTimeoutMS())*time.Millisecond,
	)
	defer cancel()

	pluginMaps, err := s.launcher.Launch(ctxWithTimeout)
	if err != nil {
		return nil, err
	}

	// Ensure the internal providers and transformers have been populated
	// with the plugin maps.
	// This will allow the internal function registry and resource deploy service
	// to resolve the plugins to make calls to.
	for providerNamespace, provider := range pluginMaps.Providers {
		s.providers[providerNamespace] = provider
	}
	for transformerNamespace, transformer := range pluginMaps.Transformers {
		s.transformers[transformerNamespace] = transformer
	}

	return pluginMaps, nil
}

func (s *serviceImpl) Close() {
	if s.closePluginService != nil {
		s.closePluginService()
	}
}
