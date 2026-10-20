package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

// Operations are held in a map and timed to the second, so two for one category can share
// a started time. A deploy engine reads the latest to decide whether to skip a scheduled
// pass, and the rolling window deletes by the same ordering, so neither may depend on map
// iteration order.
type MemFileStateContainerCleanupOperationsSuite struct {
	container *StateContainer
	suite.Suite
}

func (s *MemFileStateContainerCleanupOperationsSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *MemFileStateContainerCleanupOperationsSuite) Test_the_latest_of_two_sharing_a_second_is_the_greater_id() {
	operations := s.container.CleanupOperations()
	for _, id := range []string{"op-a", "op-c", "op-b"} {
		s.Require().NoError(
			operations.Save(context.Background(), s.operation(id, 1000)),
		)
	}

	latest, err := operations.GetLatestByType(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)

	s.Assert().Equal("op-c", latest.ID)
}

// Reading it repeatedly has to give the same answer, which is the property a scheduled
// pass depends on when it decides whether a category was cleaned up recently.
func (s *MemFileStateContainerCleanupOperationsSuite) Test_the_latest_does_not_change_between_reads() {
	operations := s.container.CleanupOperations()
	for i := range 10 {
		s.Require().NoError(
			operations.Save(context.Background(), s.operation(fmt.Sprintf("op-%02d", i), 1000)),
		)
	}

	first, err := operations.GetLatestByType(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)

	for range 20 {
		again, err := operations.GetLatestByType(context.Background(), manage.CleanupTypeEvents)
		s.Require().NoError(err)
		s.Require().Equal(first.ID, again.ID)
	}
}

// A later start still wins, whatever the IDs are, or the tiebreaker would have taken over
// the ordering rather than only settling ties.
func (s *MemFileStateContainerCleanupOperationsSuite) Test_a_later_start_wins_over_a_greater_id() {
	operations := s.container.CleanupOperations()
	s.Require().NoError(operations.Save(context.Background(), s.operation("op-z", 1000)))
	s.Require().NoError(operations.Save(context.Background(), s.operation("op-a", 2000)))

	latest, err := operations.GetLatestByType(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)

	s.Assert().Equal("op-a", latest.ID)
}

// The window deletes the operations outside it, so which ones those are must not depend on
// map iteration order either.
func (s *MemFileStateContainerCleanupOperationsSuite) Test_the_rolling_window_keeps_the_same_operations_by_id() {
	operations := s.container.CleanupOperations()
	total := statestore.MaxCleanupOperationsPerType + 5
	for i := range total {
		s.Require().NoError(
			operations.Save(context.Background(), s.operation(fmt.Sprintf("op-%03d", i), 1000)),
		)
	}

	// The greatest IDs are kept, since every operation shares a started time.
	for i := total - statestore.MaxCleanupOperationsPerType; i < total; i++ {
		id := fmt.Sprintf("op-%03d", i)
		_, err := operations.Get(context.Background(), id)
		s.Assert().NoError(err, "%s should have been kept", id)
	}

	for i := range total - statestore.MaxCleanupOperationsPerType {
		id := fmt.Sprintf("op-%03d", i)
		_, err := operations.Get(context.Background(), id)
		s.Assert().Error(err, "%s should have been evicted", id)
	}
}

func (s *MemFileStateContainerCleanupOperationsSuite) operation(
	id string,
	startedAt int64,
) *manage.CleanupOperation {
	return &manage.CleanupOperation{
		ID:            id,
		CleanupType:   manage.CleanupTypeEvents,
		Status:        manage.CleanupOperationStatusCompleted,
		StartedAt:     startedAt,
		EndedAt:       startedAt + 1,
		ItemsDeleted:  3,
		ThresholdDate: startedAt - 100,
	}
}

func TestMemFileStateContainerCleanupOperationsSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerCleanupOperationsSuite))
}
