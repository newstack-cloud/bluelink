package languageserver

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageservices"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

type Application struct {
	handler               *lsp.Handler
	state                 *languageservices.State
	settingsService       *languageservices.SettingsService
	functionRegistry      provider.FunctionRegistry
	resourceRegistry      resourcehelpers.Registry
	dataSourceRegistry    provider.DataSourceRegistry
	blueprintLoader       container.Loader
	completionService     *languageservices.CompletionService
	diagnosticService     *languageservices.DiagnosticsService
	signatureService      *languageservices.SignatureService
	hoverService          *languageservices.HoverService
	symbolService         *languageservices.SymbolService
	gotoDefinitionService *languageservices.GotoDefinitionService
	logger                *zap.Logger
	traceService          *lsp.TraceService
}

func NewApplication(
	state *languageservices.State,
	settingsService *languageservices.SettingsService,
	traceService *lsp.TraceService,
	functionRegistry provider.FunctionRegistry,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	blueprintLoader container.Loader,
	completionService *languageservices.CompletionService,
	diagnosticService *languageservices.DiagnosticsService,
	signatureService *languageservices.SignatureService,
	hoverService *languageservices.HoverService,
	symbolService *languageservices.SymbolService,
	gotoDefinitionService *languageservices.GotoDefinitionService,
	logger *zap.Logger,
) *Application {
	return &Application{
		state:                 state,
		settingsService:       settingsService,
		traceService:          traceService,
		functionRegistry:      functionRegistry,
		resourceRegistry:      resourceRegistry,
		dataSourceRegistry:    dataSourceRegistry,
		blueprintLoader:       blueprintLoader,
		completionService:     completionService,
		diagnosticService:     diagnosticService,
		signatureService:      signatureService,
		hoverService:          hoverService,
		symbolService:         symbolService,
		gotoDefinitionService: gotoDefinitionService,
		logger:                logger,
	}
}

func (a *Application) Setup() {
	a.handler = lsp.NewHandler(
		lsp.WithInitializeHandler(a.handleInitialise),
		lsp.WithInitializedHandler(a.handleInitialised),
		lsp.WithShutdownHandler(a.handleShutdown),
		lsp.WithTextDocumentDidOpenHandler(a.handleTextDocumentDidOpen),
		lsp.WithTextDocumentDidCloseHandler(a.handleTextDocumentDidClose),
		lsp.WithTextDocumentDidChangeHandler(a.handleTextDocumentDidChange),
		lsp.WithSetTraceHandler(a.traceService.CreateSetTraceHandler()),
		lsp.WithHoverHandler(a.handleHover),
		lsp.WithSignatureHelpHandler(a.handleSignatureHelp),
		lsp.WithCompletionHandler(a.handleCompletion),
		lsp.WithCompletionItemResolveHandler(a.handleCompletionItemResolve),
		lsp.WithDocumentSymbolHandler(a.handleDocumentSymbols),
		lsp.WithGotoDefinitionHandler(a.handleGotoDefinition),
	)
}

func (a *Application) Handler() *lsp.Handler {
	return a.handler
}
