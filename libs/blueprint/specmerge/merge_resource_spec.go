package specmerge

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// MergeResourceSpec merges a partially resolved resource with computed field values.
// Resource provider plugins can return a set of computed fields after they have deployed
// a resource.
// It is the responsibility of the blueprint container to merge these computed fields
// with the resolved resource spec containing values derived from the user-provided input
// and any default values.
func MergeResourceSpec(
	resolvedResource *provider.ResolvedResource,
	resourceName string,
	computedFieldValues map[string]*core.MappingNode,
	// The fields that are expected to be computed based on the results
	// of change staging.
	// Any computed field value that is not in this list will cause an error.
	expectedComputedFields []string,
) (*core.MappingNode, error) {
	if resolvedResource == nil {
		return nil, nil
	}

	mergedResource := core.CopyMappingNode(resolvedResource.Spec)
	for computedFieldPath, computedFieldValue := range computedFieldValues {
		if IsComputedFieldInList(expectedComputedFields, computedFieldPath) {
			err := core.InjectPathValue(
				core.ReplaceSpecWithRoot(computedFieldPath),
				computedFieldValue,
				mergedResource,
				core.MappingNodeMaxTraverseDepth,
			)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errUnexpectedComputedField(
				computedFieldPath,
				resourceName,
				expectedComputedFields,
			)
		}

	}

	return mergedResource, nil
}
