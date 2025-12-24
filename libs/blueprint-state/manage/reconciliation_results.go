package manage

import (
	"context"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
)

// ReconciliationResults is an interface that represents a service that manages
// state for reconciliation check results produced during the change staging process
// when drift or interrupted state is detected.
type ReconciliationResults interface {
	// Get retrieves a reconciliation result by its ID.
	Get(ctx context.Context, id string) (*ReconciliationResult, error)

	// GetLatestByChangesetID retrieves the most recent result for a changeset.
	// Returns ReconciliationResultNotFoundError if no results exist for the changeset.
	GetLatestByChangesetID(ctx context.Context, changesetID string) (*ReconciliationResult, error)

	// GetAllByChangesetID retrieves all results for a changeset, ordered by created desc.
	// Returns an empty slice if no results exist for the changeset.
	GetAllByChangesetID(ctx context.Context, changesetID string) ([]*ReconciliationResult, error)

	// GetLatestByInstanceID retrieves the most recent result for a blueprint instance.
	// Returns ReconciliationResultNotFoundError if no results exist for the instance.
	GetLatestByInstanceID(ctx context.Context, instanceID string) (*ReconciliationResult, error)

	// GetAllByInstanceID retrieves all results for a blueprint instance, ordered by created desc.
	// Returns an empty slice if no results exist for the instance.
	GetAllByInstanceID(ctx context.Context, instanceID string) ([]*ReconciliationResult, error)

	// Save persists a new reconciliation result.
	Save(ctx context.Context, result *ReconciliationResult) error

	// Cleanup removes all results older than the threshold date.
	Cleanup(ctx context.Context, thresholdDate time.Time) error
}

// ReconciliationResult represents a reconciliation check result that is produced
// when drift or interrupted state is detected during change staging.
type ReconciliationResult struct {
	// The ID for a reconciliation result.
	ID string `json:"id"`
	// The ID of the changeset that this result is associated with.
	ChangesetID string `json:"changesetId"`
	// The ID of the blueprint instance that was checked.
	InstanceID string `json:"instanceId"`
	// The reconciliation check result containing resources and links that need attention.
	Result *container.ReconciliationCheckResult `json:"result"`
	// The unix timestamp in seconds when the result was created.
	Created int64 `json:"created"`
}

////////////////////////////////////////////////////////////////////////////////////
// Helper method that implements the `manage.Entity` interface
// used to get common members of multiple entity types.
////////////////////////////////////////////////////////////////////////////////////

func (r *ReconciliationResult) GetID() string {
	return r.ID
}

func (r *ReconciliationResult) GetCreated() int64 {
	return r.Created
}

////////////////////////////////////////////////////////////////////////////////////
// Errors
////////////////////////////////////////////////////////////////////////////////////

// ReconciliationResultNotFound is an error type that is returned when
// a reconciliation result is not found.
type ReconciliationResultNotFound struct {
	ID          string
	ChangesetID string
	InstanceID  string
}

func (e *ReconciliationResultNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("reconciliation result with ID %s not found", e.ID)
	}
	if e.ChangesetID != "" {
		return fmt.Sprintf("reconciliation result for changeset %s not found", e.ChangesetID)
	}
	if e.InstanceID != "" {
		return fmt.Sprintf("reconciliation result for instance %s not found", e.InstanceID)
	}
	return "reconciliation result not found"
}

// ReconciliationResultNotFoundError creates a new ReconciliationResultNotFound error for an ID lookup.
func ReconciliationResultNotFoundError(id string) *ReconciliationResultNotFound {
	return &ReconciliationResultNotFound{ID: id}
}

// ReconciliationResultNotFoundForChangesetError creates a new ReconciliationResultNotFound error
// for a changeset ID lookup.
func ReconciliationResultNotFoundForChangesetError(changesetID string) *ReconciliationResultNotFound {
	return &ReconciliationResultNotFound{ChangesetID: changesetID}
}

// ReconciliationResultNotFoundForInstanceError creates a new ReconciliationResultNotFound error
// for an instance ID lookup.
func ReconciliationResultNotFoundForInstanceError(instanceID string) *ReconciliationResultNotFound {
	return &ReconciliationResultNotFound{InstanceID: instanceID}
}
