package specmerge

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// CollectComputedFields walks a resource spec definition schema and returns the
// paths of every field marked as computed, rooted at the given path (usually "spec").
//
// Array items use a "[0]" placeholder and map values a "[\"<key>\"]" placeholder,
// matching the placeholders understood by computed field checks.
func CollectComputedFields(schema *provider.ResourceDefinitionsSchema, rootPath string) []string {
	fields := []string{}
	collectComputedFields(schema, rootPath, &fields)
	return fields
}

func collectComputedFields(
	schema *provider.ResourceDefinitionsSchema,
	currentPath string,
	fields *[]string,
) {
	// A free-form map or an array with no declared item schema has no
	// sub-schema to descend into, so there are no nested computed fields to collect.
	if schema == nil {
		return
	}

	if schema.Computed {
		*fields = append(*fields, currentPath)
		return
	}

	switch schema.Type {
	case provider.ResourceDefinitionsSchemaTypeObject:
		for fieldName, fieldSchema := range schema.Attributes {
			collectComputedFields(fieldSchema, substitutions.RenderFieldPath(currentPath, fieldName), fields)
		}
	case provider.ResourceDefinitionsSchemaTypeMap:
		collectComputedFields(schema.MapValues, substitutions.RenderFieldPath(currentPath, "<key>"), fields)
	case provider.ResourceDefinitionsSchemaTypeArray:
		// 0 is a placeholder for any array index.
		collectComputedFields(schema.Items, fmt.Sprintf("%s[%d]", currentPath, 0), fields)
	case provider.ResourceDefinitionsSchemaTypeUnion:
		for _, unionSchema := range schema.OneOf {
			collectComputedFields(unionSchema, currentPath, fields)
		}
	}
}
