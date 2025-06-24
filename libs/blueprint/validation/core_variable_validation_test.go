package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
)

type CoreVariableValidationTestSuite struct{}

var _ = Suite(&CoreVariableValidationTestSuite{})

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_integer_variable(c *C) {
	maxRetries := 5
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"maxRetries": {
				IntValue: &maxRetries,
			},
		},
	}

	description := "Maximum number of retries for interacting with the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_float_variable(c *C) {
	timeoutInSeconds := 30.5
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"timeoutInSeconds": {
				FloatValue: &timeoutInSeconds,
			},
		},
	}

	description := "The timeout for the requests for the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_string_variable(c *C) {
	region := "us-east-1"
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &region,
			},
		},
	}

	allowedValue1 := "us-east-1"
	allowedValue2 := "us-west-1"
	description := "The region to deploy the blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			StringValue: &allowedValue1,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_bool_variable(c *C) {
	experimentalFeatures := true
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"experimentalFeatures": {
				BoolValue: &experimentalFeatures,
			},
		},
	}

	description := "Whether or not the application should include experimental features."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_when_value_is_not_provided_for_a_string_variable_with_a_default_value(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	allowedValue1 := "us-east-1"
	allowedValue2 := "us-west-1"
	defaultRegion := allowedValue1
	description := "The region to deploy the blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			StringValue: &defaultRegion,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_when_value_is_not_provided_for_an_integer_variable_with_a_default_value(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultMaxRetries := 5
	description := "Maximum number of retries for interacting with the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			IntValue: &defaultMaxRetries,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_when_value_is_not_provided_for_a_float_variable_with_a_default_value(c *C) {

	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultTimeoutInSeconds := 43.21
	description := "The timeout for the requests for the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			FloatValue: &defaultTimeoutInSeconds,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_succeeds_with_no_errors_when_value_is_not_provided_for_a_boolean_variable_with_a_default_value(c *C) {

	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultExperimentalFeatures := true
	description := "Whether or not the application should include experimental features."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			BoolValue: &defaultExperimentalFeatures,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, IsNil)
}

