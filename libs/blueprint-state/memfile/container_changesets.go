package memfile

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

type changesetsContainerImpl struct {
	changesets map[string]*manage.Changeset
	fs         afero.Fs
	persister  *statePersister
	logger     core.Logger
	mu         *sync.RWMutex
}

func (c *changesetsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.Changeset, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	changeset, ok := c.changesets[id]
	if !ok {
		return nil, manage.ChangesetNotFoundError(id)
	}

	changesetCopy, err := copyChangeset(changeset)
	if err != nil {
		return nil, err
	}

	return &changesetCopy, nil
}

func (c *changesetsContainerImpl) Save(
	ctx context.Context,
	changeset *manage.Changeset,
) error {
	c.mu.Lock()
	// Defer unlock to ensure that modifications are not made to in-memory
	// state during persistence.
	defer c.mu.Unlock()

	return c.save(changeset)
}

func (c *changesetsContainerImpl) save(
	changeset *manage.Changeset,
) error {
	changesetLogger := c.logger.WithFields(
		core.StringLogField("changesetId", changeset.ID),
	)
	_, alreadyExists := c.changesets[changeset.ID]
	c.changesets[changeset.ID] = changeset

	if alreadyExists {
		changesetLogger.Debug("persisting change set update")
		return c.persister.updateChangeset(changeset)
	}

	changesetLogger.Debug("persisting new change set")
	return c.persister.createChangeset(changeset)
}

func (c *changesetsContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// As the persister manages how change sets are stored,
	// it is responsible for cleaning up the change sets.
	// The in-memory state of the container implementation
	// doesn't require change sets to be in any particular order
	// but the persister is expected to maintain order by timestamp
	// when saving change sets to files.
	newLookup, err := c.persister.cleanupChangesets(thresholdDate)
	if err != nil {
		return err
	}
	c.changesets = newLookup

	return nil
}

func copyChangeset(
	changeset *manage.Changeset,
) (manage.Changeset, error) {
	changesCopy, err := copyChanges(changeset.Changes)
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

func copyChanges(
	changesetChanges *changes.BlueprintChanges,
) (*changes.BlueprintChanges, error) {
	if changesetChanges == nil {
		return nil, nil
	}

	// Marshalling and unmarshalling the changeset changes
	// is a convenient way to create a deep copy.
	// For large change structures, the inefficiency of this approach
	// may be a concern, it's worth monitoring performance in real world usage
	// and considering a more efficient deep copy approach if necessary
	// in the future.
	changesCopy := &changes.BlueprintChanges{}
	changesBytes, err := json.Marshal(changesetChanges)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(changesBytes, changesCopy)
	if err != nil {
		return nil, err
	}

	return changesCopy, nil
}
