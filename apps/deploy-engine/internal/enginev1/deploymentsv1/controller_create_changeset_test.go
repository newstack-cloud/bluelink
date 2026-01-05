package deploymentsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/r3labs/sse/v2"
)

func (s *ControllerTestSuite) Test_create_changeset_handler() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrl.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.Changeset]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	changeset := wrappedResponse.Data
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(changeset.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		manage.ChangesetStatusStarting,
		changeset.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		changeset.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		changeset.Created,
	)

	expectedEvents := changeStagingEventSequence()
	actualEvents, err := s.streamChangeStagingEvents(changeset.ID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Len(actualEvents, len(expectedEvents))
	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_changeset_handler_with_stream_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrlStreamErrors.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.Changeset]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	changeset := wrappedResponse.Data
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(changeset.ID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		manage.ChangesetStatusStarting,
		changeset.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		changeset.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		changeset.Created,
	)

	expectedEvents := []testutils.ChangeStagingEvent{
		{
			Error: errors.New("error: change staging error"),
		},
	}
	actualEvents, err := s.streamChangeStagingEvents(changeset.ID, len(expectedEvents))
	s.Require().NoError(err)

	s.Assert().Len(actualEvents, len(expectedEvents))
	s.Assert().Equal(
		expectedEvents,
		actualEvents,
	)
}

func (s *ControllerTestSuite) Test_create_changeset_handler_fails_for_invalid_input() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrl.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			// "files" is not a valid scheme.
			FileSourceScheme: "files",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) Test_create_changeset_handler_fails_due_to_id_gen_error() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		s.ctrlFailingIDGenerators.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	responseError := map[string]string{}
	err = json.Unmarshal(respData, &responseError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
	s.Assert().Equal(
		utils.UnexpectedErrorMessage,
		responseError["message"],
	)
}

func (s *ControllerTestSuite) streamChangeStagingEvents(
	changesetID string,
	expectedCount int,
) ([]testutils.ChangeStagingEvent, error) {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes/{id}/stream",
		s.ctrl.StreamChangesetEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/changes/%s/stream",
		streamServer.URL,
		changesetID,
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

	return extractChangeStagingEvents(collected)
}

