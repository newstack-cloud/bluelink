package pluginutils

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetCurrentResourceStateSpecData returns the spec data for the current
// resource state from the changes object.
func GetCurrentResourceStateSpecData(changes *provider.Changes) *core.MappingNode {
	if changes == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	appliedResourceInfo := changes.AppliedResourceInfo
	if appliedResourceInfo.CurrentResourceState == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return appliedResourceInfo.CurrentResourceState.SpecData
}

// GetResolvedResourceSpecData returns the resolved spec data for the resource
// from changes.
func GetResolvedResourceSpecData(changes *provider.Changes) *core.MappingNode {
	if changes == nil || changes.AppliedResourceInfo.ResourceWithResolvedSubs == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec
}

// GetCurrentStateSpecDataFromResourceInfo extracts the current resource state's
// spec data from the resource info.
// If the resource info or current resource state is nil, it returns a MappingNode with an empty set of fields.
func GetCurrentStateSpecDataFromResourceInfo(
	resourceInfo *provider.ResourceInfo,
) *core.MappingNode {
	if resourceInfo == nil || resourceInfo.CurrentResourceState == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return resourceInfo.CurrentResourceState.SpecData
}

// HasModifiedField checks if the specified field path has been modified in the
// changes object.
func HasModifiedField(
	changes *provider.Changes,
	fieldPath string,
) bool {
	return GetModifiedField(changes, fieldPath) != nil
}

// GetModifiedField retrieves the modified field from the changes object
// based on the specified field path. If no such field exists, it returns nil.
func GetModifiedField(
	changes *provider.Changes,
	fieldPath string,
) *provider.FieldChange {
	if changes == nil {
		return nil
	}

	return getFieldChange(changes.ModifiedFields, fieldPath)
}

// HasNewField checks if the specified field path has been added as a new field
// in the changes object.
func HasNewField(
	changes *provider.Changes,
	fieldPath string,
) bool {
	return GetNewField(changes, fieldPath) != nil
}

// GetNewField retrieves the new field from the changes object based on the
// specified field path. If no such field exists, it returns nil.
func GetNewField(
	changes *provider.Changes,
	fieldPath string,
) *provider.FieldChange {
	if changes == nil {
		return nil
	}

	return getFieldChange(changes.NewFields, fieldPath)
}

func getFieldChange(
	fields []provider.FieldChange,
	fieldPath string,
) *provider.FieldChange {
	for _, field := range fields {
		if field.FieldPath == fieldPath {
			return &field
		}
	}
	return nil
}

// HasChildModifiedField checks if there is a modified field
// that is a child of the specified field path.
func HasChildModifiedField(
	changes *provider.Changes,
	fieldPathPrefix string,
) bool {
	if changes == nil {
		return false
	}

	return hasFieldWithPrefix(changes.ModifiedFields, fieldPathPrefix)
}

func hasFieldWithPrefix(
	fields []provider.FieldChange,
	prefix string,
) bool {
	dotNotationPrefix := prefix + "."
	bracketNotationPrefix := prefix + "["

	for _, modifiedField := range fields {

		if strings.HasPrefix(modifiedField.FieldPath, dotNotationPrefix) ||
			strings.HasPrefix(modifiedField.FieldPath, bracketNotationPrefix) {
			return true
		}
	}

	return false
}
