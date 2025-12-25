package deploymentsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	integrationTestInstanceID   = "int-test-8582991a-9df3-4f7a-a649-344294aff656"
	integrationTestInstanceName = "integration-test-instance"
	integrationTestChangesetID  = "int-test-changeset-id"
)

type reconciliationIntegrationTestDeps struct {
	ctrl           *Controller
	stateContainer state.Container
	changesetStore manage.Changesets
	router         *mux.Router
}

func (s *ControllerTestSuite) setupReconciliationIntegrationTest(
	checkResult *container.ReconciliationCheckResult,
	applyResult *container.ApplyReconciliationResult,
) *reconciliationIntegrationTestDeps {
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	var opts []testutils.MockBlueprintLoaderOption
	if checkResult != nil {
		opts = append(opts, testutils.WithCheckReconciliationResult(checkResult))
	}
	if applyResult != nil {
		opts = append(opts, testutils.WithApplyReconciliationResult(applyResult))
	}

	blueprintLoader := testutils.NewMockBlueprintLoader(
		[]*core.Diagnostic{},
		clock,
		stateContainer.Instances(),
		deployEventSequence(""),
		changeStagingEventSequence(),
		opts...,
	)

	changesetStore := testutils.NewMockChangesetStore(
		map[string]*manage.Changeset{},
	)

	reconciliationResultsStore := testutils.NewMockReconciliationResultsStore(
		map[string]*manage.ReconciliationResult{},
	)

	dependencies := &typesv1.Dependencies{
		EventStore: testutils.NewMockEventStore(
			map[string]*manage.Event{},
		),
		ValidationStore: testutils.NewMockBlueprintValidationStore(
			map[string]*manage.BlueprintValidation{},
		),
		ChangesetStore:             changesetStore,
		ReconciliationResultsStore: reconciliationResultsStore,
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

	ctrl := NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		dependencies,
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	return &reconciliationIntegrationTestDeps{
		ctrl:           ctrl,
		stateContainer: stateContainer,
		changesetStore: changesetStore,
		router:         router,
	}
}

// Test_drift_workflow_update tests the full drift detection and resolution workflow for updates:
// 1. Changeset has DRIFT_DETECTED status
// 2. Update is blocked with 409
// 3. User calls check reconciliation
// 4. User calls apply reconciliation
// 5. Update succeeds with force=true
func (s *ControllerTestSuite) Test_drift_workflow_update() {
	checkResult := &container.ReconciliationCheckResult{
		InstanceID: integrationTestInstanceID,
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
	applyResult := &container.ApplyReconciliationResult{
		InstanceID:       integrationTestInstanceID,
		ResourcesUpdated: 1,
		LinksUpdated:     0,
		Errors:           []container.ReconciliationError{},
	}

	deps := s.setupReconciliationIntegrationTest(checkResult, applyResult)

	// Save the test instance
	err := deps.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:                integrationTestInstanceID,
			InstanceName:              integrationTestInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)
	s.Require().NoError(err)

	// Save a changeset with DRIFT_DETECTED status
	err = deps.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                integrationTestChangesetID,
			InstanceID:        integrationTestInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Created:           testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	// Step 1: Attempt update - should be blocked with 409
	updatePayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: integrationTestChangesetID,
	}
	updateBytes, err := json.Marshal(updatePayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s", integrationTestInstanceID)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(updateBytes))
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusConflict, result.StatusCode, "Update should be blocked with 409 Conflict")

	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)
	driftBlockedResp := &DriftBlockedResponse{}
	err = json.Unmarshal(respData, driftBlockedResp)
	s.Require().NoError(err)
	s.Assert().Equal(integrationTestInstanceID, driftBlockedResp.InstanceID)
	s.Assert().Contains(driftBlockedResp.Hint, "force=true")

	// Step 2: Call check reconciliation
	checkPayload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		Scope: "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	checkBytes, err := json.Marshal(checkPayload)
	s.Require().NoError(err)

	checkPath := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", integrationTestInstanceID)
	req = httptest.NewRequest("POST", checkPath, bytes.NewReader(checkBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusOK, result.StatusCode, "Check reconciliation should succeed")

	respData, err = io.ReadAll(result.Body)
	s.Require().NoError(err)
	var checkResp container.ReconciliationCheckResult
	err = json.Unmarshal(respData, &checkResp)
	s.Require().NoError(err)
	s.Assert().True(checkResp.HasDrift)

	// Step 3: Call apply reconciliation
	applyPayload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "accept_external",
				NewStatus:  "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	applyBytes, err := json.Marshal(applyPayload)
	s.Require().NoError(err)

	applyPath := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", integrationTestInstanceID)
	req = httptest.NewRequest("POST", applyPath, bytes.NewReader(applyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusOK, result.StatusCode, "Apply reconciliation should succeed")

	// Step 4: Update with force=true should succeed
	updatePayloadWithForce := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: integrationTestChangesetID,
		Force:       true,
	}
	updateBytes, err = json.Marshal(updatePayloadWithForce)
	s.Require().NoError(err)

	req = httptest.NewRequest("PATCH", path, bytes.NewReader(updateBytes))
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusAccepted, result.StatusCode, "Update with force=true should succeed")
}

