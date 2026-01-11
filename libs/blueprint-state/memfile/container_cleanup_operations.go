package memfile

import (
	"context"
	"sort"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

const maxCleanupOperationsPerType = 50

type cleanupOperationsContainerImpl struct {
	operations map[string]*manage.CleanupOperation
	mu         *sync.RWMutex
}

func newCleanupOperationsContainer() *cleanupOperationsContainerImpl {
	return &cleanupOperationsContainerImpl{
		operations: make(map[string]*manage.CleanupOperation),
		mu:         &sync.RWMutex{},
	}
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

func (c *cleanupOperationsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.CleanupOperation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	op, exists := c.operations[id]
	if !exists {
		return nil, manage.CleanupOperationNotFoundError(id)
	}

	return copyCleanupOperation(op), nil
}

func (c *cleanupOperationsContainerImpl) GetLatestByType(
	ctx context.Context,
	cleanupType manage.CleanupType,
) (*manage.CleanupOperation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var latest *manage.CleanupOperation
	for _, op := range c.operations {
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

func (c *cleanupOperationsContainerImpl) Save(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.operations[operation.ID] = copyCleanupOperation(operation)
	c.enforceRollingWindow(operation.CleanupType)

	return nil
}

func (c *cleanupOperationsContainerImpl) Update(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.operations[operation.ID]; !exists {
		return manage.CleanupOperationNotFoundError(operation.ID)
	}

	c.operations[operation.ID] = copyCleanupOperation(operation)
	return nil
}

func (c *cleanupOperationsContainerImpl) enforceRollingWindow(cleanupType manage.CleanupType) {
	var typeOps []*manage.CleanupOperation
	for _, op := range c.operations {
		if op.CleanupType == cleanupType {
			typeOps = append(typeOps, op)
		}
	}

	if len(typeOps) <= maxCleanupOperationsPerType {
		return
	}

	sort.Slice(typeOps, func(i, j int) bool {
		return typeOps[i].StartedAt > typeOps[j].StartedAt
	})

	for i := maxCleanupOperationsPerType; i < len(typeOps); i++ {
		delete(c.operations, typeOps[i].ID)
	}
}
