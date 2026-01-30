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
	s.resourceRegistry = internal.NewResourceRegistryMock(
		map[string]provider.Resource{},
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
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
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
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
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
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
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
		&ValidationContext{
			BpSchema:           blueprint,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidInclude)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing or empty path for include \"coreInfra\"",
	)
}

func (s *IncludeValidationTestSuite) Test_resolves_pure_string_include_path(c *C) {
	fileName := "child.yaml"
	path := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{StringValue: &fileName},
		},
	}
	resolveWorkingDir := func() (string, error) { return "/home/user", nil }

	resolved, ok := TryResolveIncludePath(path, resolveWorkingDir)
	c.Assert(ok, Equals, true)
	c.Assert(resolved, Equals, "child.yaml")
}

func (s *IncludeValidationTestSuite) Test_resolves_cwd_plus_string_include_path(c *C) {
	fileName := "/child.yaml"
	path := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					Function: &substitutions.SubstitutionFunctionExpr{
						FunctionName: "cwd",
					},
				},
			},
			{StringValue: &fileName},
		},
	}
	resolveWorkingDir := func() (string, error) { return "/home/user", nil }

	resolved, ok := TryResolveIncludePath(path, resolveWorkingDir)
	c.Assert(ok, Equals, true)
	c.Assert(resolved, Equals, "/home/user/child.yaml")
}

func (s *IncludeValidationTestSuite) Test_returns_false_for_unresolvable_substitution_in_path(c *C) {
	path := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					Variable: &substitutions.SubstitutionVariable{
						VariableName: "envName",
					},
				},
			},
		},
	}
	resolveWorkingDir := func() (string, error) { return "/home/user", nil }

	_, ok := TryResolveIncludePath(path, resolveWorkingDir)
	c.Assert(ok, Equals, false)
}

func (s *IncludeValidationTestSuite) Test_returns_false_for_nil_path(c *C) {
	_, ok := TryResolveIncludePath(nil, nil)
	c.Assert(ok, Equals, false)
}

func (s *IncludeValidationTestSuite) Test_returns_false_for_empty_path_values(c *C) {
	path := &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{},
	}
	_, ok := TryResolveIncludePath(path, nil)
	c.Assert(ok, Equals, false)
}

func (s *IncludeValidationTestSuite) Test_identifies_remote_include_from_path_prefix(c *C) {
	remotePaths := []string{
		"https://example.com/child.yaml",
		"http://example.com/child.yaml",
		"s3://my-bucket/child.yaml",
		"gs://my-bucket/child.yaml",
	}
	for _, remotePath := range remotePaths {
		path := remotePath
		include := &schema.Include{
			Path: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: &path},
				},
			},
		}
		c.Assert(IsRemoteInclude(include), Equals, true, Commentf("expected remote for path %q", path))
	}

	localPaths := []string{
		"/home/user/child.yaml",
		"child.yaml",
	}
	for _, localPath := range localPaths {
		path := localPath
		include := &schema.Include{
			Path: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: &path},
				},
			},
		}
		c.Assert(IsRemoteInclude(include), Equals, false, Commentf("expected local for path %q", path))
	}
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_returns_no_diagnostics_for_existing_file(c *C) {
	// Use this test file itself as an existing file.
	filePath := "include_validation_test.go"
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &filePath},
			},
		},
	}
	resolveWorkingDir := func() (string, error) { return ".", nil }

	diagnostics, err := ValidateIncludePathExists("testInclude", include, resolveWorkingDir, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_returns_error_for_missing_file(c *C) {
	filePath := "nonexistent-child-blueprint.yaml"
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &filePath},
			},
		},
	}
	resolveWorkingDir := func() (string, error) { return ".", nil }

	diagnostics, err := ValidateIncludePathExists("testInclude", include, resolveWorkingDir, nil)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeIncludePathNotFound)
	c.Assert(
		loadErr.Error(),
		Matches,
		`.*include path for "testInclude" resolving to .* which does not exist.*`,
	)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_returns_warning_diagnostic_for_directory(c *C) {
	// Use "." as a path that resolves to a directory.
	dirPath := "."
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &dirPath},
			},
		},
	}
	resolveWorkingDir := func() (string, error) { return ".", nil }

	diagnostics, err := ValidateIncludePathExists("testInclude", include, resolveWorkingDir, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Matches,
		`Include path for "testInclude" resolves to a directory .*`,
	)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_skips_remote_paths(c *C) {
	remotePath := "https://example.com/child.yaml"
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &remotePath},
			},
		},
	}

	diagnostics, err := ValidateIncludePathExists("testInclude", include, nil, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_skips_unresolvable_paths(c *C) {
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "envName",
						},
					},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludePathExists("testInclude", include, nil, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_handles_nil_include(c *C) {
	diagnostics, err := ValidateIncludePathExists("testInclude", nil, nil, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_detects_remote_include_from_sourceType_metadata(c *C) {
	sourceType := "aws/s3"
	include := &schema.Include{
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
			},
		},
	}

	c.Assert(IsRemoteInclude(include), Equals, true)
}

func (s *IncludeValidationTestSuite) Test_detects_remote_include_from_type_metadata(c *C) {
	typeVal := "gcs"
	include := &schema.Include{
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"type": {
					Scalar: &core.ScalarValue{
						StringValue: &typeVal,
					},
				},
			},
		},
	}

	c.Assert(IsRemoteInclude(include), Equals, true)
}

