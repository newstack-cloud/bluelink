package languageservices

import (
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// State holds the state shared between language server services
// to provide functionality for working with blueprint documents.
type State struct {
	hasWorkspaceFolderCapability            bool
	hasConfigurationCapability              bool
	hasHierarchicalDocumentSymbolCapability bool
	hasLinkSupportCapability                bool
	documentSettings                        map[string]*DocSettings
	documentContent                         map[string]string
	documentContexts                        map[string]*docmodel.DocumentContext
	positionEncodingKind                    lsp.PositionEncodingKind
	lock                                    sync.Mutex
}

// NewState creates a new instance of the state service
// for the language server.
func NewState() *State {
	return &State{
		documentSettings: make(map[string]*DocSettings),
		documentContent:  make(map[string]string),
		documentContexts: make(map[string]*docmodel.DocumentContext),
	}
}

// DocSettings holds settings for a document.
type DocSettings struct {
	Trace               DocTraceSettings `json:"trace"`
	MaxNumberOfProblems int              `json:"maxNumberOfProblems"`
}

// DocTraceSettings holds settings for tracing in a document.
type DocTraceSettings struct {
	Server string `json:"server"`
}

// HasWorkspaceFolderCapability returns true if the language server
// has the capability to handle workspace folders.
func (s *State) HasWorkspaceFolderCapability() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.hasWorkspaceFolderCapability
}

// SetWorkspaceFolderCapability sets the capability to handle workspace folders.
func (s *State) SetWorkspaceFolderCapability(value bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.hasWorkspaceFolderCapability = value
}

// HasConfigurationCapability returns true if the language server
// has the capability to handle configuration.
func (s *State) HasConfigurationCapability() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.hasConfigurationCapability
}

// SetConfigurationCapability sets the capability to handle configuration.
func (s *State) SetConfigurationCapability(value bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.hasConfigurationCapability = value
}

// HasHierarchicalDocumentSymbolCapability returns true if the language server
// has the capability to handle hierarchical document symbols.
func (s *State) HasHierarchicalDocumentSymbolCapability() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.hasHierarchicalDocumentSymbolCapability
}

// SetHierarchicalDocumentSymbolCapability sets the capability to handle hierarchical document symbols.
func (s *State) SetHierarchicalDocumentSymbolCapability(value bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.hasHierarchicalDocumentSymbolCapability = value
}

// HasLinkSupportCapability returns true if the language server
// has the capability to handle links using the LocationLink result type.
func (s *State) HasLinkSupportCapability() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.hasLinkSupportCapability
}

// SetLinkSupportCapability sets the capability to handle links using the LocationLink result type.
func (s *State) SetLinkSupportCapability(value bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.hasLinkSupportCapability = value
}

// SetPositionEncodingKind sets the encoding kind for positions in documents
// as specified by the client.
func (s *State) SetPositionEncodingKind(value lsp.PositionEncodingKind) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.positionEncodingKind = value
}

// GetPositionEncodingKind returns the encoding kind for positions in documents
// as specified by the client.
func (s *State) GetPositionEncodingKind() lsp.PositionEncodingKind {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.positionEncodingKind
}

// GetDocumentContent retrieves the content of a document by its URI.
func (s *State) GetDocumentContent(uri string) *string {
	s.lock.Lock()
	defer s.lock.Unlock()
	content, ok := s.documentContent[uri]
	if !ok {
		return nil
	}
	return &content
}

// SetDocumentContent sets the content of a document by its URI.
func (s *State) SetDocumentContent(uri string, content string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentContent[uri] = content
}

// GetDocumentContext retrieves the DocumentContext for a document by its URI.
func (s *State) GetDocumentContext(uri string) *docmodel.DocumentContext {
	s.lock.Lock()
	defer s.lock.Unlock()
	ctx, ok := s.documentContexts[uri]
	if !ok {
		return nil
	}
	return ctx
}

// SetDocumentContext sets the DocumentContext for a document by its URI.
func (s *State) SetDocumentContext(uri string, ctx *docmodel.DocumentContext) {
	if ctx == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentContexts[uri] = ctx
}

// GetDocumentSchema retrieves the parsed blueprint schema for a document by its URI.
func (s *State) GetDocumentSchema(uri string) *schema.Blueprint {
	s.lock.Lock()
	defer s.lock.Unlock()
	ctx, ok := s.documentContexts[uri]
	if !ok || ctx == nil {
		return nil
	}
	return ctx.Blueprint
}

// GetDocumentTree retrieves the document tree for a document by its URI.
func (s *State) GetDocumentTree(uri string) *schema.TreeNode {
	s.lock.Lock()
	defer s.lock.Unlock()
	ctx, ok := s.documentContexts[uri]
	if !ok || ctx == nil {
		return nil
	}
	return ctx.SchemaTree
}

// GetDocumentSettings retrieves the settings for a document by its URI.
func (s *State) GetDocumentSettings(uri string) *DocSettings {
	s.lock.Lock()
	defer s.lock.Unlock()
	settings, ok := s.documentSettings[uri]
	if !ok {
		return nil
	}
	return settings
}

// SetDocumentSettings sets the settings for a document by its URI.
func (s *State) SetDocumentSettings(uri string, settings *DocSettings) {
	if settings == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentSettings[uri] = settings
}

// ClearDocSettings clears settings for all documents.
func (s *State) ClearDocSettings() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentSettings = make(map[string]*DocSettings)
}
