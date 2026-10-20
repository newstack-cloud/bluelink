package httpmw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// What a log line says about the request it was written while handling, and the
// correlation ID that ties those lines to the request's audit record.
type RequestContextMiddlewareSuite struct {
	suite.Suite
}

func (s *RequestContextMiddlewareSuite) Test_enriches_the_logger_with_the_request() {
	entry := s.logOneLine(func(r *http.Request) {
		r.Host = "engine.example.com"
		r.Header.Set("User-Agent", "bluelink-cli/1.0")
		r.Header.Set(httpmw.RequestIDHeader, "request-from-the-caller")
	})

	s.Assert().Equal("request-from-the-caller", entry["requestId"])
	s.Assert().Equal("POST", entry["method"])
	s.Assert().Equal("/deployments/instances", entry["path"])
	s.Assert().Equal("engine.example.com", entry["host"])
	s.Assert().Equal("bluelink-cli/1.0", entry["userAgent"])
}

// A caller's own ID is kept so that a request can be followed across the components it
// passed through.
func (s *RequestContextMiddlewareSuite) Test_keeps_a_correlation_id_the_caller_supplied() {
	var resolved string
	s.serve(
		func(r *http.Request) { r.Header.Set(httpmw.RequestIDHeader, "caller-id") },
		func(r *http.Request) { resolved, _ = httpmw.RequestIDFromContext(r.Context()) },
	)
	s.Assert().Equal("caller-id", resolved)
}

func (s *RequestContextMiddlewareSuite) Test_mints_a_correlation_id_when_none_was_supplied() {
	var resolved string
	var found bool
	s.serve(nil, func(r *http.Request) {
		resolved, found = httpmw.RequestIDFromContext(r.Context())
	})
	s.Require().True(found)
	s.Assert().NotEmpty(resolved)
}

// The logger and the request ID have to agree, or a log line points at a record that
// does not exist.
func (s *RequestContextMiddlewareSuite) Test_the_logged_id_is_the_one_in_the_context() {
	var fromContext string
	entry := s.logOneLineWith(nil, func(r *http.Request) {
		fromContext, _ = httpmw.RequestIDFromContext(r.Context())
		httpmw.LoggerFromContext(r.Context(), bpcore.NewNopLogger()).Info("handled")
	})
	s.Require().NotEmpty(fromContext)
	s.Assert().Equal(fromContext, entry["requestId"])
}

// Later middleware adds what only it knows, without this package having to know about
// it.
func (s *RequestContextMiddlewareSuite) Test_fields_added_later_appear_on_the_logger() {
	entry := s.logOneLineWith(nil, func(r *http.Request) {
		ctx := httpmw.WithLoggerFields(
			r.Context(),
			bpcore.StringLogField("userId", "someone@example.com"),
		)
		httpmw.LoggerFromContext(ctx, bpcore.NewNopLogger()).Info("handled")
	})
	s.Assert().Equal("someone@example.com", entry["userId"])
	s.Assert().Equal("POST", entry["method"], "the request fields must survive")
}

// A caller should never have to decide whether logging is worth attempting.
func (s *RequestContextMiddlewareSuite) Test_falls_back_when_the_middleware_has_not_run() {
	request := httptest.NewRequest("POST", "/deployments/instances", nil)
	fallback := bpcore.NewNopLogger()

	s.Assert().Same(fallback, httpmw.LoggerFromContext(request.Context(), fallback))
	s.Assert().Equal(
		request.Context(),
		httpmw.WithLoggerFields(request.Context(), bpcore.StringLogField("a", "b")),
		"adding fields without the middleware should leave the context alone",
	)
}

func (s *RequestContextMiddlewareSuite) logOneLine(
	prepare func(*http.Request),
) map[string]any {
	return s.logOneLineWith(prepare, func(r *http.Request) {
		httpmw.LoggerFromContext(r.Context(), bpcore.NewNopLogger()).Info("handled")
	})
}

func (s *RequestContextMiddlewareSuite) logOneLineWith(
	prepare func(*http.Request),
	handle func(*http.Request),
) map[string]any {
	recorded, logger := observedLogger()

	handler := httpmw.NewRequestContextMiddleware(logger).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handle(r)
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest("POST", "/deployments/instances", nil)
	if prepare != nil {
		prepare(request)
	}
	handler.ServeHTTP(httptest.NewRecorder(), request)

	entries := recorded.All()
	s.Require().Len(entries, 1, "exactly one line should have been logged")

	return entries[0].ContextMap()
}

func (s *RequestContextMiddlewareSuite) serve(
	prepare func(*http.Request),
	handle func(*http.Request),
) {
	_, logger := observedLogger()
	handler := httpmw.NewRequestContextMiddleware(logger).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handle(r)
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest("POST", "/deployments/instances", nil)
	if prepare != nil {
		prepare(request)
	}
	handler.ServeHTTP(httptest.NewRecorder(), request)
}

func observedLogger() (*observer.ObservedLogs, bpcore.Logger) {
	core, recorded := observer.New(zapcore.DebugLevel)

	return recorded, bpcore.NewLoggerFromZap(zap.New(core))
}

func TestRequestContextMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(RequestContextMiddlewareSuite))
}
