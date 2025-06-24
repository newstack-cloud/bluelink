package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
)

type MiddlewareSuite struct {
	suite.Suite
}

func (s *MiddlewareSuite) Test_calls_next_handler_for_request_that_passes_auth_check() {
	middleware, err := NewMiddleware(
		checkerFunc(testChecker1),
		[]*mux.Route{},
	)
	s.Require().NoError(err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(BluelinkAPIKeyHeaderName, "checker-1-key")
	w := httptest.NewRecorder()
	middleware.Middleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message":"success"}`))
			},
		),
	).ServeHTTP(w, req)
	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))
	s.Assert().Equal("{\"message\":\"success\"}", w.Body.String())
}

func (s *MiddlewareSuite) Test_returns_401_unauthorised_response_for_failed_auth_check() {
	middleware, err := NewMiddleware(
		checkerFunc(testChecker1),
		[]*mux.Route{},
	)
	s.Require().NoError(err)

	req := httptest.NewRequest("GET", "/test2", nil)
	req.Header.Set(BluelinkAPIKeyHeaderName, "invalid-key")
	w := httptest.NewRecorder()
	middleware.Middleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message":"success"}`))
			},
		),
	).ServeHTTP(w, req)
	s.Assert().Equal(http.StatusUnauthorized, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))
	s.Assert().Equal("{\"message\":\"Unauthorized\"}", w.Body.String())
}

func (s *MiddlewareSuite) Test_calls_next_handler_for_route_in_exclude_list() {
	router := mux.NewRouter()
	healthRoute := router.HandleFunc(
		"/test/health",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"pong":true}`))
		},
	).Methods("GET")

	middleware, err := NewMiddleware(
		checkerFunc(testChecker1),
		[]*mux.Route{
			healthRoute,
		},
	)
	s.Require().NoError(err)

	req := httptest.NewRequest("GET", "/test/health", nil)
	// No API key is specified, but should pass any as auth check should
	// be skipped for this route.
	w := httptest.NewRecorder()
	middleware.Middleware(router).ServeHTTP(w, req)
	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))
	s.Assert().Equal("{\"pong\":true}", w.Body.String())
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
