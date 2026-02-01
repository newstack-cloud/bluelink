package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type StateSuite struct {
	suite.Suite
	state *State
}

func (s *StateSuite) SetupTest() {
	s.state = NewState()
}

func (s *StateSuite) Test_workspace_folder_capability() {
	s.False(s.state.HasWorkspaceFolderCapability())
	s.state.SetWorkspaceFolderCapability(true)
	s.True(s.state.HasWorkspaceFolderCapability())
	s.state.SetWorkspaceFolderCapability(false)
	s.False(s.state.HasWorkspaceFolderCapability())
}

func (s *StateSuite) Test_configuration_capability() {
	s.False(s.state.HasConfigurationCapability())
	s.state.SetConfigurationCapability(true)
	s.True(s.state.HasConfigurationCapability())
}

func (s *StateSuite) Test_hierarchical_document_symbol_capability() {
	s.False(s.state.HasHierarchicalDocumentSymbolCapability())
	s.state.SetHierarchicalDocumentSymbolCapability(true)
	s.True(s.state.HasHierarchicalDocumentSymbolCapability())
}

func (s *StateSuite) Test_link_support_capability() {
	s.False(s.state.HasLinkSupportCapability())
	s.state.SetLinkSupportCapability(true)
	s.True(s.state.HasLinkSupportCapability())
}

func (s *StateSuite) Test_position_encoding_kind() {
	s.state.SetPositionEncodingKind(lsp.PositionEncodingKindUTF16)
	s.Equal(lsp.PositionEncodingKindUTF16, s.state.GetPositionEncodingKind())

	s.state.SetPositionEncodingKind(lsp.PositionEncodingKindUTF32)
	s.Equal(lsp.PositionEncodingKindUTF32, s.state.GetPositionEncodingKind())
}

func (s *StateSuite) Test_document_content_get_nonexistent_returns_nil() {
	s.Nil(s.state.GetDocumentContent("file:///missing.yaml"))
}

func (s *StateSuite) Test_document_content_set_and_get() {
	s.state.SetDocumentContent("file:///test.yaml", "version: 2024\n")
	content := s.state.GetDocumentContent("file:///test.yaml")
	s.Require().NotNil(content)
	s.Equal("version: 2024\n", *content)
}

func (s *StateSuite) Test_document_context_get_nonexistent_returns_nil() {
	s.Nil(s.state.GetDocumentContext("file:///missing.yaml"))
}

func (s *StateSuite) Test_document_context_set_nil_is_noop() {
	s.state.SetDocumentContext("file:///test.yaml", nil)
	s.Nil(s.state.GetDocumentContext("file:///test.yaml"))
}

func (s *StateSuite) Test_document_context_set_and_get() {
	docCtx := docmodel.NewDocumentContextFromSchema(
		"file:///test.yaml",
		nil,
		nil,
	)
	s.state.SetDocumentContext("file:///test.yaml", docCtx)
	retrieved := s.state.GetDocumentContext("file:///test.yaml")
	s.NotNil(retrieved)
}

func (s *StateSuite) Test_document_schema_returns_nil_for_nonexistent() {
	s.Nil(s.state.GetDocumentSchema("file:///missing.yaml"))
}

func (s *StateSuite) Test_document_schema_returns_blueprint_from_context() {
	bp, err := schema.LoadString("version: 2024-01-01\n", schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(bp)

	docCtx := docmodel.NewDocumentContextFromSchema("file:///test.yaml", bp, tree)
	s.state.SetDocumentContext("file:///test.yaml", docCtx)

	retrieved := s.state.GetDocumentSchema("file:///test.yaml")
	s.NotNil(retrieved)
}

func (s *StateSuite) Test_document_tree_returns_nil_for_nonexistent() {
	s.Nil(s.state.GetDocumentTree("file:///missing.yaml"))
}

func (s *StateSuite) Test_document_tree_returns_tree_from_context() {
	bp, err := schema.LoadString("version: 2024-01-01\n", schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(bp)

	docCtx := docmodel.NewDocumentContextFromSchema("file:///test.yaml", bp, tree)
	s.state.SetDocumentContext("file:///test.yaml", docCtx)

	retrieved := s.state.GetDocumentTree("file:///test.yaml")
	s.NotNil(retrieved)
}

func (s *StateSuite) Test_document_settings_get_nonexistent_returns_nil() {
	s.Nil(s.state.GetDocumentSettings("file:///missing.yaml"))
}

func (s *StateSuite) Test_document_settings_set_nil_is_noop() {
	s.state.SetDocumentSettings("file:///test.yaml", nil)
	s.Nil(s.state.GetDocumentSettings("file:///test.yaml"))
}

func (s *StateSuite) Test_document_settings_set_and_get() {
	settings := &DocSettings{MaxNumberOfProblems: 50}
	s.state.SetDocumentSettings("file:///test.yaml", settings)
	retrieved := s.state.GetDocumentSettings("file:///test.yaml")
	s.Require().NotNil(retrieved)
	s.Equal(50, retrieved.MaxNumberOfProblems)
}

func (s *StateSuite) Test_clear_doc_settings() {
	s.state.SetDocumentSettings("file:///a.yaml", &DocSettings{MaxNumberOfProblems: 10})
	s.state.SetDocumentSettings("file:///b.yaml", &DocSettings{MaxNumberOfProblems: 20})
	s.NotNil(s.state.GetDocumentSettings("file:///a.yaml"))

	s.state.ClearDocSettings()
	s.Nil(s.state.GetDocumentSettings("file:///a.yaml"))
	s.Nil(s.state.GetDocumentSettings("file:///b.yaml"))
}

func (s *StateSuite) Test_enhanced_diagnostics_get_nonexistent_returns_nil() {
	s.Nil(s.state.GetEnhancedDiagnostics("file:///missing.yaml"))
}

func (s *StateSuite) Test_enhanced_diagnostics_set_and_get() {
	diagnostics := []*EnhancedDiagnostic{
		{Diagnostic: lsp.Diagnostic{Message: "test error"}},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", diagnostics)
	retrieved := s.state.GetEnhancedDiagnostics("file:///test.yaml")
	s.Require().Len(retrieved, 1)
	s.Equal("test error", retrieved[0].Diagnostic.Message)
}

func TestStateSuite(t *testing.T) {
	suite.Run(t, new(StateSuite))
}
