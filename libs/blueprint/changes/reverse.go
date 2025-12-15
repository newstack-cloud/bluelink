package changes

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	// ErrorReasonCodeMaxReverseDepthExceeded is provided when the reason for an error
	// during reverse changeset generation is due to the maximum recursion depth being exceeded.
	// This protects against infinite recursion from cyclic child blueprint references.
	ErrorReasonCodeMaxReverseDepthExceeded errors.ErrorReasonCode = "max_reverse_depth_exceeded"

	// MaxReverseChangesetDepth is the maximum depth allowed for recursion when reversing
	// a changeset. This matches the MaxBlueprintDepth used elsewhere in the blueprint library
	// to protect against cycles in nested blueprint structures.
	MaxReverseChangesetDepth = 5
)

// ReverseChangeset generates a changeset that would undo the original changes.
// This is used for rolling back failed updates and destroys to restore the previous state.
//
// For update rollback: reverses field changes by swapping PrevValue/NewValue
// For destroy rollback: recreates removed resources from previousState
//
// Returns nil, nil if the original changeset or previous state is nil.
// Returns an error if the maximum recursion depth is exceeded.
func ReverseChangeset(
	original *BlueprintChanges,
	previousState *state.InstanceState,
) (*BlueprintChanges, error) {
	return reverseChangesetWithDepth(original, previousState, 0)
}

func reverseChangesetWithDepth(
	original *BlueprintChanges,
	previousState *state.InstanceState,
	depth int,
) (*BlueprintChanges, error) {
	if original == nil || previousState == nil {
		return nil, nil
	}

	if depth >= MaxReverseChangesetDepth {
		return nil, errMaxReverseDepthExceeded(depth)
	}

	reversed := newEmptyReversedChangeset(original)

	populateReversedResourceChanges(reversed, original, previousState)
	reversed.RemovedLinks = collectLinksToRemove(original)

	if err := populateReversedChildChanges(reversed, original, previousState, depth); err != nil {
		return nil, err
	}

	populateReversedExportChanges(reversed, original, previousState)

	return reversed, nil
}

// newEmptyReversedChangeset creates a new BlueprintChanges with initialized maps
// and slices, copying over unchanged fields from the original.
func newEmptyReversedChangeset(original *BlueprintChanges) *BlueprintChanges {
	return &BlueprintChanges{
		NewResources:     make(map[string]provider.Changes),
		ResourceChanges:  make(map[string]provider.Changes),
		RemovedResources: []string{},
		RemovedLinks:     []string{},
		NewChildren:      make(map[string]NewBlueprintDefinition),
		ChildChanges:     make(map[string]BlueprintChanges),
		RemovedChildren:  []string{},
		NewExports:       make(map[string]provider.FieldChange),
		ExportChanges:    make(map[string]provider.FieldChange),
		UnchangedExports: original.UnchangedExports,
		RemovedExports:   []string{},
		MetadataChanges:  reverseMetadataChanges(&original.MetadataChanges),
		ResolveOnDeploy:  []string{}, // Rollback should have all values already resolved
	}
}

// populateReversedResourceChanges handles all resource-related reversals:
// - Reverses field changes in modified resources
// - Marks new resources as removed
// - Recreates removed resources from previous state
func populateReversedResourceChanges(
	reversed *BlueprintChanges,
	original *BlueprintChanges,
	previousState *state.InstanceState,
) {
	// Reverse resource changes (swap PrevValue and NewValue in field changes)
	for resourceName, changes := range original.ResourceChanges {
		reversed.ResourceChanges[resourceName] = reverseResourceChanges(changes)
	}

	// New resources in original become removed resources in reverse
	for resourceName := range original.NewResources {
		reversed.RemovedResources = append(reversed.RemovedResources, resourceName)
	}

	// Removed resources in original become new resources in reverse
	// (recreate from previous state)
	for _, resourceName := range original.RemovedResources {
		resourceID, hasID := previousState.ResourceIDs[resourceName]
		if !hasID {
			continue
		}
		prevResource, hasResource := previousState.Resources[resourceID]
		if hasResource && prevResource != nil {
			reversed.NewResources[resourceName] = buildChangesFromResourceState(prevResource)
		}
	}
}

// populateReversedChildChanges handles all child blueprint reversals:
// - Recursively reverses changes in modified children
// - Marks new children as removed
// - Recreates removed children from previous state
func populateReversedChildChanges(
	reversed *BlueprintChanges,
	original *BlueprintChanges,
	previousState *state.InstanceState,
	depth int,
) error {
	// Recursively reverse child blueprint changes
	for childName, childChanges := range original.ChildChanges {
		childState := previousState.ChildBlueprints[childName]
		if childState != nil {
			reversedChild, err := reverseChangesetWithDepth(&childChanges, childState, depth+1)
			if err != nil {
				return err
			}
			if reversedChild != nil {
				reversed.ChildChanges[childName] = *reversedChild
			}
		}
	}

	// New children in original become removed children in reverse
	for childName := range original.NewChildren {
		reversed.RemovedChildren = append(reversed.RemovedChildren, childName)
	}

	// Removed children in original become new children in reverse
	for _, childName := range original.RemovedChildren {
		childState := previousState.ChildBlueprints[childName]
		if childState != nil {
			newChild, err := buildNewChildFromStateWithDepth(childState, depth+1)
			if err != nil {
				return err
			}
			reversed.NewChildren[childName] = newChild
		}
	}

	return nil
}

