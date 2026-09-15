package transformutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// GetAnnotation retrieves an annotation from a resource
// using the provided key and fallback key.
//
// This is particularly useful for transformer plugins that need to extract
// annotations before resolution as a part of the transformation logic while retaining
// unresolved substitutions. `pluginutils` provides equivalent helpers
// targeted at provider plugins that extract annotations from resources after
// substitutions have been resolved.
//
// Provide an empty string for fallbackKey if you don't want to use
// a fallback.
func GetAnnotation(
	resource *schema.Resource,
	key string,
	fallbackKey string,
) (*core.MappingNode, bool) {
	if resource.Metadata == nil ||
		resource.Metadata.Annotations == nil {
		return nil, false
	}

	if annotation, ok := resource.Metadata.Annotations.Values[key]; ok {
		return mappingNodeFromStringSub(annotation), true
	}

	if fallbackKey != "" {
		if annotation, ok := resource.Metadata.Annotations.Values[fallbackKey]; ok {
			return mappingNodeFromStringSub(annotation), true
		}
	}

	return nil, false
}

func mappingNodeFromStringSub(
	stringOrSubs *substitutions.StringOrSubstitutions,
) *core.MappingNode {
	if len(stringOrSubs.Values) == 0 {
		return nil
	}

	if len(stringOrSubs.Values) == 1 {
		value := stringOrSubs.Values[0]
		if value != nil && value.StringValue != nil {
			return core.MappingNodeFromString(*value.StringValue)
		}
	}

	return &core.MappingNode{
		StringWithSubstitutions: stringOrSubs,
	}
}

// GetBoolAnnotation retrieves an annotation as a boolean flag, using the provided key and
// fallback key, falling back to defaultValue and reporting whether a boolean was found.
func GetBoolAnnotation(
	resource *schema.Resource,
	key string,
	fallbackKey string,
	defaultValue bool,
) (bool, bool) {
	annotation, found := GetAnnotation(resource, key, fallbackKey)
	if !found || annotation == nil || annotation.Scalar == nil {
		return defaultValue, false
	}

	if annotation.Scalar.BoolValue != nil {
		return *annotation.Scalar.BoolValue, true
	}

	if annotation.Scalar.StringValue == nil {
		return defaultValue, false
	}

	switch *annotation.Scalar.StringValue {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return defaultValue, false
	}
}
