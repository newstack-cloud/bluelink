package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// Converts a blueprint source range to an LSP range.
//
// Blueprint ranges may be open ended: the tree's root node, for example,
// deliberately carries no end position because the last leaf's range runs to
// the end of the document. Such a range collapses to a zero width range at its
// start rather than being reported with an incorrect end.
func rangeToLSPRange(bpRange *source.Range) *lsp.Range {
	if bpRange == nil || bpRange.Start == nil {
		return nil
	}

	start := lsp.Position{
		Line:      uint32(bpRange.Start.Line - 1),
		Character: uint32(bpRange.Start.Column - 1),
	}

	end := start
	if bpRange.End != nil {
		end = lsp.Position{
			Line:      uint32(bpRange.End.Line - 1),
			Character: uint32(bpRange.End.Column - 1),
		}
	}

	return &lsp.Range{Start: start, End: end}
}
