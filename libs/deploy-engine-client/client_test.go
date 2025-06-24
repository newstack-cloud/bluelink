package deployengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/common/sigv1"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/stretchr/testify/suite"
)

const (
	testAPIKey                   = "test-api-key"
	testClientID                 = "test-client-id"
	testClientSecret             = "test-client-secret"
	internalServerErrorTrigger   = "fail.blueprint.yml"
	networkErrorTrigger          = "network-fail.blueprint.yml"
	deserialiseErrorTrigger      = "deserialise-fail.blueprint.yml"
	internalServerErrorTriggerID = "fail-id"
	networkErrorTriggerID        = "network-fail-id"
	deserialiseErrorTriggerID    = "deserialise-fail-id"
	failingStreamTriggerID       = "failing-stream-id"
	testFailingStreamEventID     = "test-failing-stream-event-id"
)

var (
	testBluelinkSignatureKeyPair = &sigv1.KeyPair{
		KeyID:     "test-key-id",
		SecretKey: "test-secret-key",
	}
	// Tuesday, 6 May 2025 12:34:22 UTC
	testTime = time.Unix(1746534862, 0)
)

type ClientSuite struct {
	oauthServer                *httptest.Server
	failingOAuthServer         *httptest.Server
	deployEngineServer         *httptest.Server
	deployEngineServerOnSocket *httptest.Server
	clock                      *testutils.MockClock
	testSocketPath             string
	stubValidationEvents       []*manage.Event
	stubChangeStagingEvents    []*manage.Event
	stubDeploymentEvents       []*manage.Event
	suite.Suite
}

func (s *ClientSuite) SetupSuite() {
	oauthServer, err := testutils.CreateOAuthServer(
		testClientID,
		testClientSecret,
		"oauth2",
	)
	s.Require().NoError(err)
	s.oauthServer = oauthServer

	failingOAuthServer := testutils.CreateFailingServer()
	s.Require().NoError(err)
	s.failingOAuthServer = failingOAuthServer

	s.setupStreamStubEvents()

	clock := &testutils.MockClock{
		TimeSequence: []time.Time{
			testTime,
		},
	}
	s.clock = clock
	s.deployEngineServer = testutils.CreateDeployEngineServer(
		&testutils.TestServerConfig{
			AllowedAPIKeys: []string{testAPIKey},
			// See internal/testutils/token_server.go for the
			// static token returned by the token server.
			AllowedBearerTokens: []string{"test-token-1"},
			AllowedBluelinkSignatureKeyPairs: map[string]*sigv1.KeyPair{
				testBluelinkSignatureKeyPair.KeyID: testBluelinkSignatureKeyPair,
			},
			InternalServerErrorTrigger:   internalServerErrorTrigger,
			NetworkErrorTrigger:          networkErrorTrigger,
			DeserialiseErrorTrigger:      deserialiseErrorTrigger,
			InternalServerErrorTriggerID: internalServerErrorTriggerID,
			NetworkErrorTriggerID:        networkErrorTriggerID,
			DeserialiseErrorTriggerID:    deserialiseErrorTriggerID,
			FailingStreamTriggerID:       failingStreamTriggerID,
		},
		s.stubValidationEvents,
		s.stubChangeStagingEvents,
		s.stubDeploymentEvents,
		clock,
	)
	s.prepareTestSocketDir()
	s.deployEngineServerOnSocket = testutils.CreateDeployEngineServer(
		&testutils.TestServerConfig{
			AllowedAPIKeys: []string{testAPIKey},
			// See internal/testutils/token_server.go for the
			// static token returned by the token server.
			AllowedBearerTokens: []string{"test-token-1"},
			AllowedBluelinkSignatureKeyPairs: map[string]*sigv1.KeyPair{
				testBluelinkSignatureKeyPair.KeyID: testBluelinkSignatureKeyPair,
			},
			InternalServerErrorTrigger:   internalServerErrorTrigger,
			NetworkErrorTrigger:          networkErrorTrigger,
			DeserialiseErrorTrigger:      deserialiseErrorTrigger,
			InternalServerErrorTriggerID: internalServerErrorTriggerID,
			NetworkErrorTriggerID:        networkErrorTriggerID,
			DeserialiseErrorTriggerID:    deserialiseErrorTriggerID,
			FailingStreamTriggerID:       failingStreamTriggerID,
			UseUnixDomainSocket:          true,
			UnixDomainSocketPath:         s.testSocketPath,
		},
		s.stubValidationEvents,
		s.stubChangeStagingEvents,
		s.stubDeploymentEvents,
		clock,
	)
}

