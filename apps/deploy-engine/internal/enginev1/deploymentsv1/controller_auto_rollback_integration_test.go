package deploymentsv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

// Test_auto_rollback_triggered_on_deploy_failed tests that auto-rollback
// is triggered when a new deployment fails with DeployFailed status.
func (s *ControllerTestSuite) Test_auto_rollback_triggered_on_deploy_failed() {
	// Create a destroy tracker to verify rollback was triggered
	destroyTracker := testutils.NewDestroyTracker()

	// Create a controller with a deployment that will fail
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployFailed,
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true, // Enable auto-rollback
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
	s.Assert().NoError(err, "ID should be a valid UUID")

	// Wait for the deployment to complete and auto-rollback to trigger
	time.Sleep(200 * time.Millisecond)

	// Verify that Destroy was called with rollback=true
	s.Assert().True(
		destroyTracker.WasDestroyCalled(),
		"Destroy should have been called for auto-rollback",
	)

	rollbackCalls := destroyTracker.GetRollbackCalls()
	s.Assert().Len(rollbackCalls, 1, "Should have exactly one rollback call")
	s.Assert().Equal(
		instance.InstanceID,
		rollbackCalls[0].InstanceID,
		"Rollback should be for the correct instance",
	)
	s.Assert().True(
		rollbackCalls[0].Rollback,
		"Destroy call should have Rollback=true",
	)
}

// Test_auto_rollback_not_triggered_when_disabled tests that auto-rollback
// is NOT triggered when the AutoRollback flag is set to false.
func (s *ControllerTestSuite) Test_auto_rollback_not_triggered_when_disabled() {
	// Create a destroy tracker to verify rollback was NOT triggered
	destroyTracker := testutils.NewDestroyTracker()

	// Create a controller with a deployment that will fail
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployFailed,
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: false, // Auto-rollback disabled
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the deployment to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that Destroy was NOT called
	s.Assert().False(
		destroyTracker.WasDestroyCalled(),
		"Destroy should NOT have been called when auto-rollback is disabled",
	)
}

// Test_auto_rollback_not_triggered_on_success tests that auto-rollback
// is NOT triggered when the deployment succeeds.
func (s *ControllerTestSuite) Test_auto_rollback_not_triggered_on_success() {
	// Create a destroy tracker to verify rollback was NOT triggered
	destroyTracker := testutils.NewDestroyTracker()

	// Create a controller with a successful deployment
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployed, // Success status
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true, // Auto-rollback enabled
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the deployment to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that Destroy was NOT called
	s.Assert().False(
		destroyTracker.WasDestroyCalled(),
		"Destroy should NOT have been called when deployment succeeds",
	)
}

// Test_auto_rollback_triggered_on_update_failed tests that auto-rollback
// is triggered when an update fails (UpdateFailed status).
// This triggers a revert rollback (not destroy) using a reverse changeset.
func (s *ControllerTestSuite) Test_auto_rollback_triggered_on_update_failed() {
	// Create trackers to verify rollback behavior
	destroyTracker := testutils.NewDestroyTracker()
	deployTracker := testutils.NewDeployTracker()

	// Create a controller with a deployment that will fail with UpdateFailed
	// We need to use the shared instances container so the test instance is accessible
	ctrl := s.createAutoRollbackTestControllerWithTrackers(
		core.InstanceStatusUpdateFailed,
		destroyTracker,
		deployTracker,
		s.instances,
	)

	// Create the existing blueprint instance to update
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	err = s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true, // Auto-rollback enabled
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s", testInstanceID)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the deployment to complete and auto-rollback to trigger
	time.Sleep(300 * time.Millisecond)

	// Verify that Destroy was NOT called (update rollback uses Deploy, not Destroy)
	s.Assert().False(
		destroyTracker.WasDestroyCalled(),
		"Destroy should NOT have been called for UpdateFailed rollback",
	)

	// Verify that a rollback Deploy was called
	rollbackDeployCalls := deployTracker.GetRollbackDeployCalls()
	s.Assert().GreaterOrEqual(
		len(rollbackDeployCalls),
		1,
		"At least one rollback Deploy should have been called for UpdateFailed",
	)

	// Verify the rollback deploy was for the correct instance
	if len(rollbackDeployCalls) > 0 {
		s.Assert().Equal(
			testInstanceID,
			rollbackDeployCalls[0].InstanceID,
			"Rollback deploy should be for the correct instance",
		)
		s.Assert().True(
			rollbackDeployCalls[0].Rollback,
			"Deploy call should have Rollback=true",
		)
	}
}

