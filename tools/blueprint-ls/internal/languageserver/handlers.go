package languageserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/pluginhost"
	common "github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

func (a *Application) handleInitialise(ctx *common.LSPContext, params *lsp.InitializeParams) (any, error) {
	a.logger.Debug("Initialising server...")
	clientCapabilities := params.Capabilities
	capabilities := a.handler.CreateServerCapabilities()
	// Take the first position encoding as the one with the highest priority as per the spec.
	// this language server supports all three position encodings. (UTF-16, UTF-8, UTF-32)
	capabilities.PositionEncoding = params.Capabilities.General.PositionEncodings[0]
	a.state.SetPositionEncodingKind(capabilities.PositionEncoding)

	capabilities.SignatureHelpProvider = &lsp.SignatureHelpOptions{
		TriggerCharacters: []string{"(", ","},
	}
	capabilities.CompletionProvider = &lsp.CompletionOptions{
		TriggerCharacters: []string{"{", ",", "\"", "'", "(", "=", ".", " ", ":", "["},
		ResolveProvider:   &lsp.True,
	}
	capabilities.CodeActionProvider = &lsp.CodeActionOptions{
		CodeActionKinds: []lsp.CodeActionKind{
			lsp.CodeActionKindQuickFix,
		},
	}

	hasWorkspaceFolderCapability := clientCapabilities.Workspace != nil && clientCapabilities.Workspace.WorkspaceFolders != nil
	a.state.SetWorkspaceFolderCapability(hasWorkspaceFolderCapability)

	hasConfigurationCapability := clientCapabilities.Workspace != nil && clientCapabilities.Workspace.Configuration != nil
	a.state.SetConfigurationCapability(hasConfigurationCapability)

	hasHierarchicalDocumentSymbolCapability := clientCapabilities.TextDocument != nil &&
		clientCapabilities.TextDocument.DocumentSymbol != nil &&
		*clientCapabilities.TextDocument.DocumentSymbol.HierarchicalDocumentSymbolSupport
	a.state.SetHierarchicalDocumentSymbolCapability(hasHierarchicalDocumentSymbolCapability)

	hasLinkSupportCapability := clientCapabilities.TextDocument != nil &&
		clientCapabilities.TextDocument.Definition != nil &&
		*clientCapabilities.TextDocument.Definition.LinkSupport
	a.state.SetLinkSupportCapability(hasLinkSupportCapability)

	// Parse initializationOptions for plugin configuration
	var initOpts *pluginhost.InitializationOptions
	if params.InitializationOptions != nil {
		optBytes, err := json.Marshal(params.InitializationOptions)
		if err == nil {
			if err := json.Unmarshal(optBytes, &initOpts); err != nil {
				a.logger.Warn("Failed to parse initializationOptions", zap.Error(err))
			}
		}
	}

	// Load plugins if configured (only once per server lifetime)
	pluginConfig := pluginhost.NewDefaultConfig().WithInitOptions(initOpts)
	if a.pluginHostService == nil && pluginConfig.IsEnabled() && pluginConfig.GetPluginPath() != "" {
		a.loadPlugins(context.Background(), pluginConfig)
	}

	result := lsp.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &lsp.InitializeResultServerInfo{
			Name:    Name,
			Version: &Version,
		},
	}

	if hasWorkspaceFolderCapability {
		result.Capabilities.Workspace = &lsp.ServerWorkspaceCapabilities{
			WorkspaceFolders: &lsp.WorkspaceFoldersServerCapabilities{
				Supported: &hasWorkspaceFolderCapability,
			},
		}
	}

	return result, nil
}

func (a *Application) handleInitialised(ctx *common.LSPContext, params *lsp.InitializedParams) error {
	if a.state.HasConfigurationCapability() {
		a.handler.SetWorkspaceDidChangeConfigurationHandler(
			a.handleWorkspaceDidChangeConfiguration,
		)
	}
	return nil
}

