package statestore

import (
	"context"
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// MaxCleanupOperationsPerType caps how many cleanup operations of a given
// type are retained. Older entries are evicted from in-memory state and
// storage when the cap is exceeded.
const MaxCleanupOperationsPerType = 50

// CleanupOperationsContainer implements manage.CleanupOperations.
// Storage-backed via the Persister; in-memory rolling window enforced
// internally and mirrored to Storage on eviction.
type CleanupOperationsContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewCleanupOperationsContainer(st *State, persister *Persister, logger core.Logger) *CleanupOperationsContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &CleanupOperationsContainer{state: st, persister: persister, logger: logger}
}

func (c *CleanupOperationsContainer) Get(
	ctx context.Context,
	id string,
) (*manage.CleanupOperation, error) {
	op, ok, err := c.state.LookupCleanupOperation(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, manage.CleanupOperationNotFoundError(id)
	}
	return copyCleanupOperation(op), nil
}

func (c *CleanupOperationsContainer) GetLatestByType(
	ctx context.Context,
	cleanupType manage.CleanupType,
) (*manage.CleanupOperation, error) {
	c.state.RLock()
	defer c.state.RUnlock()

	var latest *manage.CleanupOperation
	for _, op := range c.state.cleanupOps {
		if op.CleanupType != cleanupType {
			continue
		}
		if latest == nil || op.StartedAt > latest.StartedAt {
			latest = op
		}
	}
	if latest == nil {
		return nil, manage.CleanupOperationNotFoundForTypeError(cleanupType)
	}
	return copyCleanupOperation(latest), nil
}

func (c *CleanupOperationsContainer) Save(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	c.state.cleanupOps[operation.ID] = copyCleanupOperation(operation)
	if err := c.persister.CreateCleanupOperation(ctx, operation); err != nil {
		return err
	}
	return c.enforceRollingWindow(ctx, operation.CleanupType)
}

func (c *CleanupOperationsContainer) Update(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	if _, exists := c.state.cleanupOps[operation.ID]; !exists {
		return manage.CleanupOperationNotFoundError(operation.ID)
	}
	c.state.cleanupOps[operation.ID] = copyCleanupOperation(operation)
	return c.persister.UpdateCleanupOperation(ctx, operation)
}

// enforceRollingWindow caps on-disk + in-memory history for a given cleanup
// type at MaxCleanupOperationsPerType, evicting the oldest by StartedAt.
// Persisted evictions flow through the persister so disk and memory stay
// in sync.
func (c *CleanupOperationsContainer) enforceRollingWindow(
	ctx context.Context,
	cleanupType manage.CleanupType,
) error {
	var typeOps []*manage.CleanupOperation
	for _, op := range c.state.cleanupOps {
		if op.CleanupType == cleanupType {
			typeOps = append(typeOps, op)
		}
	}

	if len(typeOps) <= MaxCleanupOperationsPerType {
		return nil
	}

	sort.Slice(typeOps, func(i, j int) bool {
		return typeOps[i].StartedAt > typeOps[j].StartedAt
	})

	for i := MaxCleanupOperationsPerType; i < len(typeOps); i++ {
		evictedID := typeOps[i].ID
		delete(c.state.cleanupOps, evictedID)
		if err := c.persister.RemoveCleanupOperation(ctx, evictedID); err != nil {
			return err
		}
	}

	return nil
}

func copyCleanupOperation(op *manage.CleanupOperation) *manage.CleanupOperation {
	if op == nil {
		return nil
	}
	return &manage.CleanupOperation{
		ID:            op.ID,
		CleanupType:   op.CleanupType,
		Status:        op.Status,
		StartedAt:     op.StartedAt,
		EndedAt:       op.EndedAt,
		ItemsDeleted:  op.ItemsDeleted,
		ErrorMessage:  op.ErrorMessage,
		ThresholdDate: op.ThresholdDate,
	}
}
