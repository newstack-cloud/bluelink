package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

////////////////////////////////////////////////////////////////////////////
// Allowed Values Constraint Tests
////////////////////////////////////////////////////////////////////////////

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitutions_used_in_string_field_with_fixed_allowed_values(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testStrPrefix-"
	resource.Spec.Fields["allowedStringValues"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.allowedStringValues\" contains substitutions and can not be validated against the allowed values. "+
			"When substitutions are resolved, this value must match one of the allowed values: allowedValue1, allowedValue2, allowedValue3",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitution_used_in_integer_field_with_fixed_allowed_values(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["allowedIntValues"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.allowedIntValues\" contains substitutions and can not be validated against the allowed values. "+
			"When substitutions are resolved, this value must match one of the allowed values: 10, 202, 300",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitution_used_in_float_field_with_fixed_allowed_values(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["allowedFloatValues"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.allowedFloatValues\" contains substitutions and can not be validated against the allowed values. "+
			"When substitutions are resolved, this value must match one of the allowed values: 10.11, 202.25, 340.3234",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_interpolation_used_in_integer_field_with_fixed_allowed_values(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testIntPrefix-"
	resource.Spec.Fields["allowedIntValues"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid resource item at path "+
			"\"resources.testHandler.spec.allowedIntValues\" where the integer type was expected, but string was found",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_interpolation_used_in_float_field_with_fixed_allowed_values(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testFloatPrefix-"
	resource.Spec.Fields["allowedFloatValues"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid resource item at path "+
			"\"resources.testHandler.spec.allowedFloatValues\" where the float type was expected, but string was found",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_value_not_in_allowed_value_list(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["allowedStringValues"] = core.MappingNodeFromString("unsupportedValue")
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is not allowed being provided at path "+
			"\"resources.testHandler.spec.allowedStringValues\", the value must be one of: allowedValue1, allowedValue2, allowedValue3",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_integer_value_not_in_allowed_value_list(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["allowedIntValues"] = core.MappingNodeFromInt(9999)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is not allowed being provided at path "+
			"\"resources.testHandler.spec.allowedIntValues\", the value must be one of: 10, 202, 300",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_float_value_not_in_allowed_value_list(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["allowedFloatValues"] = core.MappingNodeFromFloat(989998989.4029482372)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is not allowed being provided at path "+
			"\"resources.testHandler.spec.allowedFloatValues\", the value must be one of: 10.11, 202.25, 340.3234",
	)
}

////////////////////////////////////////////////////////////////////////////
// Min/Max Number Constraint Tests
////////////////////////////////////////////////////////////////////////////

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitution_used_in_integer_field_with_min_max_value_constraints(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxIntValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 2)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxIntValue\" contains substitutions and can not be validated against a minimum value. "+
			"When substitutions are resolved, this value must be greater than or equal to 100.",
	)

	c.Assert(diagnostics[1].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[1].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxIntValue\" contains substitutions and can not be validated against a maximum value. "+
			"When substitutions are resolved, this value must be less than or equal to 285.",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitution_used_in_float_field_with_min_max_value_constraints(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxFloatValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 2)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxFloatValue\" contains substitutions and can not be validated against a minimum value. "+
			"When substitutions are resolved, this value must be greater than or equal to 34.1304948234793.",
	)

	c.Assert(diagnostics[1].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[1].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxFloatValue\" contains substitutions and can not be validated against a maximum value. "+
			"When substitutions are resolved, this value must be less than or equal to 183.123123123123.",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_interpolation_used_in_integer_field_with_min_max_value_constraints(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testIntPrefix-"
	resource.Spec.Fields["minMaxIntValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid resource item at path "+
			"\"resources.testHandler.spec.minMaxIntValue\" where the integer type was expected, but string was found",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_interpolation_used_in_float_field_with_min_max_value_constraints(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testFloatPrefix-"
	resource.Spec.Fields["minMaxFloatValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid resource item at path "+
			"\"resources.testHandler.spec.minMaxFloatValue\" where the float type was expected, but string was found",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_integer_value_less_than_min_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxIntValue"] = core.MappingNodeFromInt(90)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is less than the minimum constraint at path "+
			"\"resources.testHandler.spec.minMaxIntValue\", 90 provided but the value must be greater than or equal to 100",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_integer_value_greater_than_max_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxIntValue"] = core.MappingNodeFromInt(320)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is greater than the maximum constraint at path "+
			"\"resources.testHandler.spec.minMaxIntValue\", 320 provided but the value must be less than or equal to 285",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_float_value_less_than_min_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxFloatValue"] = core.MappingNodeFromFloat(34.1304948234792)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is less than the minimum constraint at path "+
			"\"resources.testHandler.spec.minMaxFloatValue\", 34.1304948234792 provided but the value must be greater than or equal to 34.1304948234793",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_float_value_greater_than_max_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxFloatValue"] = core.MappingNodeFromFloat(183.123123123124)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that is greater than the maximum constraint at path "+
			"\"resources.testHandler.spec.minMaxFloatValue\", 183.123123123124 provided but the value must be less than or equal to 183.123123123123",
	)
}

////////////////////////////////////////////////////////////////////////////
// String Pattern Constraint Tests
////////////////////////////////////////////////////////////////////////////

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitutions_used_in_string_field_with_pattern_constraint(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testStrPrefix-"
	resource.Spec.Fields["patternStringValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.patternStringValue\" contains substitutions and can not be validated against a pattern. "+
			"When substitutions are resolved, this value must match the following pattern: \"^[a-zA-Z0-9]+$\".",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_value_that_does_not_match_pattern_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["patternStringValue"] = core.MappingNodeFromString("non@lphanumeric-value")
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a value that does not match the pattern constraint at path "+
			"\"resources.testHandler.spec.patternStringValue\", the value must match the pattern: ^[a-zA-Z0-9]+$",
	)
}

////////////////////////////////////////////////////////////////////////////
// Min/Max Length Constraint Tests (Strings, Arrays and Maps)
////////////////////////////////////////////////////////////////////////////

func (s *ResourceSpecValidationTestSuite) Test_reports_warnings_for_substitutions_used_in_string_field_with_min_max_len_constraint(c *C) {
	resource := createTestValidResource()
	testStrPrefix := "testStrPrefix-"
	resource.Spec.Fields["minMaxLenStringValue"] = &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStrPrefix,
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "testVariable",
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 2)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxLenStringValue\" contains substitutions and can not be validated against a minimum length. "+
			"When substitutions are resolved, this value must have 5 or more characters.",
	)

	c.Assert(diagnostics[1].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[1].Message,
		Equals,
		"The value of \"resources.testHandler.spec.minMaxLenStringValue\" contains substitutions and can not be validated against a maximum length. "+
			"When substitutions are resolved, this value must have 20 or less characters.",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_value_that_has_less_chars_than_min_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenStringValue"] = core.MappingNodeFromString("miss")
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a string value that is shorter than the minimum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenStringValue\", 4 characters provided when there must be at least 5 characters",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_value_that_has_more_chars_than_max_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenStringValue"] = core.MappingNodeFromString(
		"this string has too many characters, more than the 20 allowed",
	)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a string value that is longer than the maximum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenStringValue\", 61 characters provided when there must be at most 20 characters",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_array_value_that_has_less_items_than_min_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenArrayValue"] = &core.MappingNode{
		Items: []*core.MappingNode{
			core.MappingNodeFromString("item1"),
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an array that has less items than the minimum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenArrayValue\", 1 item provided when there must be at least 2 items",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_array_value_that_has_more_items_than_max_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenArrayValue"] = &core.MappingNode{
		Items: []*core.MappingNode{
			core.MappingNodeFromString("item1"),
			core.MappingNodeFromString("item2"),
			core.MappingNodeFromString("item3"),
			core.MappingNodeFromString("item4"),
			core.MappingNodeFromString("item5"),
			core.MappingNodeFromString("item6"),
			core.MappingNodeFromString("item7"),
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an array that has more items than the maximum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenArrayValue\", 7 items provided when there must be at most 5 items",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_map_value_that_has_less_items_than_min_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenMapValue"] = &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key1": core.MappingNodeFromString("value1"),
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a map that has less items than the minimum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenMapValue\", 1 item provided when there must be at least 2 items",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_map_value_that_has_more_items_than_max_length_constraint(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["minMaxLenMapValue"] = &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key1": core.MappingNodeFromString("value1"),
			"key2": core.MappingNodeFromString("value2"),
			"key3": core.MappingNodeFromString("value3"),
			"key4": core.MappingNodeFromString("value4"),
			"key5": core.MappingNodeFromString("value5"),
			"key6": core.MappingNodeFromString("value6"),
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a map that has more items than the maximum length constraint at path "+
			"\"resources.testHandler.spec.minMaxLenMapValue\", 6 items provided when there must be at most 5 items",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_string_value_custom_validation_failure(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["customValidateStringValue"] = core.MappingNodeFromString("invalidCustomValue")
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: invalid value for custom validate string field, must be 'validCustomValidateString'",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_int_value_custom_validation_failure(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["customValidateIntValue"] = core.MappingNodeFromInt(9568439)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: invalid value for custom validate int field, must be 39820",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_float_value_custom_validation_failure(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["customValidateFloatValue"] = core.MappingNodeFromFloat(403928.3029)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: invalid value for custom validate float field, must be 2430.30494",
	)
}

func (s *ResourceSpecValidationTestSuite) Test_reports_error_for_boolean_value_custom_validation_failure(c *C) {
	resource := createTestValidResource()
	resource.Spec.Fields["customValidateBoolValue"] = core.MappingNodeFromBool(false)
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"testVariable": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeFloat},
				},
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testHandler",
		resource,
		resourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		/* resourceDerivedFromTemplate */ false,
		core.NewNopLogger(),
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: invalid value for custom validate bool field, must be true",
	)
}
