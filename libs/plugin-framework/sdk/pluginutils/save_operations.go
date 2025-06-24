package pluginutils

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SaveOperation is a generic interface for save operations
// that are combined as a part of a resource update or creation
// that spans multiple provider service calls.
// For example, an AWS lambda function update will often involve
// multiple calls including updating the function configuration
// and updating the function code.
type SaveOperation[Service any] interface {
	// Name of the save operation, used for logging and error messages.
	Name() string
	// Prepare an operation, primarily used to transform resource spec
	// data into a format that can be used to make service calls
	// to the upstream provider.
	Prepare(
		saveOpCtx SaveOperationContext,
		specData *core.MappingNode,
		changes *provider.Changes,
	) (bool, SaveOperationContext, error)
	// Execute an operation, used to make service calls to the upstream provider.
	Execute(
		ctx context.Context,
		saveOpCtx SaveOperationContext,
		service Service,
	) (SaveOperationContext, error)
}

// SaveOperationContext is context collected when applying multiple save operations.
// It is mostly useful when creating a resource involves multiple actions
// where a previous action returns values that are required by subsequent actions.
type SaveOperationContext struct {
	// ProviderUpstreamID is the ID of the resource in the upstream provider.
	// For example, if the resource is an AWS resource, this will be the ARN.
	ProviderUpstreamID string
	// Data collected from previous save operations.
	// It is used to pass data to subsequent save operations.
	Data map[string]any
}

// RunSaveOperations runs a list of save operations in sequence.
// It returns true if any of the save operations resulted
// in an update or resource creation.
// It returns an error if any of the save operations fail.
// An initial context can be provided (can be empty), this will be
// passed through all the save operations where each operation will
// make a copy of the context and will return a new context with
// the updated data.
func RunSaveOperations[Service any](
	ctx context.Context,
	saveOpCtx SaveOperationContext,
	operations []SaveOperation[Service],
	input *provider.ResourceDeployInput,
	service Service,
) (bool, SaveOperationContext, error) {
	resolvedResourceSpecData := GetResolvedResourceSpecData(input.Changes)

	hasValuesToSave := false
	currentSaveOpCtx := &saveOpCtx
	for _, op := range operations {
		hasOpValuesToSave, saveOpCtx, err := op.Prepare(
			*currentSaveOpCtx,
			resolvedResourceSpecData,
			input.Changes,
		)
		if err != nil {
			return false, SaveOperationContext{}, fmt.Errorf(
				"failed to prepare %s save operation: %w",
				op.Name(),
				err,
			)
		}

		if hasOpValuesToSave {
			hasValuesToSave = true
			nextSaveOpCtx, err := op.Execute(ctx, saveOpCtx, service)
			if err != nil {
				return false, SaveOperationContext{}, fmt.Errorf(
					"failed to execute %s save operation: %w",
					op.Name(),
					err,
				)
			}
			currentSaveOpCtx = &nextSaveOpCtx
		}
	}

	return hasValuesToSave, *currentSaveOpCtx, nil
}
