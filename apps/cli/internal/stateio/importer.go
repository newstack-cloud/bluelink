package stateio

import (
	"context"
	"encoding/json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// StateImporter defines the interface for importing instance state data.
// This abstraction allows different storage backends to implement
// their own import logic with optimal performance characteristics.
type StateImporter interface {
	// ImportInstances imports a slice of instance states.
	// Implementations should handle batch operations efficiently where possible.
	ImportInstances(ctx context.Context, instances []state.InstanceState) error
}

// ContainerStateImporter implements StateImporter using a state.Container.
// This works with any backend that implements the state.Container interface.
// It uses the container's SaveBatch method for efficient bulk operations.
type ContainerStateImporter struct {
	container state.Container
}

// NewContainerStateImporter creates a new importer that uses the given container.
func NewContainerStateImporter(container state.Container) *ContainerStateImporter {
	return &ContainerStateImporter{container: container}
}

// ImportInstances saves instances using the container's batch save operation.
// This is efficient for both memfile and postgres backends.
func (i *ContainerStateImporter) ImportInstances(ctx context.Context, instances []state.InstanceState) error {
	return i.container.Instances().SaveBatch(ctx, instances)
}

// ParseInstancesJSON parses a JSON array of instances from raw bytes.
func ParseInstancesJSON(data []byte) ([]state.InstanceState, error) {
	var instances []state.InstanceState
	if err := json.Unmarshal(data, &instances); err != nil {
		return nil, &ImportError{
			Code:    ErrCodeInvalidJSON,
			Message: "failed to parse instances JSON",
			Err:     err,
		}
	}
	return instances, nil
}

// ImportInstancesResult contains the result of an instances import.
type ImportInstancesResult struct {
	InstancesCount int
}

// ExecuteInstancesImport performs the instances import using the provided importer.
func ExecuteInstancesImport(
	ctx context.Context,
	importer StateImporter,
	data []byte,
) (*ImportInstancesResult, error) {
	instances, err := ParseInstancesJSON(data)
	if err != nil {
		return nil, err
	}

	if err := importer.ImportInstances(ctx, instances); err != nil {
		return nil, err
	}

	return &ImportInstancesResult{
		InstancesCount: len(instances),
	}, nil
}