// populateReversedExportChanges handles all export-related reversals:
// - Reverses field changes in modified exports
// - Marks new exports as removed
// - Recreates removed exports from previous state
func populateReversedExportChanges(
	reversed *BlueprintChanges,
	original *BlueprintChanges,
	previousState *state.InstanceState,
) {
	// Reverse export changes
	for exportName, change := range original.ExportChanges {
		reversed.ExportChanges[exportName] = reverseFieldChange(change)
	}

	// New exports become removed exports
	for exportName := range original.NewExports {
		reversed.RemovedExports = append(reversed.RemovedExports, exportName)
	}

	// Removed exports become new exports (from previous state)
	for _, exportName := range original.RemovedExports {
		prevExport, hasExport := previousState.Exports[exportName]
		if hasExport && prevExport != nil {
			reversed.NewExports[exportName] = buildFieldChangeFromExportState(prevExport)
		}
	}
}

// reverseFieldChange swaps the PrevValue and NewValue in a FieldChange.
func reverseFieldChange(original provider.FieldChange) provider.FieldChange {
	return provider.FieldChange{
		FieldPath:    original.FieldPath,
		PrevValue:    original.NewValue,  // Swap
		NewValue:     original.PrevValue, // Swap
		MustRecreate: original.MustRecreate,
		Sensitive:    original.Sensitive,
	}
}

// reverseResourceChanges reverses all field changes in a resource's Changes struct.
func reverseResourceChanges(original provider.Changes) provider.Changes {
	reversed := provider.Changes{
		AppliedResourceInfo:       original.AppliedResourceInfo,
		MustRecreate:              original.MustRecreate,
		ModifiedFields:            make([]provider.FieldChange, len(original.ModifiedFields)),
		NewFields:                 make([]provider.FieldChange, 0),
		RemovedFields:             make([]string, 0),
		UnchangedFields:           original.UnchangedFields,
		ComputedFields:            original.ComputedFields,
		FieldChangesKnownOnDeploy: []string{},
		ConditionKnownOnDeploy:    true, // Rollback should have all conditions resolved
		NewOutboundLinks:          make(map[string]provider.LinkChanges),
		OutboundLinkChanges:       make(map[string]provider.LinkChanges),
		RemovedOutboundLinks:      []string{},
	}

	// Reverse modified fields
	for i, fc := range original.ModifiedFields {
		reversed.ModifiedFields[i] = reverseFieldChange(fc)
	}

	// New fields become removed fields
	for _, fc := range original.NewFields {
		reversed.RemovedFields = append(reversed.RemovedFields, fc.FieldPath)
	}

	// Removed fields become new fields (with previous value from the original's PrevValue)
	// Note: For removed fields, we need the previous value which may have been stored
	// in the original changeset or needs to be retrieved from state
	for _, fieldPath := range original.RemovedFields {
		// When a field was removed, we don't have its previous value in the changeset
		// The previous value would need to come from the resource state
		// For now, we record that this field needs to be added back
		reversed.NewFields = append(reversed.NewFields, provider.FieldChange{
			FieldPath: fieldPath,
			// NewValue will need to be populated from resource state during deployment
		})
	}

	// Reverse link changes
	// New outbound links in original become removed in reverse
	for linkTarget := range original.NewOutboundLinks {
		reversed.RemovedOutboundLinks = append(reversed.RemovedOutboundLinks, linkTarget)
	}

	// Removed outbound links in original become new in reverse
	// Note: The actual link data would need to come from state
	for _, linkTarget := range original.RemovedOutboundLinks {
		// Mark that this link needs to be recreated
		// The actual link changes would need to come from state
		reversed.NewOutboundLinks[linkTarget] = provider.LinkChanges{}
	}

	// Reverse outbound link changes
	for linkTarget, linkChanges := range original.OutboundLinkChanges {
		reversed.OutboundLinkChanges[linkTarget] = reverseLinkChanges(linkChanges)
	}

	return reversed
}

// reverseLinkChanges reverses field changes in link changes.
func reverseLinkChanges(original provider.LinkChanges) provider.LinkChanges {
	reversed := provider.LinkChanges{
		ModifiedFields:            make([]*provider.FieldChange, len(original.ModifiedFields)),
		NewFields:                 make([]*provider.FieldChange, 0),
		RemovedFields:             make([]string, 0),
		UnchangedFields:           original.UnchangedFields,
		FieldChangesKnownOnDeploy: []string{},
	}

	// Reverse modified fields
	for i, fc := range original.ModifiedFields {
		if fc != nil {
			reversedFC := reverseFieldChange(*fc)
			reversed.ModifiedFields[i] = &reversedFC
		}
	}

	// New fields become removed
	for _, fc := range original.NewFields {
		if fc != nil {
			reversed.RemovedFields = append(reversed.RemovedFields, fc.FieldPath)
		}
	}

	// Removed fields become new (would need previous value from state)
	for _, fieldPath := range original.RemovedFields {
		reversed.NewFields = append(reversed.NewFields, &provider.FieldChange{
			FieldPath: fieldPath,
		})
	}

	return reversed
}

