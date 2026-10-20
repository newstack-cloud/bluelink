package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// An audited action has to lead to the trace of the work it did, or the trail says who
// changed something without saying what that cost or where it went wrong.
type MiddlewareTracingSuite struct {
	suite.Suite

	recorder *tracetest.SpanRecorder
	previous oteltrace.TracerProvider
}

func (s *MiddlewareTracingSuite) SetupTest() {
	s.previous = otel.GetTracerProvider()
}

func (s *MiddlewareTracingSuite) TearDownTest() {
	otel.SetTracerProvider(s.previous)
	s.recorder = nil
}

func (s *MiddlewareTracingSuite) Test_records_the_trace_the_action_was_carried_out_under() {
	s.recorder = tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(s.recorder),
	))

	record := s.recordOneAction()

	spans := s.recorder.Ended()
	s.Require().Len(spans, 1)
	s.Assert().Equal(spans[0].SpanContext().TraceID().String(), record.TraceID)
	s.Assert().Equal(spans[0].SpanContext().SpanID().String(), record.SpanID)
}

// A record carrying identifiers no backend has points at a trace that cannot be found,
// which reads as though the trace exists.
func (s *MiddlewareTracingSuite) Test_records_no_trace_where_none_was_recorded() {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	))

	record := s.recordOneAction()

	s.Assert().Empty(record.TraceID)
	s.Assert().Empty(record.SpanID)
}

// Tracing is off by default, and the trail has to carry on regardless.
func (s *MiddlewareTracingSuite) Test_records_no_trace_where_tracing_is_not_set_up() {
	otel.SetTracerProvider(noop.NewTracerProvider())

	record := s.recordOneAction()

	s.Require().Equal("instance.create", record.Action)
	s.Assert().Empty(record.TraceID)
	s.Assert().Empty(record.SpanID)
}

// The chain as the engine registers it with the request instrumentation as the entrypoint,
// so the span covers the audited action.
func (s *MiddlewareTracingSuite) recordOneAction() *Record {
	sink := &collectingSink{}

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("bluelink-deploy-engine"))
	router.Use(NewMiddleware(sink, &fixedClock{}, bpcore.NewNopLogger()).Middleware)
	router.HandleFunc(
		"/deployments/instances",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	).Methods("POST")

	request := httptest.NewRequest("POST", "/deployments/instances", nil)
	request = request.WithContext(
		auth.ContextWithPrincipal(request.Context(), &auth.Principal{
			Method:  auth.MethodAPIKey,
			Subject: "sha256:abcd",
		}),
	)
	router.ServeHTTP(httptest.NewRecorder(), request)

	s.Require().Len(sink.records, 1, "the action should have been recorded")

	return sink.records[0]
}

func TestAuditMiddlewareTracingSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTracingSuite))
}
