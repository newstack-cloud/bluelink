package languageservices

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/diagnostichelpers"
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

// BlueprintErrorToDiagnostics converts a blueprint error into LSP diagnostics.
// It returns both the standard LSP diagnostics and enhanced diagnostics that
// include error context metadata for use in code actions.
func (s *DiagnosticErrorService) BlueprintErrorToDiagnostics(
	err error,
	docURI lsp.URI,
) ([]lsp.Diagnostic, []*EnhancedDiagnostic) {
	diagnostics := []lsp.Diagnostic{}
	enhanced := []*EnhancedDiagnostic{}

	loadErr, isLoadErr := err.(*errors.LoadError)
	if isLoadErr {
		s.collectLoadErrors(loadErr, &diagnostics, &enhanced, nil, docURI)
		return diagnostics, enhanced
	}

	schemaErr, isSchemaErr := err.(*schema.Error)
	if isSchemaErr {
		s.collectSchemaError(schemaErr, &diagnostics, docURI)
		return diagnostics, enhanced
	}

	coreErr, isCoreErr := err.(*core.Error)
	if isCoreErr {
		s.collectCoreError(coreErr, &diagnostics, nil, docURI)
		return diagnostics, enhanced
	}

	_, isRunErr := err.(*errors.RunError)
	if isRunErr {
		// Skip capturing run errors during validation,
		// they are useful at runtime but may appear during validation
		// in loading provider and transformer plugins.
		return diagnostics, enhanced
	}

	if s.collectLangErrors(err, &diagnostics, docURI) {
		return diagnostics, enhanced
	}

	return s.getGeneralErrorDiagnostics(err, docURI), enhanced
}

// Handles the blueprint-language parse/lex error types, mapping
// each child's source.Meta to a positioned diagnostic. It returns true when err
// was one of the lang error types (and therefore fully handled).
func (s *DiagnosticErrorService) collectLangErrors(
	err error,
	diagnostics *[]lsp.Diagnostic,
	docURI lsp.URI,
) bool {
	switch typedErr := err.(type) {
	case *lang.Errors:
		for _, childErr := range typedErr.ChildErrors {
			s.collectLangErrors(childErr, diagnostics, docURI)
		}
		return true
	case *lang.LexErrors:
		for _, childErr := range typedErr.ChildErrors {
			s.collectLangErrors(childErr, diagnostics, docURI)
		}
		return true
	case *lang.ParseError:
		s.collectLangMetaError(typedErr.SourceMeta, typedErr.Error(), diagnostics, docURI)
		return true
	case *lang.LexError:
		s.collectLangMetaError(typedErr.SourceMeta, typedErr.Error(), diagnostics, docURI)
		return true
	default:
		return false
	}
}

func (s *DiagnosticErrorService) collectLangMetaError(
	meta *source.Meta,
	message string,
	diagnostics *[]lsp.Diagnostic,
	docURI lsp.URI,
) {
	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			blueprintErrorLocationLangMeta{meta},
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  message,
	})
}

