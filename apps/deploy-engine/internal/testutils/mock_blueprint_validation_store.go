package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

type MockBlueprintValidationStore struct {
	BlueprintValidations map[string]*manage.BlueprintValidation
	mu                   sync.Mutex
}

func NewMockBlueprintValidationStore(
	validations map[string]*manage.BlueprintValidation,
) manage.Validation {
	return &MockBlueprintValidationStore{
		BlueprintValidations: validations,
	}
}

func (s *MockBlueprintValidationStore) Get(
	ctx context.Context,
	id string,
) (*manage.BlueprintValidation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if blueprintValidation, ok := s.BlueprintValidations[id]; ok {
		return blueprintValidation, nil
	}

	return nil, manage.BlueprintValidationNotFoundError(id)
}

func (s *MockBlueprintValidationStore) Save(
	ctx context.Context,
	blueprintValidation *manage.BlueprintValidation,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.BlueprintValidations[blueprintValidation.ID] = blueprintValidation

	return nil
}

func (s *MockBlueprintValidationStore) Cleanup(ctx context.Context, thresholdDate time.Time) error {
	// This is a no-op for the mock event store.
	return nil
}
