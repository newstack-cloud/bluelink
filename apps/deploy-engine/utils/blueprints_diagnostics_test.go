package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type DiagnosticsFromErrorTestSuite struct {
	logger core.Logger
	suite.Suite
}

func (s *DiagnosticsFromErrorTestSuite) SetupTest() {
	s.logger = core.NewNopLogger()
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_empty_slice_for_a_nil_error() {
	diagnostics := DiagnosticsFromBlueprintValidationError(
		nil,
		s.logger,
		/* fallbackToGeneralDiagnostic */ false,
	)
	s.Assert().Empty(diagnostics)
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_diagnostics_extracted_from_load_error() {
	line := 1
	column := 1
	inputErr := &bperrors.LoadError{
		ReasonCode: container.ErrorReasonCodeResourceValidationErrors,
		Err:        fmt.Errorf("test error"),
		Line:       &line,
		Column:     &column,
		ChildErrors: []error{
			createChildLoadError(
				true, /* hasDescendants */
				5,    /* levels */
				1,    /* currentLevel */
			),
			createSchemaError(),
			createParseErrors(10, 30),
			createParseError(15, 25),
			createParseErrorsNoChildren(),
			createCoreError(),
			createLexErrors(14, 1),
			createLexError(320, 5),
			createLexErrorsNoChildren(),
			createGeneralError(),
			createRunError(),
		},
	}
	diagnostics := DiagnosticsFromBlueprintValidationError(
		inputErr,
		s.logger,
		/* fallbackToGeneralDiagnostic */ false,
	)

	err := testhelpers.Snapshot(
		diagnostics,
	)
	s.Require().NoError(err)
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_diagnostics_extracted_from_general_error() {
	diagnostics := DiagnosticsFromBlueprintValidationError(
		createGeneralError(),
		s.logger,
		/* fallbackToGeneralDiagnostic */ true,
	)

	err := testhelpers.Snapshot(
		diagnostics,
	)
	s.Require().NoError(err)
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_empty_slice_for_a_general_error_with_fallback_disabled() {
	diagnostics := DiagnosticsFromBlueprintValidationError(
		createGeneralError(),
		s.logger,
		/* fallbackToGeneralDiagnostic */ false,
	)
	s.Assert().Empty(diagnostics)
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_diagnostics_extracted_from_schema_error() {
	diagnostics := DiagnosticsFromBlueprintValidationError(
		createSchemaError(),
		s.logger,
		/* fallbackToGeneralDiagnostic */ false,
	)

	err := testhelpers.Snapshot(
		diagnostics,
	)
	s.Require().NoError(err)
}

func (s *DiagnosticsFromErrorTestSuite) Test_returns_diagnostics_extracted_from_run_error() {
	diagnostics := DiagnosticsFromBlueprintValidationError(
		createRunError(),
		s.logger,
		/* fallbackToGeneralDiagnostic */ false,
	)

	err := testhelpers.Snapshot(
		diagnostics,
	)
	s.Require().NoError(err)
}

func createChildLoadError(hasDescendants bool, levels int, currentLevel int) error {
	line := 10
	column := 1
	childErrors := []error{}

	if hasDescendants {
		childLevel := currentLevel + 1
		childErrors = []error{
			createChildLoadError(childLevel <= levels, levels, childLevel),
		}
	}

	return &bperrors.LoadError{
		ReasonCode:  container.ErrorReasonCodeVariableValidationErrors,
		Err:         fmt.Errorf("test child error level %d", currentLevel),
		Line:        &line,
		Column:      &column,
		ChildErrors: childErrors,
	}
}

func createSchemaError() error {
	line := 20
	column := 2
	return &schema.Error{
		Err:          fmt.Errorf("test schema error"),
		SourceLine:   &line,
		SourceColumn: &column,
	}
}

func createParseErrors(line int, column int) error {
	return &substitutions.ParseErrors{
		ChildErrors: []error{
			createParseError(line, column),
		},
	}
}

func createParseErrorsNoChildren() error {
	return &substitutions.ParseErrors{
		ChildErrors: []error{},
	}
}

func createParseError(line int, column int) error {
	return &substitutions.ParseError{
		Line:           line,
		Column:         column,
		ColumnAccuracy: substitutions.ColumnAccuracyApproximate,
	}
}

func createCoreError() error {
	line := 40
	column := 1
	return &core.Error{
		ReasonCode:   core.ErrorCoreReasonCodeMustBeScalar,
		Err:          fmt.Errorf("test core error"),
		SourceLine:   &line,
		SourceColumn: &column,
	}
}

func createLexErrors(line int, column int) error {
	return &substitutions.LexErrors{
		ChildErrors: []error{
			createLexError(line, column),
		},
	}
}

func createLexErrorsNoChildren() error {
	return &substitutions.LexErrors{
		ChildErrors: []error{},
	}
}

func createLexError(line int, column int) error {
	return &substitutions.LexError{
		Line:           line,
		Column:         column,
		ColumnAccuracy: substitutions.ColumnAccuracyApproximate,
	}
}

func createGeneralError() error {
	return errors.New("test general error")
}

func createRunError() error {
	return &bperrors.RunError{
		ReasonCode:         container.ErrorReasonCodeDeployMissingInstanceID,
		Err:                fmt.Errorf("test run error"),
		ChildBlueprintPath: "include.coreInfra",
	}
}

func TestDiagnosticsFromErrorSuite(t *testing.T) {
	suite.Run(t, new(DiagnosticsFromErrorTestSuite))
}
