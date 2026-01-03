package changes

import (
	"maps"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// SkippedRollbackItem represents a resource or link that was skipped during
// rollback because it was not in a safe state to roll back.
type SkippedRollbackItem struct {
	// Name is the resource or link name.
	Name string
	// Type indicates whether this is a "resource" or "link".
	Type string
	// ChildPath is the path to the child blueprint containing this item,
	// empty string for root-level items.
	ChildPath string
	// Status is the current status that prevented rollback.
	Status string
	// Reason explains why the item was skipped.
	Reason string
}

// RollbackFilterResult contains the filtered changeset and information about
// any items that were skipped because they were not in a safe state to rollback.
type RollbackFilterResult struct {
	// FilteredChanges contains only the changes that are safe to rollback.
	FilteredChanges *BlueprintChanges
	// SkippedItems lists resources and links that were not in a safe state
	// to rollback and were therefore excluded from the changeset.
	SkippedItems []SkippedRollbackItem
	// HasSkippedItems is true if any items were skipped.
	HasSkippedItems bool
}

// FilterReverseChangesetByCurrentState filters a reverse changeset to only include
// resources and links that are in a safe state to rollback based on the current
// instance state.
//
// Resources and links are considered safe to rollback when they have completed
// their operation successfully (Created, Updated, Destroyed, or ConfigComplete states).
// Items in failed, in-progress, or interrupted states are skipped because attempting
// to rollback an incomplete operation could lead to unpredictable behavior.
//
// The function returns both the filtered changeset and a list of skipped items
// so that the rollback operation can report partial rollback status to the user.
func FilterReverseChangesetByCurrentState(
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
) *RollbackFilterResult {
	if reverseChanges == nil || currentState == nil {
		return &RollbackFilterResult{
			FilteredChanges: reverseChanges,
			SkippedItems:    nil,
			HasSkippedItems: false,
		}
	}

	return filterChangesWithDepth(reverseChanges, currentState, "", 0)
}

func filterChangesWithDepth(
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	childPath string,
	depth int,
) *RollbackFilterResult {
	if depth >= MaxReverseChangesetDepth {
		return &RollbackFilterResult{
			FilteredChanges: reverseChanges,
			SkippedItems:    nil,
			HasSkippedItems: false,
		}
	}

	result := &RollbackFilterResult{
		FilteredChanges: newEmptyFilteredChangeset(),
		SkippedItems:    []SkippedRollbackItem{},
		HasSkippedItems: false,
	}

	filterResourceChanges(result, reverseChanges, currentState, childPath)
	filterNewResources(result, reverseChanges, currentState, childPath)
	filterRemovedResources(result, reverseChanges, currentState, childPath)
	filterRemovedLinks(result, reverseChanges, currentState, childPath)
	filterChildChanges(result, reverseChanges, currentState, childPath, depth)

	// Copy over fields that don't need filtering
	copyUnfilteredFields(result.FilteredChanges, reverseChanges)

	return result
}

func newEmptyFilteredChangeset() *BlueprintChanges {
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
		RemovedExports:   []string{},
	}
}

// filterResourceChanges filters ResourceChanges (reversed update changes).
// Only include resources that were successfully updated (Updated or UpdateConfigComplete).
func filterResourceChanges(
	result *RollbackFilterResult,
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	childPath string,
) {
	for resourceName, changes := range reverseChanges.ResourceChanges {
		resourceState := findResourceStateByName(currentState, resourceName)
		if resourceState == nil {
			result.FilteredChanges.ResourceChanges[resourceName] = changes
			continue
		}

		if isResourceUpdateSafeToRollback(resourceState) {
			result.FilteredChanges.ResourceChanges[resourceName] = changes
			continue
		}

		result.SkippedItems = append(result.SkippedItems, SkippedRollbackItem{
			Name:      resourceName,
			Type:      "resource",
			ChildPath: childPath,
			Status:    resourceState.Status.String(),
			Reason:    "resource update was not completed successfully",
		})
		result.HasSkippedItems = true
	}
}

// filterNewResources filters NewResources (reversed removed resources, i.e., recreations).
// Only include resources that were successfully destroyed.
func filterNewResources(
	result *RollbackFilterResult,
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	childPath string,
) {
	for resourceName, changes := range reverseChanges.NewResources {
		resourceState := findResourceStateByName(currentState, resourceName)

		// If resource exists in current state, it wasn't fully destroyed
		if resourceState != nil && resourceState.Status != core.ResourceStatusDestroyed {
			result.SkippedItems = append(result.SkippedItems, SkippedRollbackItem{
				Name:      resourceName,
				Type:      "resource",
				ChildPath: childPath,
				Status:    resourceState.Status.String(),
				Reason:    "resource destruction was not completed successfully",
			})
			result.HasSkippedItems = true
			continue
		}

		result.FilteredChanges.NewResources[resourceName] = changes
	}
}

