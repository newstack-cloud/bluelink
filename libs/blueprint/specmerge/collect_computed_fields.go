package specmerge

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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

// CollectComputedWhenOmittedFields walks a resource spec definition schema
// alongside a resolved spec and returns the paths of every field marked as
// computed-when-omitted that has no value in the resolved spec, rooted at the
// given path (usually "spec").
//
// Such fields are treated as computed for the current change set so that a
// provider-assigned value (e.g. an auto-generated resource name) can be merged
// into the persisted resource state after deployment.
func CollectComputedWhenOmittedFields(
	schema *provider.ResourceDefinitionsSchema,
	spec *core.MappingNode,
	rootPath string,
) []string {
	fields := []string{}
	collectComputedWhenOmittedFields(schema, spec, rootPath, &fields)
	return fields
}

func collectComputedWhenOmittedFields(
	schema *provider.ResourceDefinitionsSchema,
	spec *core.MappingNode,
	currentPath string,
	fields *[]string,
) {
	if schema == nil ||
		schema.Computed ||
		schema.Type != provider.ResourceDefinitionsSchemaTypeObject {
		return
	}

	for fieldName, fieldSchema := range schema.Attributes {
		if fieldSchema == nil {
			continue
		}
		fieldPath := substitutions.RenderFieldPath(currentPath, fieldName)
		fieldValue := specObjectField(spec, fieldName)
		if fieldSchema.ComputedWhenOmitted &&
			!fieldSchema.Computed &&
			core.IsNilMappingNode(fieldValue) {
			*fields = append(*fields, fieldPath)
			continue
		}
		collectComputedWhenOmittedFields(fieldSchema, fieldValue, fieldPath, fields)
	}
}

func specObjectField(spec *core.MappingNode, fieldName string) *core.MappingNode {
	if spec == nil || spec.Fields == nil {
		return nil
	}
	return spec.Fields[fieldName]
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
