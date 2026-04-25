package statestore

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourcesContainer implements state.ResourcesContainer against a shared
// statestore.State and Persister. Instance mutations (Save/UpdateStatus/
// Remove) flow through Persister.UpdateInstance; drift mutations flow
// through Persister.CreateResourceDrift / UpdateResourceDrift / RemoveResourceDrift.
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
		if resourceID, ok := inst.ResourceIDs[resourceName]; ok {
			c.state.RLock()
			resource := c.state.resources[resourceID]
			c.state.RUnlock()
			if resource != nil {
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
	inst.ResourceIDs[resourceState.Name] = resourceState.ResourceID
	inst.Resources[resourceState.ResourceID] = &resourceState
	c.state.resources[resourceState.ResourceID] = &resourceState

	c.logger.Debug(
		"persisting updated or newly created resource",
		core.StringLogField("resourceId", resourceState.ResourceID),
		core.StringLogField("resourceName", resourceState.Name),
		core.StringLogField("instanceId", resourceState.InstanceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ResourcesContainer) UpdateStatus(
	ctx context.Context,
	resourceID string,
	statusInfo state.ResourceStatusInfo,
) error {
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
	delete(inst.Resources, resourceID)
	delete(c.state.resources, resourceID)

	c.logger.Debug(
		"persisting removal of resource",
		core.StringLogField("resourceId", resourceID),
	)
	return *resource, c.persister.UpdateInstance(ctx, inst)
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
	return copyResourceDrift(drift), nil
}

func (c *ResourcesContainer) SaveDrift(
	ctx context.Context,
	driftState state.ResourceDriftState,
) error {
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
	c.state.Lock()
	defer c.state.Unlock()

	resource, ok := c.state.resources[resourceID]
	if !ok {
		return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
	}
	driftState, hasDrift := c.state.resourceDrift[resourceID]
	if !hasDrift {
		return state.ResourceDriftState{}, nil
	}
	inst, ok := c.state.instances[resource.InstanceID]
	if !ok {
		return state.ResourceDriftState{}, errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, driftState.ResourceID),
		)
	}

	resource.Drifted = false
	resource.LastDriftDetectedTimestamp = nil
	delete(c.state.resourceDrift, resourceID)

	c.logger.Debug(
		"persisting removal of resource drift entry",
		core.StringLogField("resourceId", resourceID),
	)
	if err := c.persister.RemoveResourceDrift(ctx, driftState); err != nil {
		return state.ResourceDriftState{}, err
	}

	c.logger.Debug(
		"persisting resource changes for removal of drift state",
		core.StringLogField("resourceId", resourceID),
	)
	if err := c.persister.UpdateInstance(ctx, inst); err != nil {
		return state.ResourceDriftState{}, err
	}
	return *driftState, nil
}

func instanceNotFoundForResourceMessage(instanceID, resourceID string) string {
	return fmt.Sprintf("instance %s not found for resource %s", instanceID, resourceID)
}
