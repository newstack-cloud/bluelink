package enginev1

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginmeta"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tagging"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/azure"
	resolverfs "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/gcs"
	resolverhttps "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/https"
	resolverrouter "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/router"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/s3"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	pluginutils "github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"github.com/spf13/afero"
)

// What the controllers are assembled from.
type dependencyParams struct {
	config *core.Config
	state  *stateServices
	// The providers and transformers the launched plugins registered.
	plugins *plugin.PluginMaps
	// The plugin manager, which is how plugin config definitions and provider metadata
	// are read.
	pluginManager pluginservicev1.Manager
	// Filled as plugin config definitions are read, so that the logger can redact the
	// values of keys a plugin declared secret.
	secretConfigKeys *pluginconfig.SecretKeyRegistry
	fileSystem       afero.Fs
	idGenerator      bpcore.IDGenerator
	clock            bpcore.Clock
	logger           bpcore.Logger
}

// Assembles everything the API's controllers are handed.
//
// One set shared by every controller, rather than each building its own, so that two
// endpoints acting on the same state act on it through the same loader, resolver and
// clock.
func buildDependencies(deps *dependencyParams) *typesv1.Dependencies {
	childResolver := createChildResolver(deps)
	validationLoaderFactory := createValidationLoaderFactory(deps)

	return &typesv1.Dependencies{
		EventStore:                 deps.state.events,
		ValidationStore:            deps.state.validation,
		ChangesetStore:             deps.state.changesets,
		ReconciliationResultsStore: deps.state.reconciliationResults,
		CleanupOperationsStore:     deps.state.cleanupOperations,
		AuditStore:                 deps.state.audit,
		CleanupRunner: cleanup.NewRunner(
			deps.state.cleanupOperations,
			deps.idGenerator,
			deps.clock,
			deps.logger,
		),
		Instances:               deps.state.container.Instances(),
		Exports:                 deps.state.container.Exports(),
		IDGenerator:             deps.idGenerator,
		EventIDGenerator:        utils.NewUUIDv7Generator(),
		ValidationLoader:        validationLoaderFactory(),
		ValidationLoaderFactory: validationLoaderFactory,
		DeploymentLoader:        createDeploymentLoader(deps, childResolver),
		BlueprintResolver:       childResolver,
		ParamsProvider: params.NewDefaultProvider(
			params.DefaultContextVars(deps.config),
		),
		PluginConfigPreparer:   createPluginConfigPreparer(deps),
		TaggingConfigProvider:  tagging.NewConfigProvider(deps.config.Version),
		ProviderMetadataLookup: pluginmeta.NewLookup(deps.pluginManager),
		Clock:                  deps.clock,
		Logger:                 deps.logger,
	}
}

// Routes a child blueprint's source to whatever can fetch it.
func createChildResolver(deps *dependencyParams) includes.ChildResolver {
	resolvers := &deps.config.Resolvers

	return resolverrouter.NewResolver(
		resolverfs.NewResolver(deps.fileSystem),
		resolverrouter.WithRoute(
			resolve.S3SourceType,
			s3.NewResolver(resolvers.S3Endpoint, resolvers.S3UsePathStyle),
		),
		resolverrouter.WithRoute(
			resolve.AzureBlobStorageSourceType,
			// Azure blob storage clients will be created on the fly
			// using default credentials and sourcing the storage account
			// name from include metadata, blueprint params or
			// environment variables.
			azure.NewResolver( /* clientFactory */ nil),
		),
		resolverrouter.WithRoute(
			resolve.GoogleCloudStorageSourceType,
			gcs.NewResolver(resolvers.GCSEndpoint),
		),
		resolverrouter.WithRoute(
			resolve.HTTPSSourceType,
			resolverhttps.NewResolver(
				httputils.NewHTTPClient(resolvers.HTTPSClientTimeout),
			),
		),
	)
}

