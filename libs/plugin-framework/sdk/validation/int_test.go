package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type IntValidationSuite struct {
	suite.Suite
}

func (s *IntValidationSuite) Test_values_in_int_range() {
	validValues := []int{0, 1, 5, 10, 100}

	for _, value := range validValues {
		diagnostics := IntRange(0, 100)("exampleField", core.ScalarFromInt(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *IntValidationSuite) Test_values_outside_int_range() {
	invalidValues := []int{-1, 101, 150}

	for _, value := range invalidValues {
		diagnostics := IntRange(0, 100)("exampleField", core.ScalarFromInt(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be between 0 and 100")
	}
}

func (s *IntValidationSuite) Test_invalid_type_for_int_range() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromFloat(43.12), // Invalid type
		core.ScalarFromString("not-an-int"),
	}

	for _, value := range invalidValues {
		diagnostics := IntRange(0, 100)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be an integer")
	}
}

func (s *IntValidationSuite) Test_max_int_value() {
	validValues := []int{0, 50, 100}

	for _, value := range validValues {
		diagnostics := MaxInt(100)("exampleField", core.ScalarFromInt(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *IntValidationSuite) Test_exceeding_max_int_value() {
	invalidValues := []int{101, 150, 200}

	for _, value := range invalidValues {
		diagnostics := MaxInt(100)("exampleField", core.ScalarFromInt(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be less than or equal to 100")
	}
}

func (s *IntValidationSuite) Test_invalid_type_for_max_int() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromFloat(42.12), // Invalid type
		core.ScalarFromString("not-an-integer"),
	}

	for _, value := range invalidValues {
		diagnostics := MaxInt(100)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be an integer")
	}
}

func (s *IntValidationSuite) Test_min_int_value() {
	validValues := []int{1, 50, 100}

	for _, value := range validValues {
		diagnostics := MinInt(1)("exampleField", core.ScalarFromInt(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *IntValidationSuite) Test_below_min_int_value() {
	invalidValues := []int{-1, 0, -50}

	for _, value := range invalidValues {
		diagnostics := MinInt(1)("exampleField", core.ScalarFromInt(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be at least 1")
	}
}

func (s *IntValidationSuite) Test_invalid_type_for_min_int() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromFloat(102.14), // Invalid type
		core.ScalarFromString("not-an-int"),
	}

	for _, value := range invalidValues {
		diagnostics := MinInt(1)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be an integer")
	}
}

func TestIntValidationSuite(t *testing.T) {
	suite.Run(t, new(IntValidationSuite))
}