// Test_auto_rollback_not_triggered_when_already_rolling_back tests that auto-rollback
// is NOT triggered when the instance is already in a rollback state.
func (s *ControllerTestSuite) Test_auto_rollback_not_triggered_when_already_rolling_back() {
	// Create a destroy tracker to verify rollback was NOT triggered
	destroyTracker := testutils.NewDestroyTracker()

	// Create a controller with a deployment that ends in DeployRollbackFailed
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployRollbackFailed,
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true, // Auto-rollback enabled
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/deployments/instances", bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	// Wait for the deployment to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that Destroy was NOT called
	// Should not trigger auto-rollback for rollback states to prevent infinite loops
	s.Assert().False(
		destroyTracker.WasDestroyCalled(),
		"Destroy should NOT have been called when already in rollback state",
	)
}

// Test_auto_rollback_stream_remains_open tests that the event stream
// remains open when auto-rollback triggers (endOfStream=false on initial failure).
func (s *ControllerTestSuite) Test_auto_rollback_stream_remains_open() {
	destroyTracker := testutils.NewDestroyTracker()
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployFailed,
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		ctrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	// Create the instance
	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true,
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

	// Wait a bit for events to be generated
	time.Sleep(100 * time.Millisecond)

	// Create SSE server to stream events
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/instances/%s/stream",
		streamServer.URL,
		instance.InstanceID,
	)

	client := sse.NewClient(url)

	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	// Collect events - we should receive the DeployFailed finish event
	// with endOfStream=false if auto-rollback is properly configured
	var collected []*manage.Event
	timeout := time.After(3 * time.Second)
eventLoop:
	for {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			// Look for the deploy finished event
			if manageEvent.Type == eventTypeDeployFinished {
				break eventLoop
			}
		case <-timeout:
			break eventLoop
		}
	}

	// Verify we got at least the preparing event and the finish event
	s.Assert().GreaterOrEqual(len(collected), 2, "Should have received multiple events")

	// Find the finish event and verify it doesn't have End=true
	// (if auto-rollback is enabled and triggers, the stream should stay open)
	var foundFinish bool
	for _, event := range collected {
		if event.Type == eventTypeDeployFinished {
			foundFinish = true
			// When auto-rollback triggers, End should be false
			s.Assert().False(
				event.End,
				"DeployFinished event should have End=false when auto-rollback triggers",
			)
		}
	}
	s.Assert().True(foundFinish, "Should have found a deploy finished event")
}

// createAutoRollbackTestController creates a controller configured for auto-rollback testing
// with the specified finish status and destroy tracker.
func (s *ControllerTestSuite) createAutoRollbackTestController(
	finishStatus core.InstanceStatus,
	destroyTracker *testutils.DestroyTracker,
) *Controller {
	stateContainer := testutils.NewMemoryStateContainer()
	return s.createAutoRollbackTestControllerWithInstances(
		finishStatus,
		destroyTracker,
		stateContainer.Instances(),
	)
}

// createAutoRollbackTestControllerWithInstances creates a controller configured for auto-rollback testing
// with the specified finish status, destroy tracker, and instances container.
// This allows tests to use a shared instances container for update tests.
func (s *ControllerTestSuite) createAutoRollbackTestControllerWithInstances(
	finishStatus core.InstanceStatus,
	destroyTracker *testutils.DestroyTracker,
	instances state.InstancesContainer,
) *Controller {
	return s.createAutoRollbackTestControllerWithTrackers(
		finishStatus,
		destroyTracker,
		nil, // no deploy tracker
		instances,
	)
}

