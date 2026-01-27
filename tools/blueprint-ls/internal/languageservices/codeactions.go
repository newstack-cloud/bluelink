package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// Reason codes for quick fixes
const (
	ReasonCodeResourceDefUnknownField         = "resource_def_unknown_field"
	ReasonCodeResourceDefMissingRequiredField = "resource_def_missing_required_field"
	ReasonCodeResourceDefNotAllowedValue      = "resource_def_not_allowed_value"
	ReasonCodeMissingVersion                  = "missing_version"
	ReasonCodeVariableValidationErrors        = "variable_validation_errors"
)

// CodeActionService provides functionality for generating LSP code actions
// (quick fixes) from enhanced diagnostics.
type CodeActionService struct {
	state  *State
	logger *zap.Logger
}

// NewCodeActionService creates a new service for generating code actions.
func NewCodeActionService(
	state *State,
	logger *zap.Logger,
) *CodeActionService {
	return &CodeActionService{
		state:  state,
		logger: logger,
	}
}

// GetCodeActions returns code actions for the given document and range.
// It filters enhanced diagnostics by the requested range and generates
// appropriate quick fix actions based on the diagnostic metadata.
func (s *CodeActionService) GetCodeActions(
	params *lsp.CodeActionParams,
) ([]lsp.CodeAction, error) {
	actions := []lsp.CodeAction{}

	enhanced := s.state.GetEnhancedDiagnostics(params.TextDocument.URI)
	if enhanced == nil {
		return actions, nil
	}

	// Filter diagnostics that overlap with the requested range
	for _, diag := range enhanced {
		if !rangesOverlap(diag.Diagnostic.Range, params.Range) {
			continue
		}

		// Generate code actions based on the error context
		diagActions := s.createActionsForDiagnostic(params.TextDocument.URI, diag)
		actions = append(actions, diagActions...)
	}

	return actions, nil
}

// createActionsForDiagnostic creates code actions for a single enhanced diagnostic.
func (s *CodeActionService) createActionsForDiagnostic(
	uri lsp.URI,
	diag *EnhancedDiagnostic,
) []lsp.CodeAction {
	actions := []lsp.CodeAction{}

	if diag.ErrorContext == nil {
		return actions
	}

	reasonCode := string(diag.ErrorContext.ReasonCode)

	switch reasonCode {
	case ReasonCodeResourceDefUnknownField:
		actions = append(actions, s.createTypoFixActions(uri, diag)...)
	case ReasonCodeResourceDefNotAllowedValue:
		actions = append(actions, s.createAllowedValueActions(uri, diag)...)
	case ReasonCodeMissingVersion:
		if action := s.createMissingVersionAction(uri, diag); action != nil {
			actions = append(actions, *action)
		}
	case ReasonCodeVariableValidationErrors:
		// Check if this is a missing variable type error
		if action := s.createMissingVariableTypeAction(uri, diag); action != nil {
			actions = append(actions, *action)
		}
	}

	return actions
}

// createTypoFixActions creates quick fix actions for unknown field typos.
// It uses the suggestions metadata to offer replacements.
func (s *CodeActionService) createTypoFixActions(
	uri lsp.URI,
	diag *EnhancedDiagnostic,
) []lsp.CodeAction {
	actions := []lsp.CodeAction{}

	if diag.ErrorContext.Metadata == nil {
		return actions
	}

	// Get the unknown field name
	unknownField, ok := diag.ErrorContext.Metadata["unknownField"].(string)
	if !ok || unknownField == "" {
		return actions
	}

	// Get suggestions
	suggestions, ok := diag.ErrorContext.Metadata["suggestions"].([]string)
	if !ok || len(suggestions) == 0 {
		return actions
	}

	format := blueprint.DetermineDocFormat(uri)
	diagRange := diag.Diagnostic.Range

	// Create an action for each suggestion
	for i, suggestion := range suggestions {
		// In JSONC, field names are quoted, so we need to include quotes in the replacement
		newText := suggestion
		if format == schema.JWCCSpecFormat {
			newText = fmt.Sprintf("\"%s\"", suggestion)
		}

		textEdit := lsp.TextEdit{
			Range:   &diagRange,
			NewText: newText,
		}

		isPreferred := i == 0
		kind := lsp.CodeActionKindQuickFix
		action := lsp.CodeAction{
			Title:       fmt.Sprintf("Replace '%s' with '%s'", unknownField, suggestion),
			Kind:        &kind,
			IsPreferred: &isPreferred,
			Edit: &lsp.WorkspaceEdit{
				Changes: map[lsp.DocumentURI][]lsp.TextEdit{
					lsp.DocumentURI(uri): {textEdit},
				},
			},
			Diagnostics: []lsp.Diagnostic{diag.Diagnostic},
		}
		actions = append(actions, action)
	}

	return actions
}

