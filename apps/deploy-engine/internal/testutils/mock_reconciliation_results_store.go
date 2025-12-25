package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

// MockReconciliationResultsStore is a mock implementation of manage.ReconciliationResults
// for testing purposes.
type MockReconciliationResultsStore struct {
	Results map[string]*manage.ReconciliationResult
	mu      sync.Mutex
}

// NewMockReconciliationResultsStore creates a new mock reconciliation results store.
func NewMockReconciliationResultsStore(
	results map[string]*manage.ReconciliationResult,
) manage.ReconciliationResults {
	return &MockReconciliationResultsStore{
		Results: results,
	}
}

func (s *MockReconciliationResultsStore) Get(
	ctx context.Context,
	id string,
) (*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if result, ok := s.Results[id]; ok {
		return result, nil
	}

	return nil, manage.ReconciliationResultNotFoundError(id)
}

func (s *MockReconciliationResultsStore) GetLatestByChangesetID(
	ctx context.Context,
	changesetID string,
) (*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var latest *manage.ReconciliationResult
	for _, result := range s.Results {
		if result.ChangesetID == changesetID {
			if latest == nil || result.Created > latest.Created {
				latest = result
			}
		}
	}

	if latest == nil {
		return nil, manage.ReconciliationResultNotFoundForChangesetError(changesetID)
	}

	return latest, nil
}

func (s *MockReconciliationResultsStore) GetAllByChangesetID(
	ctx context.Context,
	changesetID string,
) ([]*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var results []*manage.ReconciliationResult
	for _, result := range s.Results {
		if result.ChangesetID == changesetID {
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *MockReconciliationResultsStore) GetLatestByInstanceID(
	ctx context.Context,
	instanceID string,
) (*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var latest *manage.ReconciliationResult
	for _, result := range s.Results {
		if result.InstanceID == instanceID {
			if latest == nil || result.Created > latest.Created {
				latest = result
			}
		}
	}

	if latest == nil {
		return nil, manage.ReconciliationResultNotFoundForInstanceError(instanceID)
	}

	return latest, nil
}

func (s *MockReconciliationResultsStore) GetAllByInstanceID(
	ctx context.Context,
	instanceID string,
) ([]*manage.ReconciliationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var results []*manage.ReconciliationResult
	for _, result := range s.Results {
		if result.InstanceID == instanceID {
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *MockReconciliationResultsStore) Save(
	ctx context.Context,
	result *manage.ReconciliationResult,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Results[result.ID] = result

	return nil
}

func (s *MockReconciliationResultsStore) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) error {
	// This is a no-op for the mock store.
	return nil
}
