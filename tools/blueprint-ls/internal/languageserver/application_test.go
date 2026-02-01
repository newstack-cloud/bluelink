package languageserver

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/languageservices"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/newstack-cloud/ls-builder/server"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const testTimeout = 10 * time.Second

type ApplicationSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *ApplicationSuite) SetupSuite() {
	s.logger = zap.NewNop()
}

// testConnectionsContainer holds client and server connections for testing
type testConnectionsContainer struct {
	clientReceivedMessages []*json.RawMessage
	clientReceivedMethods  []string
	clientConn             *jsonrpc2.Conn
	serverConn             *jsonrpc2.Conn
	mu                     sync.Mutex
}

// testStream wraps io.Reader and io.Writer for test streams
type testStream struct {
	in  io.Reader
	out io.Writer
}

func (ts *testStream) Read(p []byte) (int, error)  { return ts.in.Read(p) }
func (ts *testStream) Write(p []byte) (int, error) { return ts.out.Write(p) }
func (ts *testStream) Close() error                { return nil }

// createTestConnectionsContainer creates in-memory client/server connections
func createTestConnectionsContainer(serverHandler jsonrpc2.Handler) *testConnectionsContainer {
	clientIn, serverOut := io.Pipe()
	serverIn, clientOut := io.Pipe()
	clientStream := &testStream{in: clientIn, out: clientOut}
	serverStream := &testStream{in: serverIn, out: serverOut}

	container := &testConnectionsContainer{
		clientReceivedMessages: []*json.RawMessage{},
		clientReceivedMethods:  []string{},
	}

	clientHandler := jsonrpc2.HandlerWithError(
		func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
			container.mu.Lock()
			defer container.mu.Unlock()
			container.clientReceivedMessages = append(container.clientReceivedMessages, req.Params)
			container.clientReceivedMethods = append(container.clientReceivedMethods, req.Method)
			return nil, nil
		},
	)
	container.serverConn = server.NewStreamConnection(serverHandler, serverStream)
	container.clientConn = server.NewStreamConnection(clientHandler, clientStream)
	return container
}

// createTestApplication creates a fully configured Application for testing
func (s *ApplicationSuite) createTestApplication() *Application {
	state := languageservices.NewState()
	settingsService := languageservices.NewSettingsService(state, "blueprintLanguageServer", s.logger)
	traceService := lsp.NewTraceService(nil)

	functionRegistry := &testutils.FunctionRegistryMock{Functions: make(map[string]provider.Function)}
	resourceRegistry := &testutils.ResourceRegistryMock{Resources: make(map[string]provider.Resource)}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{DataSources: make(map[string]provider.DataSource)}
	customVarTypeRegistry := &testutils.CustomVarTypeRegistryMock{CustomVarTypes: make(map[string]provider.CustomVariableType)}

	diagnosticErrorService := languageservices.NewDiagnosticErrorService(state, s.logger)
	signatureService := languageservices.NewSignatureService(functionRegistry, s.logger)

	completionService := languageservices.NewCompletionService(
		resourceRegistry, dataSourceRegistry, customVarTypeRegistry, functionRegistry, nil, state, s.logger,
	)
	diagnosticService := languageservices.NewDiagnosticsService(
		state, settingsService, diagnosticErrorService, nil, s.logger,
	)
	hoverService := languageservices.NewHoverService(
		functionRegistry, resourceRegistry, dataSourceRegistry, nil, signatureService, nil, s.logger,
	)
	symbolService := languageservices.NewSymbolService(state, s.logger)
	gotoDefinitionService := languageservices.NewGotoDefinitionService(state, nil /* childResolver */, s.logger)
	codeActionService := languageservices.NewCodeActionService(state, s.logger)

	debouncer := NewDocumentDebouncer(300 * time.Millisecond)

	return NewApplication(
		state, settingsService, traceService,
		functionRegistry, resourceRegistry, dataSourceRegistry, customVarTypeRegistry,
		nil, // blueprintLoader
		completionService, diagnosticService, signatureService, hoverService,
		symbolService, gotoDefinitionService, nil, /* findReferencesService */
		codeActionService,
		nil, // childResolver
		make(map[string]provider.Provider), make(map[string]transform.SpecTransformer),
		nil, s.logger,
		debouncer,
	)
}

type testServerContext struct {
	clientLSPCtx *common.LSPContext
	cancel       context.CancelFunc
}

func (s *ApplicationSuite) defaultClientCaps() lsp.ClientCapabilities {
	return lsp.ClientCapabilities{
		General: &lsp.GeneralClientCapabilities{
			PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
		},
	}
}

