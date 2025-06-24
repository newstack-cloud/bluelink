package manage

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

// Changesets is an interface that represents a service that manages
// state for change sets for blueprints produced as a part of the change
// staging process.
type Changesets interface {
	// Get a change set for the given ID.
	Get(ctx context.Context, id string) (*Changeset, error)

	// Save a new change set for a change staging request.
	Save(
		ctx context.Context,
		changeset *Changeset,
	) error

	// Cleanup removes all change sets that are older
	// than the given threshold date.
	Cleanup(ctx context.Context, thresholdDate time.Time) error
}

// Changeset represents the state of a change set for a blueprint
// that is produced as a part of the change staging process.
type Changeset struct {
	// The ID for a change set.
	ID string `json:"id"`
	// The ID of the instance that is being used for the change
	// staging process.
	// This will only be present if the change set is for updating
	// an existing blueprint instance deployment.
	InstanceID string `json:"instanceId,omitempty"`
	// Determines if the change set is for destroying the blueprint
	// instance.
	Destroy bool `json:"destroy"`
	// The status of the change staging process.
	Status ChangesetStatus `json:"status"`
	// The location of the blueprint that is being used for the change
	// staging process.
	BlueprintLocation string `json:"blueprintLocation"`
	// The changes that are produced by the change staging process.
	Changes *changes.BlueprintChanges `json:"changes,omitempty"`
	// The unix timestamp in seconds when the change set was created.
	Created int64 `json:"created"`
}

////////////////////////////////////////////////////////////////////////////////////
// Helper method that implements the `manage.Entity` interface
// used to get common members of multiple entity types.
////////////////////////////////////////////////////////////////////////////////////

func (c *Changeset) GetID() string {
	return c.ID
}

func (c *Changeset) GetCreated() int64 {
	return c.Created
}

// ChangesetStatus represents the status of a change set.
type ChangesetStatus string

const (
	// ChangesetStatusStarting indicates that the change staging process is starting.
	ChangesetStatusStarting ChangesetStatus = "STARTING"
	// ChangesetStatusStagingChanges indicates that the change staging process is running.
	ChangesetStatusStagingChanges ChangesetStatus = "STAGING_CHANGES"
	// ChangesetStatusChangesStaged indicates that the change staging process
	// has completed successfully.
	ChangesetStatusChangesStaged ChangesetStatus = "CHANGES_STAGED"
	// ChangesetStatusFailed indicates that the change staging process
	// has failed.
	ChangesetStatusFailed ChangesetStatus = "FAILED"
)
