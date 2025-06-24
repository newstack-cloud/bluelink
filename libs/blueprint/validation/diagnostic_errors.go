package validation

import (
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

// ExtractDiagnosticsAndErrors extracts diagnostics and errors from the provided diagnostics slice.
// It returns a slice of diagnostics that are not errors, and a wrapper error for multiple
// validation errors assigned the given reason code if any are found.
func ExtractDiagnosticsAndErrors(
	diagnostics []*core.Diagnostic,
	errorReasonCode bperrors.ErrorReasonCode,
) ([]*core.Diagnostic, error) {
	var nonErrorDiagnostics []*core.Diagnostic
	var errs []error
	for _, diagnostic := range diagnostics {
		if diagnostic.Level != core.DiagnosticLevelError {
			nonErrorDiagnostics = append(nonErrorDiagnostics, diagnostic)
		} else {
			errs = append(errs, validationErrorFromDiagnostic(
				diagnostic,
				errorReasonCode,
			))
		}
	}

	if len(errs) > 0 {
		return nonErrorDiagnostics, ErrMultipleValidationErrors(errs)
	}

	return nonErrorDiagnostics, nil
}

func validationErrorFromDiagnostic(
	diagnostic *core.Diagnostic,
	errorReasonCode bperrors.ErrorReasonCode,
) error {
	line, column := getDiagnosticLineAndColumn(diagnostic)
	return &bperrors.LoadError{
		ReasonCode: errorReasonCode,
		Err:        errors.New(diagnostic.Message),
		Line:       &line,
		Column:     &column,
	}
}

func getDiagnosticLineAndColumn(diagnostic *core.Diagnostic) (int, int) {
	if diagnostic.Range == nil ||
		diagnostic.Range.Start == nil {
		return 1, 1
	}

	start := diagnostic.Range.Start
	return start.Line, start.Column
}
