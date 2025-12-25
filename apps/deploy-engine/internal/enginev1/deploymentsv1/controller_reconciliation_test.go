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
	reconciliationTestInstanceID   = "recon-8582991a-9df3-4f7a-a649-344294aff656"
	reconciliationTestInstanceName = "recon-test-instance"
)

func testBlueprintDocInfo() resolve.BlueprintDocumentInfo {
	return resolve.BlueprintDocumentInfo{
		FileSourceScheme: "file",
		Directory:        "/test/dir",
		BlueprintFile:    "test.blueprint.yaml",
	}
}

func (s *ControllerTestSuite) setupReconciliationTest(
	opts ...testutils.MockBlueprintLoaderOption,
) *Controller {
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}

	allOpts := append(
		[]testutils.MockBlueprintLoaderOption{},
		opts...,
	)

	blueprintLoader := testutils.NewMockBlueprintLoader(
		[]*core.Diagnostic{},
		clock,
		stateContainer.Instances(),
		deployEventSequence(""),
		changeStagingEventSequence(),
		allOpts...,
	)

	dependencies := &typesv1.Dependencies{
		EventStore: testutils.NewMockEventStore(
			map[string]*manage.Event{},
		),
		ValidationStore: testutils.NewMockBlueprintValidationStore(
			map[string]*manage.BlueprintValidation{},
		),
		ChangesetStore: testutils.NewMockChangesetStore(
			map[string]*manage.Changeset{},
		),
		Instances:        stateContainer.Instances(),
		Exports:          stateContainer.Exports(),
		IDGenerator:      core.NewUUIDGenerator(),
		EventIDGenerator: utils.NewUUIDv7Generator(),
		ValidationLoader: blueprintLoader,
		DeploymentLoader: blueprintLoader,
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

	// Save a test instance for reconciliation tests
	_ = stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:                reconciliationTestInstanceID,
			InstanceName:              reconciliationTestInstanceName,
			Status:                    core.InstanceStatusDeployed,
			LastStatusUpdateTimestamp: int(testTime.Unix()),
		},
	)

	return NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		dependencies,
	)
}

