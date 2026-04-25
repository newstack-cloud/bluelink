package statestore

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// ValidationsContainer implements manage.Validation against a shared
// statestore.State and Persister.
type ValidationsContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewValidationsContainer(st *State, persister *Persister, logger core.Logger) *ValidationsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ValidationsContainer{state: st, persister: persister, logger: logger}
}

func (c *ValidationsContainer) Get(
	ctx context.Context,
	id string,
) (*manage.BlueprintValidation, error) {
	v, ok, err := c.state.LookupValidation(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, manage.BlueprintValidationNotFoundError(id)
	}
	copied := copyBlueprintValidation(v)
	return &copied, nil
}

func (c *ValidationsContainer) Save(
	ctx context.Context,
	validation *manage.BlueprintValidation,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	_, alreadyExists := c.state.validations[validation.ID]
	c.state.validations[validation.ID] = validation

	logger := c.logger.WithFields(core.StringLogField("blueprintValidationId", validation.ID))
	if alreadyExists {
		logger.Debug("persisting blueprint validation update")
		return c.persister.UpdateValidation(ctx, validation)
	}
	logger.Debug("persisting new blueprint validation")
	return c.persister.CreateValidation(ctx, validation)
}

func (c *ValidationsContainer) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.state.Lock()
	defer c.state.Unlock()

	originalCount := len(c.state.validations)
	newLookup, err := c.persister.CleanupValidations(ctx, thresholdDate)
	if err != nil {
		return 0, err
	}
	c.state.validations = newLookup
	return int64(originalCount - len(newLookup)), nil
}

func copyBlueprintValidation(v *manage.BlueprintValidation) manage.BlueprintValidation {
	return manage.BlueprintValidation{
		ID:                v.ID,
		Status:            v.Status,
		BlueprintLocation: v.BlueprintLocation,
		Created:           v.Created,
	}
}
