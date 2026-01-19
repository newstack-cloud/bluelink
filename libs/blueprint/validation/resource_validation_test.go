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

var (
	// True represents a boolean true value
	// that can be used as a reference in tests.
	True = true
	// False represents a boolean false value
	// that can be used as a reference in tests.
	False = false
)

type ResourceValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&ResourceValidationTestSuite{})

func (s *ResourceValidationTestSuite) SetUpTest(c *C) {
	s.funcRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim":       corefunctions.NewTrimFunction(),
			"trimprefix": corefunctions.NewTrimPrefixFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"jsondecode": corefunctions.NewJSONDecodeFunction(),
			"split":      corefunctions.NewSplitFunction(),
		},
	}
	s.refChainCollector = refgraph.NewRefChainCollector()
	s.resourceRegistry = internal.NewResourceRegistryMock(
		map[string]provider.Resource{
			"aws/ecs/service": newTestECSServiceResource(),
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

func (s *ResourceValidationTestSuite) Test_reports_error_when_substitution_provided_in_resource_name(c *C) {
	description := "EC2 instance for the application"
	resourceSchema := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "${variables.awsEC2InstanceName}"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"${variables.awsEC2InstanceName}": resourceSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"${variables.awsEC2InstanceName}": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	err := ValidateResourceName("${variables.awsEC2InstanceName}", resourceMap)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: ${..} substitutions can not be used in resource names, "+
			"found in resource \"${variables.awsEC2InstanceName}\"",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_errors_when_substitutions_used_in_spec_mapping_keys(c *C) {
	version := "1.0.0"
	resourceSchema := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "celerity/api"},
		Spec: &core.MappingNode{
			Items: []*core.MappingNode{
				{
					Fields: map[string]*core.MappingNode{
						"${variables.version}": {
							Scalar: &core.ScalarValue{
								StringValue: &version,
							},
						},
					},
					SourceMeta: &source.Meta{Position: source.Position{
						Line:   1,
						Column: 1,
					}},
					FieldsSourceMeta: map[string]*source.Meta{
						"${variables.version}": {Position: source.Position{
							Line:   1,
							Column: 1,
						}},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"api": resourceSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"api": {Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		},
	}
	err := PreValidateResourceSpec(context.TODO(), "api", resourceSchema, resourceMap)
	c.Assert(err, NotNil)
}

func (s *ResourceValidationTestSuite) Test_reports_errors_when_resource_type_is_not_supported(c *C) {
	name := "Unknown Resource"
	handler := "unknown.handler"

	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/unknown"},
		Metadata: &schema.Metadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &name,
					},
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"handler": {
					Scalar: &core.ScalarValue{
						StringValue: &handler,
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"unknownHandler": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"unknownHandler",
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
		"blueprint load error: validation failed due to resource \"unknownHandler\" having an "+
			"unsupported type \"aws/lambda/unknown\", this type is not made available by any of the loaded plugins",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_providing_a_display_name_with_wrong_sub_type(c *C) {
	resource := newTestInvalidDisplayNameResource()
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in "+
			"\"resources.testService\", resolved type \"object\" is not supported by display names, "+
			"only values that resolve as primitives are supported",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_providing_a_description_with_wrong_sub_type(c *C) {
	resource := newTestInvalidDescriptionResource()
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in "+
			"\"resources.testService\", resolved type \"object\" is not supported by descriptions, "+
			"only values that resolve as primitives are supported",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_metadata_label_key_has_substitution(c *C) {
	resource := newTestValidResource()
	resource.Metadata.Labels.Values["${variables.labelKey}"] = "test"
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to a label key containing a substitution in resource \"testService\", "+
			"the label key \"${variables.labelKey}\" can not contain substitutions",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_metadata_label_value_has_substitution(c *C) {
	resource := newTestValidResource()
	resource.Metadata.Labels.Values["app"] = "${variables.labelValue}"
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to a label value containing a substitution in resource \"testService\", "+
			"the label \"app\" with value \"${variables.labelValue}\" can not contain substitutions",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_annotation_key_has_substitution(c *C) {
	resource := newTestValidResource()
	annotationValue := "v1"
	resource.Metadata.Annotations.Values["${variables.annotationKey}"] = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &annotationValue,
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to an annotation key containing a substitution in resource \"testService\", "+
			"the annotation key \"${variables.annotationKey}\" can not contain substitutions",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_nested_condition_is_empty(c *C) {
	resource := newTestValidResource()
	// Empty nested condition added to "and" list.
	resource.Condition.And = append(resource.Condition.And, &schema.Condition{})
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to a nested condition for resource \"testService\" "+
			"being empty, all nested conditions must have a value defined",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_condition_resolves_incorrect_type(c *C) {
	resource := newTestValidResource()
	resource.Condition.And = append(resource.Condition.And, &schema.Condition{
		StringValue: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "object",
							Arguments:    []*substitutions.SubstitutionFunctionArg{},
						},
					},
				},
			},
		},
	})
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in "+
			"\"resources.testService\", resolved type \"object\" is not supported by conditions, "+
			"only values that resolve as booleans are supported",
	)
}

func (s *ResourceValidationTestSuite) Test_produces_warning_when_condition_resolves_any_type(c *C) {
	resource := newTestValidResource()
	boolJSON := "true"
	resource.Condition.And = append(resource.Condition.And, &schema.Condition{
		StringValue: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "jsondecode",
							Arguments: []*substitutions.SubstitutionFunctionArg{
								{
									Value: &substitutions.Substitution{
										StringValue: &boolJSON,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"Substitution returns \"any\" type, this may produce unexpected output "+
			"in the condition, conditions are expected to be boolean values",
	)
}

func (s *ResourceValidationTestSuite) Test_produces_warning_when_each_resolves_any_type(c *C) {
	resource := newTestValidResource()
	arrJSON := "[]"
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					Function: &substitutions.SubstitutionFunctionExpr{
						FunctionName: "jsondecode",
						Arguments: []*substitutions.SubstitutionFunctionArg{
							{
								Value: &substitutions.Substitution{
									StringValue: &arrJSON,
								},
							},
						},
					},
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"Substitution returns \"any\" type, this may produce unexpected output "+
			"in each, an array is expected",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_each_resolves_incorrect_type(c *C) {
	resource := newTestValidResource()
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					BoolValue: &True,
				},
			},
		},
	}
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in "+
			"\"resources.testService\", resolved type \"boolean\" is not supported in each, "+
			"only values that resolve as arrays are supported",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_link_selector_label_key_has_substitution(c *C) {
	resource := newTestValidResource()
	resource.LinkSelector.ByLabel.Values["${variables.labelKey}"] = "test"
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to a link selector \"byLabel\" key containing a substitution in resource \"testService\", "+
			"the link selector label key \"${variables.labelKey}\" can not contain substitutions",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_link_selector_label_value_has_substitution(c *C) {
	resource := newTestValidResource()
	resource.LinkSelector.ByLabel.Values["app"] = "${variables.labelValue}"
	resourceMap := &schema.ResourceMap{
		Values: map[string]*schema.Resource{
			"testService": resource,
		},
	}

	blueprint := &schema.Blueprint{
		Resources: resourceMap,
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
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
		"blueprint load error: validation failed due to a link selector \"byLabel\" value containing a substitution in resource \"testService\", "+
			"the link selector label \"app\" with value \"${variables.labelValue}\" can not contain substitutions",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_has_a_missing_dependency(c *C) {
	resource := newTestValidResource()
	resource.DependsOn = &schema.DependsOnList{
		StringList: schema.StringList{
			Values: []string{"missingResource"},
			SourceMeta: []*source.Meta{
				{
					Position: source.Position{
						Line:   1,
						Column: 1,
					},
				},
			},
		},
	}
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testService": resource,
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
		resource,
		blueprint.Resources,
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
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMissingResourceDependency)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing dependency \"missingResource\" "+
			"for resource \"testService\"",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_dependency_contains_substitution(c *C) {
	resource := newTestValidResource()
	resource.DependsOn = &schema.DependsOnList{
		StringList: schema.StringList{
			Values: []string{"resource-${variables.environment}"},
			SourceMeta: []*source.Meta{
				{
					Position: source.Position{
						Line:   1,
						Column: 1,
					},
				},
			},
		},
	}
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testService": resource,
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
		resource,
		blueprint.Resources,
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
		"blueprint load error: validation failed due to a dependency \"resource-${variables.environment}\" "+
			"containing a substitution in resource \"testService\", the dependency name \"resource-${variables.environment}\" "+
			"can not contain substitutions and must be a resource in the same blueprint",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_depends_on_itself(c *C) {
	resource := newTestValidResource()
	resource.DependsOn = &schema.DependsOnList{
		StringList: schema.StringList{
			Values: []string{"testService"},
			SourceMeta: []*source.Meta{
				{
					Position: source.Position{
						Line:   1,
						Column: 1,
					},
				},
			},
		},
	}
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testService": resource,
			},
		},
	}

	diagnostics, err := ValidateResource(
		context.Background(),
		"testService",
		resource,
		blueprint.Resources,
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
		"blueprint load error: validation failed due to a self-referencing dependency in resource \"testService\"",
	)
}

const (
	testClusterID = "resources.testCluster"
)

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_each_directly_references_another_resource(c *C) {
	resource := newTestValidResource()
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ResourceProperty: &substitutions.SubstitutionResourceProperty{
						ResourceName: "testService",
						Path: []*substitutions.SubstitutionPathItem{
							{FieldName: "spec"},
							{FieldName: "id"},
						},
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 10,
			},
		},
	}
	testServiceResource := newTestValidResource()
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testCluster": resource,
				"testService": testServiceResource,
			},
		},
	}

	refChainCollector := refgraph.NewRefChainCollector()
	refChainCollector.Collect(
		"resources.testService",
		testServiceResource,
		testClusterID,
		[]string{CreateSubRefPropTag(testClusterID, "each")},
	)

	err := ValidateResourceEachDependencies(
		blueprint,
		refChainCollector,
	)

	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeEachResourceDependency)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a resource \"resources.testCluster\" having a direct or transitive "+
			"dependency \"resources.testService\" in the each property, the each property can not depend on resources",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_indirectly_depends_on_another_resource(c *C) {
	resource := newTestValidResource()
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ValueReference: &substitutions.SubstitutionValueReference{
						ValueName: "testServiceIdWithPrefix",
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 10,
			},
		},
	}
	testServiceResource := newTestValidResource()
	testServiceIDWithPrefixValue := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: "string"},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							ResourceProperty: &substitutions.SubstitutionResourceProperty{
								ResourceName: "testService",
								Path: []*substitutions.SubstitutionPathItem{
									{
										FieldName: "spec",
									},
									{
										FieldName: "id",
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
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testCluster": resource,
				"testService": testServiceResource,
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"testServiceIdWithPrefix": testServiceIDWithPrefixValue,
			},
		},
	}

	testServiceIDWithPrefixValueID := "values.testServiceIdWithPrefix"
	refChainCollector := refgraph.NewRefChainCollector()
	refChainCollector.Collect(
		testServiceIDWithPrefixValueID,
		testServiceIDWithPrefixValue,
		testClusterID,
		[]string{CreateSubRefPropTag(testClusterID, "each")},
	)
	refChainCollector.Collect(
		"resources.testService",
		testServiceResource,
		testServiceIDWithPrefixValueID,
		[]string{CreateSubRefTag(testServiceIDWithPrefixValueID)},
	)

	err := ValidateResourceEachDependencies(
		blueprint,
		refChainCollector,
	)

	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeEachResourceDependency)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a resource \"resources.testCluster\" having a direct "+
			"or transitive dependency \"resources.testService\" in the each property, the each property can not depend on resources",
	)
}

