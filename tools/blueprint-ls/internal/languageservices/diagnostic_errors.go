package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// DiagnosticErrorService is a service that provides functionality
// for converting validation errors into LSP diagnostics.
type DiagnosticErrorService struct {
	state  *State
	logger *zap.Logger
}

// NewDiagnosticErrorService creates a new service for
// converting validation errors into LSP diagnostics.
func NewDiagnosticErrorService(
	state *State,
	logger *zap.Logger,
) *DiagnosticErrorService {
	return &DiagnosticErrorService{
		state,
		logger,
	}
}

func (s *DiagnosticErrorService) BlueprintErrorToDiagnostics(
	err error,
	docURI lsp.URI,
) []lsp.Diagnostic {
	diagnostics := []lsp.Diagnostic{}

	loadErr, isLoadErr := err.(*errors.LoadError)
	if isLoadErr {
		s.collectLoadErrors(loadErr, &diagnostics, nil, docURI)
		return diagnostics
	}

	schemaErr, isSchemaErr := err.(*schema.Error)
	if isSchemaErr {
		s.collectSchemaError(schemaErr, &diagnostics, docURI)
		return diagnostics
	}

	coreErr, isCoreErr := err.(*core.Error)
	if isCoreErr {
		s.collectCoreError(coreErr, &diagnostics, nil, docURI)
		return diagnostics
	}

	_, isRunErr := err.(*errors.RunError)
	if isRunErr {
		// Skip capturing run errors during validation,
		// they are useful at runtime but may appear during validation
		// in loading provider and transformer plugins.
		return diagnostics
	}

	return getGeneralErrorDiagnostics(err)
}

func getGeneralErrorDiagnostics(err error) []lsp.Diagnostic {
	severity := lsp.DiagnosticSeverityError
	return []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      0,
					Character: 0,
				},
				End: lsp.Position{
					Line:      1,
					Character: 0,
				},
			},
			Severity: &severity,
			Message:  err.Error(),
		},
	}
}

func (s *DiagnosticErrorService) collectLoadErrors(
	err *errors.LoadError,
	diagnostics *[]lsp.Diagnostic,
	parentLoadErr *errors.LoadError,
	docURI lsp.URI,
) {
	for _, childErr := range err.ChildErrors {
		childLoadErr, isLoadErr := childErr.(*errors.LoadError)
		if isLoadErr {
			s.collectLoadErrors(childLoadErr, diagnostics, err, docURI)
		}

		childSchemaErr, isSchemaErr := childErr.(*schema.Error)
		if isSchemaErr {
			s.collectSchemaError(childSchemaErr, diagnostics, docURI)
		}

		childParseErrs, isParseErrs := childErr.(*substitutions.ParseErrors)
		if isParseErrs {
			s.collectParseErrors(childParseErrs, diagnostics, err, docURI)
		}

		childParseErr, isParseErr := childErr.(*substitutions.ParseError)
		if isParseErr {
			s.collectParseError(childParseErr, diagnostics, docURI)
		}

		childCoreErr, isCoreErr := childErr.(*core.Error)
		if isCoreErr {
			s.collectCoreError(childCoreErr, diagnostics, err, docURI)
		}

		childLexErrors, isLexErrs := childErr.(*substitutions.LexErrors)
		if isLexErrs {
			s.collectLexErrors(childLexErrors, diagnostics, err, docURI)
		}

		childLexError, isLexErr := childErr.(*substitutions.LexError)
		if isLexErr {
			s.collectLexError(childLexError, diagnostics, docURI)
		}

		_, isRunErr := childErr.(*errors.RunError)

		// Skip capturing run errors during validation,
		// they are useful at runtime but may appear during validation
		// in loading provider and transformer plugins.
		if !isRunErr && !isLoadErr && !isParseErrs && !isParseErr &&
			!isCoreErr && !isLexErrs && !isLexErr {
			s.collectGeneralError(childErr, diagnostics, err, docURI)
		}
	}

	if len(err.ChildErrors) == 0 {
		severity := lsp.DiagnosticSeverityError
		source := "blueprint-validator"

		// Build enhanced message with context if available
		message := err.Err.Error()
		if err.Context != nil {
			message = formatLoadErrorWithContext(err)
		}

		diag := lsp.Diagnostic{
			Range: s.rangeFromBlueprintErrorLocation(
				&blueprintErrorLocationLoadErr{err},
				&blueprintErrorLocationLoadErr{parentLoadErr},
				docURI,
			),
			Severity: &severity,
			Message:  message,
			Source:   &source,
		}

		// Add Code if ReasonCode is available
		if err.ReasonCode != "" {
			code := string(err.ReasonCode)
			diag.Code = &lsp.IntOrString{StrVal: &code}
		}

		*diagnostics = append(*diagnostics, diag)
	}
}

