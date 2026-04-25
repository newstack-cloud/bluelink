package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// MetadataContainer implements state.MetadataContainer against a shared
// statestore.State and Persister. Moved here from memfile during phase 3
// of the statestore migration — logic is backend-agnostic and is consumed
// by every backend unchanged.
type MetadataContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

// NewMetadataContainer constructs a MetadataContainer bound to the given
// state and persister. logger may be nil — the container never logs on
// a hot path; debug messages only fire in Save / Remove.
func NewMetadataContainer(st *State, persister *Persister, logger core.Logger) *MetadataContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &MetadataContainer{state: st, persister: persister, logger: logger}
}

func (c *MetadataContainer) Get(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}
	// Metadata values are mapping nodes of variable depth — a deep copy can
	// be expensive and in-place mutation is not an expected usage pattern,
	// so we return the live reference rather than copying.
	return inst.Metadata, nil
}

func (c *MetadataContainer) Save(
	ctx context.Context,
	instanceID string,
	metadata map[string]*core.MappingNode,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(instanceID)
	}
	inst.Metadata = metadata
	c.logger.Debug(
		"persisting metadata update for blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *MetadataContainer) Remove(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}
	metadata := inst.Metadata
	inst.Metadata = nil
	c.logger.Debug(
		"persisting removal of metadata for blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)
	return metadata, c.persister.UpdateInstance(ctx, inst)
}
