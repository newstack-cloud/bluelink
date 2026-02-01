package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type SymbolServiceSuite struct {
	suite.Suite
	service          *SymbolService
	blueprintContent string
	logger           *zap.Logger
}

func (s *SymbolServiceSuite) SetupTest() {
	var err error
	s.logger, err = zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	s.service = NewSymbolService(state, s.logger)
	s.blueprintContent, err = loadTestBlueprintContent("blueprint-symbols.yaml")
	s.Require().NoError(err)
}

func (s *SymbolServiceSuite) Test_creates_document_symbol_hierarchy() {
	docCtx := docmodel.NewDocumentContext(
		string(blueprintURI),
		s.blueprintContent,
		docmodel.FormatYAML,
		s.logger,
	)
	symbols, err := s.service.GetDocumentSymbolsFromContext(docCtx)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(symbols)
	s.Require().NoError(err)
}

func (s *SymbolServiceSuite) Test_nil_doc_context_returns_empty() {
	symbols, err := s.service.GetDocumentSymbolsFromContext(nil)
	s.Require().NoError(err)
	s.Assert().Empty(symbols)
}

func (s *SymbolServiceSuite) Test_doc_context_with_nil_ast_returns_empty() {
	docCtx := docmodel.NewDocumentContextFromSchema(
		string(blueprintURI),
		nil,
		nil,
	)
	symbols, err := s.service.GetDocumentSymbolsFromContext(docCtx)
	s.Require().NoError(err)
	s.Assert().Empty(symbols)
}

func TestSymbolServiceSuite(t *testing.T) {
	suite.Run(t, new(SymbolServiceSuite))
}
