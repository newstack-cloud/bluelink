package auth

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
)

type Middleware struct {
	checker              Checker
	excludeRoutes        []*mux.Route
	excludeRoutePatterns []*regexp.Regexp
}

// NewMiddleware creates a new middleware carries out
// authentication checks using the provided Checker
// that uses HTTP headers as the source of authentication data.
func NewMiddleware(checker Checker, excludeRoutes []*mux.Route) (*Middleware, error) {
	excludeRoutePatterns, err := routeToRegexpPatterns(excludeRoutes)
	if err != nil {
		return nil, err
	}

	return &Middleware{
		checker:              checker,
		excludeRoutes:        excludeRoutes,
		excludeRoutePatterns: excludeRoutePatterns,
	}, nil
}

func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exclude := m.shouldBeExcluded(r)

		if !exclude {
			err := m.checker.Check(r.Context(), r.Header)
			writtenErrResponse := handleError(err, w)
			if writtenErrResponse {
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) shouldBeExcluded(r *http.Request) bool {
	exclude := false
	requestMethod := r.Method
	i := 0
	for !exclude && i < len(m.excludeRoutePatterns) {
		excludedRoute := m.excludeRoutes[i]
		// Routes will always have a method for the deploy engine,
		// so we can safely ignore the error.
		methods, _ := excludedRoute.GetMethods()
		methodMatched := false
		j := 0
		for !methodMatched && j < len(m.excludeRoutePatterns) {
			if requestMethod == methods[j] {
				methodMatched = true
			}
			j += 1
		}

		if methodMatched {
			if m.excludeRoutePatterns[i].MatchString(r.RequestURI) {
				exclude = true
			}
		}

		i += 1
	}

	return exclude
}

func handleError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*Error)
	if !ok {
		httputils.HTTPError(
			w,
			http.StatusInternalServerError,
			utils.UnexpectedErrorMessage,
		)
		return true
	}

	httputils.HTTPError(
		w,
		http.StatusUnauthorized,
		"Unauthorized",
	)
	return true
}

func routeToRegexpPatterns(routes []*mux.Route) ([]*regexp.Regexp, error) {
	var patterns []*regexp.Regexp
	for _, route := range routes {
		pathRegexp, err := route.GetPathRegexp()
		if err != nil {
			return nil, err
		}

		pattern, err := regexp.Compile(pathRegexp)
		if err != nil {
			return nil, err
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}