func (a *Application) handleWorkspaceDidChangeConfiguration(ctx *common.LSPContext, params *lsp.DidChangeConfigurationParams) error {
	if a.state.HasConfigurationCapability() {
		// Reset all the cached document settings.
		a.state.ClearDocSettings()
	}

	return nil
}

func (a *Application) handleHover(ctx *common.LSPContext, params *lsp.HoverParams) (*lsp.Hover, error) {
	dispatcher := lsp.NewDispatcher(ctx)

	docCtx := a.state.GetDocumentContext(params.TextDocument.URI)
	if docCtx == nil || docCtx.SchemaTree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}
		docCtx = a.state.GetDocumentContext(params.TextDocument.URI)
	}

	if docCtx == nil || docCtx.Blueprint == nil {
		a.logger.Error(
			"no schema found for document, current document is not a valid blueprint",
		)
		return nil, nil
	}

	content, err := a.hoverService.GetHoverContent(
		ctx,
		docCtx,
		&params.TextDocumentPositionParams,
	)
	if err != nil {
		return nil, err
	}

	if content == nil {
		return nil, nil
	}

	return &lsp.Hover{
		Contents: lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: content.Value,
		},
		Range: content.Range,
	}, nil
}

func (a *Application) handleSignatureHelp(ctx *common.LSPContext, params *lsp.SignatureHelpParams) (*lsp.SignatureHelp, error) {
	dispatcher := lsp.NewDispatcher(ctx)

	docCtx := a.state.GetDocumentContext(params.TextDocument.URI)
	if docCtx == nil || docCtx.SchemaTree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}
		docCtx = a.state.GetDocumentContext(params.TextDocument.URI)
	}

	signatures, err := a.signatureService.GetFunctionSignatures(
		ctx,
		docCtx,
		&params.TextDocumentPositionParams,
	)
	if err != nil {
		return nil, err
	}

	return &lsp.SignatureHelp{
		Signatures: signatures,
	}, nil
}

func (a *Application) handleTextDocumentDidOpen(ctx *common.LSPContext, params *lsp.DidOpenTextDocumentParams) error {
	ctx.Notify("window/logMessage", &lsp.LogMessageParams{
		Type:    lsp.MessageTypeInfo,
		Message: "Text document opened (server received)",
	})
	dispatcher := lsp.NewDispatcher(ctx)
	a.state.SetDocumentContent(params.TextDocument.URI, params.TextDocument.Text)
	err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
	return err
}

func (a *Application) handleTextDocumentDidClose(ctx *common.LSPContext, params *lsp.DidCloseTextDocumentParams) error {
	ctx.Notify("window/logMessage", &lsp.LogMessageParams{
		Type:    lsp.MessageTypeInfo,
		Message: "Text document closed (server received)",
	})
	return nil
}

func (a *Application) handleTextDocumentDidChange(ctx *common.LSPContext, params *lsp.DidChangeTextDocumentParams) error {
	ctx.Notify("window/logMessage", &lsp.LogMessageParams{
		Type:    lsp.MessageTypeInfo,
		Message: "Text document changed (server received)",
	})
	dispatcher := lsp.NewDispatcher(ctx)
	existingContent := a.state.GetDocumentContent(params.TextDocument.URI)
	err := a.SaveDocumentContent(params, existingContent)
	if err != nil {
		return err
	}
	err = a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
	return err
}

