package cleanup_test

import (
	"context"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
	"github.com/stretchr/testify/suite"
)

// A cleanup that a request started is work the engine carries on with after answering that
// request, so it gets a trace of its own rather than hanging off the request's.
type RunnerTracingSuite struct {
	suite.Suite

	recorder *telemetry.Recorder
}

func (s *RunnerTracingSuite) SetupTest() {
	s.recorder = telemetry.NewRecorder()
	telemetry.SetTracer(s.recorder)
}

func (s *RunnerTracingSuite) TearDownTest() {
	telemetry.SetTracer(nil)
}

func (s *RunnerTracingSuite) Test_records_a_requested_cleanup_as_a_trace_of_its_own() {
	operations := testutils.NewMockCleanupOperationsStore(map[string]*manage.CleanupOperation{})
	runner := cleanup.NewRunner(
		operations,
		core.NewUUIDGenerator(),
		&testutils.MockClock{StaticTime: testTime},
		core.NewNopLogger(),
	)

	// Stands in for the request that asked for the cleanup.
	requestCtx, request := telemetry.CurrentTracer().Start(context.Background(), "request")
	removed := &recordingRemover{count: 3}
	_, err := runner.Start(requestCtx, cleanup.Task{
		Type:   manage.CleanupTypeEvents,
		Remove: removed.Remove,
	})
	s.Require().NoError(err)
	request.End()

	runner.Shutdown(context.Background())

	// The span is ended just after the work it covers, so the wait above does not
	// already guarantee it.
	var run *telemetry.RecordedSpan
	s.Require().Eventually(
		func() bool {
			recorded, found := s.recorder.Span("cleanup.run")
			if !found || !recorded.Ended {
				return false
			}
			run = recorded

			return true
		},
		time.Second,
		time.Millisecond,
		"a requested cleanup should record an ended span, recorded %v",
		s.recorder.Names(),
	)

	s.Assert().True(run.Root, "a cleanup run begins a trace of its own")
	s.Assert().Empty(run.ParentID, "it does not hang off the request")
	s.Assert().Equal(
		request.Context(),
		run.LinkedTo,
		"it names the request that asked for it",
	)

	cleanupType, _ := run.Attr("cleanupType")
	s.Assert().Equal(string(manage.CleanupTypeEvents), cleanupType)
	status, _ := run.Attr("status")
	s.Assert().Equal(string(manage.CleanupOperationStatusCompleted), status)
	deleted, _ := run.Attr("itemsDeleted")
	s.Assert().Equal(int64(3), deleted)
}

func TestRunnerTracingSuite(t *testing.T) {
	suite.Run(t, new(RunnerTracingSuite))
}
