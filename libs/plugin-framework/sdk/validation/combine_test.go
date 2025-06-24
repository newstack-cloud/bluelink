package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type CombineValidationSuite struct {
	suite.Suite
}

func (s *CombineValidationSuite) Test_all_of_combination_validation() {
	validScalar := core.ScalarFromString("https://example.com")

	diagnostics := AllOf(
		IsWebURL(),
		StringDoesNotContainChars("<>\"'"),
	)("testField", validScalar)

	s.Assert().Empty(diagnostics)
}

func (s *CombineValidationSuite) Test_all_of_fails_combination_validation() {
	invalidScalar := core.ScalarFromString("not-a-url")

	diagnostics := AllOf(
		IsWebURL(),
		StringDoesNotContainChars("<>\"'"),
	)("testField", invalidScalar)

	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(diagnostics[0].Message, "must be a valid url with a host")
}

func (s *CombineValidationSuite) Test_one_of_combination_validation() {
	validScalar := core.ScalarFromString("{\"key\": \"value<>\"}")

	diagnostics := OneOf(
		StringIsJSON(),
		StringDoesNotContainChars("<>\"'"),
	)("testField", validScalar)

	s.Assert().Empty(diagnostics)
}

func (s *CombineValidationSuite) Test_one_of_fails_combination_validation() {
	invalidScalar := core.ScalarFromString("not-json<>")

	diagnostics := OneOf(
		StringIsJSON(),
		StringDoesNotContainChars("<>\"'"),
	)("testField", invalidScalar)

	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(diagnostics[0].Message, "did not pass any of the validation checks")
}

func TestCombineValidationSuite(t *testing.T) {
	suite.Run(t, new(CombineValidationSuite))
}