func (a *Application) validateAndPublishDiagnostics(
	ctx *common.LSPContext,
	uri lsp.URI,
	dispatcher *lsp.Dispatcher,
) error {
	content := a.GetDocumentContent(uri, true)
	diagnostics, enhanced, blueprint, err := a.diagnosticService.ValidateTextDocument(
		ctx,
		uri,
	)
	if err != nil {
		return err
	}

	err = a.StoreDocumentAndDerivedStructures(uri, blueprint, *content)
	if err != nil {
		return err
	}

	// Store enhanced diagnostics for code action support
	a.state.SetEnhancedDiagnostics(uri, enhanced)

	// We must push diagnostics even if there are no errors to clear the existing ones
	// in the client.
	err = dispatcher.PublishDiagnostics(lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
	if err != nil {
		return err
	}
	return nil
}

// GetDocumentContent retrieves document content from state, optionally returning
// an empty string if the document is not found.
func (a *Application) GetDocumentContent(uri lsp.URI, fallbackToEmptyString bool) *string {
	content := a.state.GetDocumentContent(uri)
	if content == nil && fallbackToEmptyString {
		empty := ""
		return &empty
	}
	return content
}

// StoreDocumentAndDerivedStructures stores the document content, parsed blueprint,
// and derived structures (schema tree, AST) in state.
func (a *Application) StoreDocumentAndDerivedStructures(
	uri lsp.URI,
	parsed *schema.Blueprint,
	content string,
) error {
	specFormat := blueprint.DetermineDocFormat(uri)
	var docFormat docmodel.DocumentFormat
	if specFormat == schema.JWCCSpecFormat {
		docFormat = docmodel.FormatJSONC
	} else {
		docFormat = docmodel.FormatYAML
	}

	// Get existing DocumentContext to preserve last-known-good state
	existingDocCtx := a.state.GetDocumentContext(uri)

	// Create new DocumentContext with tree-sitter AST
	docCtx := docmodel.NewDocumentContext(
		string(uri),
		content,
		docFormat,
		a.logger,
	)

	// Preserve last-known-good state from existing context if current parsing failed
	if existingDocCtx != nil {
		// Preserve last valid AST for indentation-based context detection
		if existingDocCtx.LastValidAST != nil && docCtx.LastValidAST == nil {
			docCtx.LastValidAST = existingDocCtx.LastValidAST
		}
		// Preserve last valid schema for completion items
		if existingDocCtx.LastValidSchema != nil && docCtx.LastValidSchema == nil {
			docCtx.LastValidSchema = existingDocCtx.LastValidSchema
			docCtx.LastValidTree = existingDocCtx.LastValidTree
			docCtx.LastValidVersion = existingDocCtx.LastValidVersion
		}
	}

	// Add schema information if parsing succeeded
	if parsed != nil {
		tree := schema.SchemaToTree(parsed)
		docCtx.UpdateSchema(parsed, tree)
	}

	a.state.SetDocumentContext(uri, docCtx)
	return nil
}

// SaveDocumentContent processes text document change events and updates document content in state.
func (a *Application) SaveDocumentContent(params *lsp.DidChangeTextDocumentParams, existingContent *string) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}

	currentContent := ""
	if existingContent != nil {
		currentContent = *existingContent
	}

	for _, change := range params.ContentChanges {
		wholeChange, isWholeChangeEvent := change.(lsp.TextDocumentContentChangeEventWhole)
		if isWholeChangeEvent {
			a.state.SetDocumentContent(params.TextDocument.URI, wholeChange.Text)
			return nil
		}

		change, isChangeEvent := change.(lsp.TextDocumentContentChangeEvent)
		if !isChangeEvent {
			a.logger.Info(fmt.Sprintf("content change event: %+v", change))
			return errors.New(
				"content change event is not of a valid type, expected" +
					" TextDocumentContentChangeEvent or TextDocumentContentChangeEventWhole",
			)
		}

		if change.Range == nil {
			return errors.New("change range is nil")
		}

		startIndex, endIndex := change.Range.IndexesIn(*existingContent, a.state.GetPositionEncodingKind())
		currentContent = currentContent[:startIndex] + change.Text + currentContent[endIndex:]
	}

	a.state.SetDocumentContent(params.TextDocument.URI, currentContent)

	return nil
}

