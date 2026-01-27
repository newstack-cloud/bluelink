package languageservices

import (
	"fmt"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type CodeActionServiceSuite struct {
	suite.Suite
	service *CodeActionService
	state   *State
}

func (s *CodeActionServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	s.state = NewState()
	s.service = NewCodeActionService(s.state, logger)
}

func TestCodeActionServiceSuite(t *testing.T) {
	suite.Run(t, new(CodeActionServiceSuite))
}

func (s *CodeActionServiceSuite) Test_no_actions_for_empty_diagnostics() {
	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 10, Character: 0},
		},
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Assert().Empty(actions)
}

func (s *CodeActionServiceSuite) Test_typo_fix_action_for_unknown_field() {
	// Set up enhanced diagnostic with suggestions metadata
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 5, Character: 4},
		End:   lsp.Position{Line: 5, Character: 8},
	}
	severity := lsp.DiagnosticSeverityError
	code := "resource_def_unknown_field"

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "unknown field \"naem\"",
				Code:     &lsp.IntOrString{StrVal: &code},
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "resource_def_unknown_field",
				Metadata: map[string]any{
					"unknownField": "naem",
					"suggestions":  []string{"name", "namespace"},
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 2) // One action per suggestion

	// First action should be the preferred one
	s.Assert().Equal("Replace 'naem' with 'name'", actions[0].Title)
	s.Assert().True(*actions[0].IsPreferred)
	s.Assert().NotNil(actions[0].Edit)
	s.Assert().NotNil(actions[0].Edit.Changes)

	// Second action
	s.Assert().Equal("Replace 'naem' with 'namespace'", actions[1].Title)
	s.Assert().False(*actions[1].IsPreferred)
}

func (s *CodeActionServiceSuite) Test_no_action_for_diagnostics_outside_range() {
	// Set up enhanced diagnostic at line 20
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 20, Character: 4},
		End:   lsp.Position{Line: 20, Character: 8},
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "some error",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "resource_def_unknown_field",
				Metadata: map[string]any{
					"unknownField": "naem",
					"suggestions":  []string{"name"},
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", enhanced)

	// Request actions for a different range (line 0-10)
	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 10, Character: 0},
		},
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Assert().Empty(actions) // Diagnostic is outside requested range
}

func (s *CodeActionServiceSuite) Test_allowed_value_actions() {
	// Set up enhanced diagnostic with allowed values
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 10, Character: 10},
		End:   lsp.Position{Line: 10, Character: 20},
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "value not allowed",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "resource_def_not_allowed_value",
				Metadata: map[string]any{
					"allowedValuesText": "enabled, disabled, auto",
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 3) // One action per allowed value

	s.Assert().Equal("Replace with 'enabled'", actions[0].Title)
	s.Assert().True(*actions[0].IsPreferred)

	s.Assert().Equal("Replace with 'disabled'", actions[1].Title)
	s.Assert().Equal("Replace with 'auto'", actions[2].Title)
}

func (s *CodeActionServiceSuite) Test_missing_version_action() {
	// Set up enhanced diagnostic for missing version
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 1, Character: 0},
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "missing version",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "missing_version",
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 1)

	s.Assert().Equal("Add version field", actions[0].Title)
	s.Assert().True(*actions[0].IsPreferred)

	// Check that the edit inserts at the beginning of the document
	changes := actions[0].Edit.Changes
	s.Require().NotNil(changes)
	edits := changes[lsp.DocumentURI("file:///test.yaml")]
	s.Require().Len(edits, 1)
	s.Assert().Equal(fmt.Sprintf("version: \"%s\"\n", validation.Version2025_11_02), edits[0].NewText)
}

func (s *CodeActionServiceSuite) Test_ranges_overlap_correctly() {
	// Test the rangesOverlap helper function

	// Ranges that overlap
	a := lsp.Range{
		Start: lsp.Position{Line: 5, Character: 0},
		End:   lsp.Position{Line: 10, Character: 0},
	}
	b := lsp.Range{
		Start: lsp.Position{Line: 8, Character: 0},
		End:   lsp.Position{Line: 15, Character: 0},
	}
	s.Assert().True(rangesOverlap(a, b))
	s.Assert().True(rangesOverlap(b, a))

	// Ranges that don't overlap
	c := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 5, Character: 0},
	}
	d := lsp.Range{
		Start: lsp.Position{Line: 10, Character: 0},
		End:   lsp.Position{Line: 15, Character: 0},
	}
	s.Assert().False(rangesOverlap(c, d))
	s.Assert().False(rangesOverlap(d, c))

	// Adjacent ranges (don't overlap)
	e := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 5, Character: 0},
	}
	f := lsp.Range{
		Start: lsp.Position{Line: 5, Character: 0},
		End:   lsp.Position{Line: 10, Character: 0},
	}
	s.Assert().False(rangesOverlap(e, f))
}

