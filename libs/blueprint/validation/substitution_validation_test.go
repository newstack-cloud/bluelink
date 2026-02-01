package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type SubstitutionValidationTestSuite struct {
	functionRegistry   provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&SubstitutionValidationTestSuite{})

func (s *SubstitutionValidationTestSuite) SetUpTest(c *C) {
	s.functionRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim":       corefunctions.NewTrimFunction(),
			"trimprefix": corefunctions.NewTrimPrefixFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"datetime":   corefunctions.NewDateTimeFunction(&mockclock.StaticClock{}),
			"link":       corefunctions.NewLinkFunction(nil, nil),
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
			"aws/ec2/instance":           newTestEC2InstanceDataSource(),
			"aws/vpc":                    newTestVPCDataSource(),
			"aws/vpc2":                   newTestVPC2DataSource(),
			"aws/vpc3":                   newTestVPC3DataSource(),
			"celerity/exampleDataSource": newTestExampleDataSource(),
		},
	}
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_1(c *C) {
	subInputStr := "${trim(\"  hello world  \")}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			Params:       &core.ParamsImpl{},
			FuncRegistry: s.functionRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeString))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_2(c *C) {
	subInputStr := "${list(resources.exampleResource.name, datasources.exampleDataSource.name)}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
				},
			},
		},
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"exampleDataSource": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"name": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"exports.example",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeArray))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_3(c *C) {
	subInputStr := "${variables.environment}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"environment": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeInteger))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_4(c *C) {
	subInputStr := "${values.contentBuckets[0].name}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"contentBuckets": {
					Type: &schema.ValueTypeWrapper{
						Value: schema.ValueTypeArray,
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_5(c *C) {
	subInputStr := "${values.contentBuckets}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"contentBuckets": {
					Type: &schema.ValueTypeWrapper{
						Value: schema.ValueTypeArray,
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeArray))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_6(c *C) {
	subInputStr := "${object(position = 5, total = 5403.43, enabled = true)}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeObject))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_7(c *C) {
	subInputStr := "${elem.config.enabled}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					Each: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									ValueReference: &substitutions.SubstitutionValueReference{
										ValueName: "bucketConfig",
									},
								},
							},
						},
					},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_8(c *C) {
	subInputStr := "${i}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource2"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource2": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					Each: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									ValueReference: &substitutions.SubstitutionValueReference{
										ValueName: "bucketConfig",
									},
								},
							},
						},
					},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeInteger))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_9(c *C) {
	subInputStr := "${children.networking.vpcId}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	networkingBlueprintPath := "networking.blueprint.yml"
	blueprint := &schema.Blueprint{
		Include: &schema.IncludeMap{
			Values: map[string]*schema.Include{
				"networking": {
					Path: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &networkingBlueprintPath,
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_10(c *C) {
	subInputStr := "${resources.exampleResource1.spec.ids[].name}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
				},
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"ids": {
								Items: []*core.MappingNode{
									{
										Scalar: &core.ScalarValue{
											StringValue: &exampleResourceID,
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

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	// A warning is expected for the array index as bounds can't be verified
	// against the resource spec definition schema.
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeString))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_11(c *C) {
	subInputStr := "${exampleResource1}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
				},
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"ids": {
								Items: []*core.MappingNode{
									{
										Scalar: &core.ScalarValue{
											StringValue: &exampleResourceID,
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

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_passes_validation_for_valid_substitution_12(c *C) {
	// This test ensures that when a data source has `exports` set to `*`,
	// validation passes for a substitution that references a valid field
	// from the data source.
	subInputStr := "${datasources.exampleDataSourceExportsAll.name}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
				},
			},
		},
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"exampleDataSourceExportsAll": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						ExportAll: true,
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"exports.example",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 0)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeString))
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_a_var_ref_in_blueprint_without_variables(c *C) {
	subInputStr := "${variables.environment}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the variable \"environment\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_a_var_ref_is_missing_variable(c *C) {
	subInputStr := "${variables.versionName}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"environment": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the variable \"versionName\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_a_val_ref_in_blueprint_without_values(c *C) {
	subInputStr := "${values.bucketConfig}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the value \"bucketConfig\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_a_val_ref_is_missing_value(c *C) {
	subInputStr := "${values.functions}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"bucketConfig": {
					Type: &schema.ValueTypeWrapper{
						Value: schema.ValueTypeObject,
					},
					Value: &core.MappingNode{
						StringWithSubstitutions: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{
									SubstitutionValue: &substitutions.Substitution{
										DataSourceProperty: &substitutions.SubstitutionDataSourceProperty{
											DataSourceName: "buckets",
											FieldName:      "config",
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

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the value \"functions\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_self_referencing_value(c *C) {
	subInputStr := "${values.bucketConfig}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"bucketConfig": {
					Type: &schema.ValueTypeWrapper{
						Value: schema.ValueTypeObject,
					},
					Value: &core.MappingNode{
						StringWithSubstitutions: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{
									SubstitutionValue: &substitutions.Substitution{
										DataSourceProperty: &substitutions.SubstitutionDataSourceProperty{
											DataSourceName: "buckets",
											FieldName:      "config",
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

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"values.bucketConfig",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the value \"bucketConfig\" referencing itself",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_elem_ref_outside_of_resource(c *C) {
	subInputStr := "${elem}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"datasources.networking",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to an element reference being used outside of a resource",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_elem_ref_in_blueprint_without_resources(c *C) {
	subInputStr := "${elem}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.ReasonCode,
		Equals,
		ErrorReasonCodeInvalidSubstitution,
	)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to an element reference being used outside of a resource",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_elem_ref_in_missing_resource(c *C) {
	subInputStr := "${elem}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource2"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource2": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					Each: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									ValueReference: &substitutions.SubstitutionValueReference{
										ValueName: "bucketConfig",
									},
								},
							},
						},
					},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the resource \"exampleResource1\" for element reference "+
			"not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_elem_ref_in_resource_without_each(c *C) {
	subInputStr := "${elem}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource3"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource3": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					// Missing "Each" property, this isn't a valid resource template,
					// therefore "elem" cannot be used.
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the resource \"exampleResource3\" for element reference "+
			"not being a resource template, a resource template must have the `each` property defined",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_resource_prop_ref_for_blueprint_with_no_resources(c *C) {
	// Global identifiers are treated as resources if no functions are found
	// with the same name.
	subInputStr := "${exampleResource1.spec.id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.ReasonCode,
		Equals,
		ErrorReasonCodeInvalidSubstitution,
	)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the resource \"exampleResource1\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_resource_prop_ref_for_missing_resource(c *C) {
	// Global identifiers are treated as resources if no functions are found
	// with the same name.
	subInputStr := "${exampleResource1.spec.id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource3"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource3": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the resource \"exampleResource1\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_self_referencing_resource(c *C) {
	subInputStr := "${resources.exampleResource2.spec.id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource2"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource2": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the resource \"exampleResource2\" referencing itself",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_referencing_a_resource_index_for_a_non_template_resource(c *C) {
	subInputStr := "${resources.exampleResource3[0]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource3"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource3": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
					// Missing "Each" property, this isn't a valid resource template,
					// therefore "resources.exampleResources[0]" cannot be used.
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the index 0 is accessed for resource \"exampleResource3\" which is not "+
			"a resource template, a resource template must have the `each` property defined",
	)
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_diagnostic_when_referencing_a_resource_using_an_unknown_type(c *C) {
	subInputStr := "${resources.exampleResource1.spec[\"id\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/unknown"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, diagnostics, _ := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"Resource type \"celerity/unknown\" is not currently loaded, "+
			"when staging changes and deploying, you will need to make sure"+
			" the provider for the resource type is loaded.",
	)
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_diagnostic_when_referencing_a_non_core_function_not_in_registry(c *C) {
	subInputStr := "${customFunction(datasources.networking.vpcId)}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	_, diagnostics, _ := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource2",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"Function \"customFunction\" is not a core function, when staging changes and deploying, you "+
			"will need to make sure the provider is loaded.",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_referenced_resource_type_missing_spec(c *C) {
	subInputStr := "${resources.exampleResource1.spec[\"id\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResourceMissingSpecDefinition"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to a missing spec definition for resource \"exampleResource1\" "+
			"of type \"exampleResourceMissingSpecDefinition\" referenced in substitution: spec definition is nil",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_referenced_resource_type_missing_spec_schema(c *C) {
	subInputStr := "${resources.exampleResource1.spec[\"id\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceID := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResourceMissingSpecSchema"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to a missing spec definition schema for resource \"exampleResource1\" "+
			"of type \"exampleResourceMissingSpecSchema\" referenced in substitution",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_property_in_resource_spec_1(c *C) {
	subInputStr := "${resources.exampleResource1.spec[\"name\"].id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as [\"spec\"][\"name\"][\"id\"] is not valid for resource \"exampleResource1\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_property_in_resource_spec_2(c *C) {
	// spec without child properties is not a valid resource spec reference.
	subInputStr := "${resources.exampleResource1.spec}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the spec reference for resource \"exampleResource1\" is not valid",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_property_in_resource_metadata_1(c *C) {
	subInputStr := "${resources.exampleResource1.metadata}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource1"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata reference for resource \"exampleResource1\" is not valid",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_property_in_resource_metadata_2(c *C) {
	subInputStr := "${resources.exampleResource10.metadata[\"id\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource10"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource10": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource9",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata property \"id\" provided "+
			"for resource \"exampleResource10\" is not valid",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_property_of_display_name_in_resource_metadata_3(c *C) {
	subInputStr := "${resources.exampleResource8.metadata.displayName.id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource8"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource8": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource9",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata display name reference for resource \"exampleResource8\" "+
			"provided can not have children",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_resource_metadata_annotation(c *C) {
	subInputStr := "${resources.exampleResource12.metadata.annotations[\"http.v1\"].id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource12"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource12": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource11",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata annotations reference for resource \"exampleResource12\" was invalid, "+
			"must be of the form `metadata.annotations.<key>` or `metadata.annotations[\"<key>\"]`",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_missing_resource_metadata_annotation(c *C) {
	subInputStr := "${resources.exampleResource15.metadata.annotations[\"http.handler.v1\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource15"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource15": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource14",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata annotation \"http.handler.v1\""+
			" for resource \"exampleResource15\" was not found",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_invalid_resource_metadata_label(c *C) {
	subInputStr := "${resources.exampleResource10.metadata.labels[\"app\"].config.id}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource10"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource10": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource9",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata labels reference for resource \"exampleResource10\" was invalid, "+
			"must be of the form `metadata.labels.<key>` or `metadata.labels[\"<key>\"]`",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_accessing_missing_resource_metadata_label(c *C) {
	subInputStr := "${resources.exampleResource12.metadata.labels[\"app\"]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	exampleResourceName := "exampleResource12"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource12": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {
								Scalar: &core.ScalarValue{
									StringValue: &exampleResourceName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource11",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the metadata label \"app\""+
			" for resource \"exampleResource12\" was not found",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_an_incorrect_number_of_args_passed_to_function(c *C) {
	subInputStr := "${trimprefix(\"[1]-hello\", \"[1]-\", \"unexpected\")}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to an invalid number of arguments being provided for substitution function \"trimprefix\", "+
			"expected 2 but got 3",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_an_incorrect_string_choice_for_function_arg(c *C) {
	subInputStr := "${datetime(\"rfc3339nano\")}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to an invalid argument value being provided for substitution function "+
			"\"datetime\", expected argument 0 to be one of the following choices: unix, rfc3339, tag, tagcompact "+
			"but got \"rfc3339nano\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_in_blueprint_without_data_sources(c *C) {
	subInputStr := "${datasources.networking.vpc}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the data source \"networking\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_for_missing_data_source(c *C) {
	subInputStr := "${datasources.networking2.vpc}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the data source \"networking2\" not existing in the blueprint",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_self_referencing_data_source(c *C) {
	subInputStr := "${datasources.networking.vpc}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"datasources.networking",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the data source \"networking\" referencing itself",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_to_data_source_missing_exports(c *C) {
	subInputStr := "${datasources.networking.vpc}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource1",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to no fields being exported for data source \"networking\" referenced in substitution",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_to_data_source_missing_field_export(c *C) {
	subInputStr := "${datasources.networking.vpc}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the field \"vpc\" referenced in the substitution "+
			"not being an exported field for data source \"networking\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_to_data_source_export_missing_type(c *C) {
	subInputStr := "${datasources.networking.vpcId}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to the field \"vpcId\" referenced in the substitution "+
			"not having a type defined for data source \"networking\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_data_source_ref_array_accessor_for_string_field(c *C) {
	subInputStr := "${datasources.networking.vpcId[1]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed as the field \"vpcId\" being referenced with index \"1\" in the substitution "+
			"is not an array for data source \"networking\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_fails_validation_for_a_link_func_arg_referencing_a_resource_that_does_not_exist(c *C) {
	subInputStr := "${link(\"exampleResource1\", \"exampleResource2\")}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"vpcId": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeString,
								},
							},
						},
					},
				},
			},
		},
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/exampleResource"},
				},
			},
		},
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeSubFuncLinkArgResourceNotFound)
	c.Assert(
		loadErr.Err.Error(),
		Equals,
		"validation failed due to a missing resource \"exampleResource2\" being referenced "+
			"in the link function call argument at position 1 in \"resources.exampleResource3\"",
	)
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_for_resource_spec_array_index(c *C) {
	subInputStr := "${resources.exampleResource1.spec.ids[0].name}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(diagnostics[0].Context, NotNil)
	c.Assert(diagnostics[0].Context.ReasonCode, Equals, errors.ErrorReasonCodeAnyTypeWarning)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeString))
}

func (s *SubstitutionValidationTestSuite) Test_resolves_item_type_for_terminal_resource_spec_array_index(c *C) {
	subInputStr := "${resources.exampleResource1.spec.ids[0]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"exampleResource1": {
					Type: &schema.ResourceTypeWrapper{Value: "exampleResource"},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	// Terminal array index should resolve to the items type (object), not array.
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeObject))
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_for_data_source_array_index(c *C) {
	subInputStr := "${datasources.networking.hosts[0]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"networking": {
					Type: &schema.DataSourceTypeWrapper{Value: "celerity/exampleDataSource"},
					Exports: &schema.DataSourceFieldExportMap{
						Values: map[string]*schema.DataSourceFieldExport{
							"hosts": {
								Type: &schema.DataSourceFieldTypeWrapper{
									Value: schema.DataSourceFieldTypeArray,
								},
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(diagnostics[0].Context, NotNil)
	c.Assert(diagnostics[0].Context.ReasonCode, Equals, errors.ErrorReasonCodeAnyTypeWarning)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_for_child_export_array_index(c *C) {
	subInputStr := "${children.coreInfra.hosts[0].hostname}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	networkingBlueprintPath := "core-infra.blueprint.yml"
	blueprint := &schema.Blueprint{
		Include: &schema.IncludeMap{
			Values: map[string]*schema.Include{
				"coreInfra": {
					Path: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &networkingBlueprintPath,
							},
						},
					},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:     blueprint,
			Params:       &core.ParamsImpl{},
			FuncRegistry: s.functionRegistry,
			RefChainCollector: s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
			ChildExportLookup: func(
				childName string, exportName string, location *source.Meta,
			) (*schema.Export, error) {
				if childName == "coreInfra" && exportName == "hosts" {
					return &schema.Export{
						Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeArray},
					}, nil
				}
				return nil, nil
			},
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource3",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(diagnostics[0].Context, NotNil)
	c.Assert(diagnostics[0].Context.ReasonCode, Equals, errors.ErrorReasonCodeAnyTypeWarning)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_produces_warning_for_function_return_array_index(c *C) {
	subInputStr := "${jsondecode(variables.config)[0].host}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	blueprint := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"config": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	resolveType, diagnostics, err := ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, IsNil)
	// One warning for [0] array index on function return.
	c.Assert(len(diagnostics), Equals, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(diagnostics[0].Context, NotNil)
	c.Assert(diagnostics[0].Context.ReasonCode, Equals, errors.ErrorReasonCodeAnyTypeWarning)
	c.Assert(resolveType, Equals, string(substitutions.ResolvedSubExprTypeAny))
}

func (s *SubstitutionValidationTestSuite) Test_fails_for_function_return_index_on_string(c *C) {
	subInputStr := "${trim(\"  hello  \")[0]}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeSubFuncPathIndexOnNonArray)
}

func (s *SubstitutionValidationTestSuite) Test_fails_for_function_return_field_on_string(c *C) {
	subInputStr := "${trim(\"  hello  \").field}"
	stringOrSubs := &substitutions.StringOrSubstitutions{}
	err := yaml.Unmarshal([]byte(subInputStr), stringOrSubs)
	if err != nil {
		c.Fatalf("Failed to parse substitution: %v", err)
	}

	_, _, err = ValidateSubstitution(
		context.TODO(),
		stringOrSubs.Values[0].SubstitutionValue,
		/* nextLocation */ nil,
		&ValidationContext{
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.functionRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		/* usedInResourceDerivedFromTemplate */ false,
		"resources.exampleResource",
		"",
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeSubFuncPathFieldOnNonObject)
}