func (s *ApplicationSuite) createInitializedServer(
	caps lsp.ClientCapabilities,
) *testServerContext {
	app := s.createTestApplication()
	app.Setup()
	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())
	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	clientLSPCtx := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{Capabilities: caps}
	var result lsp.InitializeResult
	err := clientLSPCtx.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	return &testServerContext{
		clientLSPCtx: clientLSPCtx,
		cancel:       cancel,
	}
}

// Tests

func (s *ApplicationSuite) TestInitialize_ReturnsServerCapabilities() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
			},
		},
	}

	var result lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	s.Equal(Name, result.ServerInfo.Name)
	s.NotNil(result.Capabilities.SignatureHelpProvider)
	s.NotNil(result.Capabilities.CompletionProvider)
	s.NotNil(result.Capabilities.CodeActionProvider)
}

func (s *ApplicationSuite) TestInitialize_SetsPositionEncoding() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF32},
			},
		},
	}

	var result lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	s.Equal(lsp.PositionEncodingKindUTF32, result.Capabilities.PositionEncoding)
}

func (s *ApplicationSuite) TestInitialize_SetsWorkspaceFolderCapability() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
			},
			Workspace: &lsp.ClientWorkspaceCapabilities{
				WorkspaceFolders: &lsp.True,
			},
		},
	}

	var result lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	s.NotNil(result.Capabilities.Workspace)
	s.NotNil(result.Capabilities.Workspace.WorkspaceFolders)
	s.True(*result.Capabilities.Workspace.WorkspaceFolders.Supported)
}

// Note: Tests for didOpen/didChange that require full diagnostic processing
// are omitted here as they require a configured blueprint loader.
// The handlers_test.go covers the document content storage logic directly.

func (s *ApplicationSuite) TestShutdown_Succeeds() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	// Initialize first
	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
			},
		},
	}
	var initResult lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &initResult)
	s.Require().NoError(err)

	// Shutdown
	err = clientLSPContext.Call(lsp.MethodShutdown, nil, nil)
	s.Require().NoError(err)
}

func (s *ApplicationSuite) TestCompletionProvider_ConfiguredCorrectly() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
			},
		},
	}

	var result lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	s.NotNil(result.Capabilities.CompletionProvider)
	s.Contains(result.Capabilities.CompletionProvider.TriggerCharacters, "{")
	s.Contains(result.Capabilities.CompletionProvider.TriggerCharacters, ".")
	s.True(*result.Capabilities.CompletionProvider.ResolveProvider)
}

func (s *ApplicationSuite) TestSignatureHelpProvider_ConfiguredCorrectly() {
	app := s.createTestApplication()
	app.Setup()

	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())

	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientLSPContext := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{
			General: &lsp.GeneralClientCapabilities{
				PositionEncodings: []lsp.PositionEncodingKind{lsp.PositionEncodingKindUTF16},
			},
		},
	}

	var result lsp.InitializeResult
	err := clientLSPContext.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)

	s.NotNil(result.Capabilities.SignatureHelpProvider)
	s.Contains(result.Capabilities.SignatureHelpProvider.TriggerCharacters, "(")
	s.Contains(result.Capabilities.SignatureHelpProvider.TriggerCharacters, ",")
}

// Initialize capability tests

func (s *ApplicationSuite) TestInitialize_SetsConfigCapability() {
	caps := s.defaultClientCaps()
	caps.Workspace = &lsp.ClientWorkspaceCapabilities{
		Configuration: &lsp.True,
	}
	srvCtx := s.createInitializedServer(caps)
	defer srvCtx.cancel()

	// Send initialized notification to exercise handleInitialised config branch
	err := srvCtx.clientLSPCtx.Notify(lsp.MethodInitialized, &lsp.InitializedParams{})
	s.Require().NoError(err)
}

func (s *ApplicationSuite) TestInitialize_SetsHierarchicalDocSymbolCapability() {
	caps := s.defaultClientCaps()
	caps.TextDocument = &lsp.TextDocumentClientCapabilities{
		DocumentSymbol: &lsp.DocumentSymbolClientCapabilities{
			HierarchicalDocumentSymbolSupport: &lsp.True,
		},
	}
	srvCtx := s.createInitializedServer(caps)
	defer srvCtx.cancel()
}

func (s *ApplicationSuite) TestInitialize_SetsLinkSupportCapability() {
	caps := s.defaultClientCaps()
	caps.TextDocument = &lsp.TextDocumentClientCapabilities{
		Definition: &lsp.DefinitionClientCapabilities{
			LinkSupport: &lsp.True,
		},
	}
	srvCtx := s.createInitializedServer(caps)
	defer srvCtx.cancel()
}

