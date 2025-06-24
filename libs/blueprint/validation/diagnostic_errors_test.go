package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"

	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type DiagnosticErrorSuite struct {
	suite.Suite
}

func (s *DiagnosticErrorSuite) Test_extracts_diagnostics_and_validation_errors_from_diag_slice() {
	input := []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "This is a test error message.",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
				},
			},
		},
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "This is a test warning message.",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 5,
					},
				},
			},
		},
		{
			Level:   core.DiagnosticLevelError,
			Message: "This is another test error message.",
			Range: &core.DiagnosticRange{
				Start: nil, // No specific position, should default to line 1, column 1
			},
		},
	}

	errReasonCode := bperrors.ErrorReasonCode("test_error_reason")
	diagnostics, err := ExtractDiagnosticsAndErrors(input, errReasonCode)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "This is a test warning message.",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   2,
							Column: 5,
						},
					},
				},
			},
		},
		diagnostics,
	)
	s.Assert().Error(err)
	loadErr, isLoadErr := err.(*bperrors.LoadError)
	s.Assert().True(isLoadErr)
	s.Assert().Len(loadErr.ChildErrors, 2)

	childErr1, isChildErr1LoadErr := loadErr.ChildErrors[0].(*bperrors.LoadError)
	s.Assert().True(isChildErr1LoadErr)
	s.Assert().Equal(errReasonCode, childErr1.ReasonCode)
	s.Assert().Equal("This is a test error message.", childErr1.Err.Error())
	s.Assert().Equal(1, *childErr1.Line)
	s.Assert().Equal(3, *childErr1.Column)

	childErr2, isChildErr2LoadErr := loadErr.ChildErrors[1].(*bperrors.LoadError)
	s.Assert().True(isChildErr2LoadErr)
	s.Assert().Equal(errReasonCode, childErr2.ReasonCode)
	s.Assert().Equal("This is another test error message.", childErr2.Err.Error())
	s.Assert().Equal(1, *childErr2.Line)
	s.Assert().Equal(1, *childErr2.Column)
}

func TestDiagnosticErrorSuite(t *testing.T) {
	suite.Run(t, new(DiagnosticErrorSuite))
}
