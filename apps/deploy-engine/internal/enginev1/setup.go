package enginev1

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/deploymentsv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

// Setup wires up the v1 API on the given router and returns the callback that shuts
// everything it started back down.
//
// The order below is the order things depend on each other in; where a step depends on
// its position rather than only on having run, that is noted.
func Setup(
	router *mux.Router,
	config *core.Config,
	pluginServiceListener net.Listener,
) (io.WriteCloser, func(), error) {
	// Created before the logger and filled later, as plugin config definitions are read,
	// the logger consults it when it writes an entry, not when it is built.
	secretConfigKeys := pluginconfig.NewSecretKeyRegistry()

	logger, err := core.CreateLogger(config, secretConfigKeys)
	if err != nil {
		return nil, nil, err
	}

	clock := &bpcore.SystemClock{}
	fileSystem := afero.NewOsFs()
	idGenerator := bpcore.NewUUIDGenerator()

	// Before the state services, which reach object storage and key management through
	// SDK clients that are traced, so that the spans their startup produces are
	// exported rather than falling to the implementation that does nothing.
	stopTracing, err := setupTracing(context.Background(), config, clock, logger)
	if err != nil {
		return nil, nil, err
	}

	// Stopped here as well as in the shutdown callback, since a later step failing
	// returns before the caller is given that callback, and the exporter holds a
	// goroutine and a connection until something stops it.
	startupSucceeded := false
	defer func() {
		if !startupSucceeded {
			stopTracing()
		}
	}()

	stateServices, closeStateService, err := loadStateServices(
		context.Background(),
		fileSystem,
		logger,
		&config.State,
	)
	if err != nil {
		return nil, nil, err
	}

	pluginHostService, pluginMaps, err := startPluginHost(&pluginHostParams{
		config:                config,
		stateContainer:        stateServices.container,
		pluginServiceListener: pluginServiceListener,
		fileSystem:            fileSystem,
		idGenerator:           idGenerator,
		clock:                 clock,
		logger:                logger,
	})
	if err != nil {
		return nil, nil, err
	}

	dependencies := buildDependencies(&dependencyParams{
		config:           config,
		state:            stateServices,
		plugins:          pluginMaps,
		pluginManager:    pluginHostService.Manager(),
		secretConfigKeys: secretConfigKeys,
		fileSystem:       fileSystem,
		idGenerator:      idGenerator,
		clock:            clock,
		logger:           logger,
	})

	handlers := setupHandlers(router, dependencies, config)

	closeAuditSink, err := setupMiddleware(&middlewareParams{
		router:          router,
		config:          config,
		auditStore:      stateServices.audit,
		unauthenticated: handlers.unauthenticated,
		clock:           clock,
		logger:          logger,
	})
	if err != nil {
		return nil, nil, err
	}

	// Shares the runner the cleanup endpoints use, so a scheduled pass and a requested
	// one cannot run over the same category at once.
	stopScheduledCleanup := setupScheduledCleanup(
		config,
		dependencies.CleanupRunner,
		stateServices,
		clock,
		logger,
	)

	startupSucceeded = true

	return nil, createServerCleanupFunc(&serverShutdown{
		drainDeployments: drainInFlightDeploymentsFunc(
			handlers.deployments,
			config.GetShutdownDrainTimeout(),
		),
		stopScheduledCleanup: stopScheduledCleanup,
		closeState:           closeStateService,
		closePluginHost:      pluginHostService.Close,
		closeAuditSink:       closeAuditSink,
		stopTracing:          stopTracing,
	}), nil
}

func drainInFlightDeploymentsFunc(
	deploymentCtrl *deploymentsv1.Controller,
	drainTimeout time.Duration,
) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()
		deploymentCtrl.Shutdown(ctx)
	}
}

// What has to be stopped when the engine shuts down.
type serverShutdown struct {
	drainDeployments     func()
	stopScheduledCleanup func()
	closeState           func()
	closePluginHost      func()
	closeAuditSink       func()
	stopTracing          func()
}

// Stops what Setup started, in the order the dependencies between them require.
func createServerCleanupFunc(shutdown *serverShutdown) func() {
	ordered := []func(){
		// In-flight operations first, so that each is given the chance to persist a
		// terminal status while what it writes to is still open.
		shutdown.drainDeployments,
		// Before the state service closes, since an in-flight pass is still writing to it.
		shutdown.stopScheduledCleanup,
		shutdown.closeState,
		shutdown.closePluginHost,
		// After the drain, so that a record is written for every action still in flight
		// when the shutdown began.
		shutdown.closeAuditSink,
		// Last, so that the spans describing everything above are handed over rather
		// than discarded with the provider.
		shutdown.stopTracing,
	}

	return func() {
		for _, stop := range ordered {
			if stop != nil {
				stop()
			}
		}
	}
}
