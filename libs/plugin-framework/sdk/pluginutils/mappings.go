package pluginutils

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// GetValueByPath is a helper function to extract a value from a mapping node
// that is a thin wrapper around the blueprint framework's `core.GetPathValue` function.
// Unlike core.GetPathValue, this function will not return an error,
// instead it will return nil and false if the value is not found
// or the provided path is not valid.
func GetValueByPath(
	fieldPath string,
	specData *core.MappingNode,
) (*core.MappingNode, bool) {
	value, err := core.GetPathValue(
		fieldPath,
		specData,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		return nil, false
	}

	return value, value != nil
}

// ShallowCopy creates a shallow copy of a map of MappingNodes, excluding
// the keys in the ignoreKeys slice.
func ShallowCopy(
	fields map[string]*core.MappingNode,
	ignoreKeys ...string,
) map[string]*core.MappingNode {
	copy := make(map[string]*core.MappingNode, len(fields))
	for k, v := range fields {
		if !slices.Contains(ignoreKeys, k) {
			copy[k] = v
		}
	}
	return copy
}
