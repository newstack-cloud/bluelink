package languageservices

import (
	"sync"

	"github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"gopkg.in/yaml.v3"
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
	documentSchemas                         map[string]*schema.Blueprint
	// YAML document hierarchy representation to extract document symbols
	// from.
	documentYAMLNodes map[string]*yaml.Node
	// JSON document hierarchy representation to extract document symbols
	// from.
	documentJSONNodes    map[string]*json.Node
	documentPositionMaps map[string]map[string][]*schema.TreeNode
	documentTrees        map[string]*schema.TreeNode
	positionEncodingKind lsp.PositionEncodingKind
	lock                 sync.Mutex
}

// NewState creates a new instance of the state service
// for the language server.
func NewState() *State {
	return &State{
		documentSettings:     make(map[string]*DocSettings),
		documentContent:      make(map[string]string),
		documentSchemas:      make(map[string]*schema.Blueprint),
		documentPositionMaps: make(map[string]map[string][]*schema.TreeNode),
		documentYAMLNodes:    make(map[string]*yaml.Node),
		documentJSONNodes:    make(map[string]*json.Node),
		documentTrees:        make(map[string]*schema.TreeNode),
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

// GetDocumentSchema retrieves the parsed blueprint schema for a document by its URI.
func (s *State) GetDocumentSchema(uri string) *schema.Blueprint {
	s.lock.Lock()
	defer s.lock.Unlock()
	blueprint, ok := s.documentSchemas[uri]
	if !ok {
		return nil
	}
	return blueprint
}

// SetDocumentSchema sets the parsed blueprint schema for a document by its URI.
func (s *State) SetDocumentSchema(uri string, blueprint *schema.Blueprint) {
	if blueprint == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentSchemas[uri] = blueprint
}

// GetDocumentYAMLNode retrieves the YAML node for a document by its URI.
func (s *State) GetDocumentYAMLNode(uri string) *yaml.Node {
	s.lock.Lock()
	defer s.lock.Unlock()
	node, ok := s.documentYAMLNodes[uri]
	if !ok {
		return nil
	}
	return node
}

// SetDocumentYAMLNode sets the YAML node for a document by its URI.
func (s *State) SetDocumentYAMLNode(uri string, node *yaml.Node) {
	if node == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentYAMLNodes[uri] = node
}

// GetDocumentJSONNode retrieves the JSON node for a document by its URI.
func (s *State) GetDocumentJSONNode(uri string) *json.Node {
	s.lock.Lock()
	defer s.lock.Unlock()
	node, ok := s.documentJSONNodes[uri]
	if !ok {
		return nil
	}
	return node
}

// SetDocumentJSONNode sets the JSON node for a document by its URI.
func (s *State) SetDocumentJSONNode(uri string, node *json.Node) {
	if node == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentJSONNodes[uri] = node
}

// SetDocumentTree sets the document tree for a document by its URI.
// The tree is ordered in position from left to right with ranges assigned
// to each node in the tree.
// This makes it easier to match elements to positions in the document
// for features such as signature help and hover.
func (s *State) SetDocumentTree(uri string, tree *schema.TreeNode) {
	if tree == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentTrees[uri] = tree
}

// GetDocumentTree retrieves the document tree for a document by its URI.
func (s *State) GetDocumentTree(uri string) *schema.TreeNode {
	s.lock.Lock()
	defer s.lock.Unlock()
	tree, ok := s.documentTrees[uri]
	if !ok {
		return nil
	}

	return tree
}

// SetDocumentPositionMap sets the document position map for a document by its URI.
// This maps positions in the document to nodes in the document tree.
func (s *State) SetDocumentPositionMap(uri string, positionMap map[string][]*schema.TreeNode) {
	if positionMap == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.documentPositionMaps[uri] = positionMap
}

// GetDocumentPositionMapNodes retrieves the nodes in the document tree for a document by its URI
// at a given position.
func (s *State) GetDocumentPositionNodes(uri string, position *source.Position) []*schema.TreeNode {
	s.lock.Lock()
	defer s.lock.Unlock()
	positionMap, ok := s.documentPositionMaps[uri]
	if !ok {
		return nil
	}

	nodes, ok := positionMap[blueprint.PositionKey(position)]
	if !ok {
		return nil
	}

	return nodes
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
