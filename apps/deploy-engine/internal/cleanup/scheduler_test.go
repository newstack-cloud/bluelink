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
	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
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
	return startScheduler(operations, now, tasks)
}

func (s *SchedulerSuite) runnerFor(
	operations manage.CleanupOperations,
	now time.Time,
) *cleanup.Runner {
	return runnerFor(operations, now)
}

func (s *SchedulerSuite) eventually(condition func() bool, message string) {
	s.Require().Eventually(condition, time.Second, time.Millisecond, message)
}

// An interval of a millisecond, so a test does not wait out a realistic one. The first
// pass still waits a full interval, which is what the scheduler does.
func startScheduler(
	operations manage.CleanupOperations,
	now time.Time,
	tasks []cleanup.Task,
) func() {
	scheduler := cleanup.NewScheduler(
		runnerFor(operations, now),
		tasks,
		time.Millisecond,
		&testutils.MockClock{StaticTime: now},
		core.NewNopLogger(),
	)

	return scheduler.Start()
}

func runnerFor(operations manage.CleanupOperations, now time.Time) *cleanup.Runner {
	return cleanup.NewRunner(
		operations,
		core.NewUUIDGenerator(),
		&testutils.MockClock{StaticTime: now},
		core.NewNopLogger(),
	)
}

func noOperations() manage.CleanupOperations {
	return testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
}

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, new(SchedulerSuite))
}

// A pass that spent its time on a single category should show that, not only how long
// the whole of it took, and the entries it wrote have to lead to the trace of it.
type SchedulerTracingSuite struct {
	suite.Suite

	recorder *telemetry.Recorder
}

func (s *SchedulerTracingSuite) SetupTest() {
	s.recorder = telemetry.NewRecorder()
	telemetry.SetTracer(s.recorder)
}

func (s *SchedulerTracingSuite) TearDownTest() {
	telemetry.SetTracer(nil)
}

func (s *SchedulerTracingSuite) Test_records_a_span_for_each_category_within_the_pass() {
	events := &recordingRemover{}
	validations := &recordingRemover{}

	stop := s.start([]cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
		{Type: manage.CleanupTypeValidations, Remove: validations.Remove},
	})
	defer stop()

	s.eventually(
		func() bool { return events.Calls() >= 1 && validations.Calls() >= 1 },
		"both categories should have been cleaned up",
	)

	pass, found := s.firstSpan("cleanup.pass")
	s.Require().True(found, "recorded %v", s.recorder.Names())
	// A scheduled pass is an operation rather than request traffic, so it begins a trace
	// of its own.
	s.Assert().True(pass.Root, "a scheduled pass begins a trace of its own")
	s.Assert().Empty(pass.LinkedTo.TraceID, "nothing asked for a scheduled pass")

	for _, cleanupType := range []manage.CleanupType{
		manage.CleanupTypeEvents,
		manage.CleanupTypeValidations,
	} {
		task := s.taskSpanFor(cleanupType)
		s.Require().NotNil(task, "no span for %q in %v", cleanupType, s.recorder.Names())
		s.Assert().Equal(
			pass.SpanID,
			task.ParentID,
			"a task span belongs to the pass that ran it",
		)
		s.Assert().True(task.Ended)
	}
}

func (s *SchedulerTracingSuite) Test_records_what_a_cleanup_removed() {
	events := &recordingRemover{count: 4}

	stop := s.start([]cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
	})
	defer stop()

	s.eventually(func() bool {
		task := s.taskSpanFor(manage.CleanupTypeEvents)
		if task == nil {
			return false
		}
		removed, found := task.Attr("itemsDeleted")

		return found && removed == int64(4)
	}, "the span should carry what was removed")
}

// A category another instance cleaned up is skipped, and a span saying so is the only
// account of why a pass did nothing.
func (s *SchedulerTracingSuite) Test_records_why_a_category_was_skipped() {
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

	stop := startScheduler(operations, testTime, []cleanup.Task{
		{Type: manage.CleanupTypeEvents, Remove: events.Remove},
	})
	defer stop()

	s.eventually(func() bool {
		task := s.taskSpanFor(manage.CleanupTypeEvents)
		if task == nil {
			return false
		}
		outcome, found := task.Attr("outcome")

		return found && outcome == "skippedCleanedUpRecently"
	}, "the span should say why the category was skipped")

	s.Assert().Zero(events.Calls(), "a category cleaned up recently should be left alone")
}

func (s *SchedulerTracingSuite) Test_marks_a_cleanup_that_failed_as_failed() {
	stop := s.start([]cleanup.Task{{
		Type: manage.CleanupTypeEvents,
		Remove: func(context.Context, time.Time) (int64, error) {
			return 0, errors.New("the state backend refused the delete")
		},
	}})
	defer stop()

	s.eventually(func() bool {
		task := s.taskSpanFor(manage.CleanupTypeEvents)

		return task != nil && task.Err != nil
	}, "the span should be marked as failed")
}

func (s *SchedulerTracingSuite) start(tasks []cleanup.Task) func() {
	return startScheduler(noOperations(), testTime, tasks)
}

func (s *SchedulerTracingSuite) eventually(condition func() bool, message string) {
	s.Require().Eventually(condition, time.Second, time.Millisecond, message)
}

// The scheduler keeps running, so there is a span per pass rather than one in total.
func (s *SchedulerTracingSuite) firstSpan(name string) (*telemetry.RecordedSpan, bool) {
	for _, span := range s.recorder.Spans() {
		if span.Name == name {
			return span, true
		}
	}

	return nil, false
}

func (s *SchedulerTracingSuite) taskSpanFor(
	cleanupType manage.CleanupType,
) *telemetry.RecordedSpan {
	for _, span := range s.recorder.Spans() {
		if span.Name != "cleanup.task" {
			continue
		}
		if recorded, found := span.Attr("cleanupType"); found &&
			recorded == string(cleanupType) {
			return span
		}
	}

	return nil
}

func TestSchedulerTracingSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTracingSuite))
}
