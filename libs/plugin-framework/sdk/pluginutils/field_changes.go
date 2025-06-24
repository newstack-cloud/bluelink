package pluginutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FieldChangesToNewValueMap converts a slice of FieldChange structs to a map
// where the keys are the field paths and the values are pointers to the
// corresponding new value for the field. This is useful for creating lookups
// for potentially large sets of field changes, allowing for quick access
// to new values by field paths.
//
// Multiple slices of field changes can be provided, as a convenience
// when merging new and modified field changes into a single map.
func FieldChangesToNewValueMap(
	changes ...[]provider.FieldChange,
) map[string]*core.MappingNode {
	allChanges := []provider.FieldChange{}
	for _, changeSlice := range changes {
		allChanges = append(allChanges, changeSlice...)
	}

	fieldChangeMap := make(map[string]*core.MappingNode, len(allChanges))
	for _, change := range allChanges {
		if change.NewValue != nil {
			fieldChangeMap[change.FieldPath] = change.NewValue
		}
	}

	return fieldChangeMap
}

// FieldChangesToPrevValueMap converts a slice of FieldChange structs to a map
// where the keys are the field paths and the values are pointers to the
// corresponding previous value for the field. This is useful for creating lookups
// for potentially large sets of field changes, allowing for quick access
// to previous values by field paths.
//
// Multiple slices of field changes can be provided, as a convenience
// when merging new and modified field changes into a single map.
func FieldChangesToPrevValueMap(
	changes ...[]provider.FieldChange,
) map[string]*core.MappingNode {
	allChanges := []provider.FieldChange{}
	for _, changeSlice := range changes {
		allChanges = append(allChanges, changeSlice...)
	}

	fieldChangeMap := make(map[string]*core.MappingNode, len(allChanges))
	for _, change := range allChanges {
		if change.PrevValue != nil {
			fieldChangeMap[change.FieldPath] = change.PrevValue
		}
	}

	return fieldChangeMap
}

// MergeFieldChanges combines multiple slices of FieldChange into a single slice.
// This is useful for combining field changes for modified and new fields.
func MergeFieldChanges(
	changes ...[]provider.FieldChange,
) []provider.FieldChange {
	mergedChanges := []provider.FieldChange{}
	for _, changeSlice := range changes {
		mergedChanges = append(mergedChanges, changeSlice...)
	}
	return mergedChanges
}
