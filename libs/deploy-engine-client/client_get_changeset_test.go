// Tests for the GetChangeset method in the DeployEngine client.
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
)

func (s *ClientSuite) Test_get_changeset() {
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

	changeset, err := client.GetChangeset(
		context.Background(),
		testChangesetID,
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.Changeset{
			ID:                testChangesetID,
			InstanceID:        testInstanceID,
			Destroy:           false,
			Status:            manage.ChangesetStatusChangesStaged,
			BlueprintLocation: testBlueprintLocation,
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

func (s *ClientSuite) Test_get_changeset_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	_, err = client.GetChangeset(
		context.Background(),
		testChangesetID,
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

func (s *ClientSuite) Test_get_changeset_fails_due_to_invalid_json_response() {
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

	_, err = client.GetChangeset(
		context.Background(),
		deserialiseErrorTriggerID,
	)
	s.Require().Error(err)

	deserialiseErr, isDeserialiseErr := err.(*errors.DeserialiseError)
	s.Require().True(isDeserialiseErr)

	s.Assert().Equal(
		"deserialise error: failed to decode response: unexpected EOF",
		deserialiseErr.Error(),
	)
}

func (s *ClientSuite) Test_get_changeset_fails_due_to_internal_server_error() {
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

	_, err = client.GetChangeset(
		context.Background(),
		internalServerErrorTriggerID,
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

func (s *ClientSuite) Test_get_changeset_fails_due_to_network_error() {
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

	_, err = client.GetChangeset(
		context.Background(),
		networkErrorTriggerID,
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.RequestError)
	s.Require().True(isClientErr)

	expectedErrorMessage := fmt.Sprintf(
		"request error: Get \"%s%s%s\": EOF",
		s.deployEngineServer.URL,
		"/v1/deployments/changes/",
		networkErrorTriggerID,
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