const (
	testNetworkingStackID = "children.testNetworkingStack"
)

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_each_directly_references_a_child_blueprint(c *C) {
	resource := newTestValidResource()
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					Child: &substitutions.SubstitutionChild{
						ChildName: "testNetworkingStack",
						Path: []*substitutions.SubstitutionPathItem{
							{FieldName: "spec"},
							{FieldName: "id"},
						},
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 10,
			},
		},
	}
	networkingStackFilePath := "stacks/networking.blueprint.yml"
	testNetworkingStack := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &networkingStackFilePath,
				},
			},
		},
	}
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testCluster": resource,
			},
		},
		Include: &schema.IncludeMap{
			Values: map[string]*schema.Include{
				"testNetworkingStack": testNetworkingStack,
			},
		},
	}

	refChainCollector := refgraph.NewRefChainCollector()
	refChainCollector.Collect(
		testNetworkingStackID,
		testNetworkingStack,
		testClusterID,
		[]string{CreateSubRefPropTag(testClusterID, "each")},
	)

	err := ValidateResourceEachDependencies(
		blueprint,
		refChainCollector,
	)

	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeEachChildDependency)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a resource \"resources.testCluster\" having a direct or "+
			"transitive dependency on a child blueprint \"children.testNetworkingStack\" in the each property, "+
			"the each property can not depend on child blueprints",
	)
}

