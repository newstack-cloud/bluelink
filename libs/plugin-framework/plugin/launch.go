package plugin

import (
	context "context"
	"errors"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/spf13/afero"
)

const (
	// DefaultPluginLaunchAttemptLimit is the default number of times to attempt
	// launching a plugin before giving up.
	DefaultPluginLaunchAttemptLimit = 5
	// DefaultLaunchWaitTimeout is the default timeout to wait for a plugin to register
	// with the host service.
	DefaultLaunchWaitTimeout = 20 * time.Millisecond
	// DefaultCheckRegisteredInterval is the default interval to check if a plugin has
	// registered with the host service.
	DefaultCheckRegisteredInterval = 5 * time.Millisecond
)

var (
	// ErrPluginRegistrationTimeout is returned when a plugin does not register
	// with the host service within the wait timeout.
	ErrPluginRegistrationTimeout = errors.New("plugin registration timeout")
)

// PluginMaps is a set of adaptors that can be used as maps of providers
// and transformers to be used to create a blueprint loader.
type PluginMaps struct {
	Providers    map[string]provider.Provider
	Transformers map[string]transform.SpecTransformer
}

// Launcher is a service that launches plugins and waits for them to register
// with the host service.
type Launcher struct {
	pluginPath              string
	manager                 pluginservicev1.Manager
	executor                PluginExecutor
	fs                      afero.Fs
	logger                  core.Logger
	launchAttemptLimit      int
	launchWaitTimeout       time.Duration
	checkRegisteredInterval time.Duration
	transformerKeyType      TransformerKeyType
}

// LauncherOption is a function that configures a Launcher.
type LauncherOption func(*Launcher)

// WithLauncherFS is a Launcher option that sets the file system to use
// when discovering plugins.
func WithLauncherFS(fs afero.Fs) LauncherOption {
	return func(l *Launcher) {
		l.fs = fs
	}
}

// WithLauncherAttemptLimit is a Launcher option that sets the number of times
// to attempt launching a plugin before giving up.
func WithLauncherAttemptLimit(attemptLimit int) LauncherOption {
	return func(l *Launcher) {
		l.launchAttemptLimit = attemptLimit
	}
}

// WithLauncherWaitTimeout is a Launcher option that sets the timeout to wait
// for a plugin to register with the host service.
func WithLauncherWaitTimeout(timeout time.Duration) LauncherOption {
	return func(l *Launcher) {
		l.launchWaitTimeout = timeout
	}
}

// WithLauncherCheckRegisteredInterval is a Launcher option that sets the interval
// to check if a plugin has registered with the host service.
func WithLauncherCheckRegisteredInterval(interval time.Duration) LauncherOption {
	return func(l *Launcher) {
		l.checkRegisteredInterval = interval
	}
}

// WithLauncherTransformerKeyType is a Launcher option that sets the key type
// to use for transformer plugins.
func WithLauncherTransformerKeyType(keyType TransformerKeyType) LauncherOption {
	return func(l *Launcher) {
		l.transformerKeyType = keyType
	}
}

// NewLauncher creates a new Launcher.
func NewLauncher(
	pluginPath string,
	manager pluginservicev1.Manager,
	executor PluginExecutor,
	logger core.Logger,
	opts ...LauncherOption,
) *Launcher {
	launcher := &Launcher{
		pluginPath:              pluginPath,
		manager:                 manager,
		executor:                executor,
		fs:                      afero.NewOsFs(),
		logger:                  logger,
		launchAttemptLimit:      DefaultPluginLaunchAttemptLimit,
		launchWaitTimeout:       DefaultLaunchWaitTimeout,
		checkRegisteredInterval: DefaultCheckRegisteredInterval,
		transformerKeyType:      TransformerKeyTypeTransformName,
	}

	for _, opt := range opts {
		opt(launcher)
	}

	return launcher
}

// Launch discovers, executes plugin binaries
// and waits for the plugins to have registered with the host service.
// This returns a set of adaptors that can be used as maps of providers
// and transformers to be used to create a blueprint loader.
//
// The provided plugin path is expected to be a list of root directories
// separated by os.PathListSeparator (colon on Unix, semicolon on Windows).
//
// The provided context should set a deadline to avoid waiting
// indefinitely for plugins to register with the host service.
func (l *Launcher) Launch(ctx context.Context) (*PluginMaps, error) {
	l.logger.Info(
		"discovering plugins",
		core.StringLogField("pluginPath", l.pluginPath),
	)
	plugins, err := DiscoverPlugins(l.pluginPath, l.fs, l.logger)
	if err != nil {
		return nil, err
	}

	l.logger.Info(
		fmt.Sprintf("found %d plugins, launching ...", len(plugins)),
	)
	for _, plugin := range plugins {
		err := l.launchPlugin(ctx, plugin, 1 /* attemptNumber */)
		if err != nil {
			return nil, err
		}
	}

	providerPlugins := l.manager.GetPlugins(pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER)
	providerPluginMap, err := createProviderPluginAdaptors(providerPlugins)
	if err != nil {
		return nil, err
	}

	transformerPlugins := l.manager.GetPlugins(pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER)
	transformerPluginMap, err := createTransformerPluginAdaptors(
		ctx,
		transformerPlugins,
		l.transformerKeyType,
	)
	if err != nil {
		return nil, err
	}

	return &PluginMaps{
		Providers:    providerPluginMap,
		Transformers: transformerPluginMap,
	}, nil
}

