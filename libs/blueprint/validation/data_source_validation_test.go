package validation

// This does not cover ensuring the filter.operator field is valid,
// as that validation is carried out while parsing the schema of a blueprint.

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

type DataSourceValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&DataSourceValidationTestSuite{})

func (s *DataSourceValidationTestSuite) SetUpTest(c *C) {
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

func (s *DataSourceValidationTestSuite) Test_reports_error_when_substitution_provided_in_data_source_name(c *C) {
	description := "EC2 instance for the application"
	dataSourceSchema := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"${variables.awsEC2InstanceName}": dataSourceSchema,
		},
		SourceMeta: map[string]*source.Meta{
			"${variables.awsEC2InstanceName}": {
				Position: source.Position{
					Line:   1,
					Column: 1,
				},
			},
		},
	}
	err := ValidateDataSourceName("${variables.awsEC2InstanceName}", dataSourceMap)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: ${..} substitutions can not be used in data source names, "+
			"found in data source \"${variables.awsEC2InstanceName}\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_succeeds_without_any_issues_for_a_valid_data_source(c *C) {
	dataSource := newTestValidDataSource()

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vpc": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vpc",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, IsNil)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_filter_is_missing(c *C) {
	aliasFor := "instanceConfigId"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		// Filter omitted.
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceMissingFilter)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing filter in "+
			"data source \"vmInstance\", every data source must have a filter",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_field_is_empty(c *C) {
	name1 := "Processor-Dev"
	name2 := "Processor-Prod"
	aliasFor := "instanceConfigId"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					// Field omitted.
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorIn,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &name1,
									},
								},
							},
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &name2,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceMissingFilterField)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing field in filter for "+
			"data source \"vmInstance\", field must be set for a data source filter",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_data_source_type_is_not_supported(c *C) {
	name1 := "Processor-Dev"
	name2 := "Processor-Prod"
	dataSourceField := "instanceConfigId"
	aliasFor := "instanceConfigId"

	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/unknown"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &dataSourceField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorIn,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &name1,
									},
								},
							},
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &name2,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to data source \"vmInstance\" having an "+
			"unsupported type \"aws/ec2/unknown\", this type is not made available by any of the loaded providers",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_filter_search_is_empty(c *C) {
	field := "instanceConfigId"
	aliasFor := "instanceConfigId"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorIn,
					},
					// Search omitted.
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceMissingFilterSearch)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing search in filter for "+
			"data source \"vmInstance\", at least one search value must be provided for a filter",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_data_source_has_filter_conflict(c *C) {
	field := "instanceConfigId"
	field2 := "instanceConfigId2"
	aliasFor := "instanceConfigId"
	search := "Production"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorEquals,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
				{
					Field: &core.ScalarValue{StringValue: &field2},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorEquals,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceFilterConflict)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a conflict between "+
			"the filter fields \"instanceConfigId\" and \"instanceConfigId2\" "+
			"in data source \"vmInstance\", you must use one or the other in the "+
			"filter section of the data source",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_filter_operator_is_not_provided(c *C) {
	field := "instanceConfigId"
	aliasFor := "instanceConfigId"
	search := "Production"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field:    &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						// Operator omitted.
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSourceFilterOperator)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: data source \"vmInstance\" has an empty filter operator, "+
			"you can choose from \"=\", \"!=\", \"in\", \"not in\", \"has key\", \"not has key\", \"contains\", "+
			"\"not contains\", \"starts with\", \"not starts with\", \"ends with\", \"not ends with\", \">\", \"<\", \">=\", \"<=\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_invalid_filter_operator_is_provided(c *C) {
	field := "instanceConfigId"
	aliasFor := "instanceConfigId"
	search := "Production"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperator("unknown"),
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSourceFilterOperator)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: invalid filter operator \"unknown\" has been provided in data source "+
			"\"vmInstance\", you can choose from \"=\", \"!=\", \"in\", \"not in\", \"has key\", \"not has key\", \"contains\", "+
			"\"not contains\", \"starts with\", \"not starts with\", \"ends with\", \"not ends with\", \">\", \"<\", \">=\", \"<=\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_unsupported_filter_operator_is_provided(c *C) {
	field := "instanceConfigId"
	aliasFor := "instanceConfigId"
	search := "Production"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						// DataSourceFilterOperatorContains is not supported for the `instanceConfigId`
						// filter field of the `aws/ec2/instance` data source type.
						Value: schema.DataSourceFilterOperatorContains,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeUnsupportedDataSourceFilterOperator)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: data source \"vmInstance\" does not support the filter"+
			" operator \"contains\" for field \"instanceConfigId\", supported operators are: \"=\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_unsupported_exported_field_type_is_provided(c *C) {
	searchField := "instanceConfigId"
	aliasFor := "serviceName"
	search := "Production"
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &searchField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorEquals,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"service": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldType("unknown"),
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSourceFieldType)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: unsupported field type \"unknown\" has been provided for export \"service\" in data source "+
			"\"vmInstance\", you can choose from: string, integer, float, boolean and array",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_errors_when_no_exported_fields_are_provided(c *C) {
	search := "Production"
	field := "instanceConfigId"

	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/ec2/instance"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &field},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorEquals,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				// No exports provided for the data source.
			},
		},
	}
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceMissingExports)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to missing exports for "+
			"data source \"vmInstance\", at least one field must be exported for a data source",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_providing_a_display_name_with_wrong_sub_type(c *C) {
	dataSource := newTestInvalidDisplayNameDataSource()
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
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
			"\"datasources.vmInstance\", resolved type \"object\" is not supported by display names, "+
			"only values that resolve as primitives are supported",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_providing_a_description_with_wrong_sub_type(c *C) {
	dataSource := newTestInvalidDescriptionDataSource()
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
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
			"\"datasources.vmInstance\", resolved type \"object\" is not supported by descriptions, "+
			"only values that resolve as primitives are supported",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_spec_definition_is_missing(c *C) {
	// aws/vpc2 incorrectly returns a nil spec definition.
	dataSource := newBaseVPCTestDataSource("aws/vpc2")
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceSpecDefMissing)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing spec definition for data source"+
			" \"vmInstance\" of type \"aws/vpc2\": spec definition not found during data source validation",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_no_filter_fields_are_defined(c *C) {
	// aws/vpc3 incorrectly has no filter fields.
	dataSource := newBaseVPCTestDataSource("aws/vpc3")
	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceSpecDefMissing)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to a missing fields definition for data source"+
			" \"vmInstance\" of type \"aws/vpc3\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_filter_field_is_not_supported(c *C) {
	dataSource := newTestValidDataSource()
	field := "unknownField"
	searchValue := "Production"
	dataSource.Filter = &schema.DataSourceFilters{
		Filters: []*schema.DataSourceFilter{
			{
				Field: &core.ScalarValue{StringValue: &field},
				Operator: &schema.DataSourceFilterOperatorWrapper{
					Value: schema.DataSourceFilterOperatorIn,
				},
				Search: &schema.DataSourceFilterSearch{
					Values: []*substitutions.StringOrSubstitutions{
						{
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &searchValue,
								},
							},
						},
					},
				},
			},
		},
	}

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeDataSourceFilterFieldNotSupported)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to the field \"unknownField\" in the filter for "+
			"data source \"vmInstance\" not being supported",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_invalid_search_values_are_provided(c *C) {
	dataSource := newTestInvalidSearchValuesDataSource()

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelWarning)
	c.Assert(
		diagnostics[0].Message,
		Equals,
		"Substitution returns \"any\" type, this may produce unexpected output "+
			"in the search value, search values are expected to be scalar values",
	)

	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidSubstitution)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to an invalid substitution found in \"datasources.vmInstance\", "+
			"resolved type \"object\" is not supported by search values, only values that resolve as primitives are supported",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_empty_field_export_is_provided(c *C) {
	dataSource := newTestValidDataSource()
	dataSource.Exports.Values["emptyExport"] = nil

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to the exported field \"emptyExport\" in data source "+
			"\"vmInstance\" having an empty value",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_exported_field_is_missing(c *C) {
	dataSource := newTestValidDataSource()
	aliasFor := "missingField"
	dataSource.Exports.Values["missingFieldAlias"] = &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{
			Value: schema.DataSourceFieldTypeString,
		},
		AliasFor: &core.ScalarValue{StringValue: &aliasFor},
	}

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to the exported field \"missingFieldAlias\" in data source "+
			"\"vmInstance\" not being supported, the exported field \"missingField\" is not present for data source type \"aws/vpc\"",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_when_exported_field_has_missing_type(c *C) {
	dataSource := newTestValidDataSource()
	aliasFor := "instanceConfigId"
	dataSource.Exports.Values["instanceIdAlias"] = &schema.DataSourceFieldExport{
		// Missing field type.
		AliasFor: &core.ScalarValue{StringValue: &aliasFor},
	}

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to export \"instanceIdAlias\" "+
			"in data source \"vmInstance\" missing a type",
	)
}

