package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
)

type CredentialsHelperTestSuite struct {
	tokenServer       *httptest.Server
	credentialsHelper CredentialsHelper
	suite.Suite
}

func (s *CredentialsHelperTestSuite) SetupTest() {
	tokenServer, err := testutils.CreateOAuthServer(
		testClientID,
		testClientSecret,
		"oauth2",
	)
	s.Require().NoError(err)
	s.tokenServer = tokenServer
	s.credentialsHelper = NewCredentialsHelper(
		&clientcredentials.Config{
			TokenURL: fmt.Sprintf(
				"%s/oauth2/v1/token",
				s.tokenServer.URL,
			),
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
		},
		http.DefaultClient,
		context.Background(),
	)
}

func (s *CredentialsHelperTestSuite) TearDownTest() {
	s.tokenServer.Close()
}

func (s *CredentialsHelperTestSuite) Test_retrieves_access_token() {
	accessToken, err := s.credentialsHelper.GetAccessToken()
	s.Require().NoError(err)
	s.Assert().Equal("test-token-1", accessToken)
}

func TestCredentialsHelperTestSuite(t *testing.T) {
	suite.Run(t, new(CredentialsHelperTestSuite))
}
