package testutils

import (
	"context"
	"sort"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

// MockCleanupOperationsStore is a mock implementation of manage.CleanupOperations
// for testing purposes.
type MockCleanupOperationsStore struct {
	Operations map[string]*manage.CleanupOperation
	SaveError  error
	mu         sync.Mutex
}

// NewMockCleanupOperationsStore creates a new mock cleanup operations store.
func NewMockCleanupOperationsStore(
	operations map[string]*manage.CleanupOperation,
) manage.CleanupOperations {
	return &MockCleanupOperationsStore{
		Operations: operations,
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

func (s *MockCleanupOperationsStore) Get(
	ctx context.Context,
	id string,
) (*manage.CleanupOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if op, ok := s.Operations[id]; ok {
		return copyCleanupOperation(op), nil
	}

	return nil, manage.CleanupOperationNotFoundError(id)
}

func (s *MockCleanupOperationsStore) GetLatestByType(
	ctx context.Context,
	cleanupType manage.CleanupType,
) (*manage.CleanupOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var latest *manage.CleanupOperation
	for _, op := range s.Operations {
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

func (s *MockCleanupOperationsStore) Save(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.SaveError != nil {
		return s.SaveError
	}

	s.Operations[operation.ID] = copyCleanupOperation(operation)
	s.enforceRollingWindow(operation.CleanupType)

	return nil
}

func (s *MockCleanupOperationsStore) Update(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.Operations[operation.ID]; !exists {
		return manage.CleanupOperationNotFoundError(operation.ID)
	}

	s.Operations[operation.ID] = copyCleanupOperation(operation)
	return nil
}

func (s *MockCleanupOperationsStore) enforceRollingWindow(cleanupType manage.CleanupType) {
	var typeOps []*manage.CleanupOperation
	for _, op := range s.Operations {
		if op.CleanupType == cleanupType {
			typeOps = append(typeOps, op)
		}
	}

	if len(typeOps) <= 50 {
		return
	}

	sort.Slice(typeOps, func(i, j int) bool {
		return typeOps[i].StartedAt > typeOps[j].StartedAt
	})

	for i := 50; i < len(typeOps); i++ {
		delete(s.Operations, typeOps[i].ID)
	}
}
