package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
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

// GetDocumentSymbolsFromContext returns symbols using the unified document model.
func (s *SymbolService) GetDocumentSymbolsFromContext(
	docCtx *docmodel.DocumentContext,
) ([]lsp.DocumentSymbol, error) {
	if docCtx == nil {
		return []lsp.DocumentSymbol{}, nil
	}

	ast := docCtx.GetEffectiveAST()
	if ast == nil {
		return []lsp.DocumentSymbol{}, nil
	}

	totalLines := countLines(docCtx.Content)
	return docmodel.BuildDocumentSymbols(ast, totalLines), nil
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	return len(strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n"))
}