// createAllowedValueActions creates quick fix actions for invalid enum values.
// It parses the allowedValuesText to offer valid replacements.
func (s *CodeActionService) createAllowedValueActions(
	uri lsp.URI,
	diag *EnhancedDiagnostic,
) []lsp.CodeAction {
	actions := []lsp.CodeAction{}

	if diag.ErrorContext.Metadata == nil {
		return actions
	}

	// Get allowed values text (comma-separated or formatted list)
	allowedValuesText, ok := diag.ErrorContext.Metadata["allowedValuesText"].(string)
	if !ok || allowedValuesText == "" {
		return actions
	}

	// Parse allowed values - they may be in various formats
	allowedValues := parseAllowedValues(allowedValuesText)
	if len(allowedValues) == 0 {
		return actions
	}

	format := blueprint.DetermineDocFormat(uri)
	diagRange := diag.Diagnostic.Range
	kind := lsp.CodeActionKindQuickFix

	// Create an action for each allowed value
	for i, value := range allowedValues {
		// Format the value appropriately for the document format
		newText := formatValueForEdit(value, format)

		textEdit := lsp.TextEdit{
			Range:   &diagRange,
			NewText: newText,
		}

		isPreferred := i == 0
		action := lsp.CodeAction{
			Title:       fmt.Sprintf("Replace with '%s'", value),
			Kind:        &kind,
			IsPreferred: &isPreferred,
			Edit: &lsp.WorkspaceEdit{
				Changes: map[lsp.DocumentURI][]lsp.TextEdit{
					lsp.DocumentURI(uri): {textEdit},
				},
			},
			Diagnostics: []lsp.Diagnostic{diag.Diagnostic},
		}
		actions = append(actions, action)
	}

	return actions
}

// createMissingVersionAction creates a quick fix action to add the version field.
func (s *CodeActionService) createMissingVersionAction(
	uri lsp.URI,
	diag *EnhancedDiagnostic,
) *lsp.CodeAction {
	format := blueprint.DetermineDocFormat(uri)

	var newText string
	var insertRange lsp.Range

	if format == schema.JWCCSpecFormat {
		// For JSONC, we need to insert after the opening brace
		// We insert on line 1 (after the opening {) with proper formatting
		insertRange = lsp.Range{
			Start: lsp.Position{Line: 1, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		}
		newText = fmt.Sprintf("  \"version\": \"%s\",\n", validation.Version2025_11_02)
	} else {
		// For YAML, insert at the beginning of the document
		insertRange = lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 0},
		}
		newText = fmt.Sprintf("version: \"%s\"\n", validation.Version2025_11_02)
	}

	textEdit := lsp.TextEdit{
		Range:   &insertRange,
		NewText: newText,
	}

	kind := lsp.CodeActionKindQuickFix
	isPreferred := true
	return &lsp.CodeAction{
		Title:       "Add version field",
		Kind:        &kind,
		IsPreferred: &isPreferred,
		Edit: &lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				lsp.DocumentURI(uri): {textEdit},
			},
		},
		Diagnostics: []lsp.Diagnostic{diag.Diagnostic},
	}
}

// createMissingVariableTypeAction creates a quick fix action to add a variable type.
func (s *CodeActionService) createMissingVariableTypeAction(
	uri lsp.URI,
	diag *EnhancedDiagnostic,
) *lsp.CodeAction {
	if diag.ErrorContext.Metadata == nil {
		return nil
	}

	// Check if this is a missing variable type error
	variableName, ok := diag.ErrorContext.Metadata["variableName"].(string)
	if !ok || variableName == "" {
		return nil
	}

	// Check the error message to confirm it's about missing type
	if !strings.Contains(strings.ToLower(diag.Diagnostic.Message), "type missing") &&
		!strings.Contains(strings.ToLower(diag.Diagnostic.Message), "missing") {
		return nil
	}

	format := blueprint.DetermineDocFormat(uri)

	// Insert type on the line after the variable key (Start.Line + 1).
	// The diagnostic points to the variable key (e.g., "myVar:"), so we insert the type
	// as the first property inside the variable's block.
	insertLine := diag.Diagnostic.Range.Start.Line + 1
	insertRange := lsp.Range{
		Start: lsp.Position{Line: insertLine, Character: 0},
		End:   lsp.Position{Line: insertLine, Character: 0},
	}

	var newText string
	if format == schema.JWCCSpecFormat {
		newText = "    \"type\": \"string\",\n"
	} else {
		newText = "    type: string\n"
	}

	textEdit := lsp.TextEdit{
		Range:   &insertRange,
		NewText: newText,
	}

	kind := lsp.CodeActionKindQuickFix
	isPreferred := true
	return &lsp.CodeAction{
		Title:       fmt.Sprintf("Add type: string to variable '%s'", variableName),
		Kind:        &kind,
		IsPreferred: &isPreferred,
		Edit: &lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				lsp.DocumentURI(uri): {textEdit},
			},
		},
		Diagnostics: []lsp.Diagnostic{diag.Diagnostic},
	}
}

// Checks if two LSP ranges overlap.
// Adjacent ranges (where one ends exactly where another starts) are not considered overlapping.
func rangesOverlap(a, b lsp.Range) bool {
	// Check if range 'a' ends before or at where 'b' starts
	if a.End.Line < b.Start.Line ||
		(a.End.Line == b.Start.Line && a.End.Character <= b.Start.Character) {
		return false
	}

	// Check if range 'b' ends before or at where 'a' starts
	if b.End.Line < a.Start.Line ||
		(b.End.Line == a.Start.Line && b.End.Character <= a.Start.Character) {
		return false
	}

	return true
}

// parseAllowedValues parses a comma-separated or formatted list of allowed values.
func parseAllowedValues(text string) []string {
	// Remove any surrounding brackets or quotes
	text = strings.Trim(text, "[]{}\"'")

	// Split by comma
	parts := strings.Split(text, ",")

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		// Remove surrounding quotes
		trimmed = strings.Trim(trimmed, "\"'")
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}

// formatValueForEdit formats a value for insertion in a YAML/JSONC document.
func formatValueForEdit(value string, format schema.SpecFormat) string {
	if format == schema.JWCCSpecFormat {
		// JSONC always requires string values to be quoted
		return fmt.Sprintf("\"%s\"", value)
	}

	// For YAML, only quote if the value contains special characters
	if strings.ContainsAny(value, " \t:{}[],'\"") {
		return fmt.Sprintf("\"%s\"", value)
	}
	return value
}
