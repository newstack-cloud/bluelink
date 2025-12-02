package testutils

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/engine"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

type testDeployEngine struct {
	validationEvents []*types.BlueprintValidationEvent
}

func NewTestDeployEngine(stubValidationEvents []*types.BlueprintValidationEvent) engine.DeployEngine {
	return &testDeployEngine{
		validationEvents: stubValidationEvents,
	}
}

func (d *testDeployEngine) CreateBlueprintValidation(
	ctx context.Context,
	payload *types.CreateBlueprintValidationPayload,
	query *types.CreateBlueprintValidationQuery,
) (*manage.BlueprintValidation, error) {
	return &manage.BlueprintValidation{
		ID:                "test-validation-id",
		Status:            manage.BlueprintValidationStatusStarting,
		BlueprintLocation: payload.BlueprintFile,
		Created:           time.Now().Unix(),
	}, nil
}

func (d *testDeployEngine) GetBlueprintValidation(
	ctx context.Context,
	validationID string,
) (*manage.BlueprintValidation, error) {
	return &manage.BlueprintValidation{
		ID:      "test-validation-id",
		Status:  manage.BlueprintValidationStatusValidated,
		Created: time.Now().Unix(),
	}, nil
}

func (d *testDeployEngine) StreamBlueprintValidationEvents(
	ctx context.Context,
	validationID string,
	streamTo chan<- types.BlueprintValidationEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, validationEvent := range d.validationEvents {
			streamTo <- *validationEvent
		}
	}()

	return nil
}

func (d *testDeployEngine) CleanupBlueprintValidations(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CleanupChangesets(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CleanupEvents(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*manage.Changeset, error) {
	return nil, nil
}

func (d *testDeployEngine) GetChangeset(
	ctx context.Context,
	changesetID string,
) (*manage.Changeset, error) {
	return nil, nil
}

func (d *testDeployEngine) StreamChangeStagingEvents(
	ctx context.Context,
	changesetID string,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) error {
	return nil
}

func (d *testDeployEngine) CreateBlueprintInstance(
	ctx context.Context,
	payload *types.BlueprintInstancePayload,
) (*state.InstanceState, error) {
	return nil, nil
}

func (d *testDeployEngine) UpdateBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.BlueprintInstancePayload,
) (*state.InstanceState, error) {
	return nil, nil
}

func (d *testDeployEngine) GetBlueprintInstance(
	ctx context.Context,
	instanceID string,
) (*state.InstanceState, error) {
	return nil, nil
}

func (d *testDeployEngine) GetBlueprintInstanceExports(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	return nil, nil
}

func (d *testDeployEngine) DestroyBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.DestroyBlueprintInstancePayload,
) (*state.InstanceState, error) {
	return nil, nil
}

func (d *testDeployEngine) StreamBlueprintInstanceEvents(
	ctx context.Context,
	instanceID string,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) error {
	return nil
}
