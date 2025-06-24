package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageserver"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageservices"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/newstack-cloud/ls-builder/server"
	"github.com/sourcegraph/jsonrpc2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger, logFile, err := setupLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	state := languageservices.NewState()
	settingsService := languageservices.NewSettingsService(
		state,
		languageserver.ConfigSection,
		logger,
	)
	traceService := lsp.NewTraceService(logger)

	providers, err := blueprint.LoadProviders(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	transformers, err := blueprint.LoadTransformers(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	functionRegistry := provider.NewFunctionRegistry(providers)
	resourceRegistry := resourcehelpers.NewRegistry(
		providers,
		transformers,
		// The polling timeout is not used for the language server but needs to be provided
		// to create a new resource registry.
		/* stabilisationPollingTimeout */
		time.Second,
		nil,
	)
	frameworkLogger := core.NewLoggerFromZap(logger)
	dataSourceRegistry := provider.NewDataSourceRegistry(providers, core.SystemClock{}, frameworkLogger)
	customVarTypeRegistry := provider.NewCustomVariableTypeRegistry(providers)

	completionService := languageservices.NewCompletionService(
		resourceRegistry,
		dataSourceRegistry,
		customVarTypeRegistry,
		functionRegistry,
		state,
		logger,
	)

	blueprintLoader := container.NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{},
		/* stateContainer */ nil,
		/* childResolver */ nil,
		// Disable runtime value validation as it is not needed for diagnostics.
		container.WithLoaderValidateRuntimeValues(false),
		// Disable spec transformation as it is not needed for diagnostics.
		container.WithLoaderTransformSpec(false),
	)

	diagnosticErrorService := languageservices.NewDiagnosticErrorService(state, logger)
	diagnosticService := languageservices.NewDiagnosticsService(
		state,
		settingsService,
		diagnosticErrorService,
		blueprintLoader,
		logger,
	)

	signatureService := languageservices.NewSignatureService(
		functionRegistry,
		logger,
	)
	hoverService := languageservices.NewHoverService(
		functionRegistry,
		resourceRegistry,
		dataSourceRegistry,
		signatureService,
		logger,
	)
	symbolService := languageservices.NewSymbolService(
		state,
		logger,
	)
	gotoDefinitionService := languageservices.NewGotoDefinitionService(
		state,
		logger,
	)

	app := languageserver.NewApplication(
		state,
		settingsService,
		traceService,
		functionRegistry,
		resourceRegistry,
		dataSourceRegistry,
		blueprintLoader,
		completionService,
		diagnosticService,
		signatureService,
		hoverService,
		symbolService,
		gotoDefinitionService,
		logger,
	)
	app.Setup()

	srv := server.NewServer(app.Handler(), true, logger, nil)

	stdio := server.Stdio{}
	conn := server.NewStreamConnection(
		// Wrapping in async handler is essential to avoid a deadlock
		// when the server sends a request to the client while it is handling
		// a request from the client.
		// For example, when handling the hover request, the server may fetch
		// configuration settings from the client, without an async handler, this will
		// block until the configured timeout is reached and the context is cancelled.
		jsonrpc2.AsyncHandler(srv.NewHandler()),
		stdio,
	)
	srv.Serve(conn, logger)
}

func setupLogger() (*zap.Logger, *os.File, error) {
	logFileHandle, err := os.OpenFile("blueprint-ls.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	writerSync := zapcore.NewMultiWriteSyncer(
		// stdout and stdin are used for communication with the client
		// and should not be logged to.
		zapcore.AddSync(os.Stderr),
		zapcore.AddSync(logFileHandle),
	)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		writerSync,
		zap.DebugLevel,
	)
	logger := zap.New(core)
	return logger, logFileHandle, nil
}