func (s *CoreVariableValidationTestSuite) Test_reports_error_when_substitution_provided_in_variable_name(c *C) {
	description := "The region to deploy the blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"${variables.region}": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"${variables.region}": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	err := ValidateVariableName("${variables.region}", varMap)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: ${..} substitutions can not be used in variable names, "+
			"found in variable \"${variables.region}\"",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_string_value_is_provided(c *C) {
	invalidValue := 4391
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				IntValue: &invalidValue,
			},
		},
	}

	description := "The region to deploy the blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrect type "+
			"used for variable \"region\", expected a value of type string but one of type integer was provided",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_integer_value_is_provided(c *C) {
	invalidValue := false
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"maxRetries": {
				BoolValue: &invalidValue,
			},
		},
	}

	description := "The maximum number of retries when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrect type "+
			"used for variable \"maxRetries\", expected a value of type integer but one of type boolean was provided",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_float_value_is_provided(c *C) {
	invalidValue := "experiments"
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"timeoutInSeconds": {
				StringValue: &invalidValue,
			},
		},
	}

	description := "The timeout when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrect type "+
			"used for variable \"timeoutInSeconds\", expected a value of type float but one of type string was provided",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_bool_value_is_provided(c *C) {
	invalidValue := 4305.29
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"experimentalFeatures": {
				FloatValue: &invalidValue,
			},
		},
	}

	description := "Whether or not experimental features should be enabled."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description: &core.ScalarValue{StringValue: &description},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrect type "+
			"used for variable \"experimentalFeatures\", expected a value of type boolean but one of type float was provided",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_an_invalid_default_is_provided_for_a_string(c *C) {
	validRegion := "us-east-1"
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &validRegion,
			},
		},
	}

	invalidValue := true
	description := "The region to deploy blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			BoolValue: &invalidValue,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value "+
			"for variable \"region\", boolean was provided when string was expected",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_an_invalid_default_is_provided_for_an_integer(c *C) {
	validMaxRetries := 3
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"maxRetries": {
				IntValue: &validMaxRetries,
			},
		},
	}

	invalidValue := "experiments"
	description := "The maximum number of retries when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			StringValue: &invalidValue,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value "+
			"for variable \"maxRetries\", string was provided when integer was expected",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_an_invalid_default_is_provided_for_a_float(c *C) {
	validTimeout := 30.0
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"timeoutInSeconds": {
				FloatValue: &validTimeout,
			},
		},
	}

	invalidValue := false
	description := "The timeout when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			BoolValue: &invalidValue,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value "+
			"for variable \"timeoutInSeconds\", boolean was provided when float was expected",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_an_invalid_default_is_provided_for_a_bool(c *C) {
	validExperimentalFeatures := true
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"experimentalFeatures": {
				BoolValue: &validExperimentalFeatures,
			},
		},
	}

	invalidValue := 9205.29
	description := "Whether or not experimental features should be enabled."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			FloatValue: &invalidValue,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value "+
			"for variable \"experimentalFeatures\", float was provided when boolean was expected",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_allowed_values_are_provided_for_a_string(c *C) {
	validRegion := "us-west-1"
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &validRegion,
			},
		},
	}

	validDefaultRegion := "eu-west-2"
	invalidValue1 := true
	invalidValue2 := 9115.82
	description := "The region to deploy blueprint resources to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				BoolValue: &invalidValue1,
			},
			{
				FloatValue: &invalidValue2,
			},
		},
		Default: &core.ScalarValue{
			StringValue: &validDefaultRegion,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	// Multiple errors are expected here.
	// Instead of simply checking the error message string,
	// we want to make sure the underlying errors are provided in the error struct
	// so they can be formatted by tools that use the blueprint framework (e.g. CLIs) as they see fit.
	c.Assert(loadErr.ChildErrors, HasLen, 2)

	expectedErrorMessages := []string{
		"blueprint load error: an invalid allowed value was provided, a boolean with the value \"true\" was provided when only strings are allowed",
		"blueprint load error: an invalid allowed value was provided, a float with the value \"9115.82\" was provided when only strings are allowed",
	}

	c.Assert(
		errorsToStrings(loadErr.ChildErrors),
		DeepEquals,
		expectedErrorMessages,
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_allowed_values_are_provided_for_an_integer(c *C) {
	validMaxRetries := 5
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"maxRetries": {
				IntValue: &validMaxRetries,
			},
		},
	}

	validDefaultMaxRetries := 3
	invalidValue1 := "Not an integer"
	invalidValue2 := false
	description := "The maximum number of retries when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &invalidValue1,
			},
			{
				BoolValue: &invalidValue2,
			},
		},
		Default: &core.ScalarValue{
			IntValue: &validDefaultMaxRetries,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	// Multiple errors are expected here.
	// Instead of simply checking the error message string,
	// we want to make sure the underlying errors are provided in the error struct
	// so they can be formatted by tools that use the blueprint framework (e.g. CLIs) as they see fit.
	c.Assert(loadErr.ChildErrors, HasLen, 2)

	expectedErrorMessages := []string{
		"blueprint load error: an invalid allowed value was provided, a string with the value \"Not an integer\" was provided when only integers are allowed",
		"blueprint load error: an invalid allowed value was provided, a boolean with the value \"false\" was provided when only integers are allowed",
	}

	c.Assert(
		errorsToStrings(loadErr.ChildErrors),
		DeepEquals,
		expectedErrorMessages,
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_invalid_allowed_values_are_provided_for_a_float(c *C) {
	validTimeoutInSeconds := 45.3
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"timeoutInSeconds": {
				FloatValue: &validTimeoutInSeconds,
			},
		},
	}

	validDefaultTimeoutInSeconds := 30.5
	invalidValue1 := "Not a float"
	// An explicit integer value should not be supported for a float variable,
	// this avoids confusion and ambiguous/unexpected behavior in the implementation
	// when it comes to dealing with integers and floats.
	// This may mean the user has to provide numbers explicitly with decimal points
	// in the blueprint for them to be floats (e.g. 30.0 instead of 30).
	// Generally speaking, use cases for floats as variables are likely to be rare.
	invalidValue2 := 540

	description := "The timeout to use when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &invalidValue1,
			},
			{
				IntValue: &invalidValue2,
			},
		},
		Default: &core.ScalarValue{
			FloatValue: &validDefaultTimeoutInSeconds,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	// Multiple errors are expected here.
	// Instead of simply checking the error message string,
	// we want to make sure the underlying errors are provided in the error struct
	// so they can be formatted by tools that use the blueprint framework (e.g. CLIs) as they see fit.
	c.Assert(loadErr.ChildErrors, HasLen, 2)

	expectedErrorMessages := []string{
		"blueprint load error: an invalid allowed value was provided, a string with the value \"Not a float\" was provided when only floats are allowed",
		"blueprint load error: an invalid allowed value was provided, an integer with the value \"540\" was provided when only floats are allowed",
	}

	c.Assert(
		errorsToStrings(loadErr.ChildErrors),
		DeepEquals,
		expectedErrorMessages,
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_allowed_values_are_provided_for_a_bool(c *C) {
	// Boolean variables do not support allowed values as binary enumeration does not make much sense,
	// it is better to set boolean variables that can be true or false and use other types for enumerable lists of options.
	// This test is to help with providing a better user experience by ensuring this limitation is made clear to the user.
	experimentalFeatures := true
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"experimentalFeatures": {
				BoolValue: &experimentalFeatures,
			},
		},
	}

	allowedValue1 := true
	allowedValue2 := false
	description := "Whether or not experimental features are enabled."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description: &core.ScalarValue{StringValue: &description},
		Default: &core.ScalarValue{
			BoolValue: &experimentalFeatures,
		},
		AllowedValues: []*core.ScalarValue{
			{
				BoolValue: &allowedValue1,
			},
			{
				BoolValue: &allowedValue2,
			},
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an allowed values list being provided for "+
			"boolean variable \"experimentalFeatures\", "+
			"boolean variables do not support allowed values enumeration",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_value_that_is_not_in_the_allowed_set_is_provided_for_a_string(c *C) {
	region := "us-west-2"
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &region,
			},
		},
	}

	allowedValue1 := "eu-west-1"
	allowedValue2 := "us-east-1"
	description := "The region to deploy the application to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid value being provided for "+
			"string variable \"region\", only the following values are supported: eu-west-1, us-east-1",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_value_that_is_not_in_the_allowed_set_is_provided_for_an_integer(c *C) {
	maxRetries := 20
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"maxRetries": {
				IntValue: &maxRetries,
			},
		},
	}

	allowedValue1 := 15
	allowedValue2 := 30
	description := "The maximum number of retries allowed when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				IntValue: &allowedValue1,
			},
			{
				IntValue: &allowedValue2,
			},
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid value being provided for "+
			"integer variable \"maxRetries\", only the following values are supported: 15, 30",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_value_that_is_not_in_the_allowed_set_is_provided_for_a_float(c *C) {
	timeoutInSeconds := 45.5
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"timeoutInSeconds": {
				FloatValue: &timeoutInSeconds,
			},
		},
	}

	allowedValue1 := 30.5
	allowedValue2 := 32.6
	description := "The maximum number of retries allowed when calling the core API."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				FloatValue: &allowedValue1,
			},
			{
				FloatValue: &allowedValue2,
			},
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid value being provided for "+
			"float variable \"timeoutInSeconds\", only the following values are supported: 30.50, 32.60",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_default_value_that_is_not_in_the_allowed_set_is_provided_for_a_string(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultRegion := "us-north-3"
	allowedValue1 := "eu-west-1"
	allowedValue2 := "us-east-1"
	description := "The region to deploy the application to."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			StringValue: &defaultRegion,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an"+
			" invalid default value being provided for "+
			"string variable \"region\", only the following values are supported: eu-west-1, us-east-1",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_default_value_that_is_not_in_the_allowed_set_is_provided_for_an_integer(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultBatchSize := 50000
	allowedValue1 := 10
	allowedValue2 := 25
	description := "The maximum size for a batch events streamed to order processing."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				IntValue: &allowedValue1,
			},
			{
				IntValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			IntValue: &defaultBatchSize,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxBatchSize": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxBatchSize": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxBatchSize", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value being provided for "+
			"integer variable \"maxBatchSize\", only the following values are supported: 10, 25",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_a_default_value_that_is_not_in_the_allowed_set_is_provided_for_a_float(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	defaultSampleRate := 0.5
	allowedValue1 := 0.1
	allowedValue2 := 0.3
	description := "Sample rate for distributed traces in the app."
	variableSchema := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description: &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{
			{
				FloatValue: &allowedValue1,
			},
			{
				FloatValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			FloatValue: &defaultSampleRate,
		},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"sampleRate": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"sampleRate": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "sampleRate", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value being provided for "+
			"float variable \"sampleRate\", only the following values are supported: 0.10, 0.30",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_errors_when_string_variable_with_explicit_empty_value_is_provided(c *C) {
	emptyRegion := ""
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &emptyRegion,
			},
		},
	}

	description := "The region to deploy the application to."
	variableSchema := &schema.Variable{
		Type:          &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description:   &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an empty value being provided for "+
			"variable \"region\", please provide a valid string value that is not empty",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_error_when_string_variable_without_a_default_value_is_not_provided(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	description := "The region to deploy the application to."
	variableSchema := &schema.Variable{
		Type:          &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description:   &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"region": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"region": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "region", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value not being provided for the "+
			"required variable \"region\", as it does not have a default",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_error_when_integer_variable_without_a_default_value_is_not_provided(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	description := "The maximum number of retries allowed when calling the core API."
	variableSchema := &schema.Variable{
		Type:          &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description:   &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"maxRetries": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"maxRetries": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "maxRetries", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value not being provided for the "+
			"required variable \"maxRetries\", as it does not have a default",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_error_when_float_variable_without_a_default_value_is_not_provided(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	description := "The timeout in seconds used when calling the core API."
	variableSchema := &schema.Variable{
		Type:          &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
		Description:   &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"timeoutInSeconds": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"timeoutInSeconds": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "timeoutInSeconds", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value not being provided for the "+
			"required variable \"timeoutInSeconds\", as it does not have a default",
	)
}

func (s *CoreVariableValidationTestSuite) Test_reports_error_when_boolean_variable_without_a_default_value_is_not_provided(c *C) {
	params := &core.ParamsImpl{
		BlueprintVariables: map[string]*core.ScalarValue{},
	}

	description := "Whether or not to enable experimental features in the app."
	variableSchema := &schema.Variable{
		Type:          &schema.VariableTypeWrapper{Value: schema.VariableTypeBoolean},
		Description:   &core.ScalarValue{StringValue: &description},
		AllowedValues: []*core.ScalarValue{},
	}
	varMap := &schema.VariableMap{
		Values: map[string]*schema.Variable{
			"experimentalFeatures": variableSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"experimentalFeatures": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	_, err := ValidateCoreVariable(context.Background(), "experimentalFeatures", variableSchema, varMap, params, true)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value not being provided for the "+
			"required variable \"experimentalFeatures\", as it does not have a default",
	)
}

func errorsToStrings(errs []error) []string {
	var result []string
	for _, err := range errs {
		result = append(result, err.Error())
	}
	return result
}