func (l *Launcher) launchPlugin(
	ctx context.Context,
	plugin *PluginPathInfo,
	attemptNumber int,
) error {
	pluginLogger := l.logger.WithFields(
		core.StringLogField("plugin", plugin.ID),
		core.StringLogField("pluginPath", plugin.AbsolutePath),
		core.StringLogField("pluginType", plugin.PluginType),
		core.IntegerLogField("attemptNumber", int64(attemptNumber)),
	)
	pluginLogger.Debug(
		"launching plugin",
	)
	pluginProcess, err := l.executor.Execute(plugin.ID, plugin.AbsolutePath)
	if err != nil {
		return err
	}

	err = l.waitForPluginRegistration(
		ctx,
		plugin,
		pluginLogger,
		pluginProcess.Kill,
	)
	if err != nil {
		if errors.Is(err, ErrPluginRegistrationTimeout) {
			if attemptNumber <= l.launchAttemptLimit {
				pluginLogger.Debug(
					"timed out waiting for plugin registration",
				)
				return l.launchPlugin(ctx, plugin, attemptNumber+1)
			}
		}
		return err
	}

	return nil
}

func (l *Launcher) waitForPluginRegistration(
	ctx context.Context,
	plugin *PluginPathInfo,
	pluginLogger core.Logger,
	stop func() error,
) error {
	startTime := time.Now()
	for time.Since(startTime) < l.launchWaitTimeout {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			pluginInstance := l.manager.GetPlugin(
				pluginservicev1.PluginTypeFromString(plugin.PluginType),
				plugin.ID,
			)
			if pluginInstance != nil {
				pluginLogger.Debug(
					"plugin has been succsefully registered",
				)
				return nil
			}
			time.Sleep(l.checkRegisteredInterval)
		}
	}

	err := stop()
	if err != nil {
		return err
	}

	return ErrPluginRegistrationTimeout
}

func createProviderPluginAdaptors(
	providerPlugins []*pluginservicev1.PluginInstance,
) (map[string]provider.Provider, error) {
	providerPluginMap := make(map[string]provider.Provider)
	for _, providerPluginInstance := range providerPlugins {
		providerPlugin, ok := providerPluginInstance.Client.(provider.Provider)
		if !ok {
			return nil, fmt.Errorf(
				"plugin %s is not an instance of provider.Provider",
				providerPluginInstance.Info.ID,
			)
		}
		providerNamespace := utils.ExtractPluginNamespace(providerPluginInstance.Info.ID)
		providerPluginMap[providerNamespace] = providerPlugin
	}
	return providerPluginMap, nil
}

func createTransformerPluginAdaptors(
	ctx context.Context,
	transformerPlugins []*pluginservicev1.PluginInstance,
	transformerKeyType TransformerKeyType,
) (map[string]transform.SpecTransformer, error) {
	transformerPluginMap := make(map[string]transform.SpecTransformer)
	for _, transformerPluginInstance := range transformerPlugins {
		transformerPlugin, ok := transformerPluginInstance.Client.(transform.SpecTransformer)
		if !ok {
			return nil, fmt.Errorf(
				"plugin %s is not an instance of transform.SpecTransformer",
				transformerPluginInstance.Info.ID,
			)
		}

		transformerKey, err := getTransformerKey(
			ctx,
			transformerPluginInstance.Info.ID,
			transformerPlugin,
			transformerKeyType,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get transformer key for plugin %s: %w",
				transformerPluginInstance.Info.ID,
				err,
			)
		}
		transformerPluginMap[transformerKey] = transformerPlugin
	}
	return transformerPluginMap, nil
}

func getTransformerKey(
	ctx context.Context,
	pluginID string,
	transformerPlugin transform.SpecTransformer,
	transformerKeyType TransformerKeyType,
) (string, error) {
	if transformerKeyType == TransformerKeyTypePluginName {
		return utils.ExtractPluginNamespace(pluginID), nil
	}

	// The string used in a blueprint in the `transform` section
	// will be used to resolve the correct transformer plugin when loading a blueprint.
	// This is useful for blueprint loading as the only reference to a transformer
	// in a blueprint is the transform name string.
	return transformerPlugin.GetTransformName(ctx)
}
