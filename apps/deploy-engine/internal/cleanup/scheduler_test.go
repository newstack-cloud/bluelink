package cleanup_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type SchedulerSuite struct {
	suite.Suite
}

func (s *SchedulerSuite) Test_runs_every_task_on_each_pass() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	events := &recordingRemover{}
	validations := &recordingRemover{}

	stop := s.start(operations, testTime, []cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
		{Type: manage.CleanupTypeValidations, Remove: validations.Remove},
	})
	defer stop()

	s.eventually(func() bool {
		return events.Calls() >= 1 && validations.Calls() >= 1
	}, "both categories should have been cleaned up")
}

func (s *SchedulerSuite) Test_records_an_operation_for_each_scheduled_run() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	events := &recordingRemover{}

	stop := s.start(operations, testTime, []cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
	})
	defer stop()

	s.eventually(func() bool { return events.Calls() >= 1 }, "the category should have been cleaned up")

	recorded, err := operations.GetLatestByType(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)
	s.Assert().Equal(manage.CleanupOperationStatusCompleted, recorded.Status)
}

// Every instance sharing a state backend runs its own scheduler, so a category another
// one cleaned up within the interval is left alone.
func (s *SchedulerSuite) Test_skips_a_category_cleaned_up_within_the_interval() {
	operations := testutils.NewMockCleanupOperationsStore(
		map[string]*manage.CleanupOperation{
			"recent-operation": {
				ID:          "recent-operation",
				CleanupType: manage.CleanupTypeEvents,
				Status:      manage.CleanupOperationStatusCompleted,
				StartedAt:   testTime.Unix(),
			},
		},
	)
	events := &recordingRemover{}

	stop := s.start(operations, testTime, []cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
	})
	defer stop()

	s.Assert().Never(
		func() bool { return events.Calls() > 0 },
		50*time.Millisecond,
		time.Millisecond,
		"a category cleaned up within the interval should be skipped",
	)
}

// Shutdown must not leave a pass half-done with its operation still reading "running".
func (s *SchedulerSuite) Test_stopping_waits_for_an_in_flight_pass() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})

	entered := make(chan struct{})
	var once sync.Once
	finished := false
	var mu sync.Mutex

	scheduler := cleanup.NewScheduler(
		s.runnerFor(operations, testTime),
		[]cleanup.Task{{
			Type: manage.CleanupTypeEvents,
			Remove: func(context.Context, time.Time) (int64, error) {
				once.Do(func() { close(entered) })
				time.Sleep(20 * time.Millisecond)
				mu.Lock()
				finished = true
				mu.Unlock()
				return 0, nil
			},
		}},
		time.Millisecond,
		&testutils.MockClock{StaticTime: testTime},
		core.NewNopLogger(),
	)

	stop := scheduler.Start()
	<-entered
	stop()

	mu.Lock()
	defer mu.Unlock()
	s.Assert().True(finished, "stopping should wait for the in-flight pass to finish")
}

func (s *SchedulerSuite) start(
	operations manage.CleanupOperations,
	now time.Time,
	tasks []cleanup.Task,
) func() {
	scheduler := cleanup.NewScheduler(
		s.runnerFor(operations, now),
		tasks,
		time.Millisecond,
		&testutils.MockClock{StaticTime: now},
		core.NewNopLogger(),
	)

	return scheduler.Start()
}

func (s *SchedulerSuite) runnerFor(
	operations manage.CleanupOperations,
	now time.Time,
) *cleanup.Runner {
	return cleanup.NewRunner(
		operations,
		core.NewUUIDGenerator(),
		&testutils.MockClock{StaticTime: now},
		core.NewNopLogger(),
	)
}

func (s *SchedulerSuite) eventually(condition func() bool, message string) {
	s.Require().Eventually(condition, time.Second, time.Millisecond, message)
}

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, new(SchedulerSuite))
}
