package statev1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Saturday, 3 May 2025 14:27:22 UTC
var testTime = time.Unix(1746282442, 0).UTC()

type ControllerTestSuite struct {
	suite.Suite
	housekeeping *recordingHousekeeping
	router       *mux.Router
}

func (s *ControllerTestSuite) SetupTest() {
	s.housekeeping = &recordingHousekeeping{deleted: 3}
	cleanupOperationsStore := testutils.NewMockCleanupOperationsStore(
		map[string]*manage.CleanupOperation{},
	)
	runner := cleanup.NewRunner(
		cleanupOperationsStore,
		core.NewUUIDGenerator(),
		&testutils.MockClock{StaticTime: testTime},
		core.NewNopLogger(),
	)
	ctrl := NewController(&typesv1.Dependencies{
		StateHousekeeping:      s.housekeeping,
		CleanupOperationsStore: cleanupOperationsStore,
		CleanupRunner:          runner,
		Logger:                 core.NewNopLogger(),
	})

	router := mux.NewRouter()
	router.HandleFunc("/state/housekeeping", ctrl.HousekeepingHandler).Methods("POST")
	router.HandleFunc("/state/housekeeping/{id}", ctrl.GetCleanupStatusHandler).Methods("GET")
	s.router = router
}

func (s *ControllerTestSuite) Test_housekeeping_runs_the_backends_housekeeping_and_reports_what_it_deleted() {
	result := s.serve("POST", "/state/housekeeping")
	defer result.Body.Close()
	s.Require().Equal(http.StatusAccepted, result.StatusCode)

	started := &helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{}
	s.decode(result.Body, started)
	s.Assert().Equal(manage.CleanupTypeStateHousekeeping, started.Data.CleanupType)

	operation := &manage.CleanupOperation{}
	s.Require().Eventually(func() bool {
		*operation = *s.operationStatus(started.Data.ID)
		return operation.Status == manage.CleanupOperationStatusCompleted
	}, 5*time.Second, 10*time.Millisecond)

	s.Assert().Equal(int64(3), operation.ItemsDeleted)
	s.Assert().Equal(1, s.housekeeping.runs())
}

func (s *ControllerTestSuite) Test_status_reports_a_missing_operation_as_not_found() {
	result := s.serve("GET", "/state/housekeeping/missing-operation")
	defer result.Body.Close()
	s.Assert().Equal(http.StatusNotFound, result.StatusCode)
}

func (s *ControllerTestSuite) operationStatus(operationID string) *manage.CleanupOperation {
	result := s.serve("GET", "/state/housekeeping/"+operationID)
	defer result.Body.Close()
	s.Require().Equal(http.StatusOK, result.StatusCode)

	operation := &manage.CleanupOperation{}
	s.decode(result.Body, operation)
	return operation
}

func (s *ControllerTestSuite) serve(method string, target string) *http.Response {
	request := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, request)

	return recorder.Result()
}

func (s *ControllerTestSuite) decode(body io.Reader, into any) {
	data, err := io.ReadAll(body)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(data, into))
}

type recordingHousekeeping struct {
	mu       sync.Mutex
	runCount int
	deleted  int64
}

func (h *recordingHousekeeping) Run(context.Context) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.runCount += 1
	return h.deleted, nil
}

func (h *recordingHousekeeping) runs() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runCount
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
