package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type SymbolServiceSuite struct {
	suite.Suite
	service          *SymbolService
	blueprintContent string
}

func (s *SymbolServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	s.service = NewSymbolService(state, logger)
	s.blueprintContent, err = loadTestBlueprintContent("blueprint-symbols.yaml")
	s.Require().NoError(err)
}

func (s *SymbolServiceSuite) Test_creates_document_symbol_hierarchy() {
	symbols, err := s.service.GetDocumentSymbols(blueprintURI, s.blueprintContent)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(symbols)
	s.Require().NoError(err)
}

func TestSymbolServiceSuite(t *testing.T) {
	suite.Run(t, new(SymbolServiceSuite))
}
