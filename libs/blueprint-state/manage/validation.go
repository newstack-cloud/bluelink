package manage

import (
	"context"
	"time"
)

// Validation is an interface that represents a service that manages
// state for blueprint validation.
type Validation interface {
	// Get a blueprint validation for the given ID.
	Get(ctx context.Context, id string) (*BlueprintValidation, error)

	// Save a new blueprint validation for a request to validate a blueprint.
	Save(
		ctx context.Context,
		validation *BlueprintValidation,
	) error

	// Cleanup removes all blueprint validations that are older than the given threshold date.
	// Returns the number of validations deleted.
	Cleanup(ctx context.Context, thresholdDate time.Time) (int64, error)
}

// BlueprintValidation represents the state of a blueprint validation request.
type BlueprintValidation struct {
	// The ID for a blueprint validation request.
	ID string `json:"id"`
	// The status of the blueprint validation.
	Status BlueprintValidationStatus `json:"status"`
	// The location of the blueprint that is being validated.
	BlueprintLocation string `json:"blueprintLocation"`
	// The unix timestamp in seconds when the validation request was created.
	Created int64 `json:"created"`
}

////////////////////////////////////////////////////////////////////////////////////
// Helper method that implements the `manage.Entity` interface
// used to get common members of multiple entity types.
////////////////////////////////////////////////////////////////////////////////////

func (c *BlueprintValidation) GetID() string {
	return c.ID
}

func (c *BlueprintValidation) GetCreated() int64 {
	return c.Created
}

// BlueprintValidationStatus represents the status of a blueprint validation.
type BlueprintValidationStatus string

const (
	// BlueprintValidationStatusStarting indicates that the validation process is starting.
	BlueprintValidationStatusStarting BlueprintValidationStatus = "STARTING"
	// BlueprintValidationStatusValidating indicates that the validation process is running.
	BlueprintValidationStatusRunning BlueprintValidationStatus = "VALIDATING"
	// BlueprintValidationStatusValidated indicates that the validation process
	// has completed successfully.
	BlueprintValidationStatusValidated BlueprintValidationStatus = "VALIDATED"
	// BlueprintValidationStatusFailed indicates that the validation process
	// has failed.
	BlueprintValidationStatusFailed BlueprintValidationStatus = "FAILED"
)