// A factory rather than a loader, because an endpoint that validates has per-request
// flags to honour and must not mutate a loader shared with every other request.
//
// No state container and no child resolver as validation reads neither, and a loader
// given them would reach for them on a blueprint that happens to reference one.
func createValidationLoaderFactory(deps *dependencyParams) typesv1.ValidationLoaderFactory {
	return func(extraOpts ...container.LoaderOption) container.Loader {
		baseOpts := []container.LoaderOption{
			container.WithLoaderTransformSpec(false),
			container.WithLoaderValidateAfterTransform(false),
			container.WithLoaderValidateRuntimeValues(false),
			container.WithLoaderLogger(deps.logger),
		}

		return container.NewDefaultLoader(
			deps.plugins.Providers,
			deps.plugins.Transformers,
			/* stateContainer */ nil,
			/* childResolver */ nil,
			append(baseOpts, extraOpts...)...,
		)
	}
}

func createDeploymentLoader(
	deps *dependencyParams,
	childResolver includes.ChildResolver,
) container.Loader {
	return container.NewDefaultLoader(
		deps.plugins.Providers,
		deps.plugins.Transformers,
		deps.state.container,
		childResolver,
		container.WithLoaderTransformSpec(true),
		// Sometimes users of the deploy engine will want to debug issues
		// caused by a transformer plugin that may be producing invalid output.
		container.WithLoaderValidateAfterTransform(
			deps.config.Blueprints.ValidateAfterTransform,
		),
		container.WithLoaderValidateRuntimeValues(true),
		container.WithLoaderIDGenerator(deps.idGenerator),
		container.WithLoaderDefaultRetryPolicy(
			parseDefaultRetryPolicy(
				deps.config.Blueprints.DefaultRetryPolicy,
				deps.logger.Named("init"),
			),
		),
		container.WithLoaderResourceStabilityPollingConfig(
			createResourceStabilityPollingConfig(deps.config),
		),
		container.WithLoaderLogger(deps.logger),
	)
}

func createPluginConfigPreparer(deps *dependencyParams) pluginconfig.Preparer {
	return pluginconfig.NewDefaultPreparer(
		pluginconfig.ToConfigDefinitionProviders(deps.plugins.Providers),
		pluginconfig.ToConfigDefinitionProviders(
			transformerConfigProvidersByPluginName(deps.pluginManager),
		),
		deps.pluginManager,
		deps.secretConfigKeys,
	)
}

// Returns the installed transformer
// plugins keyed by their simplified plugin name (namespace), e.g. the plugin
// "newstack-cloud/celerity" is keyed as "celerity". This matches how
// transformer config is keyed in a blueprint operation config, and is distinct
// from the transform-name keying (e.g. "celerity-2025-08-01") used by the
// blueprint loader.
func transformerConfigProvidersByPluginName(
	manager pluginservicev1.Manager,
) map[string]transform.SpecTransformer {
	transformers := map[string]transform.SpecTransformer{}
	instances := manager.GetPlugins(pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER)
	for _, instance := range instances {
		transformer, ok := instance.Client.(transform.SpecTransformer)
		if !ok {
			continue
		}
		transformers[pluginutils.ExtractPluginNamespace(instance.Info.ID)] = transformer
	}
	return transformers
}

func createResourceStabilityPollingConfig(
	config *core.Config,
) *container.ResourceStabilityPollingConfig {
	return &container.ResourceStabilityPollingConfig{
		PollingInterval: time.Duration(
			config.Blueprints.ResourceStabilisationPollingIntervalMS,
		) * time.Millisecond,
		PollingTimeout: time.Duration(
			config.PluginsV1.ResourceStabilisationPollingTimeoutMS,
		) * time.Millisecond,
	}
}

func parseDefaultRetryPolicy(
	serialised string,
	logger bpcore.Logger,
) *provider.RetryPolicy {
	if strings.TrimSpace(serialised) == "" {
		return provider.DefaultRetryPolicy
	}

	retryPolicy := &provider.RetryPolicy{}
	err := json.Unmarshal([]byte(serialised), retryPolicy)
	if err != nil {
		logger.Warn(
			"failed to parse default retry policy from config, "+
				"using default policy from the provider package",
			bpcore.ErrorLogField("error", err),
		)
		return provider.DefaultRetryPolicy
	}

	return retryPolicy
}
