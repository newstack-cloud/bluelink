package languageserver

import (
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageservices"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type HandlersSuite struct {
	suite.Suite
	logger *zap.Logger
	app    *Application
	state  *languageservices.State
}

func (s *HandlersSuite) SetupSuite() {
	s.logger = zap.NewNop()
}

func (s *HandlersSuite) SetupTest() {
	s.state = languageservices.NewState()
	s.app = s.createTestApplication(s.state)
}

// Tests for GetDocumentContent helper method

func (s *HandlersSuite) TestGetDocumentContent_ReturnsNilWhenNotSet() {
	content := s.app.GetDocumentContent("file:///nonexistent.yaml", false)
	s.Nil(content)
}

func (s *HandlersSuite) TestGetDocumentContent_ReturnsEmptyStringWhenFallbackEnabled() {
	content := s.app.GetDocumentContent("file:///nonexistent.yaml", true)
	s.NotNil(content)
	s.Equal("", *content)
}

func (s *HandlersSuite) TestGetDocumentContent_ReturnsStoredContent() {
	uri := lsp.URI("file:///test.yaml")
	expectedContent := "version: 2024-01-01\n"
	s.state.SetDocumentContent(uri, expectedContent)

	content := s.app.GetDocumentContent(uri, false)
	s.NotNil(content)
	s.Equal(expectedContent, *content)
}

func (s *HandlersSuite) TestGetDocumentContent_ReturnStoredContentWithFallbackEnabled() {
	uri := lsp.URI("file:///test.yaml")
	expectedContent := "version: 2024-01-01\n"
	s.state.SetDocumentContent(uri, expectedContent)

	content := s.app.GetDocumentContent(uri, true)
	s.NotNil(content)
	s.Equal(expectedContent, *content)
}

// Tests for SaveDocumentContent helper method

func (s *HandlersSuite) TestSaveDocumentContent_NoChanges() {
	params := &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: "file:///test.yaml",
			},
		},
		ContentChanges: []any{},
	}

	err := s.app.SaveDocumentContent(params, nil)
	s.NoError(err)
}

func (s *HandlersSuite) TestSaveDocumentContent_WholeDocumentChange() {
	uri := lsp.URI("file:///test.yaml")
	params := &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: uri,
			},
		},
		ContentChanges: []any{
			lsp.TextDocumentContentChangeEventWhole{
				Text: "new content",
			},
		},
	}

	err := s.app.SaveDocumentContent(params, nil)
	s.NoError(err)

	content := s.state.GetDocumentContent(uri)
	s.NotNil(content)
	s.Equal("new content", *content)
}

func (s *HandlersSuite) TestSaveDocumentContent_IncrementalChange() {
	uri := lsp.URI("file:///test.yaml")
	existingContent := "hello world"

	// Set up position encoding (required for incremental changes)
	s.state.SetPositionEncodingKind(lsp.PositionEncodingKindUTF16)

	params := &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: uri,
			},
		},
		ContentChanges: []any{
			lsp.TextDocumentContentChangeEvent{
				Range: &lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 5},
				},
				Text: "goodbye",
			},
		},
	}

	err := s.app.SaveDocumentContent(params, &existingContent)
	s.NoError(err)

	content := s.state.GetDocumentContent(uri)
	s.NotNil(content)
	s.Equal("goodbye world", *content)
}

func (s *HandlersSuite) TestSaveDocumentContent_NilRange_ReturnsError() {
	uri := lsp.URI("file:///test.yaml")
	existingContent := "hello world"

	params := &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: uri,
			},
		},
		ContentChanges: []any{
			lsp.TextDocumentContentChangeEvent{
				Range: nil, // nil range should cause error
				Text:  "new text",
			},
		},
	}

	err := s.app.SaveDocumentContent(params, &existingContent)
	s.Error(err)
	s.Contains(err.Error(), "change range is nil")
}

