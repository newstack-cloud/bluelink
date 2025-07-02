package enginev1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/deploymentsv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/eventsv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/validationv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginhostv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/azure"
	resolverfs "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/gcs"
	resolverhttps "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/https"
	resolverrouter "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/router"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/s3"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/spf13/afero"
)

func Setup(
	router *mux.Router,
	config *core.Config,
	pluginServiceListener net.Listener,
) (io.WriteCloser, func(), error) {
	logger, err := core.CreateLogger(config)
	if err != nil {
		return nil, nil, err
	}

	fileSystem := afero.NewOsFs()
	idGenerator := bpcore.NewUUIDGenerator()

	stateServices, closeStateService, err := loadStateServices(
		context.Background(),
		fileSystem,
		logger,
		&config.State,
	)
	if err != nil {
		return nil, nil, err
	}

	clock := &bpcore.SystemClock{}
	initialProviders := map[string]provider.Provider{
		"core": providerhelpers.NewCoreProvider(
			stateServices.container.Links(),
			bpcore.BlueprintInstanceIDFromContext,
			os.Getwd,
			clock,
		),
	}
	pluginExectorEnvVars := getPluginExecutorEnvVars(
		pluginServiceListener,
	)
	pluginExecutor := plugin.NewOSCmdExecutor(
		config.PluginsV1.LogFileRootDir,
		pluginExectorEnvVars,
	)
	pluginHostService, err := pluginhostv1.LoadDefaultService(
		&pluginhostv1.LoadDependencies{
			Executor:         pluginExecutor,
			InstanceFactory:  plugin.CreatePluginInstance,
			PluginHostConfig: config,
			StateContainer:   stateServices.container,
		},
		pluginhostv1.WithServiceLogger(logger),
		pluginhostv1.WithServiceFS(fileSystem),
		pluginhostv1.WithIDGenerator(idGenerator),
		pluginhostv1.WithInitialProviders(initialProviders),
		pluginhostv1.WithPluginServiceListener(pluginServiceListener),
	)
	if err != nil {
		return nil, nil, err
	}

	pluginMaps, err := pluginHostService.LoadPlugins(
		context.Background(),
	)
	if err != nil {
		return nil, nil, err
	}

	fsResolver := resolverfs.NewResolver(fileSystem)
	s3Resolver := s3.NewResolver(config.Resolvers.S3Endpoint, false)
	gcsResolver := gcs.NewResolver(config.Resolvers.GCSEndpoint)
	httpClient := httputils.NewHTTPClient(config.Resolvers.HTTPSClientTimeout)
	httpsResolver := resolverhttps.NewResolver(
		httpClient,
	)
	// Azure blob storage clients will be created on the fly
	// using default credentials and sourcing the storage account
	// name from include metadata, blueprint params or
	// environment variables.
	azureObjectResolver := azure.NewResolver( /* clientFactory */ nil)
	childResolver := resolverrouter.NewResolver(
		fsResolver,
		resolverrouter.WithRoute(resolve.S3SourceType, s3Resolver),
		resolverrouter.WithRoute(resolve.AzureBlobStorageSourceType, azureObjectResolver),
		resolverrouter.WithRoute(resolve.GoogleCloudStorageSourceType, gcsResolver),
		resolverrouter.WithRoute(resolve.HTTPSSourceType, httpsResolver),
	)

	pluginConfigPreparer := pluginconfig.NewDefaultPreparer(
		pluginconfig.ToConfigDefinitionProviders(pluginMaps.Providers),
		pluginconfig.ToConfigDefinitionProviders(pluginMaps.Transformers),
	)

	validateLoader := container.NewDefaultLoader(
		pluginMaps.Providers,
		pluginMaps.Transformers,
		/* stateContainer */ nil,
		/* childResolver */ nil,
		container.WithLoaderTransformSpec(false),
		container.WithLoaderValidateAfterTransform(false),
		container.WithLoaderValidateRuntimeValues(false),
		container.WithLoaderLogger(logger),
	)

	defaultRetryPolicy := parseDefaultRetryPolicy(
		config.Blueprints.DefaultRetryPolicy,
		logger.Named("init"),
	)

	deployLoader := container.NewDefaultLoader(
		pluginMaps.Providers,
		pluginMaps.Transformers,
		stateServices.container,
		childResolver,
		container.WithLoaderTransformSpec(true),
		// Sometimes users of the deploy engine will want to debug issues
		// caused by a transformer plugin that may be producing invalid output.
		container.WithLoaderValidateAfterTransform(config.Blueprints.ValidateAfterTransform),
		container.WithLoaderValidateRuntimeValues(true),
		container.WithLoaderDriftCheckEnabled(config.Blueprints.EnableDriftCheck),
		container.WithLoaderIDGenerator(idGenerator),
		container.WithLoaderDefaultRetryPolicy(defaultRetryPolicy),
		container.WithLoaderResourceStabilityPollingConfig(
			createResourceStabilityPollingConfig(config),
		),
		container.WithLoaderLogger(logger),
	)

	paramsProvider := params.NewDefaultProvider(
		params.DefaultContextVars(config),
	)

	dependencies := &typesv1.Dependencies{
		EventStore:           stateServices.events,
		ValidationStore:      stateServices.validation,
		ChangesetStore:       stateServices.changesets,
		Instances:            stateServices.container.Instances(),
		IDGenerator:          idGenerator,
		EventIDGenerator:     utils.NewUUIDv7Generator(),
		ValidationLoader:     validateLoader,
		DeploymentLoader:     deployLoader,
		BlueprintResolver:    childResolver,
		ParamsProvider:       paramsProvider,
		PluginConfigPreparer: pluginConfigPreparer,
		Clock:                clock,
		Logger:               logger,
	}

	healthHandler := setupHealthHandler(
		router,
	)

	helpersv1.SetupRequestBodyValidator()

	setupValidationHandlers(
		router,
		dependencies,
		config,
	)

	setupDeploymentHandlers(
		router,
		dependencies,
		config,
	)

	setupEventManagementHandlers(
		router,
		dependencies,
		config,
	)

	authMiddleware, err := setupAuth(
		&config.Auth,
		clock,
		/* excludedRoutes */ []*mux.Route{healthHandler},
	)
	if err != nil {
		return nil, nil, err
	}

	router.Use(authMiddleware.Middleware)

	return nil, createServerCleanupFunc(
		closeStateService,
		pluginHostService.Close,
	), nil
}