func (s *CodeActionServiceSuite) Test_parse_allowed_values() {
	// Test parseAllowedValues helper
	s.Assert().Equal([]string{"a", "b", "c"}, parseAllowedValues("a, b, c"))
	s.Assert().Equal([]string{"enabled", "disabled"}, parseAllowedValues("enabled, disabled"))
	s.Assert().Equal([]string{"one", "two", "three"}, parseAllowedValues("[one, two, three]"))
	s.Assert().Equal([]string{"value"}, parseAllowedValues("\"value\""))
	s.Assert().Empty(parseAllowedValues(""))
}

func (s *CodeActionServiceSuite) Test_typo_fix_action_for_jsonc_document() {
	// Set up enhanced diagnostic with suggestions metadata for JSONC document
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 5, Character: 4},
		End:   lsp.Position{Line: 5, Character: 10}, // Includes quotes: "naem"
	}
	severity := lsp.DiagnosticSeverityError
	code := "resource_def_unknown_field"

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "unknown field \"naem\"",
				Code:     &lsp.IntOrString{StrVal: &code},
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "resource_def_unknown_field",
				Metadata: map[string]any{
					"unknownField": "naem",
					"suggestions":  []string{"name"},
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.jsonc", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.jsonc",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 1)

	// For JSONC, the replacement should include quotes
	edits := actions[0].Edit.Changes[lsp.DocumentURI("file:///test.jsonc")]
	s.Require().Len(edits, 1)
	s.Assert().Equal("\"name\"", edits[0].NewText)
}

func (s *CodeActionServiceSuite) Test_missing_version_action_for_jsonc_document() {
	// Set up enhanced diagnostic for missing version in JSONC
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 1, Character: 0},
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "missing version",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "missing_version",
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.jsonc", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.jsonc",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 1)

	// Check that the edit inserts JSONC-formatted version
	changes := actions[0].Edit.Changes
	s.Require().NotNil(changes)
	edits := changes[lsp.DocumentURI("file:///test.jsonc")]
	s.Require().Len(edits, 1)
	// For JSONC, version should be inserted on line 1 (after opening brace) with quotes
	s.Assert().Equal(fmt.Sprintf("  \"version\": \"%s\",\n", validation.Version2025_11_02), edits[0].NewText)
	s.Assert().Equal(lsp.UInteger(1), edits[0].Range.Start.Line)
}

func (s *CodeActionServiceSuite) Test_missing_variable_type_inserts_after_variable_key() {
	// Set up enhanced diagnostic for a variable. The diagnostic points to the variable key
	// (e.g., "myVar:" on line 5), not the content inside the variable.
	// We insert at Start.Line + 1 to place the type as the first property inside the variable.
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 5, Character: 2}, // Variable key "myVar:" starts here
		End:   lsp.Position{Line: 5, Character: 8}, // Variable key ends here
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "variable type missing for myVar",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "variable_validation_errors",
				Metadata: map[string]any{
					"variableName": "myVar",
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.yaml", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.yaml",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 1)

	edits := actions[0].Edit.Changes[lsp.DocumentURI("file:///test.yaml")]
	s.Require().Len(edits, 1)

	// The type should be inserted at line 6 (Start.Line + 1), inside the variable block
	s.Assert().Equal(lsp.UInteger(6), edits[0].Range.Start.Line)
	s.Assert().Equal("    type: string\n", edits[0].NewText)
}

func (s *CodeActionServiceSuite) Test_allowed_value_actions_for_jsonc_document() {
	// Set up enhanced diagnostic with allowed values for JSONC
	diagRange := lsp.Range{
		Start: lsp.Position{Line: 10, Character: 10},
		End:   lsp.Position{Line: 10, Character: 20},
	}
	severity := lsp.DiagnosticSeverityError

	enhanced := []*EnhancedDiagnostic{
		{
			Diagnostic: lsp.Diagnostic{
				Range:    diagRange,
				Severity: &severity,
				Message:  "value not allowed",
			},
			ErrorContext: &errors.ErrorContext{
				ReasonCode: "resource_def_not_allowed_value",
				Metadata: map[string]any{
					"allowedValuesText": "enabled, disabled",
				},
			},
		},
	}
	s.state.SetEnhancedDiagnostics("file:///test.jsonc", enhanced)

	params := &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.jsonc",
		},
		Range: diagRange,
	}

	actions, err := s.service.GetCodeActions(params)

	s.Require().NoError(err)
	s.Require().Len(actions, 2)

	// For JSONC, values should be quoted
	edits := actions[0].Edit.Changes[lsp.DocumentURI("file:///test.jsonc")]
	s.Require().Len(edits, 1)
	s.Assert().Equal("\"enabled\"", edits[0].NewText)

	edits = actions[1].Edit.Changes[lsp.DocumentURI("file:///test.jsonc")]
	s.Require().Len(edits, 1)
	s.Assert().Equal("\"disabled\"", edits[0].NewText)
}