// Tests for StoreDocumentAndDerivedStructures

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_YAMLDocument_NilBlueprint() {
	uri := lsp.URI("file:///test.yaml")
	content := "version: 2024-01-01\n"

	// Test with nil blueprint (simulates parse failure)
	err := s.app.StoreDocumentAndDerivedStructures(uri, nil, content)
	s.NoError(err)

	docCtx := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx)
	s.Nil(docCtx.Blueprint) // Blueprint should be nil when not provided
}

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_JSONDocument_NilBlueprint() {
	uri := lsp.URI("file:///test.jsonc")
	content := `{"version": "2024-01-01"}`

	err := s.app.StoreDocumentAndDerivedStructures(uri, nil, content)
	s.NoError(err)

	docCtx := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx)
}

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_CreatesDocumentContext() {
	uri := lsp.URI("file:///test.yaml")
	content := "version: 2024-01-01\n"

	// Initially no context
	s.Nil(s.state.GetDocumentContext(uri))

	err := s.app.StoreDocumentAndDerivedStructures(uri, nil, content)
	s.NoError(err)

	// Now context exists
	docCtx := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx)
	s.Equal(content, docCtx.Content)
}

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_WithParsedBlueprint() {
	uri := lsp.URI("file:///test.yaml")
	content := "version: 2024-01-01\n"

	// Load a real blueprint to avoid tree conversion issues
	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	s.NoError(err)

	err = s.app.StoreDocumentAndDerivedStructures(uri, blueprint, content)
	s.NoError(err)

	docCtx := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx)
	s.NotNil(docCtx.Blueprint)
	s.NotNil(docCtx.SchemaTree)
}

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_PreservesLastValidState() {
	uri := lsp.URI("file:///test.yaml")

	// First, store a valid blueprint using LoadString for proper source metadata
	validContent := "version: 2024-01-01\n"
	validBlueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.NoError(err)

	err = s.app.StoreDocumentAndDerivedStructures(uri, validBlueprint, validContent)
	s.NoError(err)

	// Get the first context to verify it has the schema
	docCtx1 := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx1)
	s.NotNil(docCtx1.Blueprint)

	// Now store an invalid blueprint (nil) - simulates parse failure
	invalidContent := "invalid: yaml: :"
	err = s.app.StoreDocumentAndDerivedStructures(uri, nil, invalidContent)
	s.NoError(err)

	// The context should still exist but without a current blueprint
	docCtx2 := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx2)
	// The new context won't have blueprint since we passed nil
	s.Nil(docCtx2.Blueprint)
	// But LastValidSchema should be preserved
	s.NotNil(docCtx2.LastValidSchema)
}

// Tests for state capability methods used by handlers

func (s *HandlersSuite) TestState_WorkspaceFolderCapability() {
	s.False(s.state.HasWorkspaceFolderCapability())

	s.state.SetWorkspaceFolderCapability(true)
	s.True(s.state.HasWorkspaceFolderCapability())

	s.state.SetWorkspaceFolderCapability(false)
	s.False(s.state.HasWorkspaceFolderCapability())
}

func (s *HandlersSuite) TestState_ConfigurationCapability() {
	s.False(s.state.HasConfigurationCapability())

	s.state.SetConfigurationCapability(true)
	s.True(s.state.HasConfigurationCapability())
}

func (s *HandlersSuite) TestState_HierarchicalDocumentSymbolCapability() {
	s.False(s.state.HasHierarchicalDocumentSymbolCapability())

	s.state.SetHierarchicalDocumentSymbolCapability(true)
	s.True(s.state.HasHierarchicalDocumentSymbolCapability())
}

func (s *HandlersSuite) TestState_LinkSupportCapability() {
	s.False(s.state.HasLinkSupportCapability())

	s.state.SetLinkSupportCapability(true)
	s.True(s.state.HasLinkSupportCapability())
}

func (s *HandlersSuite) TestState_PositionEncodingKind() {
	s.state.SetPositionEncodingKind(lsp.PositionEncodingKindUTF16)
	s.Equal(lsp.PositionEncodingKindUTF16, s.state.GetPositionEncodingKind())

	s.state.SetPositionEncodingKind(lsp.PositionEncodingKindUTF32)
	s.Equal(lsp.PositionEncodingKindUTF32, s.state.GetPositionEncodingKind())
}

func (s *HandlersSuite) TestState_ClearDocSettings() {
	uri := "file:///test.yaml"
	settings := &languageservices.DocSettings{
		MaxNumberOfProblems: 100,
	}
	s.state.SetDocumentSettings(uri, settings)
	s.NotNil(s.state.GetDocumentSettings(uri))

	s.state.ClearDocSettings()
	s.Nil(s.state.GetDocumentSettings(uri))
}

// Tests for enhanced diagnostics storage

func (s *HandlersSuite) TestState_EnhancedDiagnostics() {
	uri := "file:///test.yaml"

	// Initially nil
	s.Nil(s.state.GetEnhancedDiagnostics(uri))

	// Set diagnostics
	diagnostics := []*languageservices.EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Message: "test error",
			},
		},
	}
	s.state.SetEnhancedDiagnostics(uri, diagnostics)

	retrieved := s.state.GetEnhancedDiagnostics(uri)
	s.NotNil(retrieved)
	s.Len(retrieved, 1)
	s.Equal("test error", retrieved[0].Diagnostic.Message)
}

// Tests for HandleShutdown - it should work with nil context

func (s *HandlersSuite) TestHandleShutdown_NoPluginHost() {
	// When pluginHostService is nil, shutdown should still succeed
	s.Nil(s.app.pluginHostService)

	err := s.app.HandleShutdown(nil)
	s.NoError(err)
}

// Test ReinitialiseRegistries