// Test_drift_workflow_destroy tests the full drift detection and resolution workflow for destroy:
// 1. Changeset has DRIFT_DETECTED status
// 2. Destroy is blocked with 409
// 3. User calls check reconciliation
// 4. User calls apply reconciliation
// 5. Destroy succeeds with force=true
func (s *ControllerTestSuite) Test_drift_workflow_destroy() {
	checkResult := &container.ReconciliationCheckResult{
		InstanceID: integrationTestInstanceID,
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
	applyResult := &container.ApplyReconciliationResult{
		InstanceID:       integrationTestInstanceID,
		ResourcesUpdated: 1,
		LinksUpdated:     0,
		Errors:           []container.ReconciliationError{},
	}

	deps := s.setupReconciliationIntegrationTest(checkResult, applyResult)

	// Save the test instance
	err := deps.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:                integrationTestInstanceID,
			InstanceName:              integrationTestInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)
	s.Require().NoError(err)

	// Save a changeset with DRIFT_DETECTED status
	err = deps.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                integrationTestChangesetID,
			InstanceID:        integrationTestInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Created:           testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	// Step 1: Attempt destroy - should be blocked with 409
	destroyPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: integrationTestChangesetID,
	}
	destroyBytes, err := json.Marshal(destroyPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/destroy", integrationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(destroyBytes))
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusConflict, result.StatusCode, "Destroy should be blocked with 409 Conflict")

	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)
	driftBlockedResp := &DriftBlockedResponse{}
	err = json.Unmarshal(respData, driftBlockedResp)
	s.Require().NoError(err)
	s.Assert().Equal(integrationTestInstanceID, driftBlockedResp.InstanceID)
	s.Assert().Contains(driftBlockedResp.Hint, "force=true")

	// Step 2: Call check reconciliation
	checkPayload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		Scope: "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	checkBytes, err := json.Marshal(checkPayload)
	s.Require().NoError(err)

	checkPath := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", integrationTestInstanceID)
	req = httptest.NewRequest("POST", checkPath, bytes.NewReader(checkBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusOK, result.StatusCode, "Check reconciliation should succeed")

	// Step 3: Call apply reconciliation
	applyPayload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "accept_external",
				NewStatus:  "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	applyBytes, err := json.Marshal(applyPayload)
	s.Require().NoError(err)

	applyPath := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", integrationTestInstanceID)
	req = httptest.NewRequest("POST", applyPath, bytes.NewReader(applyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusOK, result.StatusCode, "Apply reconciliation should succeed")

	// Step 4: Destroy with force=true should succeed
	destroyPayloadWithForce := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: integrationTestChangesetID,
		Force:       true,
	}
	destroyBytes, err = json.Marshal(destroyPayloadWithForce)
	s.Require().NoError(err)

	req = httptest.NewRequest("POST", path, bytes.NewReader(destroyBytes))
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusAccepted, result.StatusCode, "Destroy with force=true should succeed")
}