// filterRemovedResources filters RemovedResources (reversed new resources, i.e., deletions).
// Only include resources that were successfully created.
func filterRemovedResources(
	result *RollbackFilterResult,
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	childPath string,
) {
	for _, resourceName := range reverseChanges.RemovedResources {
		resourceState := findResourceStateByName(currentState, resourceName)
		if resourceState == nil {
			// Resource doesn't exist, skip silently (already removed or never created)
			continue
		}

		if isResourceCreateSafeToRollback(resourceState) {
			result.FilteredChanges.RemovedResources = append(
				result.FilteredChanges.RemovedResources,
				resourceName,
			)
			continue
		}

		result.SkippedItems = append(result.SkippedItems, SkippedRollbackItem{
			Name:      resourceName,
			Type:      "resource",
			ChildPath: childPath,
			Status:    resourceState.Status.String(),
			Reason:    "resource creation was not completed successfully",
		})
		result.HasSkippedItems = true
	}
}

// filterRemovedLinks filters RemovedLinks (reversed new links, i.e., deletions).
// Only include links that were successfully created.
func filterRemovedLinks(
	result *RollbackFilterResult,
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	childPath string,
) {
	for _, linkName := range reverseChanges.RemovedLinks {
		linkState := findLinkStateByName(currentState, linkName)
		if linkState == nil {
			// Link doesn't exist, skip silently (already removed or never created)
			continue
		}

		if core.LinkStatusIsSafeToRollback(linkState.Status) {
			result.FilteredChanges.RemovedLinks = append(
				result.FilteredChanges.RemovedLinks,
				linkName,
			)
			continue
		}

		result.SkippedItems = append(result.SkippedItems, SkippedRollbackItem{
			Name:      linkName,
			Type:      "link",
			ChildPath: childPath,
			Status:    linkState.Status.String(),
			Reason:    "link creation was not completed successfully",
		})
		result.HasSkippedItems = true
	}
}

// filterChildChanges recursively filters child blueprint changes.
func filterChildChanges(
	result *RollbackFilterResult,
	reverseChanges *BlueprintChanges,
	currentState *state.InstanceState,
	parentChildPath string,
	depth int,
) {
	for childName, childChanges := range reverseChanges.ChildChanges {
		childState := currentState.ChildBlueprints[childName]
		if childState == nil {
			result.FilteredChanges.ChildChanges[childName] = childChanges
			continue
		}

		childPath := buildChildPath(parentChildPath, childName)
		childResult := filterChangesWithDepth(&childChanges, childState, childPath, depth+1)

		if childResult.FilteredChanges != nil {
			result.FilteredChanges.ChildChanges[childName] = *childResult.FilteredChanges
		}

		result.SkippedItems = append(result.SkippedItems, childResult.SkippedItems...)
		if childResult.HasSkippedItems {
			result.HasSkippedItems = true
		}
	}

	// Copy NewChildren and RemovedChildren as-is
	// (child blueprint lifecycle is handled at the instance level)
	maps.Copy(result.FilteredChanges.NewChildren, reverseChanges.NewChildren)
	result.FilteredChanges.RemovedChildren = append(
		result.FilteredChanges.RemovedChildren,
		reverseChanges.RemovedChildren...,
	)
}

func copyUnfilteredFields(filtered, original *BlueprintChanges) {
	// Copy exports (these don't have individual status tracking)
	maps.Copy(filtered.NewExports, original.NewExports)
	maps.Copy(filtered.ExportChanges, original.ExportChanges)
	filtered.RemovedExports = append(filtered.RemovedExports, original.RemovedExports...)
	filtered.UnchangedExports = original.UnchangedExports

	filtered.MetadataChanges = original.MetadataChanges
	filtered.ResolveOnDeploy = original.ResolveOnDeploy
}

func findResourceStateByName(instanceState *state.InstanceState, name string) *state.ResourceState {
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

func findLinkStateByName(instanceState *state.InstanceState, name string) *state.LinkState {
	return instanceState.Links[name]
}

// isResourceUpdateSafeToRollback checks if a resource that was updated is safe to rollback.
// Uses PreciseStatus for more accurate detection of config-complete states.
func isResourceUpdateSafeToRollback(resource *state.ResourceState) bool {
	if core.ResourceStatusIsSafeToRollback(resource.Status) {
		return true
	}
	return core.PreciseResourceStatusIsSafeToRollback(resource.PreciseStatus)
}

// isResourceCreateSafeToRollback checks if a newly created resource is safe to rollback (destroy).
func isResourceCreateSafeToRollback(resource *state.ResourceState) bool {
	if resource.Status == core.ResourceStatusCreated {
		return true
	}
	// Also allow config-complete resources to be destroyed
	return resource.PreciseStatus == core.PreciseResourceStatusConfigComplete ||
		resource.PreciseStatus == core.PreciseResourceStatusCreated
}

func buildChildPath(parentPath, childName string) string {
	if parentPath == "" {
		return childName
	}
	return parentPath + "." + childName
}
