package statestore

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourcesContainer implements state.ResourcesContainer against a shared
// statestore.State and Persister. Instance mutations flow through
// Persister.UpdateInstanceEntities when they add or remove a resource, and
// Persister.UpdateInstance otherwise. Drift mutations flow through
// Persister.CreateResourceDrift / UpdateResourceDrift / RemoveResourceDrift.
type ResourcesContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewResourcesContainer(st *State, persister *Persister, logger core.Logger) *ResourcesContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ResourcesContainer{state: st, persister: persister, logger: logger}
}

func (c *ResourcesContainer) Get(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	r, ok, err := c.state.LookupResource(ctx, resourceID)
	if err != nil {
		return state.ResourceState{}, err
	}
	if !ok {
		return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
	}
	// Lookup* methods return shared pointers, so the copy must happen under
	// the state read lock to avoid racing mutators that update entities
	// in place while holding the write lock.
	c.state.RLock()
	defer c.state.RUnlock()
	return copyResource(r), nil
}

func (c *ResourcesContainer) GetByName(
	ctx context.Context,
	instanceID string,
	resourceName string,
) (state.ResourceState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.ResourceState{}, err
	}
	if ok {
		c.state.RLock()
		defer c.state.RUnlock()
		if resourceID, ok := inst.ResourceIDs[resourceName]; ok {
			if resource := c.state.resources[resourceID]; resource != nil {
				return copyResource(resource), nil
			}
		}
	}
	itemID := idutils.ReourceInBlueprintID(instanceID, resourceName)
	return state.ResourceState{}, state.ResourceNotFoundError(itemID)
}

func (c *ResourcesContainer) Save(
	ctx context.Context,
	resourceState state.ResourceState,
) error {
	return retryOnInstanceChange(func() error {
		return c.save(ctx, resourceState)
	})
}

