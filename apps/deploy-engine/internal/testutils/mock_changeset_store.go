package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

type MockChangesetStore struct {
	Changesets map[string]*manage.Changeset
	mu         sync.Mutex
}

func NewMockChangesetStore(
	validations map[string]*manage.Changeset,
) manage.Changesets {
	return &MockChangesetStore{
		Changesets: validations,
	}
}

func (s *MockChangesetStore) Get(
	ctx context.Context,
	id string,
) (*manage.Changeset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if Changeset, ok := s.Changesets[id]; ok {
		return Changeset, nil
	}

	return nil, manage.ChangesetNotFoundError(id)
}

func (s *MockChangesetStore) Save(
	ctx context.Context,
	Changeset *manage.Changeset,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Changesets[Changeset.ID] = Changeset

	return nil
}

func (s *MockChangesetStore) Cleanup(ctx context.Context, thresholdDate time.Time) error {
	// This is a no-op for the mock event store.
	return nil
}
