// Tests for the CreateBlueprintValidation method in the DeployEngine client.
// This will include tests that use different authentication methods,
// including API key, OAuth2, and Bluelink Signature v1.
// This is the only method that tests all supported auth methods,
// tests for other DeployEngine client methods will be tested against
// a single authentication method.
// This file also tests connecting to the DeployEngine using a Unix domain socket.
package deployengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

const (
	testValidationID      = "test-validation-id"
	testBlueprintLocation = "test-blueprint-location"
)

func (s *ClientSuite) Test_create_blueprint_validation_oauth2_preconfigured_token_endpoint() {
	// Create a new client with OAuth2 that uses a unix domain socket
	// to connect to the Deploy Engine.
	client, err := NewClient(
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			TokenEndpoint: fmt.Sprintf(
				"%s/oauth2/v1/token",
				s.oauthServer.URL,
			),
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
		}),
		WithClientConnectProtocol(ConnectProtocolUnixDomainSocket),
		WithClientUnixDomainSocket(s.testSocketPath),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	blueprintValidation, err := client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.BlueprintValidation{
			ID:                testValidationID,
			BlueprintLocation: testBlueprintLocation,
			Status:            manage.BlueprintValidationStatusStarting,
			Created:           testTime.Unix(),
		},
		blueprintValidation,
	)
}

func (s *ClientSuite) Test_create_blueprint_validation_oauth2_derive_token_endpoint() {
	// Create a new client with OAuth2.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			ProviderBaseURL: s.oauthServer.URL,
			ClientID:        testClientID,
			ClientSecret:    testClientSecret,
		}),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	blueprintValidation, err := client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.BlueprintValidation{
			ID:                testValidationID,
			BlueprintLocation: testBlueprintLocation,
			Status:            manage.BlueprintValidationStatusStarting,
			Created:           testTime.Unix(),
		},
		blueprintValidation,
	)
}

func (s *ClientSuite) Test_create_blueprint_validation_api_key() {
	// Create a new client with API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey(testAPIKey),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	blueprintValidation, err := client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.BlueprintValidation{
			ID:                testValidationID,
			BlueprintLocation: testBlueprintLocation,
			Status:            manage.BlueprintValidationStatusStarting,
			Created:           testTime.Unix(),
		},
		blueprintValidation,
	)
}

func (s *ClientSuite) Test_create_blueprint_validation_bluelink_sigv1() {
	// Create a new client with Bluelink signature v1 auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodBluelinkSignatureV1),
		WithClientBluelinkSigv1KeyPair(testBluelinkSignatureKeyPair),
		WithClientClock(s.clock),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	blueprintValidation, err := client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		&manage.BlueprintValidation{
			ID:                testValidationID,
			BlueprintLocation: testBlueprintLocation,
			Status:            manage.BlueprintValidationStatusStarting,
			Created:           testTime.Unix(),
		},
		blueprintValidation,
	)
}

func (s *ClientSuite) Test_create_blueprint_validation_fails_for_unauthorised_client() {
	// Create a new client with invalid API key auth.
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodAPIKey),
		WithClientAPIKey("invalid-api-key"),
	)
	s.Require().NoError(err)

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
		},
	}

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
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

func (s *ClientSuite) Test_create_blueprint_validation_fails_for_incorrect_input() {
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

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			// "files" is not a valid file source scheme.
			FileSourceScheme: "files",
		},
	}

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
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

func (s *ClientSuite) Test_create_blueprint_validation_fails_due_to_invalid_json_response() {
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

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// the stub server to return an invalid JSON response.
			BlueprintFile: deserialiseErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().Error(err)

	deserialiseErr, isDeserialiseErr := err.(*errors.DeserialiseError)
	s.Require().True(isDeserialiseErr)

	s.Assert().Equal(
		"deserialise error: failed to decode response: unexpected EOF",
		deserialiseErr.Error(),
	)
}

func (s *ClientSuite) Test_create_blueprint_validation_fails_due_to_internal_server_error() {
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

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated internal server error.
			BlueprintFile: internalServerErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
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

func (s *ClientSuite) Test_create_blueprint_validation_fails_due_to_network_error() {
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

	// Make a request to create a blueprint validation
	payload := &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			// The blueprint file is set to a value that will trigger
			// a simulated network error by causing the server to
			// close the connection early.
			BlueprintFile: networkErrorTrigger,
		},
	}

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		payload,
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().Error(err)

	clientErr, isClientErr := err.(*errors.RequestError)
	s.Require().True(isClientErr)

	expectedErrorMessage := fmt.Sprintf(
		"request error: Post \"%s%s\": EOF",
		s.deployEngineServer.URL,
		"/v1/validations",
	)
	s.Assert().Equal(
		expectedErrorMessage,
		clientErr.Error(),
	)
}
