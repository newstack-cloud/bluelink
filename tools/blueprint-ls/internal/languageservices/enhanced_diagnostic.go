package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// EnhancedDiagnostic pairs an LSP diagnostic with its source error context
// for use in generating code actions. The error context contains metadata
// like typo suggestions, allowed values, and missing field names that
// can be used to provide quick fixes.
type EnhancedDiagnostic struct {
	// Diagnostic is the standard LSP diagnostic sent to the client
	Diagnostic lsp.Diagnostic

	// ErrorContext contains structured information for error resolution
	// including suggested actions, reason codes, and metadata
	ErrorContext *errors.ErrorContext

	// Original error location fields for precise text edits
	// These are 1-indexed (matching blueprint error conventions)
	Line       *int
	Column     *int
	EndLine    *int
	EndColumn  *int
}
