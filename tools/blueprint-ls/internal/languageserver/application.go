package languageserver

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageservices"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/pluginhost"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/sourcegraph/jsonrpc2"
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
	codeActionService     *languageservices.CodeActionService
	logger                *zap.Logger
	traceService          *lsp.TraceService

	// Plugin loading support
	builtInProviders    map[string]provider.Provider
	builtInTransformers map[string]transform.SpecTransformer
	frameworkLogger     core.Logger
	pluginHostService   pluginhost.Service

	// Custom variable type registry (used by completion service)
	customVarTypeRegistry provider.CustomVariableTypeRegistry

	// Debouncer for diagnostic publishing to reduce error flicker during typing
	debouncer *DocumentDebouncer

	// JSON-RPC connection for sending notifications outside of request handlers
	conn *jsonrpc2.Conn
}

func NewApplication(
	state *languageservices.State,
	settingsService *languageservices.SettingsService,
	traceService *lsp.TraceService,
	functionRegistry provider.FunctionRegistry,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
	customVarTypeRegistry provider.CustomVariableTypeRegistry,
	blueprintLoader container.Loader,
	completionService *languageservices.CompletionService,
	diagnosticService *languageservices.DiagnosticsService,
	signatureService *languageservices.SignatureService,
	hoverService *languageservices.HoverService,
	symbolService *languageservices.SymbolService,
	gotoDefinitionService *languageservices.GotoDefinitionService,
	codeActionService *languageservices.CodeActionService,
	builtInProviders map[string]provider.Provider,
	builtInTransformers map[string]transform.SpecTransformer,
	frameworkLogger core.Logger,
	logger *zap.Logger,
	debouncer *DocumentDebouncer,
) *Application {
	return &Application{
		state:                 state,
		settingsService:       settingsService,
		traceService:          traceService,
		functionRegistry:      functionRegistry,
		resourceRegistry:      resourceRegistry,
		dataSourceRegistry:    dataSourceRegistry,
		customVarTypeRegistry: customVarTypeRegistry,
		blueprintLoader:       blueprintLoader,
		completionService:     completionService,
		diagnosticService:     diagnosticService,
		signatureService:      signatureService,
		hoverService:          hoverService,
		symbolService:         symbolService,
		gotoDefinitionService: gotoDefinitionService,
		codeActionService:     codeActionService,
		builtInProviders:      builtInProviders,
		builtInTransformers:   builtInTransformers,
		frameworkLogger:       frameworkLogger,
		logger:                logger,
		debouncer:             debouncer,
	}
}

func (a *Application) Setup() {
	a.handler = lsp.NewHandler(
		lsp.WithInitializeHandler(a.handleInitialise),
		lsp.WithInitializedHandler(a.handleInitialised),
		lsp.WithShutdownHandler(a.HandleShutdown),
		lsp.WithTextDocumentDidOpenHandler(a.handleTextDocumentDidOpen),
		lsp.WithTextDocumentDidCloseHandler(a.handleTextDocumentDidClose),
		lsp.WithTextDocumentDidChangeHandler(a.handleTextDocumentDidChange),
		lsp.WithTextDocumentDidSaveHandler(a.handleTextDocumentDidSave),
		lsp.WithSetTraceHandler(a.traceService.CreateSetTraceHandler()),
		lsp.WithHoverHandler(a.handleHover),
		lsp.WithSignatureHelpHandler(a.handleSignatureHelp),
		lsp.WithCompletionHandler(a.handleCompletion),
		lsp.WithCompletionItemResolveHandler(a.handleCompletionItemResolve),
		lsp.WithDocumentSymbolHandler(a.handleDocumentSymbols),
		lsp.WithGotoDefinitionHandler(a.handleGotoDefinition),
		lsp.WithCodeActionHandler(a.handleCodeAction),
	)
}

func (a *Application) Handler() *lsp.Handler {
	return a.handler
}

// SetConnection sets the JSON-RPC connection for sending notifications.
// This must be called after the connection is established.
func (a *Application) SetConnection(conn *jsonrpc2.Conn) {
	a.conn = conn
}