func (s *DataSourceValidationTestSuite) Test_reports_error_for_exported_field_type_mismatch(c *C) {
	dataSource := newTestValidDataSource()
	aliasFor := "instanceConfigId"
	dataSource.Exports.Values["instanceIdAlias"] = &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{
			Value: schema.DataSourceFieldTypeInteger,
		},
		AliasFor: &core.ScalarValue{StringValue: &aliasFor},
	}

	dataSourceMap := &schema.DataSourceMap{
		Values: map[string]*schema.DataSource{
			"vmInstance": dataSource,
		},
	}

	blueprint := &schema.Blueprint{
		DataSources: dataSourceMap,
	}

	diagnostics, err := ValidateDataSource(
		context.Background(),
		"vmInstance",
		dataSource,
		dataSourceMap,
		blueprint,
		&core.ParamsImpl{},
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
		s.dataSourceRegistry,
		core.NewNopLogger(),
	)
	c.Assert(diagnostics, HasLen, 0)
	c.Assert(err, NotNil)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	c.Assert(isLoadErr, Equals, true)
	c.Assert(loadErr.ReasonCode, Equals, ErrorReasonCodeInvalidDataSource)
	c.Assert(
		loadErr.Error(),
		Equals,
		"blueprint load error: validation failed due to the exported field \"instanceIdAlias\" in data source "+
			"\"vmInstance\" having an unexpected type, the data source field \"instanceConfigId\" has a"+
			" type of \"string\", but the exported type is \"integer\"",
	)
}