func (s *ResourceValidationTestSuite) Test_reports_error_when_resource_indirectly_depends_on_a_child_blueprint(c *C) {
	resource := newTestValidResource()
	resource.Each = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ValueReference: &substitutions.SubstitutionValueReference{
						ValueName: "testNetworkingStackVPC",
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 10,
			},
		},
	}
	networkingStackFilePath := "stacks/networking.blueprint.yml"
	testNetworkingStack := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &networkingStackFilePath,
				},
			},
		},
	}
	testNetworkingStackVPCValue := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: "string"},
		Value: &core.MappingNode{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Child: &substitutions.SubstitutionChild{
								ChildName: "testNetworkingStack",
								Path: []*substitutions.SubstitutionPathItem{
									{
										FieldName: "vpcId",
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
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"testCluster": resource,
			},
		},
		Include: &schema.IncludeMap{
			Values: map[string]*schema.Include{
				"testNetworkingStack": testNetworkingStack,
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"testNetworkingStackVPC": testNetworkingStackVPCValue,
			},
		},
	}

	testNetworkingStackVPCValueID := "values.testNetworkingStackVPC"
	refChainCollector := refgraph.NewRefChainCollector()
	refChainCollector.Collect(
		testNetworkingStackVPCValueID,
		testNetworkingStackVPCValue,
		testClusterID,
		[]string{CreateSubRefPropTag(testClusterID, "each")},
	)
	refChainCollector.Collect(
		testNetworkingStackID,
		testNetworkingStack,
		testNetworkingStackVPCValueID,
		[]string{CreateSubRefTag(testNetworkingStackVPCValueID)},
	)

	err := ValidateResourceEachDependencies(
		blueprint,
		refChainCollector,
	)

	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeEachChildDependency)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a resource \"resources.testCluster\" having a direct or "+
			"transitive dependency on a child blueprint \"children.testNetworkingStack\" in the each property, "+
			"the each property can not depend on child blueprints",
	)
}

