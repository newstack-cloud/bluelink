package container

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/pkg/core"
	"github.com/two-hundred/celerity/libs/blueprint/pkg/schema"
	. "gopkg.in/check.v1"
)

type CustomVariableValidationTestSuite struct{}

var _ = Suite(&CustomVariableValidationTestSuite{})

func (s *CustomVariableValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_value_for_a_custom_variable(c *C) {
	instanceType := "t2.medium"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, IsNil)
}

func (s *CustomVariableValidationTestSuite) Test_succeeds_with_no_errors_when_value_is_not_provided_for_a_custom_variable_with_a_default_value(c *C) {
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	defaultInstanceType := "t2.large"
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		Default: &core.ScalarValue{
			StringValue: &defaultInstanceType,
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, IsNil)
}

func (s *CustomVariableValidationTestSuite) Test_reports_errors_when_multiple_value_types_are_provided_in_custom_type_options(c *C) {
	instanceType := "t2.medium"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testInvalidEC2InstanceTypeCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to mixed types "+
			"provided as options for variable type \"aws/ec2/instanceType\" used "+
			"in variable \"instanceType\", all options must be of the same scalar type",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_there_is_a_failure_to_load_custom_type_options(c *C) {
	instanceType := "t2.medium"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testFailToLoadOptionsCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error (1 child error): validation failed due to an error when loading options"+
			" for variable \"instanceType\" of custom type \"aws/ec2/instanceType\"",
	)
	c.Assert(len(loadErr.ChildErrors), Equals, 1)
	c.Assert(loadErr.ChildErrors[0].Error(), Equals, "failed to load options")
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_allowed_value_types_include_values_which_are_not_strings(c *C) {
	instanceType := "t2.micro"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	validOption := "t2.micro"
	invalidOption1 := 324
	invalidOption2 := false
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &validOption,
			},
			{
				IntValue: &invalidOption1,
			},
			{
				BoolValue: &invalidOption2,
			},
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	// Multiple errors are expected here.
	// Instead of simply checking the error message string,
	// we want to make sure the underlying errors are provided in the error struct
	// so they can be formatted by tools that use the blueprint framework (e.g. CLIs) as they see fit.
	c.Assert(loadErr.ChildErrors, HasLen, 2)

	expectedErrorMessages := []string{
		"an invalid allowed value was provided, an integer with the value \"324\" was provided when only aws/ec2/instanceTypes are allowed",
		"an invalid allowed value was provided, a boolean with the value \"false\" was provided when only aws/ec2/instanceTypes are allowed",
	}

	c.Assert(
		errorsToStrings(loadErr.ChildErrors),
		DeepEquals,
		expectedErrorMessages,
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_allowed_values_are_not_a_subset_of_the_labels_for_custom_type_options(c *C) {
	instanceType := "t2.medium"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	validOption := "t2.medium"
	invalidOption1 := "z4.large"
	invalidOption2 := "y7.medium"
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &validOption,
			},
			{
				StringValue: &invalidOption1,
			},
			{
				StringValue: &invalidOption2,
			},
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to invalid allowed values being provided"+
			" for variable \"instanceType\" of custom type \"aws/ec2/instanceType\". "+
			"See custom type documentation for possible values. "+
			"Invalid values provided: z4.large, y7.medium",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_default_value_is_not_a_string(c *C) {
	instanceType := "t2.large"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	invalidDefault := 3109
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		Default: &core.ScalarValue{
			IntValue: &invalidDefault,
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid type for a default value"+
			" for variable \"instanceType\", integer was provided when a custom variable type option of aws/ec2/instanceType was expected",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_default_value_is_not_one_of_the_custom_type_options(c *C) {
	instanceType := "t2.large"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"instanceType": {
				StringValue: &instanceType,
			},
		},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	unsupportedOptionDefault := "z4.large"
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		Default: &core.ScalarValue{
			StringValue: &unsupportedOptionDefault,
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value "+
			"for variable \"instanceType\" of custom type \"aws/ec2/instanceType\". "+
			"See custom type documentation for possible values. Invalid default value provided: z4.large",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_default_value_is_a_custom_type_option_but_is_not_an_allowed_value(c *C) {
	// This is to handle the case when a user further refines the set of possible values by combining
	// allowed values with a custom type.
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{},
	}

	customVariableType := &testEC2InstanceTypeCustomVariableType{}

	supportedDefaultNotInAllowedValues := "t2.xlarge"
	allowedValue1 := "t2.nano"
	allowedValue2 := "t2.micro"
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/ec2/instanceType"),
		Description: "The type of Amazon EC2 instance to deploy.",
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
		},
		Default: &core.ScalarValue{
			StringValue: &supportedDefaultNotInAllowedValues,
		},
	}
	err := ValidateCustomVariable(context.Background(), "instanceType", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid default value being provided for "+
			"variable \"instanceType\", only the following values are supported: t2.nano, t2.micro",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_no_value_is_provided_for_a_variable_with_no_default_value(c *C) {
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{},
	}

	customVariableType := &testRegionCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value not being provided for the "+
			"required variable \"region\", as it does not have a default",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_empty_string_is_explicitly_provided_for_a_value(c *C) {
	emptyRegion := ""
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &emptyRegion,
			},
		},
	}

	customVariableType := &testRegionCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an empty value being provided for "+
			"variable \"region\", please provide a valid aws/region value that is not empty",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_empty_string_is_explicitly_provided_for_a_default_value(c *C) {
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{},
	}

	customVariableType := &testRegionCustomVariableType{}

	emptyDefaultRegion := ""
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
		Default: &core.ScalarValue{
			StringValue: &emptyDefaultRegion,
		},
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an empty default aws/region value for "+
			"variable \"region\", you must provide a value when declaring a default in a blueprint",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_value_is_not_a_string(c *C) {
	invalidRegion := 43
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"region": {
				IntValue: &invalidRegion,
			},
		},
	}

	customVariableType := &testRegionCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrect type used for "+
			"variable \"region\", expected a value of type aws/region but one of type integer was provided",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_value_is_not_one_of_the_custom_type_options(c *C) {
	region := "eu-central-4"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &region,
			},
		},
	}

	customVariableType := &testRegionCustomVariableType{}

	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid value \"eu-central-4\" being provided for "+
			"variable \"region\", which is not a valid aws/region option, see the custom type documentation for more details",
	)
}

func (s *CustomVariableValidationTestSuite) Test_reports_error_when_provided_value_is_one_of_the_custom_type_options_but_is_not_an_allowed_value(c *C) {
	region := "eu-central-1"
	params := &testBlueprintParams{
		blueprintVariables: map[string]*core.ScalarValue{
			"region": {
				StringValue: &region,
			},
		},
	}

	customVariableType := &testRegionCustomVariableType{}

	allowedValue1 := "us-east-1"
	allowedValue2 := "us-west-2"
	allowedValue3 := "eu-west-1"
	variableSchema := &schema.Variable{
		Type:        schema.VariableType("aws/region"),
		Description: "The region to deploy the application to.",
		AllowedValues: []*core.ScalarValue{
			{
				StringValue: &allowedValue1,
			},
			{
				StringValue: &allowedValue2,
			},
			{
				StringValue: &allowedValue3,
			},
		},
	}
	err := ValidateCustomVariable(context.Background(), "region", variableSchema, params, customVariableType)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidVariable)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid value being provided for "+
			"variable \"region\", only the following values are supported: us-east-1, us-west-2, eu-west-1",
	)
}
