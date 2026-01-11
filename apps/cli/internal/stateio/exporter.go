package stateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// StateExporter defines the interface for exporting instance state data.
// This abstraction allows different storage backends to implement
// their own export logic with optimal performance characteristics.
type StateExporter interface {
	// ExportInstances exports instances by their IDs or names.
	// If instanceFilters is empty, exports all instances.
	// Returns the full instance state including child blueprints.
	ExportInstances(ctx context.Context, instanceFilters []string) ([]state.InstanceState, error)
}

// ContainerStateExporter implements StateExporter using a state.Container.
// This works with any backend that implements the state.Container interface.
type ContainerStateExporter struct {
	container state.Container
}

// NewContainerStateExporter creates a new exporter that uses the given container.
func NewContainerStateExporter(container state.Container) *ContainerStateExporter {
	return &ContainerStateExporter{container: container}
}

// ExportInstances retrieves instances using the container's batch get operation.
// If instanceFilters is empty, all instances are exported.
func (e *ContainerStateExporter) ExportInstances(
	ctx context.Context,
	instanceFilters []string,
) ([]state.InstanceState, error) {
	if len(instanceFilters) == 0 {
		return e.exportAllInstances(ctx)
	}

	instances, err := e.container.Instances().GetBatch(ctx, instanceFilters)
	if err != nil {
		return nil, createInstanceNotFoundError(err)
	}
	return instances, nil
}

func createInstanceNotFoundError(err error) *ExportError {
	var notFoundErr *state.InstancesNotFoundError
	if errors.As(err, &notFoundErr) {
		return &ExportError{
			Code:    ErrCodeInstanceNotFound,
			Message: formatMissingInstancesMessage(notFoundErr.MissingIDsOrNames),
			Err:     err,
		}
	}

	return &ExportError{
		Code:    ErrCodeInstanceNotFound,
		Message: "one or more instances not found",
		Err:     err,
	}
}

func formatMissingInstancesMessage(missing []string) string {
	if len(missing) == 1 {
		return fmt.Sprintf("instance %q not found", missing[0])
	}

	quoted := make([]string, len(missing))
	for i, m := range missing {
		quoted[i] = fmt.Sprintf("%q", m)
	}
	return fmt.Sprintf("instances not found: %s", strings.Join(quoted, ", "))
}

func (e *ContainerStateExporter) exportAllInstances(ctx context.Context) ([]state.InstanceState, error) {
	result, err := e.container.Instances().List(ctx, state.ListInstancesParams{Limit: 0})
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to list instances",
			Err:     err,
		}
	}

	if len(result.Instances) == 0 {
		return []state.InstanceState{}, nil
	}

	ids := make([]string, len(result.Instances))
	for i, summary := range result.Instances {
		ids[i] = summary.InstanceID
	}

	instances, err := e.container.Instances().GetBatch(ctx, ids)
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to retrieve instances",
			Err:     err,
		}
	}

	return instances, nil
}

// SerializeInstancesJSON serializes instances to a JSON byte array.
func SerializeInstancesJSON(instances []state.InstanceState) ([]byte, error) {
	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to serialize instances to JSON",
			Err:     err,
		}
	}
	return data, nil
}

// ExportInstancesResult contains the result of an instances export.
type ExportInstancesResult struct {
	InstancesCount int
	Data           []byte
}

// ExecuteInstancesExport performs the instances export using the provided exporter.
func ExecuteInstancesExport(
	ctx context.Context,
	exporter StateExporter,
	instanceFilters []string,
) (*ExportInstancesResult, error) {
	instances, err := exporter.ExportInstances(ctx, instanceFilters)
	if err != nil {
		return nil, err
	}

	data, err := SerializeInstancesJSON(instances)
	if err != nil {
		return nil, err
	}

	return &ExportInstancesResult{
		InstancesCount: len(instances),
		Data:           data,
	}, nil
}
