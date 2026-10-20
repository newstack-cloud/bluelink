package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
)

// MockAuditStore is a mock implementation of manage.Audit
// for testing purposes.
type MockAuditStore struct {
	Records    map[string]*manage.AuditRecord
	SaveError  error
	QueryError error
	mu         sync.Mutex
}

// NewMockAuditStore creates a new mock audit store.
func NewMockAuditStore(records map[string]*manage.AuditRecord) *MockAuditStore {
	return &MockAuditStore{
		Records: records,
	}
}

func (s *MockAuditStore) Get(
	ctx context.Context,
	id string,
) (*manage.AuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record, ok := s.Records[id]; ok {
		copied := *record
		return &copied, nil
	}

	return nil, manage.AuditRecordNotFoundError(id)
}

func (s *MockAuditStore) Save(
	ctx context.Context,
	record *manage.AuditRecord,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.SaveError != nil {
		return s.SaveError
	}

	copied := *record
	s.Records[record.ID] = &copied
	return nil
}

func (s *MockAuditStore) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.QueryError != nil {
		return nil, s.QueryError
	}

	// Narrowed through the real helper rather than a copy of the rules here, so a
	// change to the filtering or ordering manage.Audit documents shows up in the
	// tests that assert against this mock.
	records := make([]*manage.AuditRecord, 0, len(s.Records))
	for _, record := range s.Records {
		copied := *record
		records = append(records, &copied)
	}

	return statestore.NarrowAuditRecords(records, query), nil
}

func (s *MockAuditStore) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := thresholdDate.Unix()
	removed := int64(0)
	for id, record := range s.Records {
		if record.Timestamp <= threshold {
			delete(s.Records, id)
			removed++
		}
	}

	return removed, nil
}
