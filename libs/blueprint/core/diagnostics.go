package core

import (
	"github.com/two-hundred/celerity/libs/blueprint/source"
	"github.com/two-hundred/celerity/libs/blueprint/substitutions"
)

// Diagnostic provides error, warning or informational diagnostic for a blueprint.
// Blueprint validation will not use this for errors, but instead return an error,
// tools that use the blueprint framework can transform these errors into diagnostics,
// see the Blueprint Language Server or the Deploy Engine for examples.
type Diagnostic struct {
	// The level of this diagnostic.
	Level DiagnosticLevel `json:"level"`
	// The message of this diagnostic.
	Message string `json:"message"`
	// An optional text range in the source blueprint
	// that the diagnostic applies to.
	Range *DiagnosticRange `json:"range,omitempty"`
}

// DiagnosticLevel provides the level of a diagnostic.
type DiagnosticLevel int

const (
	// DiagnosticLevelError should be used for diagnostics that point out
	// errors in a blueprint.
	// This is not solely used as a source of errors, this should be combined
	// with unpacking a returned error to produce a set of error diagnostics
	// for tools that report diagnostics. (e.g. language servers)
	DiagnosticLevelError DiagnosticLevel = 1
	// DiagnosticLevelWarning should be used for diagnostics that point out
	// potential issues that may occur when executing a blueprint.
	DiagnosticLevelWarning DiagnosticLevel = 2
	// DiagnosticLevelInfo should be used for diagnostics that provide
	// informational messages about the blueprint that are worth noting
	// but do not indicate issues that may occur when executing a blueprint.
	DiagnosticLevelInfo DiagnosticLevel = 3
)

// DiagnosticRange provides a range in the source blueprint that a diagnostic applies to.
// This will only be used for source formats that allow position tracking of parsed nodes
// (i.e. YAML source documents).
type DiagnosticRange struct {
	Start *source.Meta `json:"start,omitempty"`
	End   *source.Meta `json:"end,omitempty"`
	// When the diagnostic is concerning contents of a ${..} substitution,
	// depending on the context, the column may not be accurate,
	// this gives you the option to ignore approximate columns in contexts
	// where they are likely to cause confusion for the end-user.
	// (e.g. language server diagnostics for a code editor)
	ColumnAccuracy *substitutions.ColumnAccuracy `json:"columnAccuracy,omitempty"`
}
