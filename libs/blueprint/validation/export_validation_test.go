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

type ExportValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&ExportValidationTestSuite{})

func (s *ExportValidationTestSuite) SetUpTest(c *C) {
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
	s.resourceRegistry = &internal.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/ecs/service": newTestECSServiceResource(),
		},
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

func (s *ExportValidationTestSuite) Test_succeeds_with_no_errors_for_a_valid_export(c *C) {
	description := "The endpoint information to be used to connect to a cache cluster."
	field := "resources.cacheCluster.spec.id"
	exportSchema := &schema.Export{
		Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	}
	exportMap := &schema.ExportMap{
		Values: map[string]*schema.Export{
			"cacheEndpointInfo": exportSchema,
		},
	}
	serviceName := "cache-cluster"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"cacheCluster": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
					Spec: &core.MappingNode{
						Scalar: &core.ScalarValue{
							StringValue: &serviceName,
						},
					},
				},
			},
		},
		Exports: exportMap,
	}
	diagnostics, err := ValidateExport(
		context.Background(),
		"cacheEndpointInfo",
		exportSchema,
		exportMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, IsNil)
}

func (s *ExportValidationTestSuite) Test_reports_error_when_an_unsupported_export_type_is_provided(c *C) {
	description := "The endpoint information to be used to connect to a cache cluster."
	field := "resources.cacheCluster.spec.cacheNodes.endpoints"
	exportSchema := &schema.Export{
		// mapping[string, integer] is not a supported export type.
		Type: &schema.ExportTypeWrapper{Value: schema.ExportType("mapping[string, integer]")},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	}
	exportMap := &schema.ExportMap{
		Values: map[string]*schema.Export{
			"cacheEndpointInfo": exportSchema,
		},
	}
	serviceName := "cache-cluster"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"cacheCluster": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
					Spec: &core.MappingNode{
						Scalar: &core.ScalarValue{
							StringValue: &serviceName,
						},
					},
				},
			},
		},
		Exports: exportMap,
	}
	diagnostics, err := ValidateExport(
		context.Background(),
		"cacheEndpointInfo",
		exportSchema,
		exportMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidExport)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid export type of \"mapping[string, integer]\""+
			" being provided for export \"cacheEndpointInfo\". "+
			"The following export types are supported: string, object, integer, float, array, boolean",
	)
}

func (s *ExportValidationTestSuite) Test_reports_error_when_an_empty_export_field_is_provided(c *C) {
	description := "The endpoint information to be used to connect to a cache cluster."
	exportSchema := &schema.Export{
		Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeObject},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{},
	}
	exportMap := &schema.ExportMap{
		Values: map[string]*schema.Export{
			"cacheEndpointInfo": exportSchema,
		},
	}
	serviceName := "cache-cluster"
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"cacheCluster": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/ecs/service"},
					Spec: &core.MappingNode{
						Scalar: &core.ScalarValue{
							StringValue: &serviceName,
						},
					},
				},
			},
		},
		Exports: exportMap,
	}
	diagnostics, err := ValidateExport(
		context.Background(),
		"cacheEndpointInfo",
		exportSchema,
		exportMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidExport)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an empty field string being provided for export \"cacheEndpointInfo\"",
	)
}

func (s *ExportValidationTestSuite) Test_reports_error_when_an_incorrect_reference_is_provided(c *C) {
	description := "The endpoint information to be used to connect to a cache cluster."
	field := "resources.cacheCluster."
	exportSchema := &schema.Export{
		Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeObject},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		// Missing a valid attribute that can be extracted from a resource.
		Field: &core.ScalarValue{StringValue: &field},
	}
	exportMap := &schema.ExportMap{
		Values: map[string]*schema.Export{
			"cacheEndpointInfo": exportSchema,
		},
	}
	blueprint := &schema.Blueprint{
		Exports: exportMap,
	}
	diagnostics, err := ValidateExport(
		context.Background(),
		"cacheEndpointInfo",
		exportSchema,
		exportMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidReference)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an incorrectly formed reference to a resource "+
			"(\"resources.cacheCluster.\") in \"exports.cacheEndpointInfo\". "+
			"See the spec documentation for examples and rules for references",
	)
}
