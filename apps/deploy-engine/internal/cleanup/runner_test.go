package cleanup_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Saturday, 3 May 2025 14:27:22 UTC
var testTime = time.Unix(1746282442, 0).UTC()

// One implementation serves both the endpoints and the scheduler, so what matters is
// that a run is recorded the same way whichever started it, and that two cannot run
// over one category at once.
type RunnerSuite struct {
	suite.Suite
}

func (s *RunnerSuite) Test_run_records_a_completed_operation() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	removed := &recordingRemover{count: 7}
	operation, err := runner.Run(context.Background(), cleanup.Task{
		Type:            manage.CleanupTypeEvents,
		RetentionPeriod: 7 * 24 * time.Hour,
		Remove:          removed.Remove,
		LogDescription:  "events",
	})
	s.Require().NoError(err)

	s.Assert().Equal(manage.CleanupOperationStatusCompleted, operation.Status)
	s.Assert().Equal(int64(7), operation.ItemsDeleted)
	s.Assert().Equal(manage.CleanupTypeEvents, operation.CleanupType)
	s.Assert().Equal(testTime.Unix(), operation.StartedAt)
	s.Assert().Empty(operation.ErrorMessage)
}

// The threshold is what the retention period is for, so it is the one value worth
// pinning rather than inferring from the operation's own timestamps.
func (s *RunnerSuite) Test_run_passes_the_retention_threshold_to_the_task() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	removed := &recordingRemover{}
	_, err := runner.Run(context.Background(), cleanup.Task{
		Type:            manage.CleanupTypeValidations,
		RetentionPeriod: 7 * 24 * time.Hour,
		Remove:          removed.Remove,
	})
	s.Require().NoError(err)

	s.Assert().Equal(
		testTime.Add(-7*24*time.Hour).Unix(),
		removed.thresholdDate.Unix(),
	)
}

func (s *RunnerSuite) Test_run_records_a_failed_operation() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	operation, err := runner.Run(context.Background(), cleanup.Task{
		Type: manage.CleanupTypeChangesets,
		Remove: func(context.Context, time.Time) (int64, error) {
			return 0, errors.New("the state backend is unreachable")
		},
		LogDescription: "change sets",
	})
	s.Require().NoError(err)

	s.Assert().Equal(manage.CleanupOperationStatusFailed, operation.Status)
	s.Assert().Equal("the state backend is unreachable", operation.ErrorMessage)
}

// Both drivers go through the same guard, so a requested cleanup and a scheduled one
// cannot walk the same records at once.
func (s *RunnerSuite) Test_refuses_a_second_run_while_one_is_in_progress() {
	operations := testutils.NewMockCleanupOperationsStore(
		map[string]*manage.CleanupOperation{
			"running-operation": {
				ID:          "running-operation",
				CleanupType: manage.CleanupTypeEvents,
				Status:      manage.CleanupOperationStatusRunning,
				StartedAt:   testTime.Unix(),
			},
		},
	)
	runner := s.runnerFor(operations, testTime)

	removed := &recordingRemover{}
	_, err := runner.Run(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: removed.Remove,
	})
	s.Require().ErrorIs(err, cleanup.ErrCleanupInProgress)
	s.Assert().Zero(removed.calls, "the task must not run while another is in progress")
}

// An instance that died partway through leaves an operation reading "running" for
// ever, which must not block the category permanently.
func (s *RunnerSuite) Test_runs_despite_an_operation_left_running_longer_than_a_run_may_take() {
	operations := testutils.NewMockCleanupOperationsStore(
		map[string]*manage.CleanupOperation{
			"abandoned-operation": {
				ID:          "abandoned-operation",
				CleanupType: manage.CleanupTypeEvents,
				Status:      manage.CleanupOperationStatusRunning,
				StartedAt:   testTime.Add(-2 * cleanup.DefaultTimeout).Unix(),
			},
		},
	)
	runner := s.runnerFor(operations, testTime)

	removed := &recordingRemover{}
	operation, err := runner.Run(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: removed.Remove,
	})
	s.Require().NoError(err)
	s.Assert().Equal(manage.CleanupOperationStatusCompleted, operation.Status)
	s.Assert().Equal(1, removed.calls)
}

