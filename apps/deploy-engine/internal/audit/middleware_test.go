package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Tests the requirements of the middleware which is that every action
// capable of changing something is recorded.
type MiddlewareSuite struct {
	suite.Suite
}

func (s *MiddlewareSuite) Test_records_a_named_mutating_action() {
	sink := &collectingSink{}
	s.serve(
		sink,
		"POST",
		"/deployments/instances/{id}/destroy",
		"/deployments/instances/i-1/destroy",
		http.StatusOK,
	)

	s.Require().Len(sink.records, 1)
	record := sink.records[0]
	s.Assert().Equal("instance.destroy", record.Action)
	s.Assert().Equal("i-1", record.Target)
	s.Assert().Equal(OutcomeSuccess, record.Outcome)
	s.Assert().Equal("apikey", record.PrincipalMethod)
	s.Assert().Equal("sha256:abcd", record.PrincipalSubject)
}

// A mutating route added without a name here still produces a record, so the trail is never
// silently missing an action.
func (s *MiddlewareSuite) Test_records_an_unnamed_mutating_route_under_its_path() {
	sink := &collectingSink{}
	s.serve(sink, "POST", "/deployments/something-new", "/deployments/something-new", http.StatusOK)

	s.Require().Len(sink.records, 1)
	s.Assert().Equal("POST /deployments/something-new", sink.records[0].Action)
}

// A trail of reads would add noise to the audit trail, and a read cannot change anything.
func (s *MiddlewareSuite) Test_does_not_record_a_read() {
	sink := &collectingSink{}
	s.serve(sink, "GET", "/deployments/instances/{id}", "/deployments/instances/i-1", http.StatusOK)

	s.Assert().Empty(sink.records)
}

// An action that was refused still happened as an attempt, and the status is what says it
// did not take effect.
func (s *MiddlewareSuite) Test_records_a_refused_action_as_a_failure() {
	sink := &collectingSink{}
	s.serve(sink, "POST", "/deployments/instances", "/deployments/instances", http.StatusConflict)

	s.Require().Len(sink.records, 1)
	s.Assert().Equal(OutcomeFailure, sink.records[0].Outcome)
	s.Assert().Equal(http.StatusConflict, sink.records[0].StatusCode)
}

// The action has already happened by the time the record is written, so a sink that cannot
// write must not turn a completed action into a failed request.
func (s *MiddlewareSuite) Test_a_failing_sink_does_not_fail_the_request() {
	recorder := s.serve(&failingSink{}, "POST", "/events/cleanup", "/events/cleanup", http.StatusOK)

	s.Assert().Equal(http.StatusOK, recorder.Code)
}

func (s *MiddlewareSuite) serve(
	sink Sink,
	method string,
	template string,
	path string,
	status int,
) *httptest.ResponseRecorder {
	middleware := NewMiddleware(sink, &fixedClock{}, bpcore.NewNopLogger())

	router := mux.NewRouter()
	router.Use(middleware.Middleware)
	router.HandleFunc(template, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}).Methods(method)

	request := httptest.NewRequest(method, path, nil)
	request = request.WithContext(
		auth.ContextWithPrincipal(request.Context(), &auth.Principal{
			Method:  auth.MethodAPIKey,
			Subject: "sha256:abcd",
		}),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

type collectingSink struct {
	records []*Record
}

func (s *collectingSink) Write(_ context.Context, record *Record) error {
	s.records = append(s.records, record)
	return nil
}

func (s *collectingSink) Close() error { return nil }

type failingSink struct{}

func (s *failingSink) Write(_ context.Context, _ *Record) error {
	return http.ErrBodyNotAllowed
}

func (s *failingSink) Close() error { return nil }

type fixedClock struct{}

func (c *fixedClock) Now() time.Time {
	return time.Unix(1727895652, 0)
}

func (c *fixedClock) Since(t time.Time) time.Duration {
	return 0
}

func TestAuditMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