func (s *HandlersSuite) TestReinitialiseRegistries_UpdatesServices() {
	providers := make(map[string]provider.Provider)
	transformers := make(map[string]transform.SpecTransformer)

	// This should not panic and should update internal registries
	s.app.ReinitialiseRegistries(providers, transformers)

	// Verify the registries were updated
	s.NotNil(s.app.functionRegistry)
	s.NotNil(s.app.resourceRegistry)
	s.NotNil(s.app.dataSourceRegistry)
	s.NotNil(s.app.customVarTypeRegistry)
}

// Helper method to create a test application
func (s *HandlersSuite) createTestApplication(state *languageservices.State) *Application {
	settingsService := languageservices.NewSettingsService(state, "blueprintLanguageServer", s.logger)
	traceService := lsp.NewTraceService(nil)

	functionRegistry := &testutils.FunctionRegistryMock{
		Functions: make(map[string]provider.Function),
	}
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: make(map[string]provider.Resource),
	}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{
		DataSources: make(map[string]provider.DataSource),
	}
	customVarTypeRegistry := &testutils.CustomVarTypeRegistryMock{
		CustomVarTypes: make(map[string]provider.CustomVariableType),
	}

	diagnosticErrorService := languageservices.NewDiagnosticErrorService(state, s.logger)
	signatureService := languageservices.NewSignatureService(functionRegistry, s.logger)

	completionService := languageservices.NewCompletionService(
		resourceRegistry,
		dataSourceRegistry,
		customVarTypeRegistry,
		functionRegistry,
		nil, // childResolver
		state,
		s.logger,
	)
	diagnosticService := languageservices.NewDiagnosticsService(
		state,
		settingsService,
		diagnosticErrorService,
		nil,
		s.logger,
	)
	hoverService := languageservices.NewHoverService(
		functionRegistry,
		resourceRegistry,
		dataSourceRegistry,
		nil, // linkRegistry
		signatureService,
		nil, // childResolver
		s.logger,
	)
	symbolService := languageservices.NewSymbolService(state, s.logger)
	gotoDefinitionService := languageservices.NewGotoDefinitionService(state, nil /* childResolver */, s.logger)
	codeActionService := languageservices.NewCodeActionService(state, s.logger)

	return NewApplication(
		state,
		settingsService,
		traceService,
		functionRegistry,
		resourceRegistry,
		dataSourceRegistry,
		customVarTypeRegistry,
		nil,
		completionService,
		diagnosticService,
		signatureService,
		hoverService,
		symbolService,
		gotoDefinitionService,
		nil, // findReferencesService
		codeActionService,
		nil, // childResolver
		make(map[string]provider.Provider),
		make(map[string]transform.SpecTransformer),
		nil,
		s.logger,
		NewDocumentDebouncer(300*time.Millisecond),
	)
}

// Tests for Setup and Handler

func (s *HandlersSuite) TestSetup_and_Handler_returns_non_nil() {
	s.app.Setup()
	s.NotNil(s.app.Handler())
}

func (s *HandlersSuite) TestHandler_returns_nil_before_setup() {
	// A fresh application without Setup() called should have nil handler.
	freshApp := s.createTestApplication(languageservices.NewState())
	s.Nil(freshApp.Handler())
}

// Tests for StoreDocumentAndDerivedStructures with different formats

func (s *HandlersSuite) TestStoreDocumentAndDerivedStructures_UpdatesExistingContext() {
	uri := lsp.URI("file:///test.yaml")
	content1 := "version: 2024-01-01\n"
	blueprint1, err := schema.LoadString(content1, schema.YAMLSpecFormat)
	s.NoError(err)

	err = s.app.StoreDocumentAndDerivedStructures(uri, blueprint1, content1)
	s.NoError(err)

	docCtx1 := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx1)
	s.NotNil(docCtx1.Blueprint)

	// Store a second time with different content but same URI.
	content2 := "version: 2025-01-01\n"
	blueprint2, err := schema.LoadString(content2, schema.YAMLSpecFormat)
	s.NoError(err)

	err = s.app.StoreDocumentAndDerivedStructures(uri, blueprint2, content2)
	s.NoError(err)

	docCtx2 := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx2)
	s.NotNil(docCtx2.Blueprint)
}

func (s *HandlersSuite) TestSaveDocumentContent_UnknownChangeType_ReturnsError() {
	uri := lsp.URI("file:///test.yaml")
	params := &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{URI: uri},
		},
		ContentChanges: []any{
			"invalid change type",
		},
	}
	existingContent := "hello world"
	err := s.app.SaveDocumentContent(params, &existingContent)
	s.Error(err)
	s.Contains(err.Error(), "not of a valid type")
}

func (s *HandlersSuite) TestState_DocumentContext_SetGet() {
	uri := lsp.URI("file:///test.yaml")
	s.Nil(s.state.GetDocumentContext(uri))

	content := "version: 2024-01-01\n"
	err := s.app.StoreDocumentAndDerivedStructures(uri, nil, content)
	s.NoError(err)

	docCtx := s.state.GetDocumentContext(uri)
	s.NotNil(docCtx)
}

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(HandlersSuite))
}