func (c *ResourcesContainer) save(
	ctx context.Context,
	resourceState state.ResourceState,
) error {
	if err := c.state.preloadInstances(ctx, resourceState.InstanceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[resourceState.InstanceID]
	if !ok {
		return state.InstanceNotFoundError(resourceState.InstanceID)
	}
	if inst.ResourceIDs == nil {
		inst.ResourceIDs = map[string]string{}
	}
	if inst.Resources == nil {
		inst.Resources = map[string]*state.ResourceState{}
	}

	added := InstanceEntityIDs{}
	if _, exists := inst.Resources[resourceState.ResourceID]; !exists {
		added.Resources = []string{resourceState.ResourceID}
	}

	inst.ResourceIDs[resourceState.Name] = resourceState.ResourceID
	inst.Resources[resourceState.ResourceID] = &resourceState
	c.state.resources[resourceState.ResourceID] = &resourceState

	c.logger.Debug(
		"persisting updated or newly created resource",
		core.StringLogField("resourceId", resourceState.ResourceID),
		core.StringLogField("resourceName", resourceState.Name),
		core.StringLogField("instanceId", resourceState.InstanceID),
	)
	return c.persister.UpdateInstanceEntities(ctx, inst, added, InstanceEntityIDs{})
}

func (c *ResourcesContainer) UpdateStatus(
	ctx context.Context,
	resourceID string,
	statusInfo state.ResourceStatusInfo,
) error {
	return retryOnInstanceChange(func() error {
		return c.updateStatus(ctx, resourceID, statusInfo)
	})
}

func (c *ResourcesContainer) updateStatus(
	ctx context.Context,
	resourceID string,
	statusInfo state.ResourceStatusInfo,
) error {
	if err := c.state.preloadResource(ctx, resourceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[resourceID]
	if !ok {
		return state.ResourceNotFoundError(resourceID)
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return errMalformedState(instanceNotFoundForResourceMessage(resource.InstanceID, resourceID))
	}

	resource.Status = statusInfo.Status
	resource.PreciseStatus = statusInfo.PreciseStatus
	resource.FailureReasons = statusInfo.FailureReasons
	if statusInfo.LastDeployAttemptTimestamp != nil {
		resource.LastDeployAttemptTimestamp = *statusInfo.LastDeployAttemptTimestamp
	}
	if statusInfo.LastDeployedTimestamp != nil {
		resource.LastDeployedTimestamp = *statusInfo.LastDeployedTimestamp
	}
	if statusInfo.LastStatusUpdateTimestamp != nil {
		resource.LastStatusUpdateTimestamp = *statusInfo.LastStatusUpdateTimestamp
	}
	if statusInfo.Durations != nil {
		resource.Durations = statusInfo.Durations
	}

	c.logger.Debug(
		"persisting updated resource status",
		core.StringLogField("resourceId", resourceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ResourcesContainer) Remove(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	return retryOnInstanceChangeWithResult(func() (state.ResourceState, error) {
		return c.remove(ctx, resourceID)
	})
}

func (c *ResourcesContainer) remove(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	if err := c.state.preloadResource(ctx, resourceID); err != nil {
		return state.ResourceState{}, err
	}

	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[resourceID]
	if !ok {
		return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return state.ResourceState{}, errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, resourceID),
		)
	}
	// The name→ID index must be pruned alongside the resource entry, because
	// a dangling ResourceIDs entry makes the removed resource visible to
	// every consumer of instance state (lookups by name, list-by-prefix).
	delete(inst.ResourceIDs, resource.Name)
	delete(inst.Resources, resourceID)
	delete(c.state.resources, resourceID)

	c.logger.Debug(
		"persisting removal of resource",
		core.StringLogField("resourceId", resourceID),
	)
	removed := InstanceEntityIDs{Resources: []string{resourceID}}
	return *resource, c.persister.UpdateInstanceEntities(
		ctx, inst, InstanceEntityIDs{}, removed,
	)
}

func (c *ResourcesContainer) SaveContributionFailure(
	ctx context.Context,
	resourceID string,
	failure state.ResourceLinkContributionFailure,
) error {
	return c.updateContributionFailures(
		ctx,
		resourceID,
		func(failures []state.ResourceLinkContributionFailure) []state.ResourceLinkContributionFailure {
			return state.UpsertContributionFailure(failures, failure)
		},
	)
}

func (c *ResourcesContainer) RemoveContributionFailure(
	ctx context.Context,
	resourceID string,
	layerDepth int,
) error {
	return c.updateContributionFailures(
		ctx,
		resourceID,
		func(failures []state.ResourceLinkContributionFailure) []state.ResourceLinkContributionFailure {
			return state.RemoveContributionFailureForLayer(failures, layerDepth)
		},
	)
}

// The failures are held on the resource rather than in a store of their own, so both
// operations are the same instance flush and differ only in what they do to the list.
func (c *ResourcesContainer) updateContributionFailures(
	ctx context.Context,
	resourceID string,
	update func([]state.ResourceLinkContributionFailure) []state.ResourceLinkContributionFailure,
) error {
	return retryOnInstanceChange(func() error {
		return c.applyContributionFailuresUpdate(ctx, resourceID, update)
	})
}

func (c *ResourcesContainer) applyContributionFailuresUpdate(
	ctx context.Context,
	resourceID string,
	update func([]state.ResourceLinkContributionFailure) []state.ResourceLinkContributionFailure,
) error {
	if err := c.state.preloadResource(ctx, resourceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[resourceID]
	if !ok {
		return state.ResourceNotFoundError(resourceID)
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, resourceID),
		)
	}

	resource.LinkContributionFailures = update(resource.LinkContributionFailures)

	c.logger.Debug(
		"persisting resource changes for latest link contribution failures",
		core.StringLogField("resourceId", resourceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ResourcesContainer) GetDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	if _, ok, err := c.state.LookupResource(ctx, resourceID); err != nil {
		return state.ResourceDriftState{}, err
	} else if !ok {
		return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
	}
	drift, ok, err := c.state.LookupResourceDrift(ctx, resourceID)
	if err != nil {
		return state.ResourceDriftState{}, err
	}
	if !ok {
		// Empty drift state is valid for a resource that hasn't drifted.
		return state.ResourceDriftState{}, nil
	}
	c.state.RLock()
	defer c.state.RUnlock()
	return copyResourceDrift(drift), nil
}

func (c *ResourcesContainer) SaveDrift(
	ctx context.Context,
	driftState state.ResourceDriftState,
) error {
	return retryOnInstanceChange(func() error {
		return c.saveDrift(ctx, driftState)
	})
}

func (c *ResourcesContainer) saveDrift(
	ctx context.Context,
	driftState state.ResourceDriftState,
) error {
	if err := c.state.preloadResourceWithDrift(ctx, driftState.ResourceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[driftState.ResourceID]
	if !ok {
		return state.ResourceNotFoundError(driftState.ResourceID)
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, driftState.ResourceID),
		)
	}

	resource.Drifted = true
	resource.LastDriftDetectedTimestamp = driftState.Timestamp

	_, alreadyExists := c.state.resourceDrift[driftState.ResourceID]
	c.state.resourceDrift[driftState.ResourceID] = &driftState

	c.logger.Debug(
		"persisting updated or newly created resource drift entry",
		core.StringLogField("resourceId", driftState.ResourceID),
	)
	if err := c.persistResourceDrift(ctx, &driftState, alreadyExists); err != nil {
		return err
	}

	c.logger.Debug(
		"persisting resource changes for latest drift state",
		core.StringLogField("resourceId", driftState.ResourceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ResourcesContainer) persistResourceDrift(
	ctx context.Context,
	driftState *state.ResourceDriftState,
	alreadyExists bool,
) error {
	if alreadyExists {
		return c.persister.UpdateResourceDrift(ctx, driftState)
	}
	return c.persister.CreateResourceDrift(ctx, driftState)
}

func (c *ResourcesContainer) RemoveDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	return retryOnInstanceChangeWithResult(func() (state.ResourceDriftState, error) {
		return c.removeDrift(ctx, resourceID)
	})
}

// The drift entry is deleted before the instance is rewritten, so an attempt that is retried
// after its instance write was rejected finds no drift. It still has to clear the resource's
// drifted flag, which only the instance write does.
func (c *ResourcesContainer) removeDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	if err := c.state.preloadResourceWithDrift(ctx, resourceID); err != nil {
		return state.ResourceDriftState{}, err
	}

	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[resourceID]
	if !ok {
		return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
	}
	driftState, hasDrift := c.state.resourceDrift[resourceID]
	if !hasDrift && !resource.Drifted {
		return state.ResourceDriftState{}, nil
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return state.ResourceDriftState{}, errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, resourceID),
		)
	}

	resource.Drifted = false
	resource.LastDriftDetectedTimestamp = nil
	delete(c.state.resourceDrift, resourceID)

	if hasDrift {
		c.logger.Debug(
			"persisting removal of resource drift entry",
			core.StringLogField("resourceId", resourceID),
		)
		if err := c.persister.RemoveResourceDrift(ctx, driftState); err != nil {
			return state.ResourceDriftState{}, err
		}
	}

	c.logger.Debug(
		"persisting resource changes for removal of drift state",
		core.StringLogField("resourceId", resourceID),
	)
	if err := c.persister.UpdateInstance(ctx, inst); err != nil {
		return state.ResourceDriftState{}, err
	}
	if !hasDrift {
		return state.ResourceDriftState{}, nil
	}

	return *driftState, nil
}

func instanceNotFoundForResourceMessage(instanceID, resourceID string) string {
	return fmt.Sprintf("instance %s not found for resource %s", instanceID, resourceID)
}
