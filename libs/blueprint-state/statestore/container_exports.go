package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ExportsContainer implements state.ExportsContainer against a shared
// statestore.State and Persister. Backend-agnostic; every backend consumes
// it unchanged.
type ExportsContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewExportsContainer(st *State, persister *Persister, logger core.Logger) *ExportsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ExportsContainer{state: st, persister: persister, logger: logger}
}

func (c *ExportsContainer) GetAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}
	return copyExports(inst.Exports), nil
}

func (c *ExportsContainer) Get(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.ExportState{}, err
	}

	if !ok {
		return state.ExportState{}, state.InstanceNotFoundError(instanceID)
	}

	export, ok := inst.Exports[exportName]
	if !ok {
		return state.ExportState{}, state.ExportNotFoundError(instanceID, exportName)
	}

	return *copyExport(export), nil
}

func (c *ExportsContainer) SaveAll(
	ctx context.Context,
	instanceID string,
	exports map[string]*state.ExportState,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(instanceID)
	}
	inst.Exports = exports
	c.logger.Debug(
		"persisting update to all exports for blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)

	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ExportsContainer) Save(
	ctx context.Context,
	instanceID string,
	exportName string,
	export state.ExportState,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(instanceID)
	}
	if inst.Exports == nil {
		inst.Exports = map[string]*state.ExportState{}
	}
	inst.Exports[exportName] = &export
	c.logger.Debug(
		"persisting updated export for blueprint instance",
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("exportName", exportName),
	)

	return c.persister.UpdateInstance(ctx, inst)
}

func (c *ExportsContainer) RemoveAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}
	exports := inst.Exports
	inst.Exports = nil
	c.logger.Debug(
		"persisting removal of all exports for blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)

	return exports, c.persister.UpdateInstance(ctx, inst)
}

func (c *ExportsContainer) Remove(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.ExportState{}, state.InstanceNotFoundError(instanceID)
	}
	export, ok := inst.Exports[exportName]
	if !ok {
		return state.ExportState{}, state.ExportNotFoundError(instanceID, exportName)
	}
	delete(inst.Exports, exportName)
	c.logger.Debug(
		"persisting removal of export for blueprint instance",
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("exportName", exportName),
	)

	return *export, c.persister.UpdateInstance(ctx, inst)
}

func copyExports(exports map[string]*state.ExportState) map[string]*state.ExportState {
	if exports == nil {
		return nil
	}
	out := make(map[string]*state.ExportState, len(exports))
	for name, export := range exports {
		out[name] = copyExport(export)
	}
	return out
}

func copyExport(export *state.ExportState) *state.ExportState {
	if export == nil {
		return nil
	}
	return &state.ExportState{
		Value: export.Value,
		Type:  export.Type,
		Field: export.Field,
	}
}