func (s *ControllerTestSuite) Test_create_changeset_drift_detected_blocks_staging() {
	// Setup a controller with drift detection configured
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	// Configure the mock to return drift detected
	driftResult := &container.ReconciliationCheckResult{
		InstanceID: testInstanceID,
		Resources: []container.ResourceReconcileResult{
			{
				ResourceID:   "resource-1",
				ResourceName: "testResource",
				Type:         container.ReconciliationTypeDrift,
			},
		},
		Links:          []container.LinkReconcileResult{},
		HasDrift:       true,
		HasInterrupted: false,
	}

	tracker := testutils.NewReconciliationTracker()
	blueprintLoader := testutils.NewMockBlueprintLoader(
		nil,
		clock,
		stateContainer.Instances(),
		deployEventSequence(""),
		changeStagingEventSequence(),
		testutils.WithReconciliationTracker(tracker),
		testutils.WithCheckReconciliationResult(driftResult),
	)

	ctrl := s.setupControllerWithLoader(stateContainer, clock, blueprintLoader)

	// Save an existing instance to trigger drift check
	s.saveTestInstanceInContainer(stateContainer)

	// Save a changeset associated with the instance
	changesetID := "drift-changeset-123"
	err := s.changesetStore.Save(context.TODO(), &manage.Changeset{
		ID:         changesetID,
		InstanceID: testInstanceID,
		Status:     manage.ChangesetStatusStarting,
		Created:    testTime.Unix(),
	})
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		ctrl.CreateChangesetHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/changes/{id}/stream",
		ctrl.StreamChangesetEventsHandler,
	).Methods("GET")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		InstanceID: testInstanceID,
		// skipDriftCheck is false by default
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.Changeset]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	changeset := wrappedResponse.Data
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Stream events and verify drift detected event is received
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/changes/%s/stream",
		streamServer.URL,
		changeset.ID,
	)

	client := sse.NewClient(url)
	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	// We expect exactly 1 drift detected event
	collected := []*manage.Event{}
	for len(collected) < 1 {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			s.Require().NotNil(event)
		case <-time.After(5 * time.Second):
			s.Fail("Timed out waiting for drift detected event")
		}
	}

	actualEvents, err := extractChangeStagingEvents(collected)
	s.Require().NoError(err)

	// Verify we received exactly 1 drift detected event
	s.Require().Len(actualEvents, 1)
	s.Require().NotNil(actualEvents[0].DriftDetectedEvent)

	// Verify the drift detected event contains the full reconciliation result
	driftEvent := actualEvents[0].DriftDetectedEvent
	s.Assert().Contains(driftEvent.Message, "Drift")
	s.Assert().Equal(testTime.Unix(), driftEvent.Timestamp)
	s.Require().NotNil(driftEvent.ReconciliationResult)
	s.Assert().Equal(testInstanceID, driftEvent.ReconciliationResult.InstanceID)
	s.Assert().True(driftEvent.ReconciliationResult.HasDrift)
	s.Assert().False(driftEvent.ReconciliationResult.HasInterrupted)
	s.Require().Len(driftEvent.ReconciliationResult.Resources, 1)
	s.Assert().Equal("resource-1", driftEvent.ReconciliationResult.Resources[0].ResourceID)
	s.Assert().Equal("testResource", driftEvent.ReconciliationResult.Resources[0].ResourceName)
	s.Assert().Equal(container.ReconciliationTypeDrift, driftEvent.ReconciliationResult.Resources[0].Type)

	// Verify reconciliation check was called
	s.Assert().True(tracker.WasCheckCalled())

	// Verify the changeset was saved with DRIFT_DETECTED status
	savedChangeset, err := s.changesetStore.Get(context.TODO(), changeset.ID)
	s.Require().NoError(err)
	s.Assert().Equal(manage.ChangesetStatusDriftDetected, savedChangeset.Status)
}

func (s *ControllerTestSuite) Test_create_changeset_skip_drift_check_bypasses_detection() {
	// Setup a controller with drift detection configured
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	// Configure the mock to return drift detected
	driftResult := &container.ReconciliationCheckResult{
		InstanceID:     testInstanceID,
		HasDrift:       true,
		HasInterrupted: false,
	}

	tracker := testutils.NewReconciliationTracker()
	blueprintLoader := testutils.NewMockBlueprintLoader(
		nil,
		clock,
		stateContainer.Instances(),
		deployEventSequence(""),
		changeStagingEventSequence(),
		testutils.WithReconciliationTracker(tracker),
		testutils.WithCheckReconciliationResult(driftResult),
	)

	ctrl := s.setupControllerWithLoader(stateContainer, clock, blueprintLoader)

	// Save an existing instance
	s.saveTestInstanceInContainer(stateContainer)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		ctrl.CreateChangesetHandler,
	).Methods("POST")

	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		InstanceID:     testInstanceID,
		SkipDriftCheck: true, // Skip drift check
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.Changeset]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the async process to complete
	time.Sleep(100 * time.Millisecond)

	// Verify reconciliation check was NOT called because skipDriftCheck=true
	s.Assert().False(tracker.WasCheckCalled())
}

func (s *ControllerTestSuite) Test_create_changeset_new_instance_skips_drift_check() {
	// Setup a controller with drift detection configured
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	tracker := testutils.NewReconciliationTracker()
	blueprintLoader := testutils.NewMockBlueprintLoader(
		nil,
		clock,
		stateContainer.Instances(),
		deployEventSequence(""),
		changeStagingEventSequence(),
		testutils.WithReconciliationTracker(tracker),
	)

	ctrl := s.setupControllerWithLoader(stateContainer, clock, blueprintLoader)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/changes",
		ctrl.CreateChangesetHandler,
	).Methods("POST")

	// Request without InstanceID (new deployment)
	reqPayload := &CreateChangesetRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		// No InstanceID - this is for a new deployment
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/changes", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the async process to complete
	time.Sleep(100 * time.Millisecond)

	// Verify reconciliation check was NOT called because there's no existing instance
	s.Assert().False(tracker.WasCheckCalled())
}