// A category whose previous run finished is free, whatever that run's outcome was.
func (s *RunnerSuite) Test_runs_after_a_previous_run_finished() {
	for _, status := range []manage.CleanupOperationStatus{
		manage.CleanupOperationStatusCompleted,
		manage.CleanupOperationStatusFailed,
	} {
		operations := testutils.NewMockCleanupOperationsStore(
			map[string]*manage.CleanupOperation{
				"previous-operation": {
					ID:          "previous-operation",
					CleanupType: manage.CleanupTypeEvents,
					Status:      status,
					StartedAt:   testTime.Unix(),
				},
			},
		)
		runner := s.runnerFor(operations, testTime)

		removed := &recordingRemover{}
		_, err := runner.Run(context.Background(), cleanup.Task{
			Type:   manage.CleanupTypeEvents,
			Remove: removed.Remove,
		})
		s.Require().NoError(err, status)
		s.Assert().Equal(1, removed.calls, status)
	}
}

// Within one engine the guard is exact, not best effort: the scheduler and the cleanup
// endpoints share a Runner, and reading the recorded operations is not atomic with
// recording one, so two concurrent starts would otherwise both see the category free.
func (s *RunnerSuite) Test_only_one_of_many_concurrent_starts_over_a_category_runs() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	release := make(chan struct{})
	remover := &blockingRemover{release: release}

	const attempts = 20
	var waitGroup sync.WaitGroup
	var mu sync.Mutex
	started := 0
	refused := 0

	for range attempts {
		waitGroup.Go(func() {
			_, err := runner.Start(context.Background(), cleanup.Task{
				Type:   manage.CleanupTypeEvents,
				Remove: remover.Remove,
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				started++
				return
			}
			if errors.Is(err, cleanup.ErrCleanupInProgress) {
				refused++
			}
		})
	}
	waitGroup.Wait()

	mu.Lock()
	s.Assert().Equal(1, started, "exactly one start should have been accepted")
	s.Assert().Equal(attempts-1, refused, "every other start should have been refused")
	mu.Unlock()

	close(release)
	s.Require().Eventually(
		func() bool { return remover.Calls() == 1 },
		time.Second,
		time.Millisecond,
		"the single accepted run should have carried out the task once",
	)
}

// The category has to be free again once a run ends, or it could only ever be cleaned
// up once per process.
func (s *RunnerSuite) Test_a_category_can_be_run_again_after_a_run_ends() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	first := &recordingRemover{}
	_, err := runner.Run(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: first.Remove,
	})
	s.Require().NoError(err)

	// The recorded run completed at the same instant on a static clock, so it is the
	// in-flight set rather than the recorded operation that has to have let go.
	second := &recordingRemover{}
	_, err = runner.Run(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: second.Remove,
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, second.calls)
}

// Start hands back the operation as recorded before the work begins, so a caller can
// serialise it into a response while the run is still updating its own copy.
func (s *RunnerSuite) Test_start_returns_the_operation_without_waiting_for_the_work() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	release := make(chan struct{})
	finished := make(chan struct{})
	operation, err := runner.Start(context.Background(), cleanup.Task{
		Type: manage.CleanupTypeAuditRecords,
		Remove: func(context.Context, time.Time) (int64, error) {
			<-release
			close(finished)
			return 3, nil
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(manage.CleanupOperationStatusRunning, operation.Status)
	s.Assert().NotEmpty(operation.ID)

	close(release)
	<-finished
}

// A cleanup started by a request outlives that request, so shutdown has to wait for it
// rather than closing the state service it is deleting through.
func (s *RunnerSuite) Test_shutdown_waits_for_a_cleanup_started_by_a_request() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	release := make(chan struct{})
	remover := &blockingRemover{release: release}
	_, err := runner.Start(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: remover.Remove,
	})
	s.Require().NoError(err)

	returned := make(chan struct{})
	go func() {
		runner.Shutdown(context.Background())
		close(returned)
	}()

	select {
	case <-returned:
		s.Require().Fail("shutdown returned while a cleanup was still running")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		s.Require().Fail("shutdown did not return once the cleanup had finished")
	}

	// Completed rather than cut short, the remover having been released rather than
	// cancelled.
	latest, found, err := runner.LatestRun(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Assert().Equal(manage.CleanupOperationStatusCompleted, latest.Status)
}

// The wait has to cover the closing status write and not only the deletion, or
// shutdown could close the state service while a run was still recording its outcome,
// leaving the operation reading as running for good.
func (s *RunnerSuite) Test_shutdown_waits_for_the_closing_status_write() {
	operations := &testutils.MockCleanupOperationsStore{
		Operations: map[string]*manage.CleanupOperation{},
	}
	updating := make(chan struct{})
	release := make(chan struct{})
	operations.BeforeUpdate = func() {
		close(updating)
		<-release
	}
	runner := s.runnerFor(operations, testTime)

	_, err := runner.Start(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: (&recordingRemover{}).Remove,
	})
	s.Require().NoError(err)

	// The deletion itself is already done by the time the update is reached.
	<-updating

	returned := make(chan struct{})
	go func() {
		runner.Shutdown(context.Background())
		close(returned)
	}()

	select {
	case <-returned:
		s.Require().Fail("shutdown returned while a run was still recording its outcome")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		s.Require().Fail("shutdown did not return once the outcome had been recorded")
	}
}