func (s *ClientSuite) TearDownSuite() {
	s.oauthServer.Close()
	s.failingOAuthServer.Close()
	s.deployEngineServer.Close()
	s.deployEngineServerOnSocket.Close()

	// Remove the socket file if it exists, ignore errors if the
	// file does not exist.
	os.Remove(s.testSocketPath)
}

func (s *ClientSuite) Test_fails_to_setup_client_due_to_failing_oauth_provider_discovery_doc() {
	_, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			ProviderBaseURL: s.failingOAuthServer.URL,
			ClientID:        testClientID,
			ClientSecret:    testClientSecret,
		}),
		// Override the default HTTP transport to opt out of retry behaviour.
		WithClientHTTPRoundTripper(testutils.CreateDefaultTransport),
	)
	s.Require().Error(err)
	authInitErr, isAuthInitErr := err.(*errors.AuthInitError)
	s.Require().True(isAuthInitErr)
	s.Assert().Equal(
		"auth init error: failed to get token endpoint from provider: "+
			"failed to fetch discovery document: 500 Internal Server Error",
		authInitErr.Error(),
	)
}

func (s *ClientSuite) Test_fails_request_due_to_an_invalid_auth_method() {
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		// 100 is an invalid auth method.
		WithClientAuthMethod(AuthMethod(100)),
	)
	s.Require().NoError(err)

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		&types.CreateBlueprintValidationPayload{
			BlueprintDocumentInfo: types.BlueprintDocumentInfo{
				FileSourceScheme: "file",
				BlueprintFile:    "test.blueprint.yml",
			},
		},
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().Error(err)
	authPrepErr, isAuthPrepErr := err.(*errors.AuthPrepError)
	s.Require().True(isAuthPrepErr)
	s.Assert().Equal(
		"auth prep error: no valid authentication method configured",
		authPrepErr.Error(),
	)
}

func (s *ClientSuite) Test_fails_to_setup_client_due_to_failing_oauth_provider_token_endpoint() {
	client, err := NewClient(
		WithClientEndpoint(s.deployEngineServer.URL),
		WithClientAuthMethod(AuthMethodOAuth2),
		WithClientOAuth2Config(&OAuth2Config{
			TokenEndpoint: fmt.Sprintf(
				"%s/oauth2/v1/token",
				s.failingOAuthServer.URL,
			),
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
		}),
		// Override the default HTTP transport to opt out of retry behaviour.
		WithClientHTTPRoundTripper(testutils.CreateDefaultTransport),
	)
	s.Require().NoError(err)

	_, err = client.CreateBlueprintValidation(
		context.Background(),
		&types.CreateBlueprintValidationPayload{
			BlueprintDocumentInfo: types.BlueprintDocumentInfo{
				FileSourceScheme: "file",
				BlueprintFile:    "test.blueprint.yml",
			},
		},
		&types.CreateBlueprintValidationQuery{},
	)
	s.Require().Error(err)
	authPrepErr, isAuthPrepErr := err.(*errors.AuthPrepError)
	s.Require().True(isAuthPrepErr)
	s.Assert().Contains(
		authPrepErr.Error(),
		"auth prep error: failed to get access token: oauth2: cannot fetch token: ",
	)
}

func (s *ClientSuite) prepareTestSocketDir() {
	workingDir, err := os.Getwd()
	s.Require().NoError(err)

	socketDir := path.Join(
		workingDir,
		"tmp",
	)
	err = os.MkdirAll(socketDir, 0755)
	s.Require().NoError(err)

	s.testSocketPath = path.Join(
		socketDir,
		"bluelink.sock",
	)
}

func (s *ClientSuite) setupStreamStubEvents() {
	s.setupValidationStreamStubEvents()
	s.setupChangeStagingStreamStubEvents()
	s.setupDeploymentStreamStubEvents()
}

