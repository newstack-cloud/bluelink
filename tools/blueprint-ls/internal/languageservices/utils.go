package languageservices

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// safeContext extracts a context.Context from an LSPContext, falling back
// to context.Background() if the Context field is nil.
// The ls-builder library may not always set the Context field.
func safeContext(lspCtx *common.LSPContext) context.Context {
	if lspCtx != nil && lspCtx.Context != nil {
		return lspCtx.Context
	}
	return context.Background()
}

func rangeToLSPRange(bpRange *source.Range) *lsp.Range {
	if bpRange == nil {
		return nil
	}

	return &lsp.Range{
		Start: lsp.Position{
			Line:      uint32(bpRange.Start.Line - 1),
			Character: uint32(bpRange.Start.Column - 1),
		},
		End: lsp.Position{
			Line:      uint32(bpRange.End.Line - 1),
			Character: uint32(bpRange.End.Column - 1),
		},
	}
}
