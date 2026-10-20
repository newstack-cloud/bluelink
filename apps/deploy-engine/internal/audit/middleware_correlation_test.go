package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// A record and the log lines written while handling the same request have to carry the
// same correlation ID. Two consumers each minting one does not correlate anything.
type MiddlewareCorrelationSuite struct {
	suite.Suite
}

func (s *MiddlewareCorrelationSuite) Test_a_record_and_its_log_lines_share_one_correlation_id() {
	record, entry := s.serve(nil)

	s.Require().NotEmpty(record.RequestID)
	s.Assert().Equal(record.RequestID, entry["requestId"])
}

func (s *MiddlewareCorrelationSuite) Test_a_correlation_id_the_caller_supplied_is_used_by_both() {
	record, entry := s.serve(func(r *http.Request) {
		r.Header.Set(httpmw.RequestIDHeader, "id-from-the-caller")
	})

	s.Assert().Equal("id-from-the-caller", record.RequestID)
	s.Assert().Equal("id-from-the-caller", entry["requestId"])
}

func (s *MiddlewareCorrelationSuite) serve(
	prepare func(*http.Request),
) (*Record, map[string]any) {
	observed, recorded := observer.New(zapcore.DebugLevel)
	logger := bpcore.NewLoggerFromZap(zap.New(observed))
	sink := &collectingSink{}

	router := mux.NewRouter()
	router.Use(httpmw.NewExtractIPMiddleware().Middleware)
	router.Use(httpmw.NewRequestContextMiddleware(logger).Middleware)
	router.Use(NewMiddleware(sink, &fixedClock{}, logger).Middleware)
	router.HandleFunc("/validations", func(w http.ResponseWriter, r *http.Request) {
		httpmw.LoggerFromContext(r.Context(), logger).Info("handled a request")
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	request := httptest.NewRequest("POST", "/validations", nil)
	request.RemoteAddr = "10.0.0.1:54321"
	if prepare != nil {
		prepare(request)
	}
	router.ServeHTTP(httptest.NewRecorder(), request)

	s.Require().Len(sink.records, 1)
	entries := recorded.All()
	s.Require().Len(entries, 1)

	return sink.records[0], entries[0].ContextMap()
}

func TestMiddlewareCorrelationSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareCorrelationSuite))
}