func createServerCleanupFunc(
	cleanupFuncs ...func(),
) func() {
	return func() {
		for _, cleanup := range cleanupFuncs {
			if cleanup != nil {
				cleanup()
			}
		}
	}
}

func setupHealthHandler(
	router *mux.Router,
) *mux.Route {
	return router.HandleFunc("/health", HealthHandler).Methods("GET")
}

func setupValidationHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.BlueprintValidationRetentionPeriod,
	) * time.Second

	validationCtrl := validationv1.NewController(
		retentionPeriod,
		dependencies,
	)

	router.HandleFunc(
		"/validations",
		validationCtrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/validations/{id}",
		validationCtrl.GetBlueprintValidationHandler,
	).Methods("GET")

	router.HandleFunc(
		"/validations/{id}/stream",
		validationCtrl.StreamEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/validations/cleanup",
		validationCtrl.CleanupBlueprintValidationsHandler,
	).Methods("POST")
}

func setupDeploymentHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.ChangesetRetentionPeriod,
	) * time.Second

	deployTimeout := time.Duration(
		config.Blueprints.DeploymentTimeout,
	) * time.Second

	deploymentCtrl := deploymentsv1.NewController(
		retentionPeriod,
		deployTimeout,
		dependencies,
	)

	router.HandleFunc(
		"/deployments/changes",
		deploymentCtrl.CreateChangesetHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/changes/{id}/stream",
		deploymentCtrl.StreamChangesetEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/changes/{id}",
		deploymentCtrl.GetChangesetHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/changes/cleanup",
		deploymentCtrl.CleanupChangesetsHandler,
	)

	router.HandleFunc(
		"/deployments/instances",
		deploymentCtrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		deploymentCtrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}",
		deploymentCtrl.GetBlueprintInstanceHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}",
		deploymentCtrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

	router.HandleFunc(
		"/deployments/instances/{id}/exports",
		deploymentCtrl.GetBlueprintInstanceExportsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		deploymentCtrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")
}

func setupEventManagementHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.EventsRetentionPeriod,
	) * time.Second

	eventsCtrl := eventsv1.NewController(
		retentionPeriod,
		dependencies,
	)

	router.HandleFunc(
		"/events/cleanup",
		eventsCtrl.CleanupEventsHandler,
	)
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
