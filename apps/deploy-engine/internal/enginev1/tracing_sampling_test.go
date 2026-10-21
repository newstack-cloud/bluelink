package enginev1_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
	"github.com/newstack-cloud/bluelink/libs/telemetry-otel/tracing"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
)

// Which of the engine's requests are traced in full and which follow the volume rate is
// decided by the request's method, and the method only reaches the sampler because
// otelmux puts it on the span at the moment it starts one.
//
// Driven through the real router instrumentation rather than by passing the attribute by
// hand, since the rule depends on otelmux and the tracing library naming that attribute
// identically. They agree today, both on semconv v1.43.0, and a release that moved
// either side would otherwise leave every mutating request silently thinned. It lives
// here rather than with the sampler because otelmux is the engine's choice of router
// instrumentation, and a library that took it as a test dependency would put gorilla/mux
// in the build list of every plugin importing the library.
type TracingSamplingSuite struct {
	suite.Suite
}

func (s *TracingSamplingSuite) Test_traces_a_mutating_request_at_the_operation_rate() {
	s.install(0, 1)

	s.Assert().True(
		s.routeIsRecorded("POST", "/deployments/instances"),
		"a request that starts an operation should be traced at the operation rate",
	)
}

// The polls and the lists are the volume the request rate is for, so they follow it even
// where operations are traced in full.
func (s *TracingSamplingSuite) Test_traces_a_read_request_at_the_request_rate() {
	s.install(0, 1)

	s.Assert().False(
		s.routeIsRecorded("GET", "/deployments/instances/{id}"),
		"a read should follow the request rate, not the operation rate",
	)
}

func (s *TracingSamplingSuite) install(requestRate float64, operationRate float64) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	s.T().Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
		telemetry.SetTracer(nil)
	})

	shutdown, err := tracing.Setup(
		context.Background(),
		tracing.Settings{
			OTLPEndpoint:          "http://localhost:4317",
			ServiceName:           "bluelink-deploy-engine",
			SamplingRate:          requestRate,
			OperationSamplingRate: operationRate,
		},
		&testutils.MockClock{StaticTime: time.Now()},
	)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		// Bounded, and its error ignored, because no collector is listening: the flush
		// this attempts cannot succeed and its failure says nothing about the test.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_ = shutdown(ctx)
	})
}

// Whether the span otelmux starts for a request to the given route is one a backend will
// have. The handler reads it from its own context, which is where anything the engine
// traces during the request would find it.
func (s *TracingSamplingSuite) routeIsRecorded(method string, route string) bool {
	recorded := false
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("bluelink-deploy-engine"))
	router.HandleFunc(route, func(_ http.ResponseWriter, r *http.Request) {
		recorded = tracing.SpanContextFromContext(r.Context()).Recorded()
	}).Methods(method)

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(
			method,
			strings.NewReplacer("{id}", "instance-1").Replace(route),
			nil,
		),
	)

	return recorded
}

func TestTracingSamplingSuite(t *testing.T) {
	suite.Run(t, new(TracingSamplingSuite))
}
