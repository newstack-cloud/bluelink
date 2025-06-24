// Tests for the CreateBlueprintInstance method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

func (s *ClientSuite) Test_create_blueprint_instance() {
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

	// Make a request to create a blueprint instance
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	blueprintInstance, err := client.CreateBlueprintInstance(
		context.Background(),
		payload,
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&state.InstanceState{
			InstanceID:   "test-instance-id",
			InstanceName: "test-instance-name",
			Status:       core.InstanceStatusDeploying,
		},
		blueprintInstance,
	)
}

func (s *ClientSuite) Test_create_blueprint_instance_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint instance
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	_, err = client.CreateBlueprintInstance(
		context.Background(),
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

func (s *ClientSuite) Test_create_blueprint_instance_fails_for_incorrect_input() {
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

	// Make a request to create a blueprint instance
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			// "files" is not a valid file source scheme.
			FileSourceScheme: "files",
		},
	}

	_, err = client.CreateBlueprintInstance(
		context.Background(),
		payload,
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.ClientError)
	s.Require().True(isClientErr)

	s.Assert().Equal(
		http.StatusUnprocessableEntity,
		clientErr.StatusCode,
	)
	s.Assert().Equal(
		"fileSourceScheme must be \"file\"",
		clientErr.Message,
	)
	s.Assert().Equal(
		[]*errors.ValidationError{
			{
				Location: "fileSourceScheme",
				Message:  "fileSourceScheme must be \"file\"",
				Type:     "invalid",
			},
		},
		clientErr.ValidationErrors,
	)
}

func (s *ClientSuite) Test_create_blueprint_instance_fails_due_to_invalid_json_response() {
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

	// Make a request to create a blueprint instance
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// the stub server to return an invalid JSON response.
			BlueprintFile: deserialiseErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintInstance(
		context.Background(),
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

func (s *ClientSuite) Test_create_blueprint_instance_fails_due_to_internal_server_error() {
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

	// Make a request to create a blueprint instance.
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated internal server error.
			BlueprintFile: internalServerErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintInstance(
		context.Background(),
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

func (s *ClientSuite) Test_create_blueprint_instance_fails_due_to_network_error() {
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

	// Make a request to create a blueprint instance
	payload := &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated network error by causing the server to
			// close the connection early.
			BlueprintFile: networkErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintInstance(
		context.Background(),
		payload,
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.RequestError)
	s.Require().True(isClientErr)

	expectedErrorMessage := fmt.Sprintf(
		"request error: Post \"%s%s\": EOF",
		s.deployEngineServer.URL,
		"/v1/deployments/instances",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