func newTestValidDataSource() *schema.DataSource {
	search := "Production"

	displayName := "VPC"
	description := "The VPC that resources in this blueprint will belong to"
	extrasEnabled := true
	x := 10
	y := 20
	filterField := "tags"
	aliasFor := "instanceConfigId"
	return &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						StringValue: &description,
					},
				},
			},
		},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &filterField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorHasKey,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										SubstitutionValue: &substitutions.Substitution{
											StringValue: &search,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		DataSourceMetadata: &schema.DataSourceMetadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &displayName,
						},
					},
				},
			},
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: map[string]*substitutions.StringOrSubstitutions{
					"networking.extras.v1": {
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									BoolValue: &extrasEnabled,
								},
							},
						},
					},
				},
			},
			Custom: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"visuals": {
						Fields: map[string]*core.MappingNode{
							"x": {
								Scalar: &core.ScalarValue{
									IntValue: &x,
								},
							},
							"y": {
								Scalar: &core.ScalarValue{
									IntValue: &y,
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
}

func newTestInvalidDisplayNameDataSource() *schema.DataSource {
	search := "Production"

	displayNamePrefix := "VPC"
	filterField := "tags"
	aliasFor := "instanceConfigId"
	return &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &filterField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorHasKey,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		DataSourceMetadata: &schema.DataSourceMetadata{
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
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
}

func newTestInvalidDescriptionDataSource() *schema.DataSource {
	search := "Production"
	filterField := "tags"
	aliasFor := "instanceConfigId"

	return &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
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
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &filterField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorHasKey,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
}

func newTestInvalidSearchValuesDataSource() *schema.DataSource {
	search := "Production"
	jsonToDecode := "{\"key\": \"value\"}"
	filterField := "tags"
	aliasFor := "instanceConfigId"

	return &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &filterField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorHasKey,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
									{
										// Object not supported for a search value.
										SubstitutionValue: &substitutions.Substitution{
											Function: &substitutions.SubstitutionFunctionExpr{
												FunctionName: "object",
												Arguments:    []*substitutions.SubstitutionFunctionArg{},
											},
										},
									},
									{
										// Any return type will produce warning diagnostic.
										SubstitutionValue: &substitutions.Substitution{
											Function: &substitutions.SubstitutionFunctionExpr{
												FunctionName: "jsondecode",
												Arguments: []*substitutions.SubstitutionFunctionArg{
													{
														Value: &substitutions.Substitution{
															StringValue: &jsonToDecode,
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
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
}

func newBaseVPCTestDataSource(dataSourceType string) *schema.DataSource {
	search := "Production"

	displayName := "VPC"
	description := "The VPC that resources in this blueprint will belong to"
	extrasEnabled := true
	x := 10
	y := 20
	filterField := "tags"
	aliasFor := "instanceConfigId"
	return &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: dataSourceType},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						StringValue: &description,
					},
				},
			},
		},
		Filter: &schema.DataSourceFilters{
			Filters: []*schema.DataSourceFilter{
				{
					Field: &core.ScalarValue{StringValue: &filterField},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorHasKey,
					},
					Search: &schema.DataSourceFilterSearch{
						Values: []*substitutions.StringOrSubstitutions{
							{
								Values: []*substitutions.StringOrSubstitution{
									{
										StringValue: &search,
									},
								},
							},
						},
					},
				},
			},
		},
		DataSourceMetadata: &schema.DataSourceMetadata{
			DisplayName: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							StringValue: &displayName,
						},
					},
				},
			},
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: map[string]*substitutions.StringOrSubstitutions{
					"networking.extras.v1": {
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									BoolValue: &extrasEnabled,
								},
							},
						},
					},
				},
			},
			Custom: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"visuals": {
						Fields: map[string]*core.MappingNode{
							"x": {
								Scalar: &core.ScalarValue{
									IntValue: &x,
								},
							},
							"y": {
								Scalar: &core.ScalarValue{
									IntValue: &y,
								},
							},
						},
					},
				},
			},
		},
		Exports: &schema.DataSourceFieldExportMap{
			Values: map[string]*schema.DataSourceFieldExport{
				"instanceId": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor: &core.ScalarValue{StringValue: &aliasFor},
				},
			},
		},
	}
}
