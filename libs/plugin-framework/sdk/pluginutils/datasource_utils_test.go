package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

type DataSourceUtilsSuite struct {
	suite.Suite
}

func (s *DataSourceUtilsSuite) Test_create_string_equals_filter() {
	filter := CreateStringEqualsFilter("testField", "testValue")

	s.Assert().Equal(
		&provider.ResolvedDataSourceFilters{
			Filters: []*provider.ResolvedDataSourceFilter{
				{
					Field: core.ScalarFromString("testField"),
					Search: &provider.ResolvedDataSourceFilterSearch{
						Values: []*core.MappingNode{
							core.MappingNodeFromString("testValue"),
						},
					},
					Operator: &schema.DataSourceFilterOperatorWrapper{
						Value: schema.DataSourceFilterOperatorEquals,
					},
				},
			},
		},
		filter,
	)
}

func (s *DataSourceUtilsSuite) Test_extract_first_match_from_filters() {
	filters := &provider.ResolvedDataSourceFilters{
		Filters: []*provider.ResolvedDataSourceFilter{
			{
				Field: core.ScalarFromString("arn"),
				Search: &provider.ResolvedDataSourceFilterSearch{
					Values: []*core.MappingNode{
						core.MappingNodeFromString("arn:aws:example:123456789012"),
					},
				},
				Operator: &schema.DataSourceFilterOperatorWrapper{
					Value: schema.DataSourceFilterOperatorEquals,
				},
			},
			{
				Field: core.ScalarFromString("name"),
				Search: &provider.ResolvedDataSourceFilterSearch{
					Values: []*core.MappingNode{
						core.MappingNodeFromString("example-name"),
					},
				},
				Operator: &schema.DataSourceFilterOperatorWrapper{
					Value: schema.DataSourceFilterOperatorEquals,
				},
			},
		},
	}

	result := ExtractFirstMatchFromFilters(filters, []string{"arn", "name"})
	s.Assert().Equal(core.MappingNodeFromString("arn:aws:example:123456789012"), result)

	result2 := ExtractFirstMatchFromFilters(filters, []string{"nonExistentField"})
	s.Assert().Nil(result2)
}

func (s *DataSourceUtilsSuite) Test_extract_match_from_filters() {
	filters := &provider.ResolvedDataSourceFilters{
		Filters: []*provider.ResolvedDataSourceFilter{
			{
				Field: core.ScalarFromString("testField"),
				Search: &provider.ResolvedDataSourceFilterSearch{
					Values: []*core.MappingNode{
						core.MappingNodeFromString("testValue"),
					},
				},
				Operator: &schema.DataSourceFilterOperatorWrapper{
					Value: schema.DataSourceFilterOperatorEquals,
				},
			},
		},
	}

	result := ExtractMatchFromFilters(filters, "testField")
	s.Assert().Equal(core.MappingNodeFromString("testValue"), result)

	result2 := ExtractMatchFromFilters(filters, "nonExistentField")
	s.Assert().Nil(result2)
}

func TestDataSourceUtilsSuite(t *testing.T) {
	suite.Run(t, new(DataSourceUtilsSuite))
}
