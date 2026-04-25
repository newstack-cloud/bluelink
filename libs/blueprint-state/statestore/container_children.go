package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ChildrenContainer implements state.ChildrenContainer against a shared
// statestore.State and Persister.
type ChildrenContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewChildrenContainer(st *State, persister *Persister, logger core.Logger) *ChildrenContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ChildrenContainer{state: st, persister: persister, logger: logger}
}

func (c *ChildrenContainer) Get(
	ctx context.Context,
	instanceID string,
	childName string,
) (state.InstanceState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}
	if !ok {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}
	child, ok := inst.ChildBlueprints[childName]
	if !ok {
		return state.InstanceState{}, state.InstanceNotFoundError(
			idutils.ChildInBlueprintID(instanceID, childName),
		)
	}
	return copyInstance(child, instanceID), nil
}

func (c *ChildrenContainer) Attach(
	ctx context.Context,
	parentInstanceID string,
	childInstanceID string,
	childName string,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	parent, ok := c.state.instances[parentInstanceID]
	if !ok {
		return state.InstanceNotFoundError(parentInstanceID)
	}
	child, ok := c.state.instances[childInstanceID]
	if !ok {
		return state.InstanceNotFoundError(childInstanceID)
	}
	if parent.ChildBlueprints == nil {
		parent.ChildBlueprints = map[string]*state.InstanceState{}
	}
	parent.ChildBlueprints[childName] = child
	c.logger.Debug(
		"persisting child instance attachment to parent instance",
		core.StringLogField("parentInstanceId", parentInstanceID),
		core.StringLogField("childInstanceId", childInstanceID),
		core.StringLogField("childName", childName),
	)
	return c.persister.UpdateInstance(ctx, parent)
}

func (c *ChildrenContainer) SaveDependencies(
	ctx context.Context,
	instanceID string,
	childName string,
	dependencies *state.DependencyInfo,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(instanceID)
	}
	if inst.ChildDependencies == nil {
		inst.ChildDependencies = map[string]*state.DependencyInfo{}
	}
	inst.ChildDependencies[childName] = dependencies
	c.logger.Debug(
		"persisting child dependencies update",
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("childName", childName),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ChildrenContainer) Detach(
	ctx context.Context,
	instanceID string,
	childName string,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(
			idutils.ChildInBlueprintID(instanceID, childName),
		)
	}
	if _, ok := inst.ChildBlueprints[childName]; !ok {
		return state.InstanceNotFoundError(
			idutils.ChildInBlueprintID(instanceID, childName),
		)
	}
	delete(inst.ChildBlueprints, childName)
	c.logger.Debug(
		"persisting child instance detachment from parent instance",
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("childName", childName),
	)
	return c.persister.UpdateInstance(ctx, inst)
}