func (s *ClientSuite) setupValidationStreamStubEvents() {
	stubValidationEvents := make(
		[]*manage.Event,
		len(stubBlueprintValidationEvents),
	)
	for i, event := range stubBlueprintValidationEvents {
		// Make a copy to remove the ID so it isn't included in the serialised
		// event data.
		eventCopy := types.BlueprintValidationEvent{
			Diagnostic: event.Diagnostic,
			Timestamp:  event.Timestamp,
			End:        event.End,
		}
		serialised, err := json.Marshal(eventCopy)
		s.Require().NoError(err)

		stubValidationEvents[i] = &manage.Event{
			ID:          event.ID,
			Type:        "diagnostic",
			ChannelType: ChannelTypeValidation,
			ChannelID:   testValidationID,
			Data:        string(serialised),
			Timestamp:   testTime.Unix(),
			End:         event.End,
		}
	}

	s.stubValidationEvents = stubValidationEvents
}

var stubBlueprintValidationEvents = []types.BlueprintValidationEvent{
	{
		ID: "test-event-1",
		Diagnostic: core.Diagnostic{
			Level:   core.DiagnosticLevelError,
			Message: "Invalid version provided for blueprint",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 10,
					},
				},
				End: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 20,
					},
				},
			},
		},
		Timestamp: testTime.Unix(),
		End:       false,
	},
	{
		ID: "test-event-2",
		Diagnostic: core.Diagnostic{
			Level:   core.DiagnosticLevelError,
			Message: "Invalid transform provided for blueprint",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 8,
					},
				},
				End: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 18,
					},
				},
			},
		},
		Timestamp: testTime.Unix(),
		End:       false,
	},
	{
		ID: "test-event-3",
		Diagnostic: core.Diagnostic{
			Level:   core.DiagnosticLevelWarning,
			Message: "No resources defined in blueprint",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 5,
					},
				},
				End: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 15,
					},
				},
			},
		},
		End: true,
	},
}

func (s *ClientSuite) setupChangeStagingStreamStubEvents() {
	events := make(
		[]*manage.Event,
		len(sourceStubChangeStagingEvents),
	)
	for i, event := range sourceStubChangeStagingEvents {
		data := getChangeStagingEventData(&event)
		serialised, err := json.Marshal(data)
		s.Require().NoError(err)

		events[i] = &manage.Event{
			ID:          event.ID,
			Type:        string(event.GetType()),
			ChannelType: ChannelTypeChangeset,
			ChannelID:   testChangesetID,
			Data:        string(serialised),
			Timestamp:   testTime.Unix(),
			End:         event.GetType() == types.ChangeStagingEventTypeCompleteChanges,
		}
	}

	s.stubChangeStagingEvents = events
}

var sourceStubChangeStagingEvents = []types.ChangeStagingEvent{
	{
		ID: "test-event-1",
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName:           "resource-1",
				Removed:                false,
				New:                    false,
				ResolveOnDeploy:        []string{"spec.id"},
				ConditionKnownOnDeploy: false,
				Changes: provider.Changes{
					NewFields: []provider.FieldChange{
						{
							FieldPath: "spec.name",
							PrevValue: core.MappingNodeFromString("old-name"),
							NewValue:  core.MappingNodeFromString("new-name"),
						},
					},
				},
			},
			Timestamp: testTime.Unix(),
		},
	},
	{
		ID: "test-event-2",
		ChildChanges: &types.ChildChangesEventData{
			ChildChangesMessage: container.ChildChangesMessage{
				ChildBlueprintName: "child-blueprint-1",
				Removed:            false,
				New:                false,
				Changes: changes.BlueprintChanges{
					RemovedResources: []string{"child-resource-1"},
				},
			},
			Timestamp: testTime.Unix(),
		},
	},
	{
		ID: "test-event-3",
		LinkChanges: &types.LinkChangesEventData{
			LinkChangesMessage: container.LinkChangesMessage{
				ResourceAName: "resource-1",
				ResourceBName: "resource-2",
				Removed:       false,
				New:           false,
				Changes: provider.LinkChanges{
					ModifiedFields: []*provider.FieldChange{
						{
							FieldPath: "resource-1.policy.name",
							PrevValue: core.MappingNodeFromString("old-policy-name"),
							NewValue:  core.MappingNodeFromString("new-policy-name"),
						},
					},
				},
			},
			Timestamp: testTime.Unix(),
		},
	},
	{
		ID: "test-event-4",
		CompleteChanges: &types.CompleteChangesEventData{
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
						OutboundLinkChanges: map[string]provider.LinkChanges{
							"resource-2": {
								ModifiedFields: []*provider.FieldChange{
									{
										FieldPath: "resource-1.policy.name",
										PrevValue: core.MappingNodeFromString("old-policy-name"),
										NewValue:  core.MappingNodeFromString("new-policy-name"),
									},
								},
							},
						},
					},
				},
				ChildChanges: map[string]changes.BlueprintChanges{
					"child-blueprint-1": {
						RemovedResources: []string{"child-resource-1"},
					},
				},
			},
			Timestamp: testTime.Unix(),
		},
	},
}