// createAutoRollbackTestControllerWithTrackers creates a controller configured for auto-rollback testing
// with the specified finish status, destroy tracker, deploy tracker, and instances container.
// This allows tests to track both destroy and deploy calls for update rollback tests.
func (s *ControllerTestSuite) createAutoRollbackTestControllerWithTrackers(
	finishStatus core.InstanceStatus,
	destroyTracker *testutils.DestroyTracker,
	deployTracker *testutils.DeployTracker,
	instances state.InstancesContainer,
) *Controller {
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	// Create deploy event sequence with the specified finish status
	deployEvents := deployEventSequenceWithStatus("", finishStatus)

	// Create destroy event sequence for rollback
	destroyEvents := []container.DeployEvent{
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				Status:          core.InstanceStatusDeployRollingBack,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:          core.InstanceStatusDeployRollbackComplete,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	}

	// Create rollback deploy event sequence for update rollback
	rollbackDeployEvents := []container.DeployEvent{
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				Status:          core.InstanceStatusUpdateRollingBack,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:          core.InstanceStatusUpdateRollbackComplete,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	}

	loaderOpts := []testutils.MockBlueprintLoaderOption{
		testutils.WithDestroyTracker(destroyTracker),
		testutils.WithDestroyEventSequence(destroyEvents),
		testutils.WithRollbackDeployEventSequence(rollbackDeployEvents),
	}
	if deployTracker != nil {
		loaderOpts = append(loaderOpts, testutils.WithDeployTracker(deployTracker))
	}

	blueprintLoader := testutils.NewMockBlueprintLoader(
		[]*core.Diagnostic{},
		clock,
		instances,
		deployEvents,
		changeStagingEventSequence(),
		loaderOpts...,
	)

	dependencies := &typesv1.Dependencies{
		EventStore:     s.eventStore,
		ChangesetStore: s.changesetStore,
		ValidationStore: testutils.NewMockBlueprintValidationStore(
			map[string]*manage.BlueprintValidation{},
		),
		Instances:         instances,
		Exports:           stateContainer.Exports(),
		IDGenerator:       core.NewUUIDGenerator(),
		EventIDGenerator:  utils.NewUUIDv7Generator(),
		ValidationLoader:  blueprintLoader,
		DeploymentLoader:  blueprintLoader,
		BlueprintResolver: &testutils.MockBlueprintResolver{},
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

// deployEventSequenceWithStatus creates a deploy event sequence with a specified finish status.
func deployEventSequenceWithStatus(instanceID string, finishStatus core.InstanceStatus) []container.DeployEvent {
	return []container.DeployEvent{
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				InstanceID:      instanceID,
				Status:          core.InstanceStatusPreparing,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				InstanceID:      instanceID,
				Status:          core.InstanceStatusDeploying,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			FinishEvent: &container.DeploymentFinishedMessage{
				InstanceID:      instanceID,
				Status:          finishStatus,
				UpdateTimestamp: testTime.Unix(),
			},
		},
	}
}

// Test_pre_rollback_state_emitted_before_auto_rollback tests that the preRollbackState event
// is emitted before auto-rollback begins, capturing the instance state for debugging.
func (s *ControllerTestSuite) Test_pre_rollback_state_emitted_before_auto_rollback() {
	destroyTracker := testutils.NewDestroyTracker()
	ctrl := s.createAutoRollbackTestController(
		core.InstanceStatusDeployFailed,
		destroyTracker,
	)

	err := s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		ctrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	// Create the instance with auto-rollback enabled
	reqPayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID:  testChangesetID,
		AutoRollback: true,
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

	// Wait for deployment to fail and auto-rollback to trigger
	time.Sleep(300 * time.Millisecond)

	// Create SSE server to stream events
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/instances/%s/stream",
		streamServer.URL,
		instance.InstanceID,
	)

	client := sse.NewClient(url)

	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	// Collect events including pre-rollback state
	collected := collectEventsUntilEnd(eventChan, 5*time.Second)

	// Verify pre-rollback state event was emitted
	foundPreRollbackState := false
	var preRollbackIndex, rollbackStartIndex int
	for i, event := range collected {
		if event.Type == eventTypePreRollbackState {
			foundPreRollbackState = true
			preRollbackIndex = i
		}
		if event.Type == eventTypeInstanceUpdate {
			var updateMsg container.DeploymentUpdateMessage
			if err := json.Unmarshal([]byte(event.Data), &updateMsg); err == nil {
				if updateMsg.Status == core.InstanceStatusDeployRollingBack {
					rollbackStartIndex = i
				}
			}
		}
	}

	s.Assert().True(foundPreRollbackState, "Should have received preRollbackState event")

	// Verify pre-rollback state event comes before rollback start event
	if foundPreRollbackState && rollbackStartIndex > 0 {
		s.Assert().Less(
			preRollbackIndex,
			rollbackStartIndex,
			"Pre-rollback state event should come before rollback start event",
		)
	}
}

// collectEventsUntilEnd collects events until an event with End=true is received or timeout.
func collectEventsUntilEnd(eventChan chan *sse.Event, timeout time.Duration) []*manage.Event {
	var collected []*manage.Event
	timer := time.After(timeout)

	for {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			if manageEvent.End {
				return collected
			}
		case <-timer:
			return collected
		}
	}
}
