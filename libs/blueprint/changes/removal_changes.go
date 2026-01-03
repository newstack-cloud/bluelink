package changes

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// MaxRemovalChangesDepth is the maximum depth for recursive child blueprint processing.
const MaxRemovalChangesDepth = 10

// RemovalChangesResult contains the removal changes and any items that were skipped
// because they were not in a safe state to be destroyed during rollback.
type RemovalChangesResult struct {
	// Changes contains the removal changes for resources, links, children, and exports
	// that are safe to destroy.
	Changes *BlueprintChanges
	// SkippedItems lists resources and links that were not in a safe state
	// to destroy and were therefore excluded from the removal changes.
	SkippedItems []SkippedRollbackItem
	// HasSkippedItems is true if any items were skipped.
	HasSkippedItems bool
}

// CreateRemovalChangesFromInstanceState creates removal changes from the current instance state.
// This is used for auto-rollback of new deployments where we need to destroy resources
// that were being created but the deployment failed.
//
// Only resources and links in a safe state (Created, ConfigComplete) are included in the
// removal changes. Items in failed or in-progress states are skipped because they cannot
// be reliably destroyed.
//
// Returns the removal changes and any items that were skipped due to unsafe state.
func CreateRemovalChangesFromInstanceState(
	instanceState *state.InstanceState,
) *RemovalChangesResult {
	if instanceState == nil {
		return &RemovalChangesResult{
			Changes:         nil,
			SkippedItems:    nil,
			HasSkippedItems: false,
		}
	}

	result := &removalChangesBuilder{
		changes: &BlueprintChanges{
			RemovedResources: make([]string, 0, len(instanceState.Resources)),
			RemovedLinks:     make([]string, 0, len(instanceState.Links)),
			RemovedChildren:  make([]string, 0, len(instanceState.ChildBlueprints)),
			RemovedExports:   make([]string, 0, len(instanceState.Exports)),
			ChildChanges:     make(map[string]BlueprintChanges),
		},
		skippedItems: []SkippedRollbackItem{},
	}

	buildRemovalChangesFromState(result, instanceState, "", 0)

	return &RemovalChangesResult{
		Changes:         result.changes,
		SkippedItems:    result.skippedItems,
		HasSkippedItems: len(result.skippedItems) > 0,
	}
}

type removalChangesBuilder struct {
	changes      *BlueprintChanges
	skippedItems []SkippedRollbackItem
}

// buildRemovalChangesFromState creates BlueprintChanges with removal entries from instance state.
// The depth parameter limits recursion to prevent stack overflow with deeply nested blueprints.
func buildRemovalChangesFromState(
	result *removalChangesBuilder,
	instanceState *state.InstanceState,
	childPath string,
	depth int,
) {
	if depth >= MaxRemovalChangesDepth {
		return
	}

	processResourcesForRemoval(result, instanceState.Resources, childPath)
	processLinksForRemoval(result, instanceState.Links, childPath)
	processChildrenForRemoval(result, instanceState.ChildBlueprints, childPath, depth)
	processExportsForRemoval(result, instanceState.Exports)
}

func processResourcesForRemoval(
	result *removalChangesBuilder,
	resources map[string]*state.ResourceState,
	childPath string,
) {
	for _, resource := range resources {
		if resource == nil {
			continue
		}

		if IsResourceSafeToDestroy(resource) {
			result.changes.RemovedResources = append(result.changes.RemovedResources, resource.Name)
			continue
		}

		result.skippedItems = append(result.skippedItems, SkippedRollbackItem{
			Name:      resource.Name,
			Type:      "resource",
			ChildPath: childPath,
			Status:    resource.Status.String(),
			Reason:    "resource creation was not completed successfully",
		})
	}
}

func processLinksForRemoval(
	result *removalChangesBuilder,
	links map[string]*state.LinkState,
	childPath string,
) {
	for linkName, link := range links {
		if link == nil {
			continue
		}

		if core.LinkStatusIsSafeToRollback(link.Status) {
			result.changes.RemovedLinks = append(result.changes.RemovedLinks, linkName)
			continue
		}

		result.skippedItems = append(result.skippedItems, SkippedRollbackItem{
			Name:      linkName,
			Type:      "link",
			ChildPath: childPath,
			Status:    link.Status.String(),
			Reason:    "link creation was not completed successfully",
		})
	}
}

func processChildrenForRemoval(
	result *removalChangesBuilder,
	children map[string]*state.InstanceState,
	parentChildPath string,
	depth int,
) {
	for childName, childState := range children {
		result.changes.RemovedChildren = append(result.changes.RemovedChildren, childName)

		if childState != nil && depth < MaxRemovalChangesDepth-1 {
			childResult := &removalChangesBuilder{
				changes: &BlueprintChanges{
					RemovedResources: make([]string, 0, len(childState.Resources)),
					RemovedLinks:     make([]string, 0, len(childState.Links)),
					RemovedChildren:  make([]string, 0, len(childState.ChildBlueprints)),
					RemovedExports:   make([]string, 0, len(childState.Exports)),
					ChildChanges:     make(map[string]BlueprintChanges),
				},
				skippedItems: []SkippedRollbackItem{},
			}

			nestedChildPath := buildChildPathForRemoval(parentChildPath, childName)
			buildRemovalChangesFromState(childResult, childState, nestedChildPath, depth+1)

			result.changes.ChildChanges[childName] = *childResult.changes
			result.skippedItems = append(result.skippedItems, childResult.skippedItems...)
		}
	}
}

func processExportsForRemoval(
	result *removalChangesBuilder,
	exports map[string]*state.ExportState,
) {
	for exportName := range exports {
		result.changes.RemovedExports = append(result.changes.RemovedExports, exportName)
	}
}

// IsResourceSafeToDestroy checks if a resource is in a safe state to be destroyed during rollback.
// A resource is safe to destroy if it was successfully created (Created status) or at least
// reached config-complete state.
func IsResourceSafeToDestroy(resource *state.ResourceState) bool {
	if resource.Status == core.ResourceStatusCreated {
		return true
	}
	return resource.PreciseStatus == core.PreciseResourceStatusConfigComplete ||
		resource.PreciseStatus == core.PreciseResourceStatusCreated
}

func buildChildPathForRemoval(parentPath, childName string) string {
	if parentPath == "" {
		return childName
	}
	return parentPath + "." + childName
}
