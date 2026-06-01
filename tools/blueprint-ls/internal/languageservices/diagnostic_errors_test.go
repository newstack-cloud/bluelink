package languageservices

import (
	"fmt"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DiagnosticErrorServiceSuite struct {
	suite.Suite
	service *DiagnosticErrorService
}

func (s *DiagnosticErrorServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	s.service = NewDiagnosticErrorService(state, logger)
}

func TestDiagnosticErrorServiceSuite(t *testing.T) {
	suite.Run(t, new(DiagnosticErrorServiceSuite))
}

func (s *DiagnosticErrorServiceSuite) Test_load_error_with_exact_end_position() {
	// When a LoadError has exact column accuracy and end positions,
	// the diagnostic range should use the precise start and end positions.
	line := 10
	col := 5
	endLine := 10
	endCol := 25
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("invalid substitution"),
		Line:           &line,
		Column:         &col,
		EndLine:        &endLine,
		EndColumn:      &endCol,
		ColumnAccuracy: &colAccuracy,
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// LSP positions are 0-indexed, blueprint positions are 1-indexed
	s.Assert().Equal(lsp.UInteger(9), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(4), diag.Range.Start.Character)
	s.Assert().Equal(lsp.UInteger(9), diag.Range.End.Line)
	s.Assert().Equal(lsp.UInteger(24), diag.Range.End.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_load_error_with_approximate_column_highlights_whole_line() {
	// When a LoadError has approximate column accuracy (e.g., YAML block literals),
	// the diagnostic range should highlight from the start position to end of next line.
	line := 10
	col := 5
	endLine := 10
	endCol := 25
	colAccuracy := source.ColumnAccuracyApproximate

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("invalid substitution in block literal"),
		Line:           &line,
		Column:         &col,
		EndLine:        &endLine,
		EndColumn:      &endCol,
		ColumnAccuracy: &colAccuracy,
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// With approximate column, start line should be used but column should be 0
	// and end should be next line
	s.Assert().Equal(lsp.UInteger(9), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.Start.Character)
	s.Assert().Equal(lsp.UInteger(10), diag.Range.End.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.End.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_load_error_without_end_position_falls_back_to_next_line() {
	// When a LoadError has no end position, the diagnostic range should
	// extend to the end of the next line as a fallback.
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("some error"),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// Start position is exact
	s.Assert().Equal(lsp.UInteger(9), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(4), diag.Range.Start.Character)
	// End falls back to next line
	s.Assert().Equal(lsp.UInteger(10), diag.Range.End.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.End.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_parse_error_with_exact_column() {
	// Parse errors have their own column accuracy tracking.
	// Only the public fields (Line, Column, ColumnAccuracy) are needed for range calculation.
	parseErr := &substitutions.ParseError{
		Line:           15,
		Column:         10,
		ColumnAccuracy: substitutions.ColumnAccuracyExact,
	}

	loadErr := &errors.LoadError{
		Err:         fmt.Errorf("parse failed"),
		ChildErrors: []error{parseErr},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// LSP positions are 0-indexed
	s.Assert().Equal(lsp.UInteger(14), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(9), diag.Range.Start.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_parse_error_with_approximate_column() {
	// Parse errors from YAML block literals have approximate column accuracy.
	parseErr := &substitutions.ParseError{
		Line:           15,
		Column:         10,
		ColumnAccuracy: substitutions.ColumnAccuracyApproximate,
	}

	loadErr := &errors.LoadError{
		Err:         fmt.Errorf("parse failed"),
		ChildErrors: []error{parseErr},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// With approximate column, start column should be 0
	s.Assert().Equal(lsp.UInteger(14), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.Start.Character)
	// End should be next line
	s.Assert().Equal(lsp.UInteger(15), diag.Range.End.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.End.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_lex_error_with_exact_column() {
	lexErr := &substitutions.LexError{
		Line:           20,
		Column:         15,
		ColumnAccuracy: substitutions.ColumnAccuracyExact,
	}

	loadErr := &errors.LoadError{
		Err:         fmt.Errorf("lex failed"),
		ChildErrors: []error{lexErr},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	s.Assert().Equal(lsp.UInteger(19), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(14), diag.Range.Start.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_lex_error_with_approximate_column() {
	lexErr := &substitutions.LexError{
		Line:           20,
		Column:         15,
		ColumnAccuracy: substitutions.ColumnAccuracyApproximate,
	}

	loadErr := &errors.LoadError{
		Err:         fmt.Errorf("lex failed"),
		ChildErrors: []error{lexErr},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// With approximate column, start column should be 0
	s.Assert().Equal(lsp.UInteger(19), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.Start.Character)
	s.Assert().Equal(lsp.UInteger(20), diag.Range.End.Line)
}

func (s *DiagnosticErrorServiceSuite) Test_error_without_location_uses_default_range() {
	loadErr := &errors.LoadError{
		Err: fmt.Errorf("some error without location"),
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]

	// Default range starts at 0,0
	s.Assert().Equal(lsp.UInteger(0), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.Start.Character)
	s.Assert().Equal(lsp.UInteger(1), diag.Range.End.Line)
	s.Assert().Equal(lsp.UInteger(0), diag.Range.End.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_nested_load_errors_collect_all_diagnostics() {
	line1 := 5
	col1 := 10
	line2 := 15
	col2 := 20
	colAccuracy := source.ColumnAccuracyExact

	childErr1 := &errors.LoadError{
		Err:            fmt.Errorf("first error"),
		Line:           &line1,
		Column:         &col1,
		ColumnAccuracy: &colAccuracy,
	}

	childErr2 := &errors.LoadError{
		Err:            fmt.Errorf("second error"),
		Line:           &line2,
		Column:         &col2,
		ColumnAccuracy: &colAccuracy,
	}

	parentErr := &errors.LoadError{
		Err:         fmt.Errorf("parent error"),
		ChildErrors: []error{childErr1, childErr2},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(parentErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 2)

	// First error
	s.Assert().Equal(lsp.UInteger(4), diagnostics[0].Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(9), diagnostics[0].Range.Start.Character)

	// Second error
	s.Assert().Equal(lsp.UInteger(14), diagnostics[1].Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(19), diagnostics[1].Range.Start.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_suggestions_includes_did_you_mean() {
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("unknown field \"handleName\""),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
		Context: &errors.ErrorContext{
			Metadata: map[string]any{
				"suggestions": []string{"handlerName"},
			},
		},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "Did you mean: handlerName?")
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_available_fields_lists_them() {
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("unknown field \"foo\""),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
		Context: &errors.ErrorContext{
			Metadata: map[string]any{
				"availableFields": []string{"code", "runtime", "timeout"},
			},
		},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "Available fields: code, runtime, timeout")
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_many_available_fields_truncates_list() {
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("unknown field \"foo\""),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
		Context: &errors.ErrorContext{
			Metadata: map[string]any{
				"availableFields": []string{
					"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l",
				},
			},
		},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "Available fields: a, b, c, d, e, f, g, h, ... (4 more)")
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_suggestions_and_available_fields() {
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		Err:            fmt.Errorf("unknown field \"timout\""),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
		Context: &errors.ErrorContext{
			Metadata: map[string]any{
				"suggestions":     []string{"timeout"},
				"availableFields": []string{"code", "runtime", "timeout"},
			},
		},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	// Both suggestions and available fields should be present
	s.Assert().Contains(diagnostics[0].Message, "Did you mean: timeout?")
	s.Assert().Contains(diagnostics[0].Message, "Available fields: code, runtime, timeout")
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_reason_code_but_no_context_creates_enhanced_diagnostic() {
	// When a LoadError has a ReasonCode but no Context, we should still
	// create an EnhancedDiagnostic with a synthesized ErrorContext for code actions.
	loadErr := &errors.LoadError{
		ReasonCode: "missing_version",
		Err:        fmt.Errorf("validation failed due to a version not being provided"),
	}

	diagnostics, enhanced := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	s.Require().Len(enhanced, 1)

	// The enhanced diagnostic should have an ErrorContext with the ReasonCode
	s.Require().NotNil(enhanced[0].ErrorContext)
	s.Assert().Equal(errors.ErrorReasonCode("missing_version"), enhanced[0].ErrorContext.ReasonCode)
}

func (s *DiagnosticErrorServiceSuite) Test_error_with_context_uses_existing_context() {
	// When a LoadError has both ReasonCode and Context, the existing Context should be used.
	line := 10
	col := 5
	colAccuracy := source.ColumnAccuracyExact

	loadErr := &errors.LoadError{
		ReasonCode:     "some_reason",
		Err:            fmt.Errorf("some error"),
		Line:           &line,
		Column:         &col,
		ColumnAccuracy: &colAccuracy,
		Context: &errors.ErrorContext{
			ReasonCode: "context_reason",
			Metadata: map[string]any{
				"key": "value",
			},
		},
	}

	diagnostics, enhanced := s.service.BlueprintErrorToDiagnostics(loadErr, "file:///test.yaml")

	s.Require().Len(diagnostics, 1)
	s.Require().Len(enhanced, 1)

	// The enhanced diagnostic should use the existing Context, not a synthesized one
	s.Require().NotNil(enhanced[0].ErrorContext)
	s.Assert().Equal(errors.ErrorReasonCode("context_reason"), enhanced[0].ErrorContext.ReasonCode)
	s.Assert().Equal("value", enhanced[0].ErrorContext.Metadata["key"])
}

func langSourceMeta(line, column int) *source.Meta {
	return &source.Meta{
		Position: source.Position{Line: line, Column: column},
	}
}

func (s *DiagnosticErrorServiceSuite) Test_lang_parse_error_maps_to_positioned_diagnostic() {
	// A blueprint-language parse error must land on its tracked line/column,
	// not collapse to the position-less whole-document fallback.
	parseErr := &lang.ParseError{
		Message:    "unexpected token",
		SourceMeta: langSourceMeta(3, 5),
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(parseErr, "file:///test.bp")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]
	// LSP positions are 0-indexed, blueprint positions are 1-indexed.
	s.Assert().Equal(lsp.UInteger(2), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(4), diag.Range.Start.Character)
	s.Assert().Contains(diag.Message, "unexpected token")
}

func (s *DiagnosticErrorServiceSuite) Test_lang_lex_error_maps_to_positioned_diagnostic() {
	lexErr := &lang.LexError{
		Message:    "unterminated string",
		SourceMeta: langSourceMeta(7, 2),
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(lexErr, "file:///test.bp")

	s.Require().Len(diagnostics, 1)
	diag := diagnostics[0]
	s.Assert().Equal(lsp.UInteger(6), diag.Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(1), diag.Range.Start.Character)
	s.Assert().Contains(diag.Message, "unterminated string")
}

func (s *DiagnosticErrorServiceSuite) Test_lang_errors_envelope_maps_each_child() {
	// The lang parser returns a *lang.Errors envelope aggregating per-error
	// diagnostics; each child must surface as its own positioned diagnostic.
	langErrs := &lang.Errors{
		ChildErrors: []error{
			&lang.ParseError{Message: "first", SourceMeta: langSourceMeta(2, 1)},
			&lang.LexError{Message: "second", SourceMeta: langSourceMeta(9, 4)},
		},
	}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(langErrs, "file:///test.bp")

	s.Require().Len(diagnostics, 2)
	s.Assert().Equal(lsp.UInteger(1), diagnostics[0].Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), diagnostics[0].Range.Start.Character)
	s.Assert().Equal(lsp.UInteger(8), diagnostics[1].Range.Start.Line)
	s.Assert().Equal(lsp.UInteger(3), diagnostics[1].Range.Start.Character)
}

func (s *DiagnosticErrorServiceSuite) Test_lang_parse_error_without_position_still_diagnoses() {
	// Some parse errors (e.g. the missing 'version' directive) carry no source
	// position; these must still produce a diagnostic rather than be dropped.
	parseErr := &lang.ParseError{Message: "missing required 'version' directive"}

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(parseErr, "file:///test.bp")

	s.Require().Len(diagnostics, 1)
	s.Assert().Contains(diagnostics[0].Message, "missing required 'version' directive")
}

func (s *DiagnosticErrorServiceSuite) Test_real_lang_parse_failure_flows_to_diagnostics() {
	// End-to-end: a genuinely invalid .bp parsed by the library must produce a
	// *lang.Errors envelope that the converter unwraps into per-child diagnostics.
	src := "version \"2025-05-12\"\n\nvariable region: string {\n    default = \"unterminated\n}\n"

	_, err := lang.ParseString(src)
	s.Require().Error(err)

	langErrs, ok := err.(*lang.Errors)
	s.Require().True(ok, "expected lang.ParseString to return *lang.Errors, got %T", err)
	s.Require().NotEmpty(langErrs.ChildErrors)

	diagnostics, _ := s.service.BlueprintErrorToDiagnostics(err, "file:///test.bp")

	// Each aggregated child error surfaces as its own diagnostic with a message.
	s.Require().Len(diagnostics, len(langErrs.ChildErrors))
	for _, diag := range diagnostics {
		s.Assert().NotEmpty(diag.Message)
	}
}
