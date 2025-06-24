package host

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/env"
)

// LaunchAndResolvePlugin launches plugins with the provided launcher
// for the host service and resolves the plugin for the provided ID.
func LaunchAndResolvePlugin(
	pluginID string,
	launcher *plugin.Launcher,
	targetProviders map[string]provider.Provider,
	targetTransformers map[string]transform.SpecTransformer,
	envConfig *env.Config,
) (any, error) {
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(envConfig.LaunchWaitTimeoutMS)*time.Millisecond,
	)
	defer cancel()

	pluginMaps, err := launcher.Launch(ctxWithTimeout)
	if err != nil {
		return nil, err
	}

	namespace := utils.ExtractPluginNamespace(pluginID)

	// Populate the target provider and transformer maps so that the
	// registries configured with the plugin service have access to the
	// launched plugins.
	for key, provider := range pluginMaps.Providers {
		targetProviders[key] = provider
	}

	for key, transformer := range pluginMaps.Transformers {
		targetTransformers[key] = transformer
	}

	if provider, isProvider := targetProviders[namespace]; isProvider {
		return provider, nil
	}

	if transformer, isTransformer := targetTransformers[namespace]; isTransformer {
		return transformer, nil
	}

	return nil, ErrPluginNotFound
}
