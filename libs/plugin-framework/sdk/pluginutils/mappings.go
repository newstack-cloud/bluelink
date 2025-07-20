package pluginutils

import (
	"fmt"
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

// AnyToMappingNode converts any JSON-like data to a MappingNode.
func AnyToMappingNode(data any) (*core.MappingNode, error) {
	switch v := data.(type) {
	case map[string]any:
		fields := make(map[string]*core.MappingNode)
		for key, value := range v {
			convertedValue, err := AnyToMappingNode(value)
			if err != nil {
				return nil, err
			}
			fields[key] = convertedValue
		}
		return &core.MappingNode{Fields: fields}, nil
	case []any:
		items := make([]*core.MappingNode, len(v))
		for i, item := range v {
			convertedItem, err := AnyToMappingNode(item)
			if err != nil {
				return nil, err
			}
			items[i] = convertedItem
		}
		return &core.MappingNode{Items: items}, nil
	case string:
		return core.MappingNodeFromString(v), nil
	case int:
		return core.MappingNodeFromInt(v), nil
	case int32:
		return core.MappingNodeFromInt(int(v)), nil
	case int64:
		return core.MappingNodeFromInt(int(v)), nil
	case float32:
		return core.MappingNodeFromFloat(float64(v)), nil
	case float64:
		return core.MappingNodeFromFloat(v), nil
	case bool:
		return core.MappingNodeFromBool(v), nil
	case nil:
		return &core.MappingNode{}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", data)
	}
}
