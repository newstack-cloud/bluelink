package statestore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// ChangesetsContainer implements manage.Changesets against a shared
// statestore.State and Persister.
type ChangesetsContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewChangesetsContainer(st *State, persister *Persister, logger core.Logger) *ChangesetsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &ChangesetsContainer{state: st, persister: persister, logger: logger}
}

func (c *ChangesetsContainer) Get(
	ctx context.Context,
	id string,
) (*manage.Changeset, error) {
	cs, ok, err := c.state.LookupChangeset(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, manage.ChangesetNotFoundError(id)
	}
	copied, err := copyChangeset(cs)
	if err != nil {
		return nil, err
	}
	return &copied, nil
}

func (c *ChangesetsContainer) Save(
	ctx context.Context,
	changeset *manage.Changeset,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	_, alreadyExists := c.state.changesets[changeset.ID]
	c.state.changesets[changeset.ID] = changeset

	logger := c.logger.WithFields(core.StringLogField("changesetId", changeset.ID))
	if alreadyExists {
		logger.Debug("persisting change set update")
		return c.persister.UpdateChangeset(ctx, changeset)
	}
	logger.Debug("persisting new change set")
	return c.persister.CreateChangeset(ctx, changeset)
}

// Cleanup removes changesets older than thresholdDate and returns the count
// deleted. The persister reconstructs on-disk chunks from scratch; the
// returned lookup replaces the in-memory map.
func (c *ChangesetsContainer) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	c.state.Lock()
	defer c.state.Unlock()

	originalCount := len(c.state.changesets)
	newLookup, err := c.persister.CleanupChangesets(ctx, thresholdDate)
	if err != nil {
		return 0, err
	}
	c.state.changesets = newLookup
	return int64(originalCount - len(newLookup)), nil
}

func copyChangeset(changeset *manage.Changeset) (manage.Changeset, error) {
	changesCopy, err := copyBlueprintChanges(changeset.Changes)
	if err != nil {
		return manage.Changeset{}, err
	}
	return manage.Changeset{
		ID:                changeset.ID,
		InstanceID:        changeset.InstanceID,
		Destroy:           changeset.Destroy,
		Status:            changeset.Status,
		BlueprintLocation: changeset.BlueprintLocation,
		Changes:           changesCopy,
		Created:           changeset.Created,
	}, nil
}

func copyBlueprintChanges(src *changes.BlueprintChanges) (*changes.BlueprintChanges, error) {
	if src == nil {
		return nil, nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	dst := &changes.BlueprintChanges{}
	if err := json.Unmarshal(data, dst); err != nil {
		return nil, err
	}
	return dst, nil
}
