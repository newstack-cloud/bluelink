package diagnostichelpers

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type TransformSuite struct {
	suite.Suite
}

func (s *TransformSuite) TestBlueprintToLSP_EmptyDiagnostics() {
	result := BlueprintToLSP([]*core.Diagnostic{}, true)
	s.Empty(result)
}

func (s *TransformSuite) TestBlueprintToLSP_SingleErrorDiagnostic() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "test error message",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 5, Column: 10},
				},
				End: &source.Meta{
					Position: source.Position{Line: 5, Column: 20},
				},
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.Equal(lsp.DiagnosticSeverityError, *result[0].Severity)
	s.Equal("test error message", result[0].Message)
	s.Equal("blueprint-validator", *result[0].Source)
	s.Equal(lsp.Position{Line: 4, Character: 9}, result[0].Range.Start)
	s.Equal(lsp.Position{Line: 4, Character: 19}, result[0].Range.End)
}

func (s *TransformSuite) TestBlueprintToLSP_SingleWarningDiagnostic() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "test warning message",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 10, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 10, Column: 15},
				},
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.Equal(lsp.DiagnosticSeverityWarning, *result[0].Severity)
	s.Equal("test warning message", result[0].Message)
}

func (s *TransformSuite) TestBlueprintToLSP_InformationLevelDiagnostic() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelInfo,
			Message: "info message",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 5},
				},
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.Equal(lsp.DiagnosticSeverityInformation, *result[0].Severity)
}

func (s *TransformSuite) TestBlueprintToLSP_NilRangeDefaultsToDocumentStart() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "error without range",
			Range:   nil,
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.Equal(lsp.Position{Line: 0, Character: 0}, result[0].Range.Start)
	s.Equal(lsp.Position{Line: 1, Character: 0}, result[0].Range.End)
}

func (s *TransformSuite) TestBlueprintToLSP_WithReasonCode() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "error with code",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 10},
				},
			},
			Context: &errors.ErrorContext{
				ReasonCode: errors.ErrorReasonCode("INVALID_TYPE"),
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.NotNil(result[0].Code)
	s.Equal("INVALID_TYPE", *result[0].Code.StrVal)
}

func (s *TransformSuite) TestBlueprintToLSP_MultipleDiagnostics() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "first error",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 5},
				},
			},
		},
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "second warning",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 10, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 10, Column: 10},
				},
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 2)
	s.Equal(lsp.DiagnosticSeverityError, *result[0].Severity)
	s.Equal(lsp.DiagnosticSeverityWarning, *result[1].Severity)
}

func (s *TransformSuite) TestBlueprintToLSP_WithSuggestedActions() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "Invalid configuration",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 5, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 5, Column: 10},
				},
			},
			Context: &errors.ErrorContext{
				ReasonCode: "CONFIG_ERROR",
				SuggestedActions: []errors.SuggestedAction{
					{Title: "Check syntax", Description: "Review the YAML syntax"},
					{Title: "Use validator"},
				},
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	expectedMessage := "Invalid configuration\n\nSuggested Actions:\n  1. Check syntax: Review the YAML syntax\n  2. Use validator\n"
	s.Equal(expectedMessage, result[0].Message)
	s.Equal("CONFIG_ERROR", *result[0].Code.StrVal)
}

func (s *TransformSuite) TestBlueprintToLSP_FiltersAnyTypeWarningsWhenDisabled() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "any type warning",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 10},
				},
			},
			Context: &errors.ErrorContext{
				ReasonCode: errors.ErrorReasonCodeAnyTypeWarning,
			},
		},
		{
			Level:   core.DiagnosticLevelError,
			Message: "real error",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 5, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 5, Column: 10},
				},
			},
		},
	}

	result := BlueprintToLSP(input, false)

	s.Len(result, 1)
	s.Equal("real error", result[0].Message)
}

func (s *TransformSuite) TestBlueprintToLSP_ShowsAnyTypeWarningsWhenEnabled() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "any type warning",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 10},
				},
			},
			Context: &errors.ErrorContext{
				ReasonCode: errors.ErrorReasonCodeAnyTypeWarning,
			},
		},
	}

	result := BlueprintToLSP(input, true)

	s.Len(result, 1)
	s.Equal("any type warning", result[0].Message)
}

func (s *TransformSuite) TestBlueprintToLSP_DoesNotFilterNonAnyTypeWarnings() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "other warning",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{Line: 1, Column: 1},
				},
				End: &source.Meta{
					Position: source.Position{Line: 1, Column: 10},
				},
			},
			Context: &errors.ErrorContext{
				ReasonCode: "some_other_reason",
			},
		},
	}

	result := BlueprintToLSP(input, false)

	s.Len(result, 1)
	s.Equal("other warning", result[0].Message)
}

func (s *TransformSuite) TestFormatDiagnosticWithContext_NilContext() {
	result := formatDiagnosticWithContext("original message", nil)
	s.Equal("original message", result)
}

func (s *TransformSuite) TestFormatDiagnosticWithContext_EmptySuggestedActions() {
	ctx := &errors.ErrorContext{
		SuggestedActions: []errors.SuggestedAction{},
	}
	result := formatDiagnosticWithContext("original message", ctx)
	s.Equal("original message", result)
}

