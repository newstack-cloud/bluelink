// Tests for the DestroyBlueprintInstance method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/errors"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/internal/testutils"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/types"
)

func (s *ClientSuite) Test_destroy_blueprint_instance() {
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

	payload := &types.DestroyBlueprintInstancePayload{
		ChangeSetID: testChangesetID,
	}

	blueprintInstance, err := client.DestroyBlueprintInstance(
		context.Background(),
		"test-instance-100",
		payload,
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&state.InstanceState{
			InstanceID:   "test-instance-100",
			InstanceName: "test-instance-name",
			Status:       core.InstanceStatusDestroying,
		},
		blueprintInstance,
	)
}

func (s *ClientSuite) Test_destroy_blueprint_instance_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	payload := &types.DestroyBlueprintInstancePayload{
		ChangeSetID: testChangesetID,
	}

	_, err = client.DestroyBlueprintInstance(
		context.Background(),
		"test-instance-100",
		payload,
	)
	s.Require().Error(err)

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

func (s *ClientSuite) Test_destroy_blueprint_instance_fails_due_to_invalid_json_response() {
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

	payload := &types.DestroyBlueprintInstancePayload{
		ChangeSetID: testChangesetID,
	}

	_, err = client.DestroyBlueprintInstance(
		context.Background(),
		deserialiseErrorTriggerID,
		payload,
	)
	s.Require().Error(err)

	deserialiseErr, isDeserialiseErr := err.(*errors.DeserialiseError)
	s.Require().True(isDeserialiseErr)

	s.Assert().Equal(
		"deserialise error: failed to decode response: unexpected EOF",
		deserialiseErr.Error(),
	)
}

func (s *ClientSuite) Test_destroy_blueprint_instance_fails_due_to_internal_server_error() {
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

	payload := &types.DestroyBlueprintInstancePayload{
		ChangeSetID: testChangesetID,
	}

	_, err = client.DestroyBlueprintInstance(
		context.Background(),
		internalServerErrorTriggerID,
		payload,
	)
	s.Require().Error(err)

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

func (s *ClientSuite) Test_destroy_blueprint_instance_fails_due_to_network_error() {
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

	payload := &types.DestroyBlueprintInstancePayload{
		ChangeSetID: testChangesetID,
	}

	_, err = client.DestroyBlueprintInstance(
		context.Background(),
		networkErrorTriggerID,
		payload,
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.RequestError)
	s.Require().True(isClientErr)

	expectedErrorMessage := fmt.Sprintf(
		"request error: Post \"%s%s%s%s\": EOF",
		s.deployEngineServer.URL,
		"/v1/deployments/instances/",
		networkErrorTriggerID,
		"/destroy",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