func getChangeStagingEventData(
	event *types.ChangeStagingEvent,
) any {
	if resourceChanges, ok := event.AsResourceChanges(); ok {
		return resourceChanges
	}

	if childChanges, ok := event.AsChildChanges(); ok {
		return childChanges
	}

	if linkChanges, ok := event.AsLinkChanges(); ok {
		return linkChanges
	}

	if completeChanges, ok := event.AsCompleteChanges(); ok {
		return completeChanges
	}

	return nil
}

func (s *ClientSuite) setupDeploymentStreamStubEvents() {
	events := make(
		[]*manage.Event,
		len(sourceStubDeploymentEvents),
	)
	for i, event := range sourceStubDeploymentEvents {
		data := getDeploymentEventData(&event)
		serialised, err := json.Marshal(data)
		s.Require().NoError(err)

		events[i] = &manage.Event{
			ID:          event.ID,
			Type:        string(event.GetType()),
			ChannelType: ChannelTypeDeployment,
			ChannelID:   testInstanceID,
			Data:        string(serialised),
			Timestamp:   testTime.Unix(),
			End:         event.GetType() == types.BlueprintInstanceEventTypeDeployFinished,
		}
	}

	s.stubDeploymentEvents = events
}

var sourceStubDeploymentEvents = []types.BlueprintInstanceEvent{
	{
		ID: "test-deploy-event-1",
		DeployEvent: container.DeployEvent{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				InstanceID:      testInstanceID,
				Status:          core.InstanceStatusPreparing,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	},
	{
		ID: "test-deploy-event-2",
		DeployEvent: container.DeployEvent{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				InstanceID:      testInstanceID,
				ResourceID:      "resource-1",
				ResourceName:    "Resource1",
				Status:          core.ResourceStatusCreating,
				PreciseStatus:   core.PreciseResourceStatusConfigComplete,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	},
	{
		ID: "test-deploy-event-3",
		DeployEvent: container.DeployEvent{
			ChildUpdateEvent: &container.ChildDeployUpdateMessage{
				ChildInstanceID:  "test-child-instance-1",
				ParentInstanceID: testInstanceID,
				ChildName:        "coreInfra",
				Status:           core.InstanceStatusDeployed,
				UpdateTimestamp:  testTime.Unix(),
			},
		},
	},
	{
		ID: "test-deploy-event-4",
		DeployEvent: container.DeployEvent{
			LinkUpdateEvent: &container.LinkDeployUpdateMessage{
				InstanceID:      testInstanceID,
				LinkID:          "link-1",
				LinkName:        "resource-1::resource-2",
				Status:          core.LinkStatusCreating,
				PreciseStatus:   core.PreciseLinkStatusUpdatingResourceA,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	},
	{
		ID: "test-deploy-event-5",
		DeployEvent: container.DeployEvent{
			FinishEvent: &container.DeploymentFinishedMessage{
				InstanceID:      testInstanceID,
				Status:          core.InstanceStatusDeployed,
				FinishTimestamp: testTime.Unix(),
				UpdateTimestamp: testTime.Unix(),
			},
		},
	},
}

func getDeploymentEventData(
	event *types.BlueprintInstanceEvent,
) any {
	if resourceUpdate, ok := event.AsResourceUpdate(); ok {
		return resourceUpdate
	}

	if childUpdate, ok := event.AsChildUpdate(); ok {
		return childUpdate
	}

	if linkUpdate, ok := event.AsLinkUpdate(); ok {
		return linkUpdate
	}

	if instanceUpdate, ok := event.AsInstanceUpdate(); ok {
		return instanceUpdate
	}

	if finishEvent, ok := event.AsFinish(); ok {
		return finishEvent
	}

	return nil
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}
