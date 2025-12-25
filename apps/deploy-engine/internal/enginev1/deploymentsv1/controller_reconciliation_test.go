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

// ============================================================================
// Child Blueprint Reconciliation Tests
// ============================================================================

func (s *ControllerTestSuite) Test_check_reconciliation_with_child_blueprint_resources() {
	expectedResult := &container.ReconciliationCheckResult{
		InstanceID: reconciliationTestInstanceID,
		Resources: []container.ResourceReconcileResult{
			{
				ResourceID:   "parent-resource-1",
				ResourceName: "parentResource",
				ChildPath:    "",
				Type:         container.ReconciliationTypeInterrupted,
			},
			{
				ResourceID:   "child-resource-1",
				ResourceName: "childResource",
				ChildPath:    "childA",
				Type:         container.ReconciliationTypeInterrupted,
			},
			{
				ResourceID:   "nested-resource-1",
				ResourceName: "nestedResource",
				ChildPath:    "childA.nestedChild",
				Type:         container.ReconciliationTypeDrift,
			},
		},
		Links:          []container.LinkReconcileResult{},
		HasDrift:       true,
		HasInterrupted: true,
		HasChildIssues: true,
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
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	var checkResult container.ReconciliationCheckResult
	err = json.Unmarshal(respData, &checkResult)
	s.Require().NoError(err)

	s.Assert().Equal(reconciliationTestInstanceID, checkResult.InstanceID)
	s.Assert().True(checkResult.HasDrift)
	s.Assert().True(checkResult.HasInterrupted)
	s.Assert().True(checkResult.HasChildIssues, "HasChildIssues should be true")
	s.Assert().Len(checkResult.Resources, 3)

	// Verify ChildPath is correctly populated
	parentResource := findResourceByID(checkResult.Resources, "parent-resource-1")
	s.Require().NotNil(parentResource)
	s.Assert().Equal("", parentResource.ChildPath)

	childResource := findResourceByID(checkResult.Resources, "child-resource-1")
	s.Require().NotNil(childResource)
	s.Assert().Equal("childA", childResource.ChildPath)

	nestedResource := findResourceByID(checkResult.Resources, "nested-resource-1")
	s.Require().NotNil(nestedResource)
	s.Assert().Equal("childA.nestedChild", nestedResource.ChildPath)
}

func (s *ControllerTestSuite) Test_check_reconciliation_with_include_children_false() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(
		testutils.WithReconciliationTracker(tracker),
	)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	includeChildren := false
	payload := CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "all",
		IncludeChildren:       &includeChildren,
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

	// Verify tracker recorded the call with IncludeChildren = false
	s.Assert().True(tracker.WasCheckCalled())
	checkCalls := tracker.GetCheckCalls()
	s.Assert().Len(checkCalls, 1)
	s.Require().NotNil(checkCalls[0].IncludeChildren)
	s.Assert().False(*checkCalls[0].IncludeChildren)
}

