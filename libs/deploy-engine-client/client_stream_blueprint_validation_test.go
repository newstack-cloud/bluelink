// Tests for the StreamBlueprintValidationEvents method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

func (s *ClientSuite) Test_stream_blueprint_validation() {
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

	streamTo := make(chan types.BlueprintValidationEvent)
	errChan := make(chan error)
	err = client.StreamBlueprintValidationEvents(
		context.Background(),
		testValidationID,
		streamTo,
		errChan,
	)
	s.Require().NoError(err)

	collected := []types.BlueprintValidationEvent{}
	channelOpen := true
	for channelOpen {
		select {
		case event, ok := <-streamTo:
			channelOpen = ok
			if channelOpen {
				collected = append(collected, event)
				s.Require().NotNil(event)
			}
		case <-time.After(5 * time.Second):
			s.Fail("Timed out waiting for events")
		}
	}

	s.Assert().Equal(
		stubBlueprintValidationEvents,
		collected,
	)
}

func (s *ClientSuite) Test_stream_blueprint_validation_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	streamTo := make(chan types.BlueprintValidationEvent)
	errChan := make(chan error)
	err = client.StreamBlueprintValidationEvents(
		context.Background(),
		testValidationID,
		streamTo,
		errChan,
	)
	s.Require().NoError(err)

	// Due to the r3labs/sse/v2 library not having a straight forward public API
	// for error handling, the best we can do is to provide an interface where
	// the caller provides an error channel to be able to handle HTTP error responses.
	select {
	case <-time.After(5 * time.Second):
		s.Fail("Timed out waiting for client error")
	case err = <-errChan:
	}

	clientErr, isClientErr := err.(*errors.ClientError)
	s.Require().True(isClientErr)

	s.Assert().Equal(
		http.StatusUnauthorized,
		clientErr.StatusCode,
	)
	s.Assert().Equal(
		"Unauthorized",
		clientErr.Message,
	)
}

func (s *ClientSuite) Test_stream_blueprint_validation_fails_due_to_internal_server_error() {
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
		// Override the default HTTP transport to opt out of retry behaviour.
		WithClientHTTPRoundTripper(testutils.CreateDefaultTransport),
	)
	s.Require().NoError(err)

	streamTo := make(chan types.BlueprintValidationEvent)
	errChan := make(chan error)
	err = client.StreamBlueprintValidationEvents(
		context.Background(),
		internalServerErrorTriggerID,
		streamTo,
		errChan,
	)
	s.Require().NoError(err)

	// Due to the r3labs/sse/v2 library not having a straight forward public API
	// for error handling, the best we can do is to provide an interface where
	// the caller provides an error channel to be able to handle HTTP error responses.
	select {
	case <-time.After(5 * time.Second):
		s.Fail("Timed out waiting for client error")
	case err = <-errChan:
	}

	clientErr, isClientErr := err.(*errors.ClientError)
	s.Require().True(isClientErr)

	s.Assert().Equal(
		http.StatusInternalServerError,
		clientErr.StatusCode,
	)
	s.Assert().Equal(
		"an unexpected error occurred",
		clientErr.Message,
	)
}