// Test_force_bypasses_drift_no_reconciliation tests that using force=true
// allows the operation to proceed without calling reconciliation endpoints
func (s *ControllerTestSuite) Test_force_bypasses_drift_no_reconciliation() {
	deps := s.setupReconciliationIntegrationTest(nil, nil)

	// Save the test instance
	err := deps.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:                integrationTestInstanceID,
			InstanceName:              integrationTestInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)
	s.Require().NoError(err)

	// Save a changeset with DRIFT_DETECTED status
	err = deps.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                integrationTestChangesetID,
			InstanceID:        integrationTestInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Created:           testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	// Update with force=true should succeed without calling reconciliation
	updatePayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: integrationTestChangesetID,
		Force:       true,
	}
	updateBytes, err := json.Marshal(updatePayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s", integrationTestInstanceID)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(updateBytes))
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusAccepted, result.StatusCode, "Update with force=true should succeed immediately")

	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)
	instance := &state.InstanceState{}
	err = json.Unmarshal(respData, instance)
	s.Require().NoError(err)
	s.Assert().Equal(integrationTestInstanceID, instance.InstanceID)
	// Note: The response returns the existing instance state before async deployment starts.
	// The status will transition to DEPLOYING asynchronously via SSE events.
	s.Assert().Equal(core.InstanceStatusDeployed, instance.Status)
}

// Test_interrupted_state_blocks_operation tests that interrupted state
// (from a previous failed operation) also blocks operations
func (s *ControllerTestSuite) Test_interrupted_state_blocks_operation() {
	checkResult := &container.ReconciliationCheckResult{
		InstanceID: integrationTestInstanceID,
		Resources: []container.ResourceReconcileResult{
			{
				ResourceID:   "resource-1",
				ResourceName: "testResource",
				Type:         container.ReconciliationTypeInterrupted,
			},
		},
		Links:          []container.LinkReconcileResult{},
		HasDrift:       false,
		HasInterrupted: true,
	}

	deps := s.setupReconciliationIntegrationTest(checkResult, nil)

	// Save the test instance
	err := deps.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:                integrationTestInstanceID,
			InstanceName:              integrationTestInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)
	s.Require().NoError(err)

	// Save a changeset with DRIFT_DETECTED status (covers both drift and interrupted)
	err = deps.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                integrationTestChangesetID,
			InstanceID:        integrationTestInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Created:           testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	// Update should be blocked
	updatePayload := &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		ChangeSetID: integrationTestChangesetID,
	}
	updateBytes, err := json.Marshal(updatePayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s", integrationTestInstanceID)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(updateBytes))
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	s.Assert().Equal(http.StatusConflict, result.StatusCode, "Update should be blocked for interrupted state")

	// Calling check should reveal the interrupted state
	checkPayload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        "/test/dir",
			BlueprintFile:    "test.blueprint.yaml",
		},
		Scope: "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	checkBytes, err := json.Marshal(checkPayload)
	s.Require().NoError(err)

	checkPath := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", integrationTestInstanceID)
	req = httptest.NewRequest("POST", checkPath, bytes.NewReader(checkBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	deps.router.ServeHTTP(w, req)

	result = w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	var checkResp container.ReconciliationCheckResult
	err = json.Unmarshal(respData, &checkResp)
	s.Require().NoError(err)
	s.Assert().True(checkResp.HasInterrupted)
	s.Assert().False(checkResp.HasDrift)
	s.Assert().Equal(container.ReconciliationTypeInterrupted, checkResp.Resources[0].Type)
}

func init() {
	helpersv1.SetupRequestBodyValidator()
}
