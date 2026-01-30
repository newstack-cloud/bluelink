package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

type ValueValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&ValueValidationTestSuite{})

func (s *ValueValidationTestSuite) SetUpTest(c *C) {
	s.funcRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim":       corefunctions.NewTrimFunction(),
			"trimprefix": corefunctions.NewTrimPrefixFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"jsondecode": corefunctions.NewJSONDecodeFunction(),
		},
	}
	s.refChainCollector = refgraph.NewRefChainCollector()
	s.resourceRegistry = internal.NewResourceRegistryMock(
		map[string]provider.Resource{
			"exampleResource":                      &testExampleResource{},
			"exampleResourceMissingSpecDefinition": &testExampleResourceMissingSpecDefinition{},
			"exampleResourceMissingSpecSchema":     &testExampleResourceMissingSpecSchema{},
		},
	)
	s.dataSourceRegistry = &internal.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/ec2/instance": newTestEC2InstanceDataSource(),
			"aws/vpc":          newTestVPCDataSource(),
			"aws/vpc2":         newTestVPC2DataSource(),
			"aws/vpc3":         newTestVPC3DataSource(),
		},
	}
}

func (s *ValueValidationTestSuite) Test_reports_error_when_substitution_provided_in_value_name(c *C) {
	description := "The collection of regions to deploy the blueprint to"
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	}
	valueMap := &schema.ValueMap{
		Values: map[string]*schema.Value{
			"${variables.region}": valueSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"${variables.region}": {
				Position: source.Position{
					Line:   1,
					Column: 1,
				},
			},
		},
	}
	err := ValidateValueName("${variables.region}", valueMap)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: ${..} substitutions can not be used in value names, "+
			"found in value \"${variables.region}\"",
	)
}

func (s *ValueValidationTestSuite) Test_passes_validation_for_a_valid_value(c *C) {
	description := "The collection of regions to deploy the blueprint to"

	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeArray},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Function: &substitutions.SubstitutionFunctionExpr{
								FunctionName: "jsondecode",
								Arguments: []*substitutions.SubstitutionFunctionArg{
									{
										Value: &substitutions.Substitution{
											Variable: &substitutions.SubstitutionVariable{
												VariableName: "regions",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	diagnostics, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *ValueValidationTestSuite) Test_reports_error_for_invalid_sub_in_description(c *C) {
	value := "[\"eu-west-1\",\"eu-west-2\"]"
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				// This resolves an any type which is not supported in descriptions,
				// types must strictly resolve to strings for descriptions.
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "jsondecode",
							Arguments: []*substitutions.SubstitutionFunctionArg{
								{
									Value: &substitutions.Substitution{
										Variable: &substitutions.SubstitutionVariable{
											VariableName: "regions",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &value,
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)
	childErr, isChildErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildErr, Equals, true)
	c.Assert(childErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		childErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in \"values.regions\","+
			" resolved type \"any\" is not supported by descriptions, "+
			"only values that resolve as primitives are supported",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_value_type_is_missing(c *C) {
	description := "The collection of regions to deploy the blueprint to"
	regions := "[\"eu-west-1\",\"eu-west-2\"]"
	valueSchema := &schema.Value{
		// Missing type.
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &regions,
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed as the value \"regions\" "+
			"is missing a type, all values must have a type defined",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_unsupported_value_type_is_provided(c *C) {
	description := "The collection of regions to deploy the blueprint to"
	regions := "[\"eu-west-1\",\"eu-west-2\"]"
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueType("unknown")},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &regions,
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidValueType)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed as an unsupported type \"unknown\" was provided for value \"regions\", "+
			"you can choose from: string, integer, float, boolean, object and array",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_for_interpolated_string_for_value(c *C) {
	strVal := "This is a string"
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeArray},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Function: &substitutions.SubstitutionFunctionExpr{
								FunctionName: "jsondecode",
								Arguments: []*substitutions.SubstitutionFunctionArg{
									{
										Value: &substitutions.Substitution{
											Variable: &substitutions.SubstitutionVariable{
												VariableName: "regions",
											},
										},
									},
								},
							},
						},
					},
					{
						StringValue: &strVal,
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an interpolated string being used in \"values.regions\", "+
			"value type \"array\" does not support interpolated strings",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_string_literal_is_provided_for_an_array_value(c *C) {
	strVal := "This is a single string"
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeArray},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &strVal,
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)
	childErr, isChildErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildErr, Equals, true)
	c.Assert(childErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		childErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an interpolated string being used in \"values.regions\", "+
			"value type \"array\" does not support interpolated strings",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_sub_that_resolves_to_string_is_provided_for_array_value(c *C) {
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeArray},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Function: &substitutions.SubstitutionFunctionExpr{
								FunctionName: "trim",
								Arguments: []*substitutions.SubstitutionFunctionArg{
									{
										Value: &substitutions.Substitution{
											Variable: &substitutions.SubstitutionVariable{
												VariableName: "regions",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)
	childErr, isChildErr := loadErr.ChildErrors[0].(*errors.LoadError)
	c.Assert(isChildErr, Equals, true)
	c.Assert(childErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		childErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in \"values.regions\", "+
			"resolved type \"string\" is not supported by value of type \"array\"",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_static_integer_is_provided_for_array_value(c *C) {
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeArray},
		Value: &core.MappingNode{
			Scalar: core.ScalarFromInt(36),
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidValue)
	c.Assert(
		err.Error(),
		Equals,
		"blueprint load error: validation failed due to incorrect value content "+
			"in \"values.regions\", the content provided is of type \"integer\" but "+
			"the expected value type is \"array\"",
	)
}

func (s *ValueValidationTestSuite) Test_reports_error_when_nested_sub_reference_is_invalid(c *C) {
	valueSchema := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeObject},
		Value: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"regions": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									Function: &substitutions.SubstitutionFunctionExpr{
										FunctionName: "trim",
										Arguments: []*substitutions.SubstitutionFunctionArg{
											{
												Value: &substitutions.Substitution{
													Variable: &substitutions.SubstitutionVariable{
														VariableName: "missingVariable",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"regions": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"regions": valueSchema,
			},
		},
	}
	params := &core.ParamsImpl{}

	_, err := ValidateValue(
		context.TODO(),
		"regions",
		valueSchema,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             params,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)
	childErr, isChildErr := internal.UnpackLoadError(loadErr.ChildErrors[0])
	c.Assert(isChildErr, Equals, true)
	c.Assert(childErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		childErr.Error(),
		Equals,
		"blueprint load error: validation failed due to the variable \"missingVariable\" not existing in the blueprint",
	)
}
