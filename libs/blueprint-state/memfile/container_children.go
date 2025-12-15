package memfile

import (
	"context"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type childrenContainerImpl struct {
	instances map[string]*state.InstanceState
	fs        afero.Fs
	persister *statePersister
	logger    core.Logger
	mu        *sync.RWMutex
}

func (c *childrenContainerImpl) Get(
	ctx context.Context,
	instanceID string,
	childName string,
) (state.InstanceState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		if child, ok := instance.ChildBlueprints[childName]; ok {
			return copyInstance(child, instanceID), nil
		} else {
			itemID := idutils.ChildInBlueprintID(instanceID, childName)
			return state.InstanceState{}, state.InstanceNotFoundError(itemID)
		}
	}

	return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
}

func (c *childrenContainerImpl) Attach(
	ctx context.Context,
	parentInstanceID string,
	childInstanceID string,
	childName string,
) error {
	childLogger := c.logger.WithFields(
		core.StringLogField("parentInstanceId", parentInstanceID),
		core.StringLogField("childInstanceId", childInstanceID),
		core.StringLogField("childName", childName),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if parentInstance, ok := getInstance(c.instances, parentInstanceID); ok {
		if childInstance, ok := getInstance(c.instances, childInstanceID); ok {
			if parentInstance.ChildBlueprints == nil {
				parentInstance.ChildBlueprints = make(map[string]*state.InstanceState)
			}
			parentInstance.ChildBlueprints[childName] = childInstance
			childLogger.Debug("persisting child instance attachment to parent instance")
			return c.persister.updateInstance(parentInstance)
		}

		return state.InstanceNotFoundError(childInstanceID)
	}

	return state.InstanceNotFoundError(parentInstanceID)
}

func (c *childrenContainerImpl) SaveDependencies(
	ctx context.Context,
	instanceId string,
	childName string,
	dependencies *state.DependencyInfo,
) error {
	childLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceId),
		core.StringLogField("childName", childName),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceId); ok {
		if instance.ChildDependencies == nil {
			instance.ChildDependencies = make(map[string]*state.DependencyInfo)
		}
		instance.ChildDependencies[childName] = dependencies
		childLogger.Debug("persisting child dependencies update")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(instanceId)
}

func (c *childrenContainerImpl) Detach(
	ctx context.Context,
	instanceID string,
	childName string,
) error {
	childLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("childName", childName),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		_, ok := instance.ChildBlueprints[childName]
		if ok {
			delete(instance.ChildBlueprints, childName)
			childLogger.Debug("persisting child instance detachment from parent instance")
			return c.persister.updateInstance(instance)
		}
	}

	itemID := idutils.ChildInBlueprintID(instanceID, childName)
	return state.InstanceNotFoundError(itemID)
}