func (s *IncludeValidationTestSuite) Test_does_not_detect_remote_include_when_metadata_is_nil(c *C) {
	include := &schema.Include{}
	c.Assert(IsRemoteInclude(include), Equals, false)
	c.Assert(IsRemoteInclude(nil), Equals, false)
}

func (s *IncludeValidationTestSuite) Test_does_not_detect_remote_include_when_no_remote_fields(c *C) {
	bucket := "my-bucket"
	include := &schema.Include{
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"bucket": {
					Scalar: &core.ScalarValue{
						StringValue: &bucket,
					},
				},
			},
		},
	}

	c.Assert(IsRemoteInclude(include), Equals, false)
}

func (s *IncludeValidationTestSuite) Test_does_not_detect_remote_include_when_sourceType_is_empty(c *C) {
	emptySourceType := ""
	include := &schema.Include{
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &emptySourceType,
					},
				},
			},
		},
	}

	c.Assert(IsRemoteInclude(include), Equals, false)
}

func (s *IncludeValidationTestSuite) Test_validate_path_exists_skips_when_remote_source_metadata(c *C) {
	// A non-existent file, but with remote source metadata, should be skipped.
	filePath := "nonexistent-child.yaml"
	sourceType := "aws/s3"
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &filePath},
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
			},
		},
	}
	resolveWorkingDir := func() (string, error) { return ".", nil }

	diagnostics, err := ValidateIncludePathExists("testInclude", include, resolveWorkingDir, nil)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_warns_on_unknown_variable(c *C) {
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"known":   {Scalar: &core.ScalarValue{StringValue: strPtr("val")}},
				"unknown": {Scalar: &core.ScalarValue{StringValue: strPtr("val2")}},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"known": {
					Type:    &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					Default: &core.ScalarValue{StringValue: strPtr("default")},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(diagnostics[0].Message, Matches, `.*"unknown".*not defined.*child blueprint`)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_errors_on_missing_required_variable(c *C) {
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"provided": {Scalar: &core.ScalarValue{StringValue: strPtr("val")}},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"provided": {
					Type:    &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					Default: &core.ScalarValue{StringValue: strPtr("default")},
				},
				"requiredVar": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					// No default → required
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeIncludeMissingRequiredVar)
	c.Assert(loadErr.Error(), Matches, `.*required variable "requiredVar".*not being provided.*"childA"`)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_errors_on_type_mismatch(c *C) {
	intVal := 42
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"region": {Scalar: &core.ScalarValue{IntValue: &intVal}},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"region": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeIncludeVarTypeMismatch)
	c.Assert(loadErr.Error(), Matches, `.*variable "region".*type "integer".*expects type "string"`)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_no_diagnostics_when_all_match(c *C) {
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"region":   {Scalar: &core.ScalarValue{StringValue: strPtr("us-east-1")}},
				"replicas": {Scalar: &core.ScalarValue{IntValue: intPtr(3)}},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"region": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
				"replicas": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_resolves_single_sub_type(c *C) {
	// Parent blueprint defines an integer variable "parentCount".
	// Include passes ${variables.parentCount} to child variable expecting string → type mismatch.
	parentBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"parentCount": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
			},
		},
	}
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"childVar": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									Variable: &substitutions.SubstitutionVariable{
										VariableName: "parentCount",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"childVar": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           parentBp,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeIncludeVarTypeMismatch)
	c.Assert(loadErr.Error(), Matches, `.*variable "childVar".*type "integer".*expects type "string"`)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_interpolated_string_is_string_type(c *C) {
	// Interpolated string "prefix-${variables.parentRegion}" should resolve to "string".
	prefix := "prefix-"
	parentBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"parentRegion": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"region": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{StringValue: &prefix},
							{
								SubstitutionValue: &substitutions.Substitution{
									Variable: &substitutions.SubstitutionVariable{
										VariableName: "parentRegion",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"region": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	_, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           parentBp,
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	// No type mismatch: interpolated string = "string", child expects "string"
	c.Assert(err, IsNil)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_warns_non_primitive_sub_in_interpolation(c *C) {
	// Interpolated string with object() substitution → warning for non-primitive in string interpolation.
	prefix := "prefix-"
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"region": {
					StringWithSubstitutions: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{StringValue: &prefix},
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
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"region": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	// Should have warning for non-primitive sub in interpolation
	hasWarning := false
	for _, d := range diagnostics {
		if d.Level == core.DiagnosticLevelWarning &&
			c.Check(d.Message, Matches, `.*non-primitive type.*string interpolation.*`) {
			hasWarning = true
		}
	}
	c.Assert(hasWarning, Equals, true)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_allows_optional_variables(c *C) {
	// Child has a variable with a default → not required.
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"optionalVar": {
					Type:    &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					Default: &core.ScalarValue{StringValue: strPtr("default-value")},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_skips_when_child_has_no_variables(c *C) {
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"someVar": {Scalar: &core.ScalarValue{StringValue: strPtr("val")}},
			},
		},
	}
	childBp := &schema.Blueprint{
		// No Variables section
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_skips_when_include_has_no_variables(c *C) {
	// Include has no variables, child has no required vars.
	includeSchema := &schema.Include{}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"optionalVar": {
					Type:    &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					Default: &core.ScalarValue{StringValue: strPtr("default-value")},
				},
			},
		},
	}

	diagnostics, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_errors_when_no_variables_but_child_requires(c *C) {
	// Include has no variables section, child has required vars → errors.
	includeSchema := &schema.Include{}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"requiredA": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
				"requiredB": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
				},
				"optionalC": {
					Type:    &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
					Default: &core.ScalarValue{StringValue: strPtr("default")},
				},
			},
		},
	}

	_, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	// Should have 2 errors (requiredA, requiredB) but not optionalC
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeMultipleValidationErrors)
	c.Assert(loadErr.ChildErrors, HasLen, 2)
}

func (s *IncludeValidationTestSuite) Test_validates_include_variables_skips_type_check_for_complex_values(c *C) {
	// Fields/Items in variable values should skip type checking.
	includeSchema := &schema.Include{
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"config": {
					Fields: map[string]*core.MappingNode{
						"key": {Scalar: &core.ScalarValue{StringValue: strPtr("value")}},
					},
				},
			},
		},
	}
	childBp := &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"config": {
					Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
				},
			},
		},
	}

	_, err := ValidateIncludeVariables(
		context.Background(), "childA", includeSchema, nil, childBp,
		&ValidationContext{
			BpSchema:           &schema.Blueprint{},
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	// No type mismatch errors: complex types skip comparison
	c.Assert(err, IsNil)
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
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
