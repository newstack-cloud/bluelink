package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// SymbolService is a service that provides functionality
// for document symbols for an LSP client.
type SymbolService struct {
	state  *State
	logger *zap.Logger
}

// NewSymbolService creates a new service for document symbols.
func NewSymbolService(
	state *State,
	logger *zap.Logger,
) *SymbolService {
	return &SymbolService{
		state,
		logger,
	}
}

// GetDocumentSymbols returns the symbols in the given blueprint schema.
func (s *SymbolService) GetDocumentSymbols(
	docURI lsp.URI,
	content string,
) ([]lsp.DocumentSymbol, error) {
	format := blueprint.DetermineDocFormat(docURI)
	if format == schema.JWCCSpecFormat {
		return s.getJSONDocumentSymbols(docURI, content)
	}

	return s.getYAMLDocumentSymbols(docURI, content)
}