func (s *ControllerTestSuite) Test_check_reconciliation_success() {
	expectedResult := &container.ReconciliationCheckResult{
		InstanceID: reconciliationTestInstanceID,
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
	ctrl := s.setupReconciliationTest(
		testutils.WithReconciliationTracker(tracker),
		testutils.WithCheckReconciliationResult(expectedResult),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"key": core.ScalarFromString("value"),
				},
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	var checkResult container.ReconciliationCheckResult
	err = json.Unmarshal(respData, &checkResult)
	s.Require().NoError(err)

	s.Assert().Equal(reconciliationTestInstanceID, checkResult.InstanceID)
	s.Assert().True(checkResult.HasDrift)
	s.Assert().False(checkResult.HasInterrupted)
	s.Assert().Len(checkResult.Resources, 1)
	s.Assert().Equal("resource-1", checkResult.Resources[0].ResourceID)

	// Verify tracker recorded the call
	s.Assert().True(tracker.WasCheckCalled())
	checkCalls := tracker.GetCheckCalls()
	s.Assert().Len(checkCalls, 1)
	s.Assert().Equal(reconciliationTestInstanceID, checkCalls[0].InstanceID)
	s.Assert().Equal(container.ReconciliationScopeAll, checkCalls[0].Scope)
}

func (s *ControllerTestSuite) Test_check_reconciliation_by_instance_name() {
	expectedResult := &container.ReconciliationCheckResult{
		InstanceID:     reconciliationTestInstanceID,
		Resources:      []container.ResourceReconcileResult{},
		Links:          []container.LinkReconcileResult{},
		HasDrift:       false,
		HasInterrupted: false,
	}

	ctrl := s.setupReconciliationTest(
		testutils.WithCheckReconciliationResult(expectedResult),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	// Use instance name instead of ID
	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", reconciliationTestInstanceName)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusOK, result.StatusCode)
}

func (s *ControllerTestSuite) Test_check_reconciliation_instance_not_found() {
	ctrl := s.setupReconciliationTest()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := "/deployments/instances/non-existent-instance/reconciliation/check"
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusNotFound, result.StatusCode)

	var errResp map[string]string
	err = json.Unmarshal(respData, &errResp)
	s.Require().NoError(err)
	s.Assert().Contains(errResp["message"], "not found")
}

func (s *ControllerTestSuite) Test_check_reconciliation_missing_config() {
	ctrl := s.setupReconciliationTest()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	// Payload without required config
	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
}

func (s *ControllerTestSuite) Test_check_reconciliation_specific_scope() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(
		testutils.WithReconciliationTracker(tracker),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "specific",
		ResourceNames:         []string{"resource1", "resource2"},
		LinkNames:             []string{"link1"},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	// Verify tracker recorded the call with specific scope
	s.Assert().True(tracker.WasCheckCalled())
	checkCalls := tracker.GetCheckCalls()
	s.Assert().Len(checkCalls, 1)
	s.Assert().Equal(container.ReconciliationScopeSpecific, checkCalls[0].Scope)
	s.Assert().Equal([]string{"resource1", "resource2"}, checkCalls[0].ResourceNames)
	s.Assert().Equal([]string{"link1"}, checkCalls[0].LinkNames)
}

func (s *ControllerTestSuite) Test_check_reconciliation_container_error() {
	ctrl := s.setupReconciliationTest(
		testutils.WithCheckReconciliationError(errors.New("provider unavailable")),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/check", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_success() {
	expectedResult := &container.ApplyReconciliationResult{
		InstanceID:       reconciliationTestInstanceID,
		ResourcesUpdated: 2,
		LinksUpdated:     1,
		Errors:           []container.ReconciliationError{},
	}

	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(
		testutils.WithReconciliationTracker(tracker),
		testutils.WithApplyReconciliationResult(expectedResult),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "accept_external",
				NewStatus:  "stable",
			},
			{
				ResourceID: "resource-2",
				Action:     "update_status",
				NewStatus:  "stable",
			},
		},
		LinkActions: []LinkReconcileActionPayload{
			{
				LinkID:    "link-1",
				Action:    "update_status",
				NewStatus: "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{
				"test-provider": {
					"key": core.ScalarFromString("value"),
				},
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	var applyResult container.ApplyReconciliationResult
	err = json.Unmarshal(respData, &applyResult)
	s.Require().NoError(err)

	s.Assert().Equal(reconciliationTestInstanceID, applyResult.InstanceID)
	s.Assert().Equal(2, applyResult.ResourcesUpdated)
	s.Assert().Equal(1, applyResult.LinksUpdated)
	s.Assert().Empty(applyResult.Errors)

	// Verify tracker recorded the call
	s.Assert().True(tracker.WasApplyCalled())
	applyCalls := tracker.GetApplyCalls()
	s.Assert().Len(applyCalls, 1)
	s.Assert().Equal(reconciliationTestInstanceID, applyCalls[0].InstanceID)
	s.Assert().Len(applyCalls[0].ResourceActions, 2)
	s.Assert().Len(applyCalls[0].LinkActions, 1)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_by_instance_name() {
	ctrl := s.setupReconciliationTest()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "update_status",
				NewStatus:  "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	// Use instance name instead of ID
	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", reconciliationTestInstanceName)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusOK, result.StatusCode)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_instance_not_found() {
	ctrl := s.setupReconciliationTest()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "update_status",
				NewStatus:  "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := "/deployments/instances/non-existent-instance/reconciliation/apply"
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusNotFound, result.StatusCode)

	var errResp map[string]string
	err = json.Unmarshal(respData, &errResp)
	s.Require().NoError(err)
	s.Assert().Contains(errResp["message"], "not found")
}

func (s *ControllerTestSuite) Test_apply_reconciliation_missing_config() {
	ctrl := s.setupReconciliationTest()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	// Payload without required config
	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "update_status",
				NewStatus:  "stable",
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_container_error() {
	ctrl := s.setupReconciliationTest(
		testutils.WithApplyReconciliationError(errors.New("failed to update resource state")),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "update_status",
				NewStatus:  "stable",
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusInternalServerError, result.StatusCode)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_with_intermediary_actions() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(
		testutils.WithReconciliationTracker(tracker),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payload := ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		LinkActions: []LinkReconcileActionPayload{
			{
				LinkID:    "link-1",
				Action:    "accept_external",
				NewStatus: "stable",
				IntermediaryActions: map[string]*IntermediaryReconcileActionPayload{
					"intermediary-1": {
						Action:    "accept_external",
						NewStatus: "stable",
					},
				},
			},
		},
		Config: &types.BlueprintOperationConfig{
			Providers: map[string]map[string]*core.ScalarValue{},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/reconciliation/apply", reconciliationTestInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	// Verify tracker recorded the call with intermediary actions
	s.Assert().True(tracker.WasApplyCalled())
	applyCalls := tracker.GetApplyCalls()
	s.Assert().Len(applyCalls, 1)
	s.Assert().Len(applyCalls[0].LinkActions, 1)
	s.Assert().Len(applyCalls[0].LinkActions[0].IntermediaryActions, 1)
}

func init() {
	helpersv1.SetupRequestBodyValidator()
}
