package auth

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

type Middleware struct {
	checker              Checker
	excludeRoutes        []*mux.Route
	excludeRoutePatterns []*regexp.Regexp
	logger               core.Logger
}

// NewMiddleware creates a new middleware carries out
// authentication checks using the provided Checker
// that uses HTTP headers as the source of authentication data.
func NewMiddleware(
	checker Checker,
	excludeRoutes []*mux.Route,
	logger core.Logger,
) (*Middleware, error) {
	excludeRoutePatterns, err := routeToRegexpPatterns(excludeRoutes)
	if err != nil {
		return nil, err
	}

	return &Middleware{
		checker:              checker,
		excludeRoutes:        excludeRoutes,
		excludeRoutePatterns: excludeRoutePatterns,
		logger:               logger,
	}, nil
}

func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exclude := m.shouldBeExcluded(r)

		if !exclude {
			principal, err := m.checker.Check(r.Context(), r.Header)
			if err != nil {
				// The only record of a request that got no further. It changed nothing,
				// so it has no place in the audit trail, but where repeated refusals are
				// coming from is exactly what an operator needs.
				httpmw.LoggerFromContext(r.Context(), m.logger).Warn(
					"refused a request that failed to authenticate",
					core.ErrorLogField("error", err),
				)
			}
			writtenErrResponse := handleError(err, w)
			if writtenErrResponse {
				return
			}

			// Carried on the request from here, since the credential it was established
			// from is not available to anything downstream.
			ctx := ContextWithPrincipal(r.Context(), principal)
			// Added to the request-scoped logger as well, so that every line logged
			// while handling the request says who it was handled for.
			r = r.WithContext(httpmw.WithLoggerFields(ctx, principalLogFields(principal)...))
		}

		next.ServeHTTP(w, r)
	})
}

// How a principal appears in logs.
//
// userId is the end user where a fronting service asserted one and the authenticated
// principal otherwise, since that is the field an operator filters on when asking what
// somebody did. The principal is kept alongside it rather than replaced, so a line still
// says which credential acted.
func principalLogFields(principal *Principal) []core.LogField {
	if principal == nil {
		return nil
	}

	userID := principal.OnBehalfOf
	if userID == "" {
		userID = principal.Subject
	}

	return []core.LogField{
		core.StringLogField("userId", userID),
		core.StringLogField("principalMethod", principal.Method),
		core.StringLogField("principalSubject", principal.Subject),
		core.StringLogField("onBehalfOf", principal.OnBehalfOf),
	}
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
