package pluginutils

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// OptionalValueExtractor is a helper struct that allows defining
// optional value extractors for a data source or the `GetExternalState`
// method of a resource.
// This is usually used to conditionally extract values that are not required
// based on some condition.
type OptionalValueExtractor[Input any] struct {
	// Name is the name of the extractor, used for logging and debugging.
	Name string
	// Condition is a function that returns true if the extractor should be applied.
	// If the condition returns false, the extractor will be skipped.
	Condition func(input Input) bool
	// Fields is a list of fields that the extractor will
	// populate in the target data map.
	Fields []string
	// Values is a function that returns the values to be extracted.
	// It should return a slice of MappingNode pointers, or an error if the extraction fails.
	// The values will be added to the target data map under the keys specified in the Fields slice
	// in the order they are defined.
	Values func(input Input) ([]*core.MappingNode, error)
}

// AdditionalValueExtractor is a helper struct that allows defining
// additional value extractors for a data source or the `GetExternalState`
// method of a resource.
// This is usually used to make additional service API calls
// to retrieve additional values that are needed to populate the data source
// or resource spec.
type AdditionalValueExtractor[Service any] struct {
	// Name is the name of the extractor, used for logging and debugging.
	Name string
	// Extract is a function that performs the service call
	// and extracts values from the service response.
	Extract func(
		ctx context.Context,
		filters *provider.ResolvedDataSourceFilters,
		targetData map[string]*core.MappingNode,
		service Service,
	) error
}

// RunOptionalValueExtractors runs a list of optional value extractors
// and returns true if any of the extractors were applied.
func RunOptionalValueExtractors[Input any](
	input Input,
	targetMap map[string]*core.MappingNode,
	extractors []OptionalValueExtractor[Input],
) error {
	for _, extractor := range extractors {
		if extractor.Condition(input) {
			for i, field := range extractor.Fields {
				values, err := extractor.Values(input)
				if err != nil {
					return fmt.Errorf(
						"%s failed to extract values for %s: %w",
						extractor.Name,
						field,
						err,
					)
				}
				targetMap[field] = values[i]
			}
		}
	}

	return nil
}

// RunAdditionalValueExtractors runs a list of additional value extractors
// and returns true if any of the extractors were applied.
func RunAdditionalValueExtractors[Service any](
	ctx context.Context,
	filters *provider.ResolvedDataSourceFilters,
	targetMap map[string]*core.MappingNode,
	extractors []AdditionalValueExtractor[Service],
	service Service,
) error {
	for _, extractor := range extractors {
		if err := extractor.Extract(ctx, filters, targetMap, service); err != nil {
			return fmt.Errorf(
				"%s failed to extract additional values: %w",
				extractor.Name,
				err,
			)
		}
	}

	return nil
}
