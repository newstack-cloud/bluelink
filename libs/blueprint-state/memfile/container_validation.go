package memfile

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

type validationContainerImpl struct {
	validations map[string]*manage.BlueprintValidation
	fs          afero.Fs
	persister   *statePersister
	logger      core.Logger
	mu          *sync.RWMutex
}

func (c *validationContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.BlueprintValidation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	validation, ok := c.validations[id]
	if !ok {
		return nil, manage.BlueprintValidationNotFoundError(id)
	}

	validationCopy := copyBlueprintValidation(validation)

	return &validationCopy, nil
}

func (c *validationContainerImpl) Save(
	ctx context.Context,
	validation *manage.BlueprintValidation,
) error {
	c.mu.Lock()
	// Defer unlock to ensure that modifications are not made to in-memory
	// state during persistence.
	defer c.mu.Unlock()

	return c.save(validation)
}

func (c *validationContainerImpl) save(
	validation *manage.BlueprintValidation,
) error {
	changesetLogger := c.logger.WithFields(
		core.StringLogField("blueprintValidationId", validation.ID),
	)
	_, alreadyExists := c.validations[validation.ID]
	c.validations[validation.ID] = validation

	if alreadyExists {
		changesetLogger.Debug("persisting blueprint validation update")
		return c.persister.updateBlueprintValidation(validation)
	}

	changesetLogger.Debug("persisting new blueprint validation")
	return c.persister.createBlueprintValidation(validation)
}

func (c *validationContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// As the persister manages how blueprint validations are stored,
	// it is responsible for cleaning them up.
	// The in-memory state of the container implementation
	// doesn't require validations to be in any particular order
	// but the persister is expected to maintain order by timestamp
	// when saving blueprint validations to files.
	newLookup, err := c.persister.cleanupBlueprintValidations(thresholdDate)
	if err != nil {
		return err
	}
	c.validations = newLookup

	return nil
}

func copyBlueprintValidation(
	validation *manage.BlueprintValidation,
) manage.BlueprintValidation {

	return manage.BlueprintValidation{
		ID:                validation.ID,
		Status:            validation.Status,
		BlueprintLocation: validation.BlueprintLocation,
		Created:           validation.Created,
	}
}
