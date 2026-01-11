package manage

import (
	"context"
	"fmt"
)

// CleanupType represents the type of cleanup operation.
type CleanupType string

const (
	CleanupTypeEvents                CleanupType = "events"
	CleanupTypeChangesets            CleanupType = "changesets"
	CleanupTypeReconciliationResults CleanupType = "reconciliation_results"
	CleanupTypeValidations           CleanupType = "validations"
)

// CleanupOperationStatus represents the status of a cleanup operation.
type CleanupOperationStatus string

const (
	CleanupOperationStatusRunning   CleanupOperationStatus = "running"
	CleanupOperationStatusCompleted CleanupOperationStatus = "completed"
	CleanupOperationStatusFailed    CleanupOperationStatus = "failed"
)

// CleanupOperations is an interface that represents a service that manages
// state for cleanup operations tracking.
type CleanupOperations interface {
	// Get retrieves a cleanup operation by its ID.
	Get(ctx context.Context, id string) (*CleanupOperation, error)

	// GetLatestByType retrieves the most recent operation for a cleanup type.
	// Returns CleanupOperationNotFoundError if no operations exist for the type.
	GetLatestByType(ctx context.Context, cleanupType CleanupType) (*CleanupOperation, error)

	// Save persists a new cleanup operation and enforces the rolling window
	// by deleting the oldest records when the count exceeds the limit per type.
	Save(ctx context.Context, operation *CleanupOperation) error

	// Update updates an existing cleanup operation (for status changes, item counts).
	Update(ctx context.Context, operation *CleanupOperation) error
}

// CleanupOperation represents a cleanup process being tracked.
type CleanupOperation struct {
	// The unique ID for the cleanup operation.
	ID string `json:"id"`
	// The type of cleanup (events, changesets, reconciliation_results, validations).
	CleanupType CleanupType `json:"cleanupType"`
	// The status of the cleanup operation.
	Status CleanupOperationStatus `json:"status"`
	// Unix timestamp when the operation started.
	StartedAt int64 `json:"startedAt"`
	// Unix timestamp when the operation ended (0 if still running).
	EndedAt int64 `json:"endedAt,omitempty"`
	// Number of items deleted by the cleanup operation.
	ItemsDeleted int64 `json:"itemsDeleted"`
	// Error message if the operation failed.
	ErrorMessage string `json:"errorMessage,omitempty"`
	// The threshold date used for determining what to clean up (unix timestamp).
	ThresholdDate int64 `json:"thresholdDate"`
}

////////////////////////////////////////////////////////////////////////////////////
// Helper methods that implement the `manage.Entity` interface
////////////////////////////////////////////////////////////////////////////////////

func (c *CleanupOperation) GetID() string {
	return c.ID
}

func (c *CleanupOperation) GetCreated() int64 {
	return c.StartedAt
}

// Duration returns the duration of the cleanup operation in seconds.
// Returns 0 if the operation is still running.
func (c *CleanupOperation) Duration() int64 {
	if c.EndedAt == 0 {
		return 0
	}
	return c.EndedAt - c.StartedAt
}

////////////////////////////////////////////////////////////////////////////////////
// Errors
////////////////////////////////////////////////////////////////////////////////////

// CleanupOperationNotFound is an error type that is returned when
// a cleanup operation is not found.
type CleanupOperationNotFound struct {
	ID          string
	CleanupType CleanupType
}

func (e *CleanupOperationNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("cleanup operation with ID %s not found", e.ID)
	}
	if e.CleanupType != "" {
		return fmt.Sprintf("cleanup operation for type %s not found", e.CleanupType)
	}
	return "cleanup operation not found"
}

// CleanupOperationNotFoundError creates a new CleanupOperationNotFound error for an ID lookup.
func CleanupOperationNotFoundError(id string) *CleanupOperationNotFound {
	return &CleanupOperationNotFound{ID: id}
}

// CleanupOperationNotFoundForTypeError creates a new CleanupOperationNotFound error
// for a cleanup type lookup.
func CleanupOperationNotFoundForTypeError(cleanupType CleanupType) *CleanupOperationNotFound {
	return &CleanupOperationNotFound{CleanupType: cleanupType}
}
