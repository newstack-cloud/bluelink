package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// The credential a request authenticated with is discarded once the check has passed, so
// the principal is the only record of who acted. Nothing above the engine can reconstruct
// it, which is why attribution is established here even though authorisation is not.
type PrincipalSuite struct {
	suite.Suite
}

func (s *PrincipalSuite) Test_an_api_key_principal_carries_a_fingerprint_not_the_key() {
	service := NewAPIKeyService(
		&core.AuthConfig{APIKeys: []string{"a-configured-key"}},
	)
	headers := http.Header{}
	headers.Set(BluelinkAPIKeyHeaderName, "a-configured-key")

	principal, err := service.Check(context.Background(), headers)

	s.Require().NoError(err)
	s.Require().NotNil(principal)
	s.Assert().Equal(MethodAPIKey, principal.Method)
	s.Assert().NotContains(
		principal.Subject,
		"a-configured-key",
		"the principal carried the key itself, so a record naming it could be replayed",
	)
	s.Assert().Regexp(`^sha256:[0-9a-f]{16}$`, principal.Subject)
}

// Two distinct keys have to be distinguishable in a record, or attribution says only that
// some configured key was used.
func (s *PrincipalSuite) Test_api_key_fingerprints_differ_between_keys() {
	s.Assert().NotEqual(
		apiKeyFingerprint("key-one"),
		apiKeyFingerprint("key-two"),
	)
	s.Assert().Equal(
		apiKeyFingerprint("key-one"),
		apiKeyFingerprint("key-one"),
	)
}

// The middleware is the only place the principal can be attached, since the credential is
// not available to anything downstream of it.
func (s *PrincipalSuite) Test_the_middleware_carries_the_principal_on_the_request() {
	middleware, err := NewMiddleware(testChecker1, []*mux.Route{}, bpcore.NewNopLogger())
	s.Require().NoError(err)

	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(BluelinkAPIKeyHeaderName, "checker-1-key")

	var carried *Principal
	var found bool
	middleware.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			carried, found = PrincipalFromContext(r.Context())
		}),
	).ServeHTTP(httptest.NewRecorder(), request)

	s.Require().True(found, "the request reached the handler with no principal")
	s.Assert().Equal("checker-1", carried.Subject)
}

// A caller has to handle a request with no principal rather than assuming one, since a
// route authentication is excluded from never establishes it.
func (s *PrincipalSuite) Test_reports_no_principal_when_none_was_established() {
	_, found := PrincipalFromContext(context.Background())

	s.Assert().False(found)
}

func (s *PrincipalSuite) Test_reports_no_principal_for_a_nil_principal() {
	_, found := PrincipalFromContext(
		ContextWithPrincipal(context.Background(), nil),
	)

	s.Assert().False(found)
}

func TestPrincipalSuite(t *testing.T) {
	suite.Run(t, new(PrincipalSuite))
}
