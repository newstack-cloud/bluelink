// Tests for the CheckReconciliation method in the DeployEngine client.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

func (s *ClientSuite) Test_check_reconciliation() {
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

	payload := &types.CheckReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		Scope: "all",
	}

	result, err := client.CheckReconciliation(
		context.Background(),
		"test-instance-100",
		payload,
	)
	s.Require().NoError(err)

	s.Assert().Equal("test-instance-100", result.InstanceID)
	s.Assert().True(result.HasDrift)
	s.Assert().True(result.HasInterrupted)
	s.Assert().Len(result.Resources, 1)
	s.Assert().Len(result.Links, 1)

	// Verify resource reconcile result
	s.Assert().Equal("resource-1-id", result.Resources[0].ResourceID)
	s.Assert().Equal("resource-1", result.Resources[0].ResourceName)
	s.Assert().Equal("test/resource", result.Resources[0].ResourceType)
	s.Assert().Equal(container.ReconciliationTypeDrift, result.Resources[0].Type)
	s.Assert().Equal(core.PreciseResourceStatusCreated, result.Resources[0].OldStatus)
	s.Assert().Equal(core.PreciseResourceStatusCreated, result.Resources[0].NewStatus)
	s.Assert().True(result.Resources[0].ResourceExists)
	s.Assert().Equal(container.ReconciliationActionAcceptExternal, result.Resources[0].RecommendedAction)

	// Verify link reconcile result
	s.Assert().Equal("link-1-id", result.Links[0].LinkID)
	s.Assert().Equal("resource-1::resource-2", result.Links[0].LinkName)
	s.Assert().Equal(container.ReconciliationTypeInterrupted, result.Links[0].Type)
	s.Assert().Equal(core.PreciseLinkStatusResourceBUpdateRollingBack, result.Links[0].OldStatus)
	s.Assert().Equal(core.PreciseLinkStatusResourceBUpdated, result.Links[0].NewStatus)
	s.Assert().Equal(container.ReconciliationActionUpdateStatus, result.Links[0].RecommendedAction)
}

func (s *ClientSuite) Test_check_reconciliation_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	payload := &types.CheckReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		Scope: "all",
	}

	_, err = client.CheckReconciliation(
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

func (s *ClientSuite) Test_check_reconciliation_fails_due_to_invalid_json_response() {
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

	payload := &types.CheckReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		Scope: "all",
	}

	_, err = client.CheckReconciliation(
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

func (s *ClientSuite) Test_check_reconciliation_fails_due_to_internal_server_error() {
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

	payload := &types.CheckReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		Scope: "all",
	}

	_, err = client.CheckReconciliation(
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

func (s *ClientSuite) Test_check_reconciliation_fails_due_to_network_error() {
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

	payload := &types.CheckReconciliationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			BlueprintFile:    "/path/to/blueprint.yaml",
		},
		Scope: "all",
	}

	_, err = client.CheckReconciliation(
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
		"/reconciliation/check",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