func (a *Application) handleCompletion(
	ctx *common.LSPContext,
	params *lsp.CompletionParams,
) (any, error) {
	dispatcher := lsp.NewDispatcher(ctx)

	docCtx := a.state.GetDocumentContext(params.TextDocument.URI)
	// Try to validate if we don't have any schema (current or last-known-good)
	if docCtx == nil || (docCtx.SchemaTree == nil && docCtx.LastValidTree == nil) {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}
		docCtx = a.state.GetDocumentContext(params.TextDocument.URI)
	}

	// Use GetEffectiveSchema() to fall back to last-known-good schema during editing
	if docCtx == nil || docCtx.GetEffectiveSchema() == nil {
		return nil, errors.New("no parsed blueprint found for document")
	}

	// Update DocumentContext with the latest content from state.
	// This handles race conditions where completion request arrives before
	// the didChange notification is fully processed and docCtx is updated.
	latestContent := a.state.GetDocumentContent(params.TextDocument.URI)
	if latestContent != nil && *latestContent != docCtx.Content {
		docCtx.UpdateContent(*latestContent, docCtx.Version+1)
	}

	completionItems, err := a.completionService.GetCompletionItems(
		ctx,
		docCtx,
		&params.TextDocumentPositionParams,
	)
	if err != nil {
		return nil, err
	}

	return completionItems, nil
}

func (a *Application) handleCompletionItemResolve(
	ctx *common.LSPContext,
	item *lsp.CompletionItem,
) (*lsp.CompletionItem, error) {

	dataMap, isDataMap := item.Data.(map[string]interface{})
	if !isDataMap {
		return item, nil
	}

	completionType, hasCompletionType := dataMap["completionType"].(string)
	if !hasCompletionType {
		return item, nil
	}

	return a.completionService.ResolveCompletionItem(ctx, item, completionType)
}

func (a *Application) handleDocumentSymbols(
	ctx *common.LSPContext,
	params *lsp.DocumentSymbolParams,
) (any, error) {
	docCtx := a.state.GetDocumentContext(params.TextDocument.URI)

	// If no stored context, create one from content for symbols
	if docCtx == nil {
		content := a.state.GetDocumentContent(params.TextDocument.URI)
		if content == nil {
			return nil, errors.New("no content found for document")
		}

		format := blueprint.DetermineDocFormat(params.TextDocument.URI)
		var docFormat docmodel.DocumentFormat
		if format == schema.JWCCSpecFormat {
			docFormat = docmodel.FormatJSONC
		} else {
			docFormat = docmodel.FormatYAML
		}

		docCtx = docmodel.NewDocumentContext(
			string(params.TextDocument.URI),
			*content,
			docFormat,
			a.logger,
		)
	}

	if a.state.HasHierarchicalDocumentSymbolCapability() {
		return a.symbolService.GetDocumentSymbolsFromContext(docCtx)
	}

	return []lsp.DocumentSymbol{}, nil
}

func (a *Application) handleGotoDefinition(
	ctx *common.LSPContext,
	params *lsp.DefinitionParams,
) (any, error) {
	dispatcher := lsp.NewDispatcher(ctx)

	docCtx := a.state.GetDocumentContext(params.TextDocument.URI)
	if docCtx == nil || docCtx.SchemaTree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}
		docCtx = a.state.GetDocumentContext(params.TextDocument.URI)
	}

	if docCtx == nil || docCtx.Blueprint == nil {
		return nil, errors.New("no parsed blueprint found for document")
	}

	if a.state.HasLinkSupportCapability() {
		return a.gotoDefinitionService.GetDefinitionsFromContext(
			docCtx,
			&params.TextDocumentPositionParams,
		)
	}

	return []lsp.Location{}, nil
}

// HandleShutdown handles the LSP shutdown request, closing the plugin host if active.
func (a *Application) HandleShutdown(ctx *common.LSPContext) error {
	a.logger.Info("Shutting down server...")
	if a.pluginHostService != nil {
		a.pluginHostService.Close()
	}
	return nil
}