func (s *ControllerTestSuite) Test_check_reconciliation_with_child_path_filter() {
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
		ChildPath:             "childA.nestedChild",
		ResourceNames:         []string{"resourceA"},
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

	// Verify tracker recorded the call with ChildPath filter
	s.Assert().True(tracker.WasCheckCalled())
	checkCalls := tracker.GetCheckCalls()
	s.Assert().Len(checkCalls, 1)
	s.Assert().Equal("childA.nestedChild", checkCalls[0].ChildPath)
	s.Assert().Equal(container.ReconciliationScopeSpecific, checkCalls[0].Scope)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_with_child_resource_actions() {
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
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "parent-resource-1",
				ChildPath:  "",
				Action:     "update_status",
				NewStatus:  "stable",
			},
			{
				ResourceID: "child-resource-1",
				ChildPath:  "childA",
				Action:     "accept_external",
				NewStatus:  "stable",
			},
			{
				ResourceID: "nested-resource-1",
				ChildPath:  "childA.nestedChild",
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

	s.Assert().Equal(http.StatusOK, result.StatusCode)

	// Verify tracker recorded the call with child paths
	s.Assert().True(tracker.WasApplyCalled())
	applyCalls := tracker.GetApplyCalls()
	s.Assert().Len(applyCalls, 1)
	s.Assert().Len(applyCalls[0].ResourceActions, 3)

	// Verify ChildPath is correctly passed through
	parentAction := findResourceActionByID(applyCalls[0].ResourceActions, "parent-resource-1")
	s.Require().NotNil(parentAction)
	s.Assert().Equal("", parentAction.ChildPath)

	childAction := findResourceActionByID(applyCalls[0].ResourceActions, "child-resource-1")
	s.Require().NotNil(childAction)
	s.Assert().Equal("childA", childAction.ChildPath)
	s.Assert().Equal(container.ReconciliationActionAcceptExternal, childAction.Action)

	nestedAction := findResourceActionByID(applyCalls[0].ResourceActions, "nested-resource-1")
	s.Require().NotNil(nestedAction)
	s.Assert().Equal("childA.nestedChild", nestedAction.ChildPath)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_with_child_link_actions() {
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
				LinkID:    "parent-link-1",
				ChildPath: "",
				Action:    "update_status",
				NewStatus: "stable",
			},
			{
				LinkID:    "child-link-1",
				ChildPath: "childA",
				Action:    "accept_external",
				NewStatus: "stable",
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

	// Verify tracker recorded the call with child paths
	s.Assert().True(tracker.WasApplyCalled())
	applyCalls := tracker.GetApplyCalls()
	s.Assert().Len(applyCalls, 1)
	s.Assert().Len(applyCalls[0].LinkActions, 2)

	// Verify ChildPath is correctly passed through for links
	parentLinkAction := findLinkActionByID(applyCalls[0].LinkActions, "parent-link-1")
	s.Require().NotNil(parentLinkAction)
	s.Assert().Equal("", parentLinkAction.ChildPath)

	childLinkAction := findLinkActionByID(applyCalls[0].LinkActions, "child-link-1")
	s.Require().NotNil(childLinkAction)
	s.Assert().Equal("childA", childLinkAction.ChildPath)
	s.Assert().Equal(container.ReconciliationActionAcceptExternal, childLinkAction.Action)
}

func (s *ControllerTestSuite) Test_check_reconciliation_with_child_blueprint_links() {
	expectedResult := &container.ReconciliationCheckResult{
		InstanceID: reconciliationTestInstanceID,
		Resources:  []container.ResourceReconcileResult{},
		Links: []container.LinkReconcileResult{
			{
				LinkID:    "parent-link-1",
				LinkName:  "resourceA::resourceB",
				ChildPath: "",
				Type:      container.ReconciliationTypeInterrupted,
			},
			{
				LinkID:    "child-link-1",
				LinkName:  "childResourceA::childResourceB",
				ChildPath: "childA",
				Type:      container.ReconciliationTypeDrift,
			},
		},
		HasDrift:       true,
		HasInterrupted: true,
		HasChildIssues: true,
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

	s.Assert().True(checkResult.HasChildIssues)
	s.Assert().Len(checkResult.Links, 2)

	// Verify ChildPath is correctly populated for links
	parentLink := findLinkByID(checkResult.Links, "parent-link-1")
	s.Require().NotNil(parentLink)
	s.Assert().Equal("", parentLink.ChildPath)

	childLink := findLinkByID(checkResult.Links, "child-link-1")
	s.Require().NotNil(childLink)
	s.Assert().Equal("childA", childLink.ChildPath)
}

// Helper functions for finding elements in slices

func findResourceByID(resources []container.ResourceReconcileResult, resourceID string) *container.ResourceReconcileResult {
	for i := range resources {
		if resources[i].ResourceID == resourceID {
			return &resources[i]
		}
	}
	return nil
}

func findLinkByID(links []container.LinkReconcileResult, linkID string) *container.LinkReconcileResult {
	for i := range links {
		if links[i].LinkID == linkID {
			return &links[i]
		}
	}
	return nil
}

func findResourceActionByID(actions []container.ResourceReconcileAction, resourceID string) *container.ResourceReconcileAction {
	for i := range actions {
		if actions[i].ResourceID == resourceID {
			return &actions[i]
		}
	}
	return nil
}

func findLinkActionByID(actions []container.LinkReconcileAction, linkID string) *container.LinkReconcileAction {
	for i := range actions {
		if actions[i].LinkID == linkID {
			return &actions[i]
		}
	}
	return nil
}

func init() {
	helpersv1.SetupRequestBodyValidator()
}
