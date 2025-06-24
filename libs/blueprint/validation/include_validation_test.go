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
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

type IncludeValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&IncludeValidationTestSuite{})

func (s *IncludeValidationTestSuite) SetUpTest(c *C) {
	getWorkingDir := func() (string, error) {
		return "/home/user", nil
	}
	s.funcRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim":       corefunctions.NewTrimFunction(),
			"trimprefix": corefunctions.NewTrimPrefixFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"jsondecode": corefunctions.NewJSONDecodeFunction(),
			"cwd":        corefunctions.NewCWDFunction(getWorkingDir),
		},
	}
	s.refChainCollector = refgraph.NewRefChainCollector()
	s.resourceRegistry = &internal.ResourceRegistryMock{
		Resources: map[string]provider.Resource{},
	}
	s.dataSourceRegistry = &internal.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/ec2/instance": newTestEC2InstanceDataSource(),
			"aws/vpc":          newTestVPCDataSource(),
			"aws/vpc2":         newTestVPC2DataSource(),
			"aws/vpc3":         newTestVPC3DataSource(),
		},
	}
}

func (s *IncludeValidationTestSuite) Test_reports_error_when_substitution_provided_in_include_name(c *C) {
	includeSchema := createTestValidInclude()
	includeMap := &schema.IncludeMap{
		Values: map[string]*schema.Include{
			"${variables.awsEC2InstanceName}": includeSchema,
		},
	}
	err := ValidateIncludeName("${variables.awsEC2InstanceName}", includeMap)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidResource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: ${..} substitutions can not be used in include names, "+
			"found in include \"${variables.awsEC2InstanceName}\"",
	)
}

func (s *IncludeValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_child_blueprint_include(c *C) {
	includeSchema := createTestValidInclude()
	includeMap := &schema.IncludeMap{
		Values: map[string]*schema.Include{
			"coreInfra": includeSchema,
		},
	}
	blueprint := &schema.Blueprint{
		Include: includeMap,
	}

	_, err := ValidateInclude(
		context.Background(),
		"coreInfra",
		includeSchema,
		includeMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(err, IsNil)
}

func (s *IncludeValidationTestSuite) Test_reports_error_when_an_invalid_sub_is_provided_in_description(c *C) {
	includeSchema := createTestValidInclude()
	includeSchema.Description = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					// object() yields an object, not a string
					Function: &substitutions.SubstitutionFunctionExpr{
						FunctionName: "object",
						Arguments:    []*substitutions.SubstitutionFunctionArg{},
					},
				},
			},
		},
	}
	includeMap := &schema.IncludeMap{
		Values: map[string]*schema.Include{
			"coreInfra": includeSchema,
		},
	}
	blueprint := &schema.Blueprint{
		Include: includeMap,
	}

	_, err := ValidateInclude(
		context.Background(),
		"coreInfra",
		includeSchema,
		includeMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in \"include.coreInfra\", "+
			"resolved type \"object\" is not supported by descriptions, only values that resolve as primitives are supported",
	)
}

func (s *IncludeValidationTestSuite) Test_reports_error_when_an_invalid_sub_is_provided_in_include_path(c *C) {
	includeSchema := createTestValidInclude()
	includeSchema.Path = &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					// object() yields an object, not a string
					Function: &substitutions.SubstitutionFunctionExpr{
						FunctionName: "object",
						Arguments:    []*substitutions.SubstitutionFunctionArg{},
					},
				},
			},
		},
	}
	includeMap := &schema.IncludeMap{
		Values: map[string]*schema.Include{
			"coreInfra": includeSchema,
		},
	}
	blueprint := &schema.Blueprint{
		Include: includeMap,
	}

	_, err := ValidateInclude(
		context.Background(),
		"coreInfra",
		includeSchema,
		includeMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in \"include.coreInfra\", "+
			"resolved type \"object\" is not supported by include paths, only values that resolve as primitives are supported",
	)
}

func (s *IncludeValidationTestSuite) Test_reports_error_for_a_child_blueprint_include_with_an_empty_path(c *C) {
	databaseName := "${variables.databaseName}"
	path := ""
	sourceType := "aws/s3"
	bucket := "order-system-blueprints"
	region := "eu-west-1"
	description := "A child blueprint that creates a core infrastructure."
	includeSchema := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &path,
				},
			},
		},
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"databaseName": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &databaseName,
							},
						},
					},
				},
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &sourceType,
							},
						},
					},
				},
				"bucket": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &bucket,
							},
						},
					},
				},
				"region": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &region,
							},
						},
					},
				},
			},
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	}
	includeMap := &schema.IncludeMap{
		Values: map[string]*schema.Include{
			"coreInfra": includeSchema,
		},
	}
	blueprint := &schema.Blueprint{
		Include: includeMap,
	}

	_, err := ValidateInclude(
		context.Background(),
		"coreInfra",
		includeSchema,
		includeMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidInclude)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an empty path being provided for include \"coreInfra\"",
	)
}

func createTestValidInclude() *schema.Include {
	databaseName := "${variables.databaseName}"
	fileName := "core-infra.yml"
	sourceType := "aws/s3"
	bucket := "order-system-blueprints"
	region := "eu-west-1"
	description := "A child blueprint that creates a core infrastructure."
	return &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Function: &substitutions.SubstitutionFunctionExpr{
							FunctionName: "cwd",
						},
					},
				},
				{
					StringValue: &fileName,
				},
			},
		},
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"databaseName": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &databaseName,
							},
						},
					},
				},
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &sourceType,
							},
						},
					},
				},
				"bucket": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &bucket,
							},
						},
					},
				},
				"region": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								StringValue: &region,
							},
						},
					},
				},
			},
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	}
}
