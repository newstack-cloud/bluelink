package memfile

import (
	"context"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type metadataContainerImpl struct {
	instances map[string]*state.InstanceState
	fs        afero.Fs
	persister *statePersister
	logger    core.Logger
	mu        *sync.RWMutex
}

func (c *metadataContainerImpl) Get(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		// Metadata values are mapping nodes which can be of variable depth
		// so a deep copy can be an expensive operation and inline manipulation
		// of metadata is not expected to be common so a reference to the metadata
		// is returned.
		return instance.Metadata, nil
	}

	return nil, state.InstanceNotFoundError(instanceID)
}

func (c *metadataContainerImpl) Save(
	ctx context.Context,
	instanceID string,
	metadata map[string]*core.MappingNode,
) error {
	metadataLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		instance.Metadata = metadata
		metadataLogger.Debug("persisting metadata update for blueprint instance")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(instanceID)
}

func (c *metadataContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	metadataLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		metadata := instance.Metadata
		instance.Metadata = nil
		metadataLogger.Debug("persisting removal of metadata for blueprint instance")
		return metadata, c.persister.updateInstance(instance)
	}

	return nil, state.InstanceNotFoundError(instanceID)
}