func (s *TransformSuite) TestFormatDiagnosticWithContext_SingleActionWithoutDescription() {
	ctx := &errors.ErrorContext{
		SuggestedActions: []errors.SuggestedAction{
			{Title: "Fix the issue"},
		},
	}
	result := formatDiagnosticWithContext("error occurred", ctx)
	s.Equal("error occurred\n\nSuggested Actions:\n  1. Fix the issue\n", result)
}

func (s *TransformSuite) TestFormatDiagnosticWithContext_SingleActionWithDescription() {
	ctx := &errors.ErrorContext{
		SuggestedActions: []errors.SuggestedAction{
			{Title: "Fix the issue", Description: "Update the configuration file"},
		},
	}
	result := formatDiagnosticWithContext("error occurred", ctx)
	s.Equal("error occurred\n\nSuggested Actions:\n  1. Fix the issue: Update the configuration file\n", result)
}

func (s *TransformSuite) TestFormatDiagnosticWithContext_MultipleActions() {
	ctx := &errors.ErrorContext{
		SuggestedActions: []errors.SuggestedAction{
			{Title: "Check syntax"},
			{Title: "Review documentation", Description: "See the API docs"},
			{Title: "Contact support"},
		},
	}
	result := formatDiagnosticWithContext("validation failed", ctx)
	expected := "validation failed\n\nSuggested Actions:\n  1. Check syntax\n  2. Review documentation: See the API docs\n  3. Contact support\n"
	s.Equal(expected, result)
}

func (s *TransformSuite) TestLspDiagnosticRangeFromBlueprintDiagnosticRange_NilRange() {
	result := lspDiagnosticRangeFromBlueprintDiagnosticRange(nil)
	s.Equal(lsp.Position{Line: 0, Character: 0}, result.Start)
	s.Equal(lsp.Position{Line: 1, Character: 0}, result.End)
}

func (s *TransformSuite) TestLspDiagnosticRangeFromBlueprintDiagnosticRange_ValidRange() {
	input := &core.DiagnosticRange{
		Start: &source.Meta{
			Position: source.Position{Line: 5, Column: 10},
		},
		End: &source.Meta{
			Position: source.Position{Line: 5, Column: 20},
		},
	}
	result := lspDiagnosticRangeFromBlueprintDiagnosticRange(input)
	s.Equal(lsp.Position{Line: 4, Character: 9}, result.Start)
	s.Equal(lsp.Position{Line: 4, Character: 19}, result.End)
}

func (s *TransformSuite) TestLspDiagnosticRangeFromBlueprintDiagnosticRange_NilEnd() {
	input := &core.DiagnosticRange{
		Start: &source.Meta{
			Position: source.Position{Line: 10, Column: 5},
		},
		End: nil,
	}
	result := lspDiagnosticRangeFromBlueprintDiagnosticRange(input)
	s.Equal(lsp.Position{Line: 9, Character: 4}, result.Start)
	s.Equal(lsp.Position{Line: 10, Character: 0}, result.End)
}

func (s *TransformSuite) TestLspDiagnosticRangeFromBlueprintDiagnosticRange_NilStart() {
	input := &core.DiagnosticRange{
		Start: nil,
		End: &source.Meta{
			Position: source.Position{Line: 5, Column: 10},
		},
	}
	result := lspDiagnosticRangeFromBlueprintDiagnosticRange(input)
	s.Equal(lsp.Position{Line: 0, Character: 0}, result.Start)
	s.Equal(lsp.Position{Line: 4, Character: 9}, result.End)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_NilSourceMetaAndNilStartPos() {
	result := lspPositionFromSourceMeta(nil, nil, nil)
	s.Equal(lsp.Position{Line: 0, Character: 0}, result)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_NilSourceMetaWithStartPos() {
	startPos := &lsp.Position{Line: 5, Character: 10}
	result := lspPositionFromSourceMeta(nil, startPos, nil)
	s.Equal(lsp.Position{Line: 6, Character: 0}, result)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_ValidSourceMetaWithNilColumnAccuracy() {
	meta := &source.Meta{
		Position: source.Position{Line: 10, Column: 15},
	}
	result := lspPositionFromSourceMeta(meta, nil, nil)
	s.Equal(lsp.Position{Line: 9, Character: 14}, result)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_ExactColumnAccuracy() {
	meta := &source.Meta{
		Position: source.Position{Line: 10, Column: 15},
	}
	exactAccuracy := substitutions.ColumnAccuracyExact
	result := lspPositionFromSourceMeta(meta, nil, &exactAccuracy)
	s.Equal(lsp.Position{Line: 9, Character: 14}, result)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_ApproximateColumnAccuracy() {
	meta := &source.Meta{
		Position: source.Position{Line: 10, Column: 15},
	}
	approxAccuracy := substitutions.ColumnAccuracyApproximate
	result := lspPositionFromSourceMeta(meta, nil, &approxAccuracy)
	s.Equal(lsp.Position{Line: 9, Character: 0}, result)
}

func (s *TransformSuite) TestLspPositionFromSourceMeta_Line1Column1ConvertsTo00() {
	meta := &source.Meta{
		Position: source.Position{Line: 1, Column: 1},
	}
	result := lspPositionFromSourceMeta(meta, nil, nil)
	s.Equal(lsp.Position{Line: 0, Character: 0}, result)
}

func TestTransformSuite(t *testing.T) {
	suite.Run(t, new(TransformSuite))
}
