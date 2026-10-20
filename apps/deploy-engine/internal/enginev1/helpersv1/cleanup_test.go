package helpersv1_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Saturday, 3 May 2025 14:27:22 UTC
var testTime = time.Unix(1746282442, 0).UTC()

// Every cleanup endpoint answers through this, so it is the one place the status codes
// for a started, a refused and a failed cleanup are decided.
type StartCleanupSuite struct {
	suite.Suite
}

func (s *StartCleanupSuite) Test_accepts_a_cleanup_and_reports_the_operation() {
	operations := &testutils.MockCleanupOperationsStore{
		Operations: map[string]*manage.CleanupOperation{},
	}
	result := s.serve(operations, s.task())
	defer result.Body.Close()

	s.Require().Equal(http.StatusAccepted, result.StatusCode)

	response := &helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{}
	s.decode(result.Body, response)
	s.Assert().Equal(manage.CleanupTypeEvents, response.Data.CleanupType)
	s.Assert().NotEmpty(response.Data.ID)
}

// A category the scheduler or another request is already cleaning up is a conflict
// rather than a second run over the same records.
func (s *StartCleanupSuite) Test_refuses_a_cleanup_already_in_progress_with_a_conflict() {
	operations := &testutils.MockCleanupOperationsStore{
		Operations: map[string]*manage.CleanupOperation{
			"running-operation": {
				ID:          "running-operation",
				CleanupType: manage.CleanupTypeEvents,
				Status:      manage.CleanupOperationStatusRunning,
				StartedAt:   testTime.Unix(),
			},
		},
	}

	result := s.serve(operations, s.task())
	defer result.Body.Close()
	s.Assert().Equal(http.StatusConflict, result.StatusCode)
}

func (s *StartCleanupSuite) Test_reports_a_failure_to_record_the_operation() {
	operations := &testutils.MockCleanupOperationsStore{
		Operations: map[string]*manage.CleanupOperation{},
	}
	operations.SaveError = http.ErrBodyNotAllowed

	result := s.serve(operations, s.task())
	defer result.Body.Close()
	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
}

func (s *StartCleanupSuite) task() cleanup.Task {
	return cleanup.Task{
		Type:    manage.CleanupTypeEvents,
		Remove:  func(context.Context, time.Time) (int64, error) { return 0, nil },
		LogName: "eventsCleanup",
	}
}

func (s *StartCleanupSuite) serve(
	operations *testutils.MockCleanupOperationsStore,
	task cleanup.Task,
) *http.Response {
	runner := cleanup.NewRunner(
		operations,
		core.NewUUIDGenerator(),
		&testutils.MockClock{StaticTime: testTime},
		core.NewNopLogger(),
	)

	recorder := httptest.NewRecorder()
	helpersv1.StartCleanup(
		recorder,
		httptest.NewRequest("POST", "/events/cleanup", nil),
		runner,
		task,
		core.NewNopLogger(),
	)

	return recorder.Result()
}

func (s *StartCleanupSuite) decode(body io.Reader, into any) {
	data, err := io.ReadAll(body)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(data, into))
}

func TestStartCleanupSuite(t *testing.T) {
	suite.Run(t, new(StartCleanupSuite))
}
