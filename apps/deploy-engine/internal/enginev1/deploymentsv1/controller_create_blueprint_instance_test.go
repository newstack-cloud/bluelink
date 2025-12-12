package deploymentsv1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/r3labs/sse/v2"
)

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler() {
	// Create the test change set to be used to start the deployment
	// process for the new blueprint instance.
	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	instance := &state.InstanceState{}
	err = json.Unmarshal(respData, instance)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(instance.InstanceID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		core.InstanceStatusPreparing,
		instance.Status,
	)
	s.Assert().Equal(
		testTime.Unix(),
		int64(instance.LastStatusUpdateTimestamp),
	)

	expectedEvents := deployEventSequence(instance.InstanceID)
	actualEvents, err := s.streamDeployEvents(instance.InstanceID, len(expectedEvents))
	s.Require().NoError(err)

	s.assertDeployEventsEqual(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler_with_instance_name() {
	// Create the test change set to be used to start the deployment
	// process for the new blueprint instance.
	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	newInstanceName := "my-new-instance"
	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		InstanceName: newInstanceName,
		ChangeSetID:  testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	instance := &state.InstanceState{}
	err = json.Unmarshal(respData, instance)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(instance.InstanceID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		newInstanceName,
		instance.InstanceName,
	)
	s.Assert().Equal(
		core.InstanceStatusPreparing,
		instance.Status,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler_with_stream_error() {
	// Create the test change set to be used to start the deployment
	// process for the new blueprint instance.
	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrlStreamErrors.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	instance := &state.InstanceState{}
	err = json.Unmarshal(respData, instance)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(instance.InstanceID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		core.InstanceStatusPreparing,
		instance.Status,
	)
	s.Assert().Equal(
		testTime.Unix(),
		int64(instance.LastStatusUpdateTimestamp),
	)

	expectedEvents := []testutils.DeployEventWrapper{
		{
			DeployEvent: &container.DeployEvent{
				DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
					InstanceID:      instance.InstanceID,
					Status:          core.InstanceStatusPreparing,
					UpdateTimestamp: testTime.Unix(),
				},
			},
		},

		{
			DeployError: errors.New(
				"error: deploy error",
			),
		},
	}
	actualEvents, err := s.streamDeployEvents(instance.InstanceID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Len(actualEvents, len(expectedEvents))
	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler_fails_for_invalid_input() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			// "files" is not a valid scheme.
			FileSourceScheme: "files",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	validationError := &inputvalidation.FormattedValidationError{}
	err = json.Unmarshal(respData, validationError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().Equal(
		"request body input validation failed",
		validationError.Message,
	)
	s.Assert().Len(validationError.Errors, 1)
	s.Assert().Equal(
		".fileSourceScheme",
		validationError.Errors[0].Location,
	)
	s.Assert().Equal(
		"the value must be one of the following: file s3 gcs azureblob https",
		validationError.Errors[0].Message,
	)
	s.Assert().Equal(
		"oneof",
		validationError.Errors[0].Type,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler_fails_for_invalid_plugin_config() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: testChangesetID,
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"aws": {
					"field1": core.ScalarFromString("invalid value"),
				},
			},
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	validationError := &typesv1.ValidationDiagnosticErrors{}
	err = json.Unmarshal(respData, validationError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().Equal(
		"plugin configuration validation failed",
		validationError.Message,
	)
	s.Assert().Equal(
		pluginConfigPreparerFixtures["invalid value"],
		validationError.ValidationDiagnostics,
	)
}

func (s *ControllerTestSuite) Test_create_blueprint_instance_handler_fails_due_to_missing_changeset() {
	// We are not saving the test change set for this test,
	// so it should not be found when the request is made.
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrlFailingIDGenerators.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	responseError := map[string]string{}
	err = json.Unmarshal(respData, &responseError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusBadRequest, result.StatusCode)
	s.Assert().Equal(
		"requested change set is missing",
		responseError["message"],
	)
}

func (s *ControllerTestSuite) assertDeployEventsEqual(
	expected []container.DeployEvent,
	actual []testutils.DeployEventWrapper,
) {
	s.Assert().Len(actual, len(expected))
	for i, event := range actual {
		s.Assert().Equal(
			expected[i],
			*event.DeployEvent,
		)
	}
}

func (s *ControllerTestSuite) streamDeployEvents(
	instanceID string,
	expectedCount int,
) ([]testutils.DeployEventWrapper, error) {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		s.ctrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/instances/%s/stream",
		streamServer.URL,
		instanceID,
	)

	client := sse.NewClient(url)

	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	collected := []*manage.Event{}
	for len(collected) < expectedCount {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			s.Require().NotNil(event)
		case <-time.After(5 * time.Second):
			s.Fail("Timed out waiting for events")
		}
	}

	return extractDeployEvents(collected)
}

func extractDeployEvents(
	collected []*manage.Event,
) ([]testutils.DeployEventWrapper, error) {
	extractedEvents := []testutils.DeployEventWrapper{}

	for _, event := range collected {
		if event.Type == eventTypeResourceUpdate {
			resourceUpdate := &container.ResourceDeployUpdateMessage{}
			err := parseEventJSON(event, resourceUpdate)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, wrapDeployEvent(
				container.DeployEvent{
					ResourceUpdateEvent: resourceUpdate,
				},
			))
		}

		if event.Type == eventTypeChildUpdate {
			childUpdate := &container.ChildDeployUpdateMessage{}
			err := parseEventJSON(event, childUpdate)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, wrapDeployEvent(
				container.DeployEvent{
					ChildUpdateEvent: childUpdate,
				},
			))
		}

		if event.Type == eventTypeLinkUpdate {
			linkUpdate := &container.LinkDeployUpdateMessage{}
			err := parseEventJSON(event, linkUpdate)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, wrapDeployEvent(
				container.DeployEvent{LinkUpdateEvent: linkUpdate},
			))
		}

		if event.Type == eventTypeInstanceUpdate {
			deploymentUpdate := &container.DeploymentUpdateMessage{}
			err := parseEventJSON(event, deploymentUpdate)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, wrapDeployEvent(
				container.DeployEvent{
					DeploymentUpdateEvent: deploymentUpdate,
				},
			))
		}

		if event.Type == eventTypeDeployFinished {
			finishEvent := &container.DeploymentFinishedMessage{}
			err := parseEventJSON(event, finishEvent)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, wrapDeployEvent(
				container.DeployEvent{
					FinishEvent: finishEvent,
				},
			))
		}

		if event.Type == eventTypeError {
			errorMessage := &errorMessageEvent{}
			err := parseEventJSON(event, errorMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.DeployEventWrapper{
				DeployError: fmt.Errorf(
					"error: %s",
					errorMessage.Message,
				),
			})
		}
	}

	return extractedEvents, nil
}

func parseEventJSON(
	event *manage.Event,
	target any,
) error {
	return json.Unmarshal(
		[]byte(event.Data),
		target,
	)
}

func wrapDeployEvent(
	event container.DeployEvent,
) testutils.DeployEventWrapper {
	return testutils.DeployEventWrapper{
		DeployEvent: &event,
	}
}
