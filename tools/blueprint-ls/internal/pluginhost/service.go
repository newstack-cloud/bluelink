package pluginhost

import (
	"context"
	"maps"
	"net"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"github.com/spf13/afero"
)

// Service provides an interface for a plugin host in the language server.
type Service interface {
	// LoadPlugins loads plugins and returns a map of plugin names to their
	// implementations that can be used with the blueprint framework.
	LoadPlugins(ctx context.Context) (*plugin.PluginMaps, error)
	// Manager returns the underlying manager for the plugin host service.
	Manager() pluginservicev1.Manager
	// Close the plugin host service and cleans up resources used by the plugin host.
	Close()
}

// LoadDependencies holds the required dependencies to set up the plugin host.
type LoadDependencies struct {
	Executor         plugin.PluginExecutor
	InstanceFactory  pluginservicev1.PluginFactory
	PluginHostConfig Config
}

type serviceImpl struct {
	executor              plugin.PluginExecutor
	instanceFactory       pluginservicev1.PluginFactory
	launcher              *plugin.Launcher
	manager               pluginservicev1.Manager
	providers             map[string]provider.Provider
	transformers          map[string]transform.SpecTransformer
	fs                    afero.Fs
	logger                core.Logger
	pluginServiceListener net.Listener
	idGenerator           core.IDGenerator
	config                Config
	closePluginService    func()
}

// ServiceOption is a function that configures the plugin host service.
type ServiceOption func(*serviceImpl)

// WithServiceFS sets the file system to be used by the plugin host service.
func WithServiceFS(fs afero.Fs) ServiceOption {
	return func(s *serviceImpl) {
		s.fs = fs
	}
}

// WithServiceLogger sets the logger to be used by the plugin host service.
func WithServiceLogger(logger core.Logger) ServiceOption {
	return func(s *serviceImpl) {
		s.logger = logger
	}
}

// WithPluginServiceListener sets the network listener for the gRPC plugin service.
func WithPluginServiceListener(listener net.Listener) ServiceOption {
	return func(s *serviceImpl) {
		s.pluginServiceListener = listener
	}
}

// WithIDGenerator sets the ID generator for the host.
func WithIDGenerator(idGenerator core.IDGenerator) ServiceOption {
	return func(s *serviceImpl) {
		s.idGenerator = idGenerator
	}
}

// WithInitialProviders sets initial providers available to plugins via the function registry.
func WithInitialProviders(providers map[string]provider.Provider) ServiceOption {
	return func(s *serviceImpl) {
		maps.Copy(s.providers, providers)
	}
}

// LoadDefaultService creates a new plugin host service using the gRPC plugin framework.
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

	err := service.initialise()
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *serviceImpl) Manager() pluginservicev1.Manager {
	return s.manager
}

func (s *serviceImpl) initialise() error {
	hostID, err := s.idGenerator.GenerateID()
	if err != nil {
		return err
	}

	s.manager = pluginservicev1.NewManager(
		map[pluginservicev1.PluginType]string{
			pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:    providerserverv1.ProtocolVersion,
			pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER: transformerserverv1.ProtocolVersion,
		},
		s.instanceFactory,
		hostID,
	)

	s.launcher = plugin.NewLauncher(
		s.config.GetPluginPath(),
		s.manager,
		s.executor,
		s.logger,
		plugin.WithLauncherWaitTimeout(
			time.Duration(s.config.GetLaunchWaitTimeoutMS())*time.Millisecond,
		),
		plugin.WithLauncherFS(s.fs),
	)

	functionRegistry := provider.NewFunctionRegistry(s.providers)
	// The language server doesn't need resource deployment capabilities,
	// but the plugin service requires a resource registry for plugin-to-plugin calls.
	resourceRegistry := resourcehelpers.NewRegistry(
		s.providers,
		s.transformers,
		time.Second, // Not used by LS
		nil,         // No state container needed
		nil,         // No params needed
	)

	pluginService := pluginservicev1.NewServiceServer(
		s.manager,
		functionRegistry,
		resourceRegistry,
		hostID,
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

	// Populate internal maps so the function registry can resolve plugin calls.
	maps.Copy(s.providers, pluginMaps.Providers)
	maps.Copy(s.transformers, pluginMaps.Transformers)

	return &plugin.PluginMaps{
		Providers:    s.providers,
		Transformers: s.transformers,
	}, nil
}

func (s *serviceImpl) Close() {
	if s.closePluginService != nil {
		s.closePluginService()
	}
}
