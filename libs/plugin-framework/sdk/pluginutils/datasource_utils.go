package pluginutils

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// GetDataSourceFilterOperator is a utility function that safely
// retrieves a data source filter operator from a resolved data source filter
// without producing a nil pointer dereference error.
// An empty string is returned if the operator is not set.
func GetDataSourceFilterOperator(
	dataSourceFilter *provider.ResolvedDataSourceFilter,
) schema.DataSourceFilterOperator {
	if dataSourceFilter == nil || dataSourceFilter.Operator == nil {
		return ""
	}

	return dataSourceFilter.Operator.Value
}

// GetDataSourceFilterSearchValues is a utility function that safely
// retrieves data source filter search values from a resolved data source filter
// without producing a nil pointer dereference error.
// It returns an empty slice if the search values are not set.
func GetDataSourceFilterSearchValues(
	dataSourceFilter *provider.ResolvedDataSourceFilter,
) []*core.MappingNode {
	if dataSourceFilter == nil || dataSourceFilter.Search == nil {
		return []*core.MappingNode{}
	}

	return dataSourceFilter.Search.Values
}

// GetDataSourceFilterSearchValue is a utility function that safely
// retrieves a data source filter search value from a resolved data source filter
// without producing a nil pointer dereference error.
// It returns nil if the search value is not set or if the index is out of bounds.
func GetDataSourceFilterSearchValue(
	dataSourceFilter *provider.ResolvedDataSourceFilter,
	index int,
) *core.MappingNode {
	if dataSourceFilter == nil || dataSourceFilter.Search == nil {
		return nil
	}

	if index < 0 || index >= len(dataSourceFilter.Search.Values) {
		return nil
	}

	return dataSourceFilter.Search.Values[index]
}

// CreateStringEqualsFilter creates a resolved data source filters object
// that contains a single equality filter for the given field and string value.
// This is useful for things like creating filters that match a specific resource ID
// to be compatible with helpers like the `AdditionalValueExtractor` used
// for extracting values for a data source or when getting the external state
// of a resource.
func CreateStringEqualsFilter(field string, value string) *provider.ResolvedDataSourceFilters {
	return &provider.ResolvedDataSourceFilters{
		Filters: []*provider.ResolvedDataSourceFilter{
			{
				Field: core.ScalarFromString(field),
				Operator: &schema.DataSourceFilterOperatorWrapper{
					Value: schema.DataSourceFilterOperatorEquals,
				},
				Search: &provider.ResolvedDataSourceFilterSearch{
					Values: []*core.MappingNode{
						core.MappingNodeFromString(value),
					},
				},
			},
		},
	}
}

// ExtractFirstMatchFromFilters extracts the first field match from the filters
// that matches one of the provided identifiers.
// It returns the first matching filter's search value as a MappingNode,
// or nil if no match is found.
// An example use case for this could be when you have an "ARN" and a unique "Name" field
// for an AWS resource, and you want to extract either value from the filters to
// use in a service call.
func ExtractFirstMatchFromFilters(
	filters *provider.ResolvedDataSourceFilters,
	fields []string,
) *core.MappingNode {
	for _, filter := range filters.Filters {
		if slices.Contains(fields, core.StringValueFromScalar(filter.Field)) &&
			GetDataSourceFilterOperator(
				filter,
			) == schema.DataSourceFilterOperatorEquals {
			return GetDataSourceFilterSearchValue(filter, 0)
		}
	}

	return nil
}

// ExtractMatchFromFilters extracts a specific field match from the filters
// that matches the provided field name with an equality operator.
// It returns the matching filter's search value as a MappingNode,
// or nil if no match is found.
func ExtractMatchFromFilters(
	filters *provider.ResolvedDataSourceFilters,
	field string,
) *core.MappingNode {
	for _, filter := range filters.Filters {
		if core.StringValueFromScalar(filter.Field) == field &&
			GetDataSourceFilterOperator(
				filter,
			) == schema.DataSourceFilterOperatorEquals {
			return GetDataSourceFilterSearchValue(filter, 0)
		}
	}

	return nil
}