func (s *ApplicationSuite) TestInitialize_WithDiagnosticInitOptions() {
	app := s.createTestApplication()
	app.Setup()
	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())
	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	clientLSPCtx := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: s.defaultClientCaps(),
		InitializationOptions: map[string]interface{}{
			"diagnostics": map[string]interface{}{
				"showAnyTypeWarnings": false,
			},
		},
	}
	var result lsp.InitializeResult
	err := clientLSPCtx.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)
	s.NotNil(result.Capabilities.CompletionProvider)
}

func (s *ApplicationSuite) TestInitialize_NilInitOptions_NoError() {
	app := s.createTestApplication()
	app.Setup()
	srv := server.NewServer(app.Handler(), true, nil, nil)
	container := createTestConnectionsContainer(srv.NewHandler())
	go srv.Serve(container.serverConn, s.logger)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	clientLSPCtx := server.NewLSPContext(ctx, container.clientConn, nil)

	initParams := lsp.InitializeParams{
		Capabilities: s.defaultClientCaps(),
	}
	var result lsp.InitializeResult
	err := clientLSPCtx.Call(lsp.MethodInitialize, initParams, &result)
	s.Require().NoError(err)
	s.NotNil(result.Capabilities.CompletionProvider)
}

// Handler nil-guard tests

func (s *ApplicationSuite) TestHover_NoDocument_ReturnsNull() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	var result *lsp.Hover
	err := srvCtx.clientLSPCtx.Call(lsp.MethodHover, lsp.HoverParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///unknown.yaml"},
			Position:     lsp.Position{Line: 0, Character: 0},
		},
	}, &result)
	s.Require().NoError(err)
	s.Nil(result)
}

func (s *ApplicationSuite) TestCompletion_NoDocument_ReturnsEmptyList() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	var result lsp.CompletionList
	err := srvCtx.clientLSPCtx.Call(lsp.MethodCompletion, lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///unknown.yaml"},
			Position:     lsp.Position{Line: 0, Character: 0},
		},
	}, &result)
	s.Require().NoError(err)
	s.Empty(result.Items)
}

func (s *ApplicationSuite) TestDocumentSymbol_NoDocument_ReturnsEmpty() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	var result []lsp.DocumentSymbol
	err := srvCtx.clientLSPCtx.Call(lsp.MethodDocumentSymbol, lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///unknown.yaml"},
	}, &result)
	s.Require().NoError(err)
	s.Empty(result)
}

func (s *ApplicationSuite) TestGotoDefinition_NoDocument_ReturnsEmpty() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	var result []lsp.Location
	err := srvCtx.clientLSPCtx.Call(lsp.MethodGotoDefinition, lsp.DefinitionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///unknown.yaml"},
			Position:     lsp.Position{Line: 0, Character: 0},
		},
	}, &result)
	s.Require().NoError(err)
	s.Empty(result)
}

func (s *ApplicationSuite) TestCodeAction_NoDocument_ReturnsEmpty() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	var result []lsp.CodeActionOrCommand
	err := srvCtx.clientLSPCtx.Call(lsp.MethodCodeAction, lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///unknown.yaml"},
		Range:        lsp.Range{},
		Context:      lsp.CodeActionContext{Diagnostics: []lsp.Diagnostic{}},
	}, &result)
	s.Require().NoError(err)
	s.Empty(result)
}

// CompletionItemResolve protocol tests

func (s *ApplicationSuite) TestCompletionResolve_NilData_ReturnsUnchanged() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	item := lsp.CompletionItem{Label: "test"}
	var result lsp.CompletionItem
	err := srvCtx.clientLSPCtx.Call(lsp.MethodCompletionItemResolve, item, &result)
	s.Require().NoError(err)
	s.Equal("test", result.Label)
	s.Nil(result.Documentation)
}

func (s *ApplicationSuite) TestCompletionResolve_NonMapData_ReturnsUnchanged() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	item := lsp.CompletionItem{Label: "test", Data: "not-a-map"}
	var result lsp.CompletionItem
	err := srvCtx.clientLSPCtx.Call(lsp.MethodCompletionItemResolve, item, &result)
	s.Require().NoError(err)
	s.Equal("test", result.Label)
	s.Nil(result.Documentation)
}

func (s *ApplicationSuite) TestCompletionResolve_NoCompletionType_ReturnsUnchanged() {
	srvCtx := s.createInitializedServer(s.defaultClientCaps())
	defer srvCtx.cancel()

	item := lsp.CompletionItem{
		Label: "test",
		Data:  map[string]any{"someKey": "someValue"},
	}
	var result lsp.CompletionItem
	err := srvCtx.clientLSPCtx.Call(lsp.MethodCompletionItemResolve, item, &result)
	s.Require().NoError(err)
	s.Equal("test", result.Label)
	s.Nil(result.Documentation)
}

func TestApplicationSuite(t *testing.T) {
	suite.Run(t, new(ApplicationSuite))
}
