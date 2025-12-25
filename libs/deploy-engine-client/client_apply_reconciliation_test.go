// Tests for the ApplyReconciliation method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

func (s *ClientSuite) Test_apply_reconciliation() {
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

	payload := &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		ResourceActions: []types.ResourceReconcileActionPayload{
			{
				ResourceID:    "resource-1-id",
				Action:        "accept_external",
				ExternalState: core.MappingNodeFromString("external-state-value"),
				NewStatus:     "created",
			},
		},
		LinkActions: []types.LinkReconcileActionPayload{
			{
				LinkID:    "link-1-id",
				Action:    "update_status",
				NewStatus: "resource_b_updated",
			},
		},
	}

	result, err := client.ApplyReconciliation(
		context.Background(),
		"test-instance-100",
		payload,
	)
	s.Require().NoError(err)

	s.Assert().Equal("test-instance-100", result.InstanceID)
	s.Assert().Equal(1, result.ResourcesUpdated)
	s.Assert().Equal(1, result.LinksUpdated)
	s.Assert().Empty(result.Errors)
}

func (s *ClientSuite) Test_apply_reconciliation_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	payload := &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		ResourceActions: []types.ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1-id",
				Action:     "accept_external",
				NewStatus:  "created",
			},
		},
	}

	_, err = client.ApplyReconciliation(
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

func (s *ClientSuite) Test_apply_reconciliation_fails_due_to_invalid_json_response() {
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

	payload := &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		ResourceActions: []types.ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1-id",
				Action:     "accept_external",
				NewStatus:  "created",
			},
		},
	}

	_, err = client.ApplyReconciliation(
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

func (s *ClientSuite) Test_apply_reconciliation_fails_due_to_internal_server_error() {
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

	payload := &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		ResourceActions: []types.ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1-id",
				Action:     "accept_external",
				NewStatus:  "created",
			},
		},
	}

	_, err = client.ApplyReconciliation(
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

func (s *ClientSuite) Test_apply_reconciliation_fails_due_to_network_error() {
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

	payload := &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		ResourceActions: []types.ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1-id",
				Action:     "accept_external",
				NewStatus:  "created",
			},
		},
	}

	_, err = client.ApplyReconciliation(
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
		"/reconciliation/apply",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
