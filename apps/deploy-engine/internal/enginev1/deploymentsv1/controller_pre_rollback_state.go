package deploymentsv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// maxPreRollbackStateDepth limits recursion depth when building child snapshots.
// This matches the MaxBlueprintDepth used in the blueprint container.
const maxPreRollbackStateDepth = 10

// capturePreRollbackState captures the current instance state before auto-rollback begins.
// This emits a preRollbackState event containing the full state snapshot for debugging/auditing.
func (c *Controller) capturePreRollbackState(
	ctx context.Context,
	instanceID string,
	failureReasons []string,
	logger core.Logger,
) {
	instanceState, err := c.instances.Get(ctx, instanceID)
	if err != nil {
		logger.Warn(
			"failed to capture pre-rollback state: could not fetch instance state",
			core.ErrorLogField("error", err),
		)
		return
	}

	capturedAt := c.clock.Now().Unix()
	msg := buildPreRollbackStateMessage(&instanceState, failureReasons, capturedAt)

	c.saveDeploymentEvent(
		ctx,
		eventTypePreRollbackState,
		msg,
		capturedAt,
		false, // endOfStream - more events will follow for the rollback
		instanceID,
		"rollback",
		logger,
	)

	logger.Info(
		"captured pre-rollback state",
		core.IntegerLogField("resourceCount", int64(len(msg.Resources))),
		core.IntegerLogField("linkCount", int64(len(msg.Links))),
		core.IntegerLogField("childCount", int64(len(msg.Children))),
	)
}

// buildPreRollbackStateMessage creates a PreRollbackStateMessage from the instance state.
func buildPreRollbackStateMessage(
	instanceState *state.InstanceState,
	failureReasons []string,
	capturedAt int64,
) *container.PreRollbackStateMessage {
	return &container.PreRollbackStateMessage{
		InstanceID:     instanceState.InstanceID,
		InstanceName:   instanceState.InstanceName,
		Status:         instanceState.Status,
		Resources:      buildResourceSnapshots(instanceState.Resources),
		Links:          buildLinkSnapshots(instanceState.Links),
		Children:       buildChildSnapshotsWithDepth(instanceState.ChildBlueprints, 0),
		FailureReasons: failureReasons,
		CapturedAt:     capturedAt,
	}
}

// buildResourceSnapshots creates resource snapshots from the resource state map.
func buildResourceSnapshots(resources map[string]*state.ResourceState) []container.ResourceSnapshot {
	if len(resources) == 0 {
		return nil
	}

	snapshots := make([]container.ResourceSnapshot, 0, len(resources))
	for _, r := range resources {
		if r == nil {
			continue
		}
		snapshots = append(snapshots, container.ResourceSnapshot{
			ResourceID:     r.ResourceID,
			ResourceName:   r.Name,
			ResourceType:   r.Type,
			Status:         r.Status,
			PreciseStatus:  r.PreciseStatus,
			FailureReasons: r.FailureReasons,
			SpecData:       r.SpecData,
			ComputedFields: r.ComputedFields,
		})
	}
	return snapshots
}

// buildLinkSnapshots creates link snapshots from the link state map.
func buildLinkSnapshots(links map[string]*state.LinkState) []container.LinkSnapshot {
	if len(links) == 0 {
		return nil
	}

	snapshots := make([]container.LinkSnapshot, 0, len(links))
	for _, l := range links {
		if l == nil {
			continue
		}
		snapshots = append(snapshots, container.LinkSnapshot{
			LinkID:         l.LinkID,
			LinkName:       l.Name,
			Status:         l.Status,
			PreciseStatus:  l.PreciseStatus,
			FailureReasons: l.FailureReasons,
		})
	}
	return snapshots
}

// buildChildSnapshotsWithDepth recursively creates child snapshots with depth limiting.
func buildChildSnapshotsWithDepth(
	children map[string]*state.InstanceState,
	depth int,
) []container.ChildSnapshot {
	if len(children) == 0 {
		return nil
	}

	if depth >= maxPreRollbackStateDepth {
		return nil
	}

	snapshots := make([]container.ChildSnapshot, 0, len(children))
	for childName, childState := range children {
		if childState == nil {
			continue
		}
		snapshots = append(snapshots, container.ChildSnapshot{
			ChildInstanceID: childState.InstanceID,
			ChildName:       childName,
			Status:          childState.Status,
			Resources:       buildResourceSnapshots(childState.Resources),
			Links:           buildLinkSnapshots(childState.Links),
			Children:        buildChildSnapshotsWithDepth(childState.ChildBlueprints, depth+1),
			FailureReasons:  nil, // Child failure reasons are tracked in child deploy events
		})
	}
	return snapshots
}

// createRemovalChangesFromInstanceState creates removal changes from the current instance state.
// This is used for auto-rollback of new deployments where we need to destroy resources
// that were being created but the deployment failed.
// Returns the filtered changes and any items that were skipped due to unsafe state.
func (c *Controller) createRemovalChangesFromInstanceState(
	ctx context.Context,
	instanceID string,
) (*changes.BlueprintChanges, []changes.SkippedRollbackItem, error) {
	instanceState, err := c.instances.Get(ctx, instanceID)
	if err != nil {
		return nil, nil, err
	}

	result := changes.CreateRemovalChangesFromInstanceState(&instanceState)
	return result.Changes, result.SkippedItems, nil
}

// logSkippedRollbackItems logs information about items that were skipped during rollback
// because they were not in a safe state to rollback.
func (c *Controller) logSkippedRollbackItems(
	skippedItems []changes.SkippedRollbackItem,
	instanceID string,
	logger core.Logger,
) {
	for _, item := range skippedItems {
		logger.Debug(
			"skipping rollback for item not in safe state",
			core.StringLogField("instanceId", instanceID),
			core.StringLogField("itemName", item.Name),
			core.StringLogField("itemType", item.Type),
			core.StringLogField("childPath", item.ChildPath),
			core.StringLogField("status", item.Status),
			core.StringLogField("reason", item.Reason),
		)
	}
}

// convertSkippedItemsToContainerType converts changes.SkippedRollbackItem to container.SkippedRollbackItem.
func convertSkippedItemsToContainerType(items []changes.SkippedRollbackItem) []container.SkippedRollbackItem {
	result := make([]container.SkippedRollbackItem, len(items))
	for i, item := range items {
		result[i] = container.SkippedRollbackItem{
			Name:      item.Name,
			Type:      item.Type,
			ChildPath: item.ChildPath,
			Status:    item.Status,
			Reason:    item.Reason,
		}
	}
	return result
}