func (s *DiagnosticErrorService) getGeneralErrorDiagnostics(
	err error,
	docURI lsp.URI,
) []lsp.Diagnostic {
	severity := lsp.DiagnosticSeverityError

	diagnostics := []lsp.Diagnostic{}
	for _, location := range s.resourceTypeLocations(err, docURI) {
		diagnostics = append(diagnostics, lsp.Diagnostic{
			Range:    s.rangeFromBlueprintErrorLocation(location, nil, docURI),
			Severity: &severity,
			Message:  err.Error(),
		})
	}

	if len(diagnostics) > 0 {
		return diagnostics
	}

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

// Errors that reach the general fallbacks often carry no location of their own,
// most commonly errors reported by a plugin for a resource type. Matching the error
// against the resource types declared in the document keeps the diagnostic on the
// resources that the error refers to instead of the start of the document.
func (s *DiagnosticErrorService) resourceTypeLocations(
	err error,
	docURI lsp.URI,
) []blueprintErrorLocation {
	blueprint := s.state.GetDocumentSchema(string(docURI))
	if blueprint == nil || blueprint.Resources == nil {
		return nil
	}

	message := err.Error()
	locations := []blueprintErrorLocation{}
	for _, resource := range blueprint.Resources.Values {
		if isResourceTypeReferencedInError(resource, message) {
			locations = append(
				locations,
				blueprintErrorLocationLangMeta{resource.Type.SourceMeta},
			)
		}
	}

	// Resources are held in a map, sort the locations so that the diagnostics
	// produced for a document are stable between validation runs.
	slices.SortFunc(locations, compareErrorLocations)

	return locations
}

func isResourceTypeReferencedInError(resource *schema.Resource, message string) bool {
	if resource == nil || resource.Type == nil ||
		resource.Type.Value == "" || resource.Type.SourceMeta == nil {
		return false
	}

	return strings.Contains(message, resource.Type.Value)
}

func compareErrorLocations(locationA, locationB blueprintErrorLocation) int {
	lineA, lineB := locationA.Line(), locationB.Line()
	if lineA != nil && lineB != nil && *lineA != *lineB {
		return cmp.Compare(*lineA, *lineB)
	}

	columnA, columnB := locationA.Column(), locationB.Column()
	if columnA != nil && columnB != nil {
		return cmp.Compare(*columnA, *columnB)
	}

	return 0
}

func (s *DiagnosticErrorService) collectLoadErrors(
	err *errors.LoadError,
	diagnostics *[]lsp.Diagnostic,
	enhanced *[]*EnhancedDiagnostic,
	parentLoadErr *errors.LoadError,
	docURI lsp.URI,
) {
	for _, childErr := range err.ChildErrors {
		childLoadErr, isLoadErr := childErr.(*errors.LoadError)
		if isLoadErr {
			s.collectLoadErrors(childLoadErr, diagnostics, enhanced, err, docURI)
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

		// A lang error may appear nested in a LoadError (e.g. a child blueprint
		// include parsed via lang); position it from its source.Meta.
		isLangErr := s.collectLangErrors(childErr, diagnostics, docURI)

		// Skip capturing run errors during validation,
		// they are useful at runtime but may appear during validation
		// in loading provider and transformer plugins.
		if !isRunErr && !isLoadErr && !isParseErrs && !isParseErr &&
			!isCoreErr && !isLexErrs && !isLexErr && !isLangErr {
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

		// Create enhanced diagnostic with error context for code actions
		if err.Context != nil {
			*enhanced = append(*enhanced, &EnhancedDiagnostic{
				Diagnostic:   diag,
				ErrorContext: err.Context,
				Line:         err.Line,
				Column:       err.Column,
				EndLine:      err.EndLine,
				EndColumn:    err.EndColumn,
			})
		} else if err.ReasonCode != "" {
			// Create a minimal ErrorContext from the ReasonCode for code actions
			// that only need the reason code (e.g., missing version)
			*enhanced = append(*enhanced, &EnhancedDiagnostic{
				Diagnostic: diag,
				ErrorContext: &errors.ErrorContext{
					ReasonCode: err.ReasonCode,
				},
				Line:      err.Line,
				Column:    err.Column,
				EndLine:   err.EndLine,
				EndColumn: err.EndColumn,
			})
		}
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

	diagnostichelpers.WriteSuggestions(&sb, err.Context.Metadata)

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
	parentLocation := &blueprintErrorLocationLoadErr{parentLoadError}
	if parentLocation.Line() == nil && parentLocation.Column() == nil {
		// The parent is usually the wrapper for multiple validation errors, which
		// carries no location of its own, leaving the diagnostic at the start of
		// the document unless it can be attributed to a resource.
		locatedDiagnostics := s.getGeneralErrorDiagnostics(err, docURI)
		*diagnostics = append(*diagnostics, locatedDiagnostics...)
		return
	}

	severity := lsp.DiagnosticSeverityError
	*diagnostics = append(*diagnostics, lsp.Diagnostic{
		Range: s.rangeFromBlueprintErrorLocation(
			parentLocation,
			nil,
			docURI,
		),
		Severity: &severity,
		Message:  err.Error(),
	})
}
