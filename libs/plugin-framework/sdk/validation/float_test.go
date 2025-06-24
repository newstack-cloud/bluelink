package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type FloatValidationSuite struct {
	suite.Suite
}

func (s *FloatValidationSuite) Test_values_in_float_range() {
	validValues := []float64{0.0, 1.0, 5.5, 10.0, 100.0}

	for _, value := range validValues {
		diagnostics := FloatRange(0.0, 100.0)("exampleField", core.ScalarFromFloat(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *FloatValidationSuite) Test_values_outside_float_range() {
	invalidValues := []float64{-1.0, 101.0, 150.5}

	for _, value := range invalidValues {
		diagnostics := FloatRange(0.0, 100.50493)("exampleField", core.ScalarFromFloat(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be between 0 and 100.50493")
	}
}

func (s *FloatValidationSuite) Test_invalid_type_for_float_range() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(42), // Invalid type
		core.ScalarFromString("not-a-float"),
	}

	for _, value := range invalidValues {
		diagnostics := FloatRange(0.0, 100.0)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a float")
	}
}

func (s *FloatValidationSuite) Test_max_float_value() {
	validValues := []float64{0.0, 50.0, 100.0}

	for _, value := range validValues {
		diagnostics := MaxFloat(100.0)("exampleField", core.ScalarFromFloat(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *FloatValidationSuite) Test_exceeding_max_float_value() {
	invalidValues := []float64{101.0, 150.5, 200.0}

	for _, value := range invalidValues {
		diagnostics := MaxFloat(100.0)("exampleField", core.ScalarFromFloat(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be less than or equal to 100")
	}
}

func (s *FloatValidationSuite) Test_invalid_type_for_max_float() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(4212), // Invalid type
		core.ScalarFromString("not-a-float"),
	}

	for _, value := range invalidValues {
		diagnostics := MaxFloat(100.0)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a float")
	}
}

func (s *FloatValidationSuite) Test_min_float_value() {
	validValues := []float64{0.84, 50.0, 100.0}

	for _, value := range validValues {
		diagnostics := MinFloat(0.5)("exampleField", core.ScalarFromFloat(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *FloatValidationSuite) Test_below_min_float_value() {
	invalidValues := []float64{-0.1, 0.4, 0.49}

	for _, value := range invalidValues {
		diagnostics := MinFloat(0.5)("exampleField", core.ScalarFromFloat(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be at least 0.5")
	}
}

func (s *FloatValidationSuite) Test_invalid_type_for_min_float() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(921), // Invalid type
		core.ScalarFromString("not-a-float"),
	}

	for _, value := range invalidValues {
		diagnostics := MinFloat(0.5)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a float")
	}
}

func TestFloatValidationSuite(t *testing.T) {
	suite.Run(t, new(FloatValidationSuite))
}
