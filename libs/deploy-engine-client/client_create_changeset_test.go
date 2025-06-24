// Tests for the CreateChangeset method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

const (
	testChangesetID = "test-changeset-id"
	testInstanceID  = "test-instance-id"
)

func (s *ClientSuite) Test_create_changeset() {
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

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	changeset, err := client.CreateChangeset(
		context.Background(),
		payload,
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.Changeset{
			ID:                testChangesetID,
			InstanceID:        testInstanceID,
			BlueprintLocation: testBlueprintLocation,
			Status:            manage.ChangesetStatusStarting,
			Destroy:           false,
			Changes: &changes.BlueprintChanges{
				ResourceChanges: map[string]provider.Changes{
					"resource-1": {
						NewFields: []provider.FieldChange{
							{
								FieldPath: "spec.name",
								PrevValue: core.MappingNodeFromString("old-name"),
								NewValue:  core.MappingNodeFromString("new-name"),
							},
						},
					},
				},
				RemovedResources: []string{"resource-2", "resource-3"},
			},
			Created: testTime.Unix(),
		},
		changeset,
	)
}

func (s *ClientSuite) Test_create_changeset_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	_, err = client.CreateChangeset(
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

func (s *ClientSuite) Test_create_changeset_fails_for_incorrect_input() {
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

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			// "files" is not a valid file source scheme.
			FileSourceScheme: "files",
		},
	}

	_, err = client.CreateChangeset(
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

func (s *ClientSuite) Test_create_changeset_fails_due_to_invalid_json_response() {
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

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// the stub server to return an invalid JSON response.
			BlueprintFile: deserialiseErrorTrigger,
		},
	}

	_, err = client.CreateChangeset(
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

func (s *ClientSuite) Test_create_changeset_fails_due_to_internal_server_error() {
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

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated internal server error.
			BlueprintFile: internalServerErrorTrigger,
		},
	}

	_, err = client.CreateChangeset(
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

func (s *ClientSuite) Test_create_changeset_fails_due_to_network_error() {
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

	// Make a request to create a change set
	payload := &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated network error by causing the server to
			// close the connection early.
			BlueprintFile: networkErrorTrigger,
		},
	}

	_, err = client.CreateChangeset(
		context.Background(),
		payload,
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.RequestError)
	s.Require().True(isClientErr)

	expectedErrorMessage := fmt.Sprintf(
		"request error: Post \"%s%s\": EOF",
		s.deployEngineServer.URL,
		"/v1/deployments/changes",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
