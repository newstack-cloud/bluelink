package memfile

import (
	"context"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type exportContainerImpl struct {
	instances map[string]*state.InstanceState
	fs        afero.Fs
	persister *statePersister
	logger    core.Logger
	mu        *sync.RWMutex
}

func (c *exportContainerImpl) GetAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		return copyExports(instance.Exports), nil
	}

	return nil, state.InstanceNotFoundError(instanceID)
}

func (c *exportContainerImpl) Get(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		if export, ok := instance.Exports[exportName]; ok {
			exportCopy := copyExport(export)
			return *exportCopy, nil
		}

		return state.ExportState{}, state.ExportNotFoundError(instanceID, exportName)
	}

	return state.ExportState{}, state.InstanceNotFoundError(instanceID)
}

func (c *exportContainerImpl) SaveAll(
	ctx context.Context,
	instanceID string,
	exports map[string]*state.ExportState,
) error {
	exportsLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		instance.Exports = exports
		exportsLogger.Debug("persisting update to all exports for blueprint instance")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(instanceID)
}

func (c *exportContainerImpl) Save(
	ctx context.Context,
	instanceID string,
	exportName string,
	export state.ExportState,
) error {
	exportLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("exportName", exportName),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		instance.Exports[exportName] = &export
		exportLogger.Debug("persisting updated export for blueprint instance")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(instanceID)
}

func (c *exportContainerImpl) RemoveAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	exportsLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		exports := instance.Exports
		instance.Exports = nil
		exportsLogger.Debug("persisting removal of all exports for blueprint instance")
		return exports, c.persister.updateInstance(instance)
	}

	return nil, state.InstanceNotFoundError(instanceID)
}

func (c *exportContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	exportLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("exportName", exportName),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		export, ok := instance.Exports[exportName]
		if ok {
			delete(instance.Exports, exportName)
			exportLogger.Debug("persisting removal of export for blueprint instance")
			return *export, c.persister.updateInstance(instance)
		}

		return state.ExportState{}, state.ExportNotFoundError(instanceID, exportName)
	}

	return state.ExportState{}, state.InstanceNotFoundError(instanceID)
}

func copyExports(
	exports map[string]*state.ExportState,
) map[string]*state.ExportState {
	if exports == nil {
		return nil
	}

	exportsCopy := make(map[string]*state.ExportState)
	for exportName, export := range exports {
		exportCopy := copyExport(export)
		exportsCopy[exportName] = exportCopy
	}

	return exportsCopy
}

func copyExport(
	exportState *state.ExportState,
) *state.ExportState {
	if exportState == nil {
		return nil
	}

	return &state.ExportState{
		Value: exportState.Value,
		Type:  exportState.Type,
		Field: exportState.Field,
	}
}