// Cancelling is what ends the work.
func (s *RunnerSuite) Test_shutdown_cuts_short_a_cleanup_started_by_a_request() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	_, err := runner.Start(context.Background(), cleanup.Task{
		Type: manage.CleanupTypeEvents,
		Remove: func(ctx context.Context, _ time.Time) (int64, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		},
	})
	s.Require().NoError(err)

	returned := make(chan struct{})
	go func() {
		runner.Shutdown(context.Background())
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		s.Require().Fail("shutdown did not cut the cleanup short")
	}

	// Failed rather than completed, so the category is left for the next pass to pick
	// up instead of being reported as cleaned.
	latest, found, err := runner.LatestRun(context.Background(), manage.CleanupTypeEvents)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Assert().Equal(manage.CleanupOperationStatusFailed, latest.Status)
}

// A start accepted after Shutdown has returned would run with nobody left to wait
// for it.
func (s *RunnerSuite) Test_refuses_a_cleanup_once_shutting_down() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	runner.Shutdown(context.Background())

	removed := &recordingRemover{}
	_, startErr := runner.Start(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: removed.Remove,
	})
	s.Require().ErrorIs(startErr, cleanup.ErrShuttingDown)

	_, runErr := runner.Run(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeValidations,
		Remove: removed.Remove,
	})
	s.Require().ErrorIs(runErr, cleanup.ErrShuttingDown)

	s.Assert().Equal(0, removed.Calls(), "a refused cleanup must not carry out the task")
}

// Cancelling a run does not oblige the backend to stop, so the wait needs a deadline of
// its own or a backend that carries on to the end of the pass would hold up process
// exit for as long as it takes.
func (s *RunnerSuite) Test_shutdown_stops_waiting_once_its_deadline_has_passed() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := s.runnerFor(operations, testTime)

	// Takes no notice of the context it is given, as the memfile backend does not.
	release := make(chan struct{})
	defer close(release)
	_, err := runner.Start(context.Background(), cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: (&blockingRemover{release: release}).Remove,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	returned := make(chan struct{})
	go func() {
		runner.Shutdown(ctx)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		s.Require().Fail("shutdown did not give up once its deadline had passed")
	}
}

func (s *RunnerSuite) runnerFor(
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

type recordingRemover struct {
	mu            sync.Mutex
	count         int64
	calls         int
	thresholdDate time.Time
}

func (r *recordingRemover) Remove(_ context.Context, thresholdDate time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.thresholdDate = thresholdDate

	return r.count, nil
}

func (r *recordingRemover) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.calls
}

type blockingRemover struct {
	mu      sync.Mutex
	calls   int
	release chan struct{}
}

func (r *blockingRemover) Remove(_ context.Context, _ time.Time) (int64, error) {
	<-r.release
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++

	return 0, nil
}

func (r *blockingRemover) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.calls
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, new(RunnerSuite))
}