// reverseMetadataChanges reverses metadata field changes.
func reverseMetadataChanges(original *MetadataChanges) MetadataChanges {
	if original == nil {
		return MetadataChanges{}
	}

	reversed := MetadataChanges{
		NewFields:       make([]provider.FieldChange, 0),
		ModifiedFields:  make([]provider.FieldChange, len(original.ModifiedFields)),
		UnchangedFields: original.UnchangedFields,
		RemovedFields:   make([]string, 0),
	}

	// Reverse modified fields
	for i, fc := range original.ModifiedFields {
		reversed.ModifiedFields[i] = reverseFieldChange(fc)
	}

	// New fields become removed fields
	for _, fc := range original.NewFields {
		reversed.RemovedFields = append(reversed.RemovedFields, fc.FieldPath)
	}

	// Removed fields become new fields
	for _, fieldPath := range original.RemovedFields {
		reversed.NewFields = append(reversed.NewFields, provider.FieldChange{
			FieldPath: fieldPath,
		})
	}

	return reversed
}

// buildChangesFromResourceState creates a Changes struct that would recreate
// a resource from its previous state.
func buildChangesFromResourceState(resource *state.ResourceState) provider.Changes {
	return provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID:   resource.ResourceID,
			ResourceName: resource.Name,
			// SpecData is restored from the resource state
		},
		// No field changes needed - this is a full recreation from state
		ModifiedFields:         []provider.FieldChange{},
		NewFields:              []provider.FieldChange{},
		RemovedFields:          []string{},
		UnchangedFields:        []string{},
		ComputedFields:         []string{},
		ConditionKnownOnDeploy: true,
		NewOutboundLinks:       make(map[string]provider.LinkChanges),
		OutboundLinkChanges:    make(map[string]provider.LinkChanges),
		RemovedOutboundLinks:   []string{},
	}
}

// buildNewChildFromStateWithDepth creates a NewBlueprintDefinition from an existing child state.
// It tracks recursion depth to protect against infinite recursion from cyclic references.
func buildNewChildFromStateWithDepth(childState *state.InstanceState, depth int) (NewBlueprintDefinition, error) {
	if depth >= MaxReverseChangesetDepth {
		return NewBlueprintDefinition{}, errMaxReverseDepthExceeded(depth)
	}

	newChild := NewBlueprintDefinition{
		NewResources: make(map[string]provider.Changes),
		NewChildren:  make(map[string]NewBlueprintDefinition),
		NewExports:   make(map[string]provider.FieldChange),
	}

	// Add all resources from the child state
	for resourceName, resourceID := range childState.ResourceIDs {
		if resource, ok := childState.Resources[resourceID]; ok && resource != nil {
			newChild.NewResources[resourceName] = buildChangesFromResourceState(resource)
		}
	}

	// Recursively add nested children
	for nestedChildName, nestedChildState := range childState.ChildBlueprints {
		if nestedChildState != nil {
			nestedChild, err := buildNewChildFromStateWithDepth(nestedChildState, depth+1)
			if err != nil {
				return NewBlueprintDefinition{}, err
			}
			newChild.NewChildren[nestedChildName] = nestedChild
		}
	}

	// Add exports
	for exportName, exportState := range childState.Exports {
		if exportState != nil {
			newChild.NewExports[exportName] = buildFieldChangeFromExportState(exportState)
		}
	}

	return newChild, nil
}

// buildFieldChangeFromExportState creates a FieldChange from an export state.
func buildFieldChangeFromExportState(export *state.ExportState) provider.FieldChange {
	return provider.FieldChange{
		FieldPath: export.Field,
		NewValue:  export.Value,
	}
}

// collectLinksToRemove collects link identifiers that should be removed in the reverse changeset.
// These are links that were created in the original changeset.
func collectLinksToRemove(original *BlueprintChanges) []string {
	result := []string{}

	// Links created in resource changes should be removed
	for _, changes := range original.ResourceChanges {
		for linkTarget := range changes.NewOutboundLinks {
			result = append(result, linkTarget)
		}
	}

	// Links created in new resources should be removed
	for _, changes := range original.NewResources {
		for linkTarget := range changes.NewOutboundLinks {
			result = append(result, linkTarget)
		}
	}

	return result
}

func errMaxReverseDepthExceeded(depth int) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMaxReverseDepthExceeded,
		Err: fmt.Errorf(
			"max reverse changeset depth exceeded at depth %d, "+
				"only %d levels of nested child blueprints are supported",
			depth,
			MaxReverseChangesetDepth,
		),
	}
}
