package languageserver

import (
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	common "github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
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
		TriggerCharacters: []string{"{", ",", "\"", "'", "(", "=", ".", " ", ":"},
		ResolveProvider:   &lsp.True,
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

	tree := a.state.GetDocumentTree(params.TextDocument.URI)
	if tree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}

		tree = a.state.GetDocumentTree(params.TextDocument.URI)
	}

	bpSchema := a.state.GetDocumentSchema(params.TextDocument.URI)
	if bpSchema == nil {
		a.logger.Error(
			"no schema found for document, current document is not a valid blueprint",
		)
		return nil, nil
	}

	content, err := a.hoverService.GetHoverContent(
		ctx,
		tree,
		bpSchema,
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

	tree := a.state.GetDocumentTree(params.TextDocument.URI)
	if tree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}

		tree = a.state.GetDocumentTree(params.TextDocument.URI)
	}

	signatures, err := a.signatureService.GetFunctionSignatures(
		ctx,
		tree,
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
	err := a.saveDocumentContent(params, existingContent)
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
	content := a.getDocumentContent(uri, true)
	diagnostics, blueprint, err := a.diagnosticService.ValidateTextDocument(
		ctx,
		uri,
	)
	if err != nil {
		return err
	}

	err = a.storeDocumentAndDerivedStructures(uri, blueprint, *content)
	if err != nil {
		return err
	}

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

func (a *Application) getDocumentContent(uri lsp.URI, fallbackToEmptyString bool) *string {
	content := a.state.GetDocumentContent(uri)
	if content == nil && fallbackToEmptyString {
		empty := ""
		return &empty
	}
	return content
}

func (a *Application) storeDocumentAndDerivedStructures(
	uri lsp.URI,
	parsed *schema.Blueprint,
	content string,
) error {
	if parsed == nil {
		return nil
	}

	docFormat := blueprint.DetermineDocFormat(uri)

	a.state.SetDocumentSchema(uri, parsed)
	tree := schema.SchemaToTree(parsed)
	// positionMap := blueprint.CreatePositionMap(tree)
	a.state.SetDocumentTree(uri, tree)
	// a.state.SetDocumentPositionMap(uri, positionMap)

	if docFormat == schema.YAMLSpecFormat {
		yamlNode, err := blueprint.ParseYAMLNode(content)
		if err != nil {
			return err
		}
		a.state.SetDocumentYAMLNode(uri, yamlNode)
	} else {
		jsonNode, err := blueprint.ParseJWCCNode(content)
		if err != nil {
			return err
		}
		a.state.SetDocumentJSONNode(uri, jsonNode)
	}
	return nil
}

func (a *Application) saveDocumentContent(params *lsp.DidChangeTextDocumentParams, existingContent *string) error {
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

	tree := a.state.GetDocumentTree(params.TextDocument.URI)
	if tree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}

		tree = a.state.GetDocumentTree(params.TextDocument.URI)
	}
	content := a.getDocumentContent(params.TextDocument.URI, true)
	bpSchema := a.state.GetDocumentSchema(params.TextDocument.URI)
	if bpSchema == nil {
		return nil, errors.New("no parsed blueprint found for document")
	}

	completionItems, err := a.completionService.GetCompletionItems(
		ctx,
		*content,
		tree,
		bpSchema,
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
	content := a.state.GetDocumentContent(params.TextDocument.URI)
	if content == nil {
		return nil, errors.New("no content found for document")
	}

	if a.state.HasHierarchicalDocumentSymbolCapability() {
		return a.symbolService.GetDocumentSymbols(params.TextDocument.URI, *content)
	}

	return []lsp.DocumentSymbol{}, nil
}

func (a *Application) handleGotoDefinition(
	ctx *common.LSPContext,
	params *lsp.DefinitionParams,
) (any, error) {
	dispatcher := lsp.NewDispatcher(ctx)

	tree := a.state.GetDocumentTree(params.TextDocument.URI)
	if tree == nil {
		err := a.validateAndPublishDiagnostics(ctx, params.TextDocument.URI, dispatcher)
		if err != nil {
			return nil, err
		}

		tree = a.state.GetDocumentTree(params.TextDocument.URI)
	}
	content := a.getDocumentContent(params.TextDocument.URI, true)
	bpSchema := a.state.GetDocumentSchema(params.TextDocument.URI)
	if bpSchema == nil {
		return nil, errors.New("no parsed blueprint found for document")
	}

	if a.state.HasLinkSupportCapability() {
		return a.gotoDefinitionService.GetDefinitions(
			*content,
			tree,
			bpSchema,
			&params.TextDocumentPositionParams,
		)
	}

	return []lsp.Location{}, nil
}

func (a *Application) handleShutdown(ctx *common.LSPContext) error {
	a.logger.Info("Shutting down server...")
	return nil
}