func (s *ControllerTestSuite) setupControllerWithLoader(
	stateContainer state.Container,
	clock *testutils.MockClock,
	blueprintLoader container.Loader,
) *Controller {
	dependencies := &typesv1.Dependencies{
		EventStore: testutils.NewMockEventStore(
			map[string]*manage.Event{},
		),
		ValidationStore: testutils.NewMockBlueprintValidationStore(
			map[string]*manage.BlueprintValidation{},
		),
		ChangesetStore:             s.changesetStore,
		ReconciliationResultsStore: s.reconciliationResultsStore,
		Instances:                  stateContainer.Instances(),
		Exports:                    stateContainer.Exports(),
		IDGenerator:                core.NewUUIDGenerator(),
		EventIDGenerator:           utils.NewUUIDv7Generator(),
		ValidationLoader:           blueprintLoader,
		DeploymentLoader:           blueprintLoader,
		BlueprintResolver:          &testutils.MockBlueprintResolver{},
		ParamsProvider: params.NewDefaultProvider(
			map[string]*core.ScalarValue{},
		),
		PluginConfigPreparer: testutils.NewMockPluginConfigPreparer(
			pluginConfigPreparerFixtures,
		),
		Clock:  clock,
		Logger: core.NewNopLogger(),
	}
	return NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		/* drainTimeout */ 100*time.Millisecond,
		dependencies,
	)
}

func (s *ControllerTestSuite) saveTestInstanceInContainer(
	stateContainer state.Container,
) {
	_ = stateContainer.Instances().Save(
		context.TODO(),
		state.InstanceState{
			InstanceID:                testInstanceID,
			InstanceName:              testInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)
}

func extractChangeStagingEvents(
	collected []*manage.Event,
) ([]testutils.ChangeStagingEvent, error) {
	extractedEvents := []testutils.ChangeStagingEvent{}

	for _, event := range collected {
		if event.Type == eventTypeResourceChanges {
			resourceChangesMessage := &container.ResourceChangesMessage{}
			err := parseEventJSON(event, resourceChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				ResourceChangesEvent: resourceChangesMessage,
			})
		}

		if event.Type == eventTypeChildChanges {
			childChangesMessage := &container.ChildChangesMessage{}
			err := parseEventJSON(event, childChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				ChildChangesEvent: childChangesMessage,
			})
		}

		if event.Type == eventTypeLinkChanges {
			LinkChangesMessage := &container.LinkChangesMessage{}
			err := parseEventJSON(event, LinkChangesMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				LinkChangesEvent: LinkChangesMessage,
			})
		}

		if event.Type == eventTypeChangeStagingComplete {
			finalChangesEvent := &changeStagingCompleteEvent{}
			err := parseEventJSON(event, finalChangesEvent)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				FinalBlueprintChanges: finalChangesEvent.Changes,
			})
		}

		if event.Type == eventTypeDriftDetected {
			driftEvent := &testutils.DriftDetectedEventData{}
			err := parseEventJSON(event, driftEvent)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				DriftDetectedEvent: driftEvent,
			})
		}

		if event.Type == eventTypeError {
			errorMessage := &errorMessageEvent{}
			err := parseEventJSON(event, errorMessage)
			if err != nil {
				return nil, err
			}
			extractedEvents = append(extractedEvents, testutils.ChangeStagingEvent{
				Error: fmt.Errorf("error: %s", errorMessage.Message),
			})
		}
	}

	return extractedEvents, nil
}
