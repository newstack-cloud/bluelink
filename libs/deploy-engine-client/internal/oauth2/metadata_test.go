package oauth2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/stretchr/testify/suite"
)

type MetadataHelperTestSuite struct {
	oidcProviderServer   *httptest.Server
	oauth2ProviderServer *httptest.Server
	oidcMetadataHelper   MetadataHelper
	oauth2MetadataHelper MetadataHelper
	suite.Suite
}

func (s *MetadataHelperTestSuite) SetupTest() {
	oidcProviderServer, err := testutils.CreateOAuthServer(
		/* clientID */ "",
		/* clientSecret */ "",
		"oidc",
	)
	s.Require().NoError(err)
	s.oidcProviderServer = oidcProviderServer
	oauth2ProviderServer, err := testutils.CreateOAuthServer(
		/* clientID */ "",
		/* clientSecret */ "",
		"oauth2",
	)
	s.Require().NoError(err)
	s.oauth2ProviderServer = oauth2ProviderServer

	s.oidcMetadataHelper = NewMetadataHelper(
		s.oidcProviderServer.URL,
		http.DefaultClient,
		core.NewNopLogger(),
	)
	s.oauth2MetadataHelper = NewMetadataHelper(
		s.oauth2ProviderServer.URL,
		http.DefaultClient,
		core.NewNopLogger(),
	)
}

func (s *MetadataHelperTestSuite) TearDownTest() {
	s.oidcProviderServer.Close()
	s.oauth2ProviderServer.Close()
}

func (s *MetadataHelperTestSuite) Test_get_token_endpoint_from_oidc_discovery_doc() {
	tokenEndpoint, err := s.oidcMetadataHelper.GetTokenEndpoint()
	s.Require().NoError(err)

	s.Assert().Equal(
		fmt.Sprintf("%s/oauth2/v1/token", s.oidcProviderServer.URL),
		tokenEndpoint,
	)
}

func (s *MetadataHelperTestSuite) Test_get_token_endpoint_from_oauth2_discovery_doc() {
	tokenEndpoint, err := s.oauth2MetadataHelper.GetTokenEndpoint()
	s.Require().NoError(err)

	s.Assert().Equal(
		fmt.Sprintf("%s/oauth2/v1/token", s.oauth2ProviderServer.URL),
		tokenEndpoint,
	)
}

func TestMetadataHelperTestSuite(t *testing.T) {
	suite.Run(t, new(MetadataHelperTestSuite))
}
