package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// How a mutation comes to be attributed.
type MiddlewareAttributionSuite struct {
	suite.Suite
}

func (s *MiddlewareAttributionSuite) Test_records_an_action_whose_handler_panicked() {
	sink := &collectingSink{}
	middleware := NewMiddleware(sink, &fixedClock{}, bpcore.NewNopLogger())

	router := mux.NewRouter()
	router.Use(middleware.Middleware)
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		func(w http.ResponseWriter, r *http.Request) {
			panic("the handler gave up part-way through")
		},
	).Methods("POST")

	request := httptest.NewRequest("POST", "/deployments/instances/i-1/destroy", nil)
	s.Assert().Panics(func() {
		router.ServeHTTP(httptest.NewRecorder(), request)
	})

	s.Require().Len(sink.records, 1)
	record := sink.records[0]
	s.Assert().Equal("instance.destroy", record.Action)
	s.Assert().Equal("i-1", record.Target)
	s.Assert().Equal(OutcomeFailure, record.Outcome)
	s.Assert().Equal(http.StatusInternalServerError, record.StatusCode)
}

// A record carries the address the extract-IP middleware resolved, rather than one the
// audit middleware works out for itself. Which header to trust is a deployment setting,
// and deriving it twice is how the two come to disagree.
func (s *MiddlewareAttributionSuite) Test_records_the_resolved_source_address() {
	record := s.recordFor(
		httpmw.NewExtractIPMiddleware(
			httpmw.WithExtractIPMiddlewareSource(httpmw.AWSALBExtractionSource),
		).Middleware,
	)
	s.Assert().Equal("203.0.113.42", record.SourceIP)
}

// The default source ignores the forwarded header, so an engine reached directly records
// the address the connection came from rather than the one the caller asked for.
func (s *MiddlewareAttributionSuite) Test_records_the_connection_address_by_default() {
	record := s.recordFor(httpmw.NewExtractIPMiddleware().Middleware)
	s.Assert().Equal("192.0.2.1", record.SourceIP)
}

// Without that middleware there is no resolved address, and a record says nothing rather
// than guessing at one.
func (s *MiddlewareAttributionSuite) Test_records_no_source_address_when_none_was_resolved() {
	record := s.recordFor(nil)
	s.Assert().Empty(record.SourceIP)
}

func (s *MiddlewareAttributionSuite) recordFor(
	extractIP func(http.Handler) http.Handler,
) *Record {
	sink := &collectingSink{}

	router := mux.NewRouter()
	if extractIP != nil {
		router.Use(extractIP)
	}
	router.Use(NewMiddleware(sink, &fixedClock{}, bpcore.NewNopLogger()).Middleware)
	router.HandleFunc("/validations", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	request := httptest.NewRequest("POST", "/validations", nil)
	request.RemoteAddr = "192.0.2.1:54321"
	request.Header.Set("X-Forwarded-For", "203.0.113.42")

	router.ServeHTTP(httptest.NewRecorder(), request)

	s.Require().Len(sink.records, 1)

	return sink.records[0]
}

func TestMiddlewareAttributionSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareAttributionSuite))
}