func newTestValidResource() *schema.Resource {
	serviceName := "testService"
	displayNamePrefix := "Service-"
	serviceAnnotationPrefix := "service.v2024."
	serviceNo := "1"
	strToSplit := "a,b,c,d"
	delimiter := ","
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
		Metadata: &schema.Metadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &displayNamePrefix,
						},
					},
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &serviceNo,
						},
					},
				},
			},
			Labels: &schema.StringMap{
				Values: map[string]string{
					"service": "test",
				},
			},
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: map[string]*substitutions.StringOrSubstitutions{
					"service.v1": {
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &serviceAnnotationPrefix,
							},
							{
								SubstitutionValue: &substitutions.Substitution{
									StringValue: &serviceNo,
								},
							},
						},
					},
				},
			},
		},
		LinkSelector: &schema.LinkSelector{
			ByLabel: &schema.StringMap{
				Values: map[string]string{
					"service": "test",
				},
			},
		},
		Each: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "split",
							Arguments: []*substitutions.SubstitutionFunctionArg{
								{
									Value: &substitutions.Substitution{
										StringValue: &strToSplit,
									},
								},
								{
									Value: &substitutions.Substitution{
										StringValue: &delimiter,
									},
								},
							},
						},
					},
				},
			},
		},
		Condition: &schema.Condition{
			And: []*schema.Condition{
				{
					Or: []*schema.Condition{
						{
							Not: &schema.Condition{
								StringValue: &substitutions.StringOrSubstitutions{
									Values: []*substitutions.StringOrSubstitution{
										{
											SubstitutionValue: &substitutions.Substitution{
												BoolValue: &True,
											},
										},
									},
								},
							},
						},
						{
							StringValue: &substitutions.StringOrSubstitutions{
								Values: []*substitutions.StringOrSubstitution{
									{
										SubstitutionValue: &substitutions.Substitution{
											BoolValue: &False,
										},
									},
								},
							},
						},
					},
				},
				{
					StringValue: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									BoolValue: &True,
								},
							},
						},
					},
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"serviceName": {
					Scalar: &core.ScalarValue{
						StringValue: &serviceName,
					},
				},
			},
		},
	}
}

func newTestInvalidDisplayNameResource() *schema.Resource {
	serviceName := "testService"
	displayNamePrefix := "Service-"
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
		Metadata: &schema.Metadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &displayNamePrefix,
						},
					},
					{
						SubstitutionValue: &substitutions.Substitution{
							Function: &substitutions.SubstitutionFunctionExpr{
								FunctionName: "object",
								Arguments:    []*substitutions.SubstitutionFunctionArg{},
							},
						},
					},
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"serviceName": {
					Scalar: &core.ScalarValue{
						StringValue: &serviceName,
					},
				},
			},
		},
	}
}

func newTestInvalidDescriptionResource() *schema.Resource {
	serviceName := "testService"
	displayName := "Test Service"
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
		Metadata: &schema.Metadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &displayName,
						},
					},
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"serviceName": {
					Scalar: &core.ScalarValue{
						StringValue: &serviceName,
					},
				},
			},
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "object",
							Arguments:    []*substitutions.SubstitutionFunctionArg{},
						},
					},
				},
			},
		},
	}
}
