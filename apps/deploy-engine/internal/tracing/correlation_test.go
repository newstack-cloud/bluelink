package tracing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// A trace shows where a request's time went and its log entries say what happened, and
// neither accounts for a deployment without the other, so every entry has to carry the
// identity of the span it was written under.
type LogCorrelationSuite struct {
	suite.Suite

	recorder *tracetest.SpanRecorder
	previous oteltrace.TracerProvider
}

func (s *LogCorrelationSuite) SetupTest() {
	s.previous = otel.GetTracerProvider()
}

func (s *LogCorrelationSuite) TearDownTest() {
	otel.SetTracerProvider(s.previous)
	s.recorder = nil
}

func (s *LogCorrelationSuite) Test_the_logged_trace_is_the_one_that_was_recorded() {
	s.recordSpans()

	entry := s.logOneLine( /* xray */ false)

	recorded := s.recorder.Ended()
	s.Require().Len(recorded, 1)
	s.Assert().Equal(recorded[0].SpanContext().TraceID().String(), entry["traceId"])
	s.Assert().Equal(recorded[0].SpanContext().SpanID().String(), entry["spanId"])
}

func (s *LogCorrelationSuite) Test_keeps_what_the_request_logger_already_carried() {
	s.recordSpans()

	entry := s.logOneLine( /* xray */ false)

	s.Assert().Equal("POST", entry["method"])
	s.Assert().Equal("/deployments/instances", entry["path"])
	s.Assert().NotEmpty(entry["requestId"])
}

// The X-Ray console will only find a trace by its own rendering of the ID, so on AWS an
// entry carrying the OpenTelemetry form alone leads nowhere.
func (s *LogCorrelationSuite) Test_adds_the_xray_form_of_the_trace_id_on_aws() {
	s.recordSpans()

	entry := s.logOneLine( /* xray */ true)

	traceID, isString := entry["traceId"].(string)
	s.Require().True(isString)
	s.Require().Len(traceID, 32)
	// Spelled out rather than taken from ToXRayTraceID, which is what produced the
	// field, so that a wrong rendering is not compared against itself.
	s.Assert().Equal("1-"+traceID[:8]+"-"+traceID[8:], entry["xrayTraceId"])
}

func (s *LogCorrelationSuite) Test_omits_the_xray_form_off_aws() {
	s.recordSpans()

	s.Assert().NotContains(s.logOneLine( /* xray */ false), "xrayTraceId")
}

// Identifiers that resolve to nothing read as though a trace exists, and send whoever
// is reading the entry looking for one.
func (s *LogCorrelationSuite) Test_adds_nothing_where_no_backend_has_the_trace() {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	))

	entry := s.logOneLine( /* xray */ true)

	s.Assert().NotContains(entry, "traceId")
	s.Assert().NotContains(entry, "spanId")
	s.Assert().NotContains(entry, "xrayTraceId")
	s.Assert().Equal("POST", entry["method"], "the request fields must survive")
}

func (s *LogCorrelationSuite) recordSpans() {
	s.recorder = tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(s.recorder),
	))
}

// The chain as the engine registers it, since what this middleware can add depends
// entirely on what has already run.
func (s *LogCorrelationSuite) logOneLine(xray bool) map[string]any {
	logCore, recorded := observer.New(zapcore.DebugLevel)
	logger := bpcore.NewLoggerFromZap(zap.New(logCore))

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("bluelink-deploy-engine"))
	router.Use(httpmw.NewRequestContextMiddleware(logger).Middleware)
	router.Use(tracing.NewLogCorrelationMiddleware(xray).Middleware)
	router.HandleFunc(
		"/deployments/instances",
		func(w http.ResponseWriter, r *http.Request) {
			httpmw.LoggerFromContext(r.Context(), bpcore.NewNopLogger()).Info("handled")
			w.WriteHeader(http.StatusOK)
		},
	).Methods("POST")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/deployments/instances", nil),
	)

	entries := recorded.All()
	s.Require().Len(entries, 1, "exactly one line should have been logged")

	return entries[0].ContextMap()
}

func TestLogCorrelationSuite(t *testing.T) {
	suite.Run(t, new(LogCorrelationSuite))
}
