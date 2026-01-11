// Tests for the CleanupEvents method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
)

func (s *ClientSuite) Test_cleanup_events() {
	// Create a new client with OAuth2.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			TokenEndpoint: fmt.Sprintf(
				"%s/oauth2/v1/token",
				s.oauthServer.URL,
			),
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
		}),
	)
	s.Require().NoError(err)

	operation, err := client.CleanupEvents(context.Background())
	s.Require().NoError(err)

	s.Assert().Equal("test-cleanup-operation-id", operation.ID)
	s.Assert().Equal(manage.CleanupTypeEvents, operation.CleanupType)
	s.Assert().Equal(manage.CleanupOperationStatusRunning, operation.Status)
}

func (s *ClientSuite) Test_cleanup_events_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	_, err = client.CleanupEvents(context.Background())
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.ClientError)
	s.Require().True(isClientErr)

	s.Assert().Equal(http.StatusUnauthorized, clientErr.StatusCode)
	s.Assert().Equal("Unauthorized", clientErr.Message)
}

func (s *ClientSuite) Test_get_cleanup_operation_events() {
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			TokenEndpoint: fmt.Sprintf(
				"%s/oauth2/v1/token",
				s.oauthServer.URL,
			),
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
		}),
	)
	s.Require().NoError(err)

	operation, err := client.GetCleanupOperation(
		context.Background(),
		manage.CleanupTypeEvents,
		"test-operation-id",
	)
	s.Require().NoError(err)

	s.Assert().Equal("test-operation-id", operation.ID)
	s.Assert().Equal(manage.CleanupTypeEvents, operation.CleanupType)
	s.Assert().Equal(manage.CleanupOperationStatusCompleted, operation.Status)
	s.Assert().Equal(int64(42), operation.ItemsDeleted)
}
