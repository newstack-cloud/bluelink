package plugintestsuites

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

func dataSourceValidateInput() *provider.DataSourceValidateInput {
	exampleAnnotationValue := "example-annotation-value"
	searchFor := "search-for"
	description := "example field description"
	return &provider.DataSourceValidateInput{
		SchemaDataSource: &schema.DataSource{
			Type: &schema.DataSourceTypeWrapper{
				Value: vpcDataSourceType,
			},
			DataSourceMetadata: &schema.DataSourceMetadata{
				Annotations: &schema.StringOrSubstitutionsMap{
					Values: map[string]*substitutions.StringOrSubstitutions{
						"example.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &exampleAnnotationValue,
								},
							},
						},
					},
				},
			},
			Filter: &schema.DataSourceFilters{
				Filters: []*schema.DataSourceFilter{
					{
						Field: core.ScalarFromString("examplefield"),
						Operator: &schema.DataSourceFilterOperatorWrapper{
							Value: schema.DataSourceFilterOperatorEquals,
						},
						Search: &schema.DataSourceFilterSearch{
							Values: []*substitutions.StringOrSubstitutions{
								{
									Values: []*substitutions.StringOrSubstitution{
										{
											StringValue: &searchFor,
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
					"example": {
						Type: &schema.DataSourceFieldTypeWrapper{
							Value: schema.DataSourceFieldTypeString,
						},
						AliasFor: core.ScalarFromString("examplefield"),
						Description: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &description,
								},
							},
						},
					},
				},
			},
		},
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceGetTypeInput() *provider.DataSourceGetTypeInput {
	return &provider.DataSourceGetTypeInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceGetTypeDescriptionInput() *provider.DataSourceGetTypeDescriptionInput {
	return &provider.DataSourceGetTypeDescriptionInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceGetSpecDefinitionInput() *provider.DataSourceGetSpecDefinitionInput {
	return &provider.DataSourceGetSpecDefinitionInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceGetFilterFieldsInput() *provider.DataSourceGetFilterFieldsInput {
	return &provider.DataSourceGetFilterFieldsInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceGetExamplesInput() *provider.DataSourceGetExamplesInput {
	return &provider.DataSourceGetExamplesInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func dataSourceFetchInput() *provider.DataSourceFetchInput {
	return &provider.DataSourceFetchInput{
		DataSourceWithResolvedSubs: &provider.ResolvedDataSource{
			Type: &schema.DataSourceTypeWrapper{
				Value: vpcDataSourceType,
			},
			DataSourceMetadata: &provider.ResolvedDataSourceMetadata{
				Annotations: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"example.annotation": core.MappingNodeFromString("example-annotation-value"),
					},
				},
			},
			Filter: &provider.ResolvedDataSourceFilters{
				Filters: []*provider.ResolvedDataSourceFilter{
					{
						Field: core.ScalarFromString("examplefield"),
						Operator: &schema.DataSourceFilterOperatorWrapper{
							Value: schema.DataSourceFilterOperatorEquals,
						},
						Search: &provider.ResolvedDataSourceFilterSearch{
							Values: []*core.MappingNode{
								core.MappingNodeFromString("search-for"),
							},
						},
					},
				},
			},
			Exports: map[string]*provider.ResolvedDataSourceFieldExport{
				"example": {
					Type: &schema.DataSourceFieldTypeWrapper{
						Value: schema.DataSourceFieldTypeString,
					},
					AliasFor:    core.ScalarFromString("examplefield"),
					Description: core.MappingNodeFromString("example field description"),
				},
			},
		},
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}
