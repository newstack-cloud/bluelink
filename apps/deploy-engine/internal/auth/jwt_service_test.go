package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/stretchr/testify/suite"
)

const (
	testAudience = "deploy-engine-app-client-id"
)

type JWTServiceSuite struct {
	suite.Suite
	serverOAuth2 *httptest.Server
	serverOIDC   *httptest.Server
	privateKey   jwk.Key
}

func (s *JWTServiceSuite) SetupSuite() {
	var err error
	s.serverOAuth2, err = newMetadataTestServer("oauth2")
	s.Require().NoError(err)
	s.serverOIDC, err = newMetadataTestServer("oidc")
	s.Require().NoError(err)

	privateKey, err := loadPrivateKey("jwk_full")
	s.Require().NoError(err)
	s.privateKey = privateKey
}

func (s *JWTServiceSuite) TearDownSuite() {
	s.serverOAuth2.Close()
	s.serverOIDC.Close()
}

func (s *JWTServiceSuite) Test_verifies_valid_jwt_from_oauth2_provider() {
	config := &core.AuthConfig{
		JWTIssuer:             stripScheme(s.serverOAuth2.URL),
		JWTIssuerSecure:       false,
		JWTAudience:           testAudience,
		JWTSignatureAlgorithm: "RS256",
	}

	service, err := LoadJWTService(config)
	s.Require().NoError(err)

	token, err := createToken(
		s.privateKey,
		"user1",
		stripScheme(s.serverOAuth2.URL),
		[]string{testAudience},
		"test-key-1",
		map[string]any{},
	)
	s.Require().NoError(err)

	headers := make(http.Header)
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().NoError(err)
}

func (s *JWTServiceSuite) Test_verifies_valid_jwt_from_oidc_provider() {
	config := &core.AuthConfig{
		JWTIssuer:             stripScheme(s.serverOIDC.URL),
		JWTIssuerSecure:       false,
		JWTAudience:           testAudience,
		JWTSignatureAlgorithm: "RS256",
	}

	service, err := LoadJWTService(config)
	s.Require().NoError(err)

	token, err := createToken(
		s.privateKey,
		"user1",
		stripScheme(s.serverOIDC.URL),
		[]string{testAudience},
		"test-key-1",
		map[string]any{},
	)
	s.Require().NoError(err)

	headers := make(http.Header)
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().NoError(err)
}

func (s *JWTServiceSuite) Test_fails_for_invalid_token() {
	config := &core.AuthConfig{
		JWTIssuer:             stripScheme(s.serverOIDC.URL),
		JWTIssuerSecure:       false,
		JWTAudience:           testAudience,
		JWTSignatureAlgorithm: "RS256",
	}

	service, err := LoadJWTService(config)
	s.Require().NoError(err)

	token, err := createToken(
		s.privateKey,
		"user1",
		"http://invalid-issuer",
		[]string{testAudience},
		"test-key-1",
		map[string]any{},
	)
	s.Require().NoError(err)

	headers := make(http.Header)
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().Error(err)
	authErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Contains(authErr.ChildErr.Error(), "invalid issuer claim (iss)")
}

func (s *JWTServiceSuite) Test_fails_for_missing_bearer_token_header() {
	config := &core.AuthConfig{
		JWTIssuer:             stripScheme(s.serverOIDC.URL),
		JWTIssuerSecure:       false,
		JWTAudience:           testAudience,
		JWTSignatureAlgorithm: "RS256",
	}

	service, err := LoadJWTService(config)
	s.Require().NoError(err)

	headers := make(http.Header)
	// Missing the Authorization header

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().Error(err)
	authErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Contains(authErr.ChildErr.Error(), "missing Authorization header")
}

func (s *JWTServiceSuite) Test_fails_for_missing_bearer_prefix_in_authorization_header() {
	config := &core.AuthConfig{
		JWTIssuer:             stripScheme(s.serverOIDC.URL),
		JWTIssuerSecure:       false,
		JWTAudience:           testAudience,
		JWTSignatureAlgorithm: "RS256",
	}

	service, err := LoadJWTService(config)
	s.Require().NoError(err)

	token, err := createToken(
		s.privateKey,
		"user1",
		stripScheme(s.serverOIDC.URL),
		[]string{testAudience},
		"test-key-1",
		map[string]any{},
	)
	s.Require().NoError(err)

	headers := make(http.Header)
	// Missing the "Bearer " prefix.
	headers.Set("Authorization", token)

	err = service.Check(
		context.Background(),
		headers,
	)
	s.Require().Error(err)
	authErr, ok := err.(*Error)
	s.Require().True(ok)
	s.Assert().Contains(
		authErr.ChildErr.Error(),
		"invalid Authorization header format",
	)
}

func newMetadataTestServer(authType string) (*httptest.Server, error) {
	mux := http.NewServeMux()

	discoveryDocument := determineDiscoveryDocument(authType)

	jwksData, err := os.ReadFile("__testdata/jwt/jwks_public.json")
	if err != nil {
		return nil, err
	}

	metadata, err := os.ReadFile(
		fmt.Sprintf("__testdata/jwt/%s", discoveryDocument),
	)
	if err != nil {
		return nil, err
	}

	var serverURL string

	discoveryEndpoint := fmt.Sprintf("/.well-known/%s", discoveryDocument)
	mux.HandleFunc(discoveryEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(replaceServerURL(metadata, serverURL))
	})

	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jwksData)
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	return server, nil
}

func determineDiscoveryDocument(authType string) string {
	if authType == "oidc" {
		return "openid-configuration"
	}
	return "oauth-authorization-server"
}

func stripScheme(url string) string {
	if strings.HasPrefix(url, "https://") {
		return url[8:]
	}

	if strings.HasPrefix(url, "http://") {
		return url[7:]
	}

	return url
}

func replaceServerURL(data []byte, serverURL string) []byte {
	return []byte(strings.ReplaceAll(string(data), "{server}", serverURL))
}

func TestJWTServiceSuite(t *testing.T) {
	suite.Run(t, new(JWTServiceSuite))
}