func (a *Application) loadPlugins(ctx context.Context, config pluginhost.Config) {
	a.logger.Info("Loading plugins...",
		zap.String("pluginPath", config.GetPluginPath()),
		zap.String("logFileRootDir", config.GetLogFileRootDir()),
	)

	pluginHostService, err := pluginhost.LoadDefaultService(
		&pluginhost.LoadDependencies{
			Executor:         plugin.NewOSCmdExecutor(config.GetLogFileRootDir(), nil),
			InstanceFactory:  plugin.CreatePluginInstance,
			PluginHostConfig: config,
		},
		pluginhost.WithServiceLogger(a.frameworkLogger),
		pluginhost.WithInitialProviders(a.builtInProviders),
	)
	if err != nil {
		a.logger.Warn("Failed to initialize plugin host", zap.Error(err))
		return
	}
	a.pluginHostService = pluginHostService

	loadCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(config.GetTotalLaunchWaitTimeoutMS())*time.Millisecond,
	)
	defer cancel()

	pluginMaps, err := pluginHostService.LoadPlugins(loadCtx)
	if err != nil {
		a.logger.Warn("Failed to load plugins", zap.Error(err))
		return
	}

	a.logger.Info("Loaded plugins",
		zap.Int("providers", len(pluginMaps.Providers)-len(a.builtInProviders)),
		zap.Int("transformers", len(pluginMaps.Transformers)-len(a.builtInTransformers)),
	)

	// Merge providers and transformers
	mergedProviders := make(map[string]provider.Provider)
	maps.Copy(mergedProviders, a.builtInProviders)
	maps.Copy(mergedProviders, pluginMaps.Providers)

	mergedTransformers := make(map[string]transform.SpecTransformer)
	maps.Copy(mergedTransformers, a.builtInTransformers)
	maps.Copy(mergedTransformers, pluginMaps.Transformers)

	// Reinitialize registries with merged providers
	a.ReinitialiseRegistries(mergedProviders, mergedTransformers)
}

// ReinitialiseRegistries updates all registries and services with new providers and transformers.
func (a *Application) ReinitialiseRegistries(
	providers map[string]provider.Provider,
	transformers map[string]transform.SpecTransformer,
) {
	a.functionRegistry = provider.NewFunctionRegistry(providers)
	a.resourceRegistry = resourcehelpers.NewRegistry(
		providers,
		transformers,
		time.Second, // Not used by LS
		nil,         // No state container needed
		nil,         // No params needed
	)
	a.dataSourceRegistry = provider.NewDataSourceRegistry(
		providers,
		core.SystemClock{},
		a.frameworkLogger,
	)
	a.customVarTypeRegistry = provider.NewCustomVariableTypeRegistry(providers)
	linkRegistry := provider.NewLinkRegistry(providers)

	// Create a new blueprint loader with merged providers
	blueprintLoader := container.NewDefaultLoader(
		providers,
		transformers,
		nil, // No state container
		nil, // No child resolver
		container.WithLoaderValidateRuntimeValues(false),
		container.WithLoaderTransformSpec(false),
	)
	a.blueprintLoader = blueprintLoader

	// Update services with new registries
	a.completionService.UpdateRegistries(
		a.resourceRegistry,
		a.dataSourceRegistry,
		a.customVarTypeRegistry,
		a.functionRegistry,
		linkRegistry,
	)
	a.diagnosticService.UpdateLoader(blueprintLoader)
	a.signatureService.UpdateRegistry(a.functionRegistry)
	a.hoverService.UpdateRegistries(
		a.functionRegistry,
		a.resourceRegistry,
		a.dataSourceRegistry,
	)
}

func (a *Application) handleCodeAction(
	ctx *common.LSPContext,
	params *lsp.CodeActionParams,
) ([]*lsp.CodeActionOrCommand, error) {
	actions, err := a.codeActionService.GetCodeActions(params)
	if err != nil {
		return nil, err
	}

	result := make([]*lsp.CodeActionOrCommand, len(actions))
	for i, action := range actions {
		actionCopy := action
		result[i] = &lsp.CodeActionOrCommand{
			CodeAction: &actionCopy,
		}
	}
	return result, nil
}
