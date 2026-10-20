package auditv1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	ctrl       *Controller
	auditStore *testutils.MockAuditStore
	router     *mux.Router
}

func (s *ControllerTestSuite) SetupTest() {
	s.auditStore = testutils.NewMockAuditStore(
		map[string]*manage.AuditRecord{
			"record-1": auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
			"record-2": auditRecord("record-2", 2000, "instance.update", "user-b", manage.OutcomeSuccess),
			"record-3": auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeFailure),
		},
	)

	cleanupOperationsStore := testutils.NewMockCleanupOperationsStore(
		map[string]*manage.CleanupOperation{},
	)

	dependencies := &typesv1.Dependencies{
		AuditStore:             s.auditStore,
		CleanupOperationsStore: cleanupOperationsStore,
		CleanupRunner: cleanup.NewRunner(
			cleanupOperationsStore,
			core.NewUUIDGenerator(),
			&testutils.MockClock{StaticTime: testTime},
			core.NewNopLogger(),
		),
		IDGenerator: core.NewUUIDGenerator(),
		Clock:       &testutils.MockClock{StaticTime: testTime},
		Logger:      core.NewNopLogger(),
	}

	s.ctrl = NewController(10*time.Second, dependencies)

	router := mux.NewRouter()
	router.HandleFunc("/audit/records/cleanup", s.ctrl.CleanupAuditRecordsHandler).Methods("POST")
	router.HandleFunc("/audit/records", s.ctrl.QueryAuditRecordsHandler).Methods("GET")
	router.HandleFunc("/audit/records/{id}", s.ctrl.GetAuditRecordHandler).Methods("GET")
	s.router = router
}

func (s *ControllerTestSuite) Test_query_returns_the_trail_oldest_first() {
	response := s.queryRecords("/audit/records")
	s.Assert().Equal(
		[]string{"record-1", "record-2", "record-3"},
		recordIDs(response.Records),
	)
}

func (s *ControllerTestSuite) Test_query_narrows_the_trail_by_time_range() {
	response := s.queryRecords("/audit/records?from=2000&to=3000")
	s.Assert().Equal([]string{"record-2"}, recordIDs(response.Records))
}

func (s *ControllerTestSuite) Test_query_narrows_the_trail_by_action() {
	response := s.queryRecords("/audit/records?action=instance.destroy")
	s.Assert().Equal([]string{"record-3"}, recordIDs(response.Records))
}

func (s *ControllerTestSuite) Test_query_narrows_the_trail_by_principal() {
	response := s.queryRecords("/audit/records?principalSubject=user-a")
	s.Assert().Equal([]string{"record-1", "record-3"}, recordIDs(response.Records))
}

func (s *ControllerTestSuite) Test_query_narrows_the_trail_by_outcome() {
	response := s.queryRecords("/audit/records?outcome=failure")
	s.Assert().Equal([]string{"record-3"}, recordIDs(response.Records))
}

func (s *ControllerTestSuite) Test_query_limits_the_page() {
	response := s.queryRecords("/audit/records?limit=2")
	s.Assert().Equal([]string{"record-1", "record-2"}, recordIDs(response.Records))
}

// A bound the engine cannot read is rejected rather than ignored: answering a
// different question would have the caller read a partial trail as a whole one.
func (s *ControllerTestSuite) Test_query_rejects_a_malformed_filter() {
	for _, target := range []string{
		"/audit/records?from=yesterday",
		"/audit/records?to=-1",
		"/audit/records?limit=0",
		"/audit/records?limit=10000",
	} {
		result := s.serve("GET", target)
		s.Assert().Equal(http.StatusBadRequest, result.StatusCode, target)
		result.Body.Close()
	}
}

func (s *ControllerTestSuite) Test_get_returns_a_single_record() {
	result := s.serve("GET", "/audit/records/record-2")
	defer result.Body.Close()
	s.Require().Equal(http.StatusOK, result.StatusCode)

	record := &manage.AuditRecord{}
	s.decode(result.Body, record)
	s.Assert().Equal("instance.update", record.Action)
}

func (s *ControllerTestSuite) Test_get_reports_a_missing_record_as_not_found() {
	result := s.serve("GET", "/audit/records/missing-record")
	defer result.Body.Close()
	s.Assert().Equal(http.StatusNotFound, result.StatusCode)
}

func (s *ControllerTestSuite) Test_cleanup_starts_an_operation_for_audit_records() {
	result := s.serve("POST", "/audit/records/cleanup")
	defer result.Body.Close()
	s.Require().Equal(http.StatusAccepted, result.StatusCode)

	response := &helpersv1.AsyncOperationResponse[*manage.CleanupOperation]{}
	s.decode(result.Body, response)

	s.Assert().Equal(manage.CleanupTypeAuditRecords, response.Data.CleanupType)
	s.Assert().NotEmpty(response.Data.ID)
	s.Assert().Contains(
		[]manage.CleanupOperationStatus{
			manage.CleanupOperationStatusRunning,
			manage.CleanupOperationStatusCompleted,
		},
		response.Data.Status,
	)
}

func (s *ControllerTestSuite) Test_query_reports_a_store_failure() {
	s.auditStore.QueryError = http.ErrBodyNotAllowed

	result := s.serve("GET", "/audit/records")
	defer result.Body.Close()
	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
}

func (s *ControllerTestSuite) queryRecords(target string) *QueryAuditRecordsResponse {
	result := s.serve("GET", target)
	defer result.Body.Close()
	s.Require().Equal(http.StatusOK, result.StatusCode)

	response := &QueryAuditRecordsResponse{}
	s.decode(result.Body, response)

	return response
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

func auditRecord(
	id string,
	timestamp int64,
	action string,
	principalSubject string,
	outcome string,
) *manage.AuditRecord {
	return &manage.AuditRecord{
		ID:               id,
		Timestamp:        timestamp,
		Action:           action,
		Outcome:          outcome,
		StatusCode:       200,
		Method:           "POST",
		Path:             "/deployments/instances",
		PrincipalMethod:  "apikey",
		PrincipalSubject: principalSubject,
	}
}

func recordIDs(records []*manage.AuditRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}
	return ids
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