// formatLoadErrorWithContext formats a LoadError with its ErrorContext,
// including suggested actions in a plain text format suitable for VS Code diagnostics.
func formatLoadErrorWithContext(err *errors.LoadError) string {
	if err.Context == nil {
		return err.Err.Error()
	}

	sb := strings.Builder{}
	sb.WriteString(err.Err.Error())

	// Add typo suggestions if available (from schema validation)
	if err.Context.Metadata != nil {
		if suggestions, ok := err.Context.Metadata["suggestions"].([]string); ok && len(suggestions) > 0 {
			sb.WriteString("\n\nDid you mean: ")
			sb.WriteString(strings.Join(suggestions, ", "))
			sb.WriteString("?")
		}

		// Add available fields hint
		if fields, ok := err.Context.Metadata["availableFields"].([]string); ok && len(fields) > 0 {
			sb.WriteString("\n\nAvailable fields: ")
			if len(fields) <= 8 {
				sb.WriteString(strings.Join(fields, ", "))
			} else {
				// Show first 8 fields with ellipsis for large schemas
				sb.WriteString(strings.Join(fields[:8], ", "))
				sb.WriteString(fmt.Sprintf(", ... (%d more)", len(fields)-8))
			}
		}
	}

	if len(err.Context.SuggestedActions) > 0 {
		sb.WriteString("\n\nSuggested Actions:\n")
		for i, action := range err.Context.SuggestedActions {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, action.Title))
			if action.Description != "" {
				sb.WriteString(fmt.Sprintf(": %s", action.Description))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (s *DiagnosticErrorService) rangeFromBlueprintErrorLocation(
	location blueprintErrorLocation,
	parentLocation blueprintErrorLocation,
	docURI lsp.URI,
) lsp.Range {
	errMissingLocation := location.Line() == nil && location.Column() == nil
	parentMissingLocation := parentLocation == nil ||
		parentLocation.Line() == nil && parentLocation.Column() == nil

	if errMissingLocation && parentMissingLocation {
		return lsp.Range{
			Start: lsp.Position{
				Line:      0,
				Character: 0,
			},
			End: lsp.Position{
				Line:      1,
				Character: 0,
			},
		}
	}

	if errMissingLocation {
		return s.rangeFromBlueprintErrorLocation(
			parentLocation,
			nil,
			docURI,
		)
	}

	startPos := getStartErrorLocation(location)

	// LSP offsets are 0-based, the blueprint package uses 1-based offsets.
	start := lsp.Position{
		Line:      lsp.UInteger(startPos.Line - 1),
		Character: lsp.UInteger(startPos.Column - 1),
	}

	// Use end position if available for precise diagnostic ranges,
	// but only when column accuracy is exact. For approximate columns
	// (e.g., YAML block strings), the start position is unreliable
	// so we highlight the whole line instead.
	colAccuracy := location.ColumnAccuracy()
	hasExactColumn := !location.UseColumnAccuracy() ||
		(colAccuracy != nil && *colAccuracy == substitutions.ColumnAccuracyExact)

	endLine := location.EndLine()
	endCol := location.EndColumn()

	if hasExactColumn && endLine != nil && endCol != nil {
		return lsp.Range{
			Start: start,
			End: lsp.Position{
				Line:      lsp.UInteger(*endLine - 1),
				Character: lsp.UInteger(*endCol - 1),
			},
		}
	}

	// Fallback: highlight to end of next line when column is approximate
	// or no end position available.
	return lsp.Range{
		Start: start,
		End: lsp.Position{
			Line:      start.Line + 1,
			Character: 0,
		},
	}
}

func getStartErrorLocation(location blueprintErrorLocation) *source.Meta {
	line := location.Line()
	if line == nil {
		firstLine := 1
		line = &firstLine
	}

	col := 1
	colAccuracy := location.ColumnAccuracy()
	if location.UseColumnAccuracy() && colAccuracy != nil {
		if *colAccuracy == substitutions.ColumnAccuracyExact {
			colPtr := location.Column()
			if colPtr != nil {
				col = *colPtr
			}
		}
	} else if !location.UseColumnAccuracy() {
		colPtr := location.Column()
		if colPtr != nil {
			col = *colPtr
		}
	}

	return &source.Meta{
		Position: source.Position{
			Line:   *line,
			Column: col,
		},
	}
}

func (s *DiagnosticErrorService) collectParseErrors(
	errs *substitutions.ParseErrors,
	diagnostics *[]lsp.Diagnostic,
	parentLoadErr *errors.LoadError,
	docURI lsp.URI,
) {
	for _, childErr := range errs.ChildErrors {
		childParseErr, isParseError := childErr.(*substitutions.ParseError)
		if isParseError {
			s.collectParseError(childParseErr, diagnostics, docURI)
		}
	}

	if len(errs.ChildErrors) == 0 {
		severity := lsp.DiagnosticSeverityError
		*diagnostics = append(*diagnostics, lsp.Diagnostic{
			Range: s.rangeFromBlueprintErrorLocation(
				&blueprintErrorLocationLoadErr{parentLoadErr},
				nil,
				docURI,
			),
			Severity: &severity,
			Message:  errs.Error(),
		})
	}
}

func (s *DiagnosticErrorService) collectParseError(
	err *substitutions.ParseError,
	diagnostics *[]lsp.Diagnostic,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			&blueprintErrorLocationParseErr{err},
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}

func (s *DiagnosticErrorService) collectCoreError(
	err *core.Error,
	diagnostics *[]lsp.Diagnostic,
	parentLoadErr *errors.LoadError,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError

	var parentLocation blueprintErrorLocation
	if parentLoadErr != nil {
		parentLocation = &blueprintErrorLocationLoadErr{parentLoadErr}
	}

	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			&blueprintErrorLocationCoreErr{err},
			parentLocation,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}

func (s *DiagnosticErrorService) collectLexErrors(
	errs *substitutions.LexErrors,
	diagnostics *[]lsp.Diagnostic,
	parentLoadErr *errors.LoadError,
	docURI lsp.URI,
) {
	for _, childErr := range errs.ChildErrors {
		childLexErr, isLexError := childErr.(*substitutions.LexError)
		if isLexError {
			s.collectLexError(childLexErr, diagnostics, docURI)
		}
	}

	if len(errs.ChildErrors) == 0 {
		severity := lsp.DiagnosticSeverityError
		*diagnostics = append(*diagnostics, lsp.Diagnostic{
			Range: s.rangeFromBlueprintErrorLocation(
				&blueprintErrorLocationLoadErr{parentLoadErr},
				nil,
				docURI,
			),
			Severity: &severity,
			Message:  errs.Error(),
		})
	}
}

func (s *DiagnosticErrorService) collectLexError(
	err *substitutions.LexError,
	diagnostics *[]lsp.Diagnostic,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			&blueprintErrorLocationLexErr{err},
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}

func (s *DiagnosticErrorService) collectSchemaError(
	err *schema.Error,
	diagnostics *[]lsp.Diagnostic,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			&blueprintErrorLocationSchemaErr{err},
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}

func (s *DiagnosticErrorService) collectGeneralError(
	err error,
	diagnostics *[]lsp.Diagnostic,
	parentLoadError *errors.LoadError,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			&blueprintErrorLocationLoadErr{parentLoadError},
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}
