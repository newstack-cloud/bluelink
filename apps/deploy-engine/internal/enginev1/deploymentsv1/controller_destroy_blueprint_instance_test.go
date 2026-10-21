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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
)

func (s *ControllerTestSuite) Test_destroy_blueprint_instance() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	// Create the test destroy change set to be used to start the destroy
	// process for the blueprint instance.
	err = s.saveDestroyChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testDestroyChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf(
		"/deployments/instances/%s/destroy",
		testInstanceID,
	)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[state.InstanceState]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	instance := wrappedResponse.Data

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	_, err = uuid.Parse(instance.InstanceID)
	s.Assert().NoError(err, "ID should be a valid UUID (as per the configured generator)")
	s.Assert().Equal(
		core.InstanceStatusDestroying,
		instance.Status,
	)
	s.Assert().Equal(
		testTime.Unix(),
		int64(instance.LastStatusUpdateTimestamp),
	)
}

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_by_name() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	// Create the test destroy change set to be used to start the destroy
	// process for the blueprint instance.
	err = s.saveDestroyChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testDestroyChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	// Use the instance name instead of the ID
	path := fmt.Sprintf(
		"/deployments/instances/%s/destroy",
		testInstanceName,
	)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	wrappedResponse := &helpersv1.AsyncOperationResponse[state.InstanceState]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	instance := wrappedResponse.Data

	s.Assert().Equal(http.StatusAccepted, result.StatusCode)
	s.Assert().Equal(
		testInstanceID,
		instance.InstanceID,
	)
	s.Assert().Equal(
		testInstanceName,
		instance.InstanceName,
	)
	s.Assert().Equal(
		core.InstanceStatusDestroying,
		instance.Status,
	)
}

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_handler_returns_404_not_found() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/destroy", nonExistentInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	responseError := map[string]string{}
	err = json.Unmarshal(respData, &responseError)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusNotFound, result.StatusCode)
	s.Assert().Equal(
		fmt.Sprintf(
			"blueprint instance %q not found",
			nonExistentInstanceID,
		),
		responseError["message"],
	)
}

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_handler_fails_for_invalid_plugin_config() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	err = s.saveDestroyChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testDestroyChangesetID,
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

	path := fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_handler_fails_due_to_missing_changeset() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	// We are not saving the test change set for this test,
	// so it should not be found when the request is made.
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrlFailingIDGenerators.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf(
		"/deployments/instances/%s/destroy",
		testInstanceID,
	)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_drift_detected_returns_409() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	// Create a reconciliation result to store in the separate store
	reconciliationResult := &container.ReconciliationCheckResult{
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

	// Create a changeset with DRIFT_DETECTED status
	driftChangesetID := "drift-changeset-for-destroy"
	err = s.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                driftChangesetID,
			InstanceID:        testInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Created:           testTime.Unix(),
			Destroy:           true,
		},
	)
	s.Require().NoError(err)

	// Save the reconciliation result to the separate store
	err = s.reconciliationResultsStore.Save(
		context.Background(),
		&manage.ReconciliationResult{
			ID:          "reconciliation-result-2",
			ChangesetID: driftChangesetID,
			InstanceID:  testInstanceID,
			Result:      reconciliationResult,
			Created:     testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: driftChangesetID,
		// Force is false by default
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	// Should return 409 Conflict with DriftBlockedResponse
	s.Assert().Equal(http.StatusConflict, result.StatusCode)

	driftBlockedResp := &DriftBlockedResponse{}
	err = json.Unmarshal(respData, driftBlockedResp)
	s.Require().NoError(err)

	s.Assert().Equal(testInstanceID, driftBlockedResp.InstanceID)
	s.Assert().Equal(driftChangesetID, driftBlockedResp.ChangesetID)
	s.Assert().Contains(driftBlockedResp.Message, "drift")
	s.Assert().Contains(driftBlockedResp.Hint, "force=true")

	// Verify the reconciliation result is included in the response
	s.Require().NotNil(driftBlockedResp.ReconciliationResult)
	s.Assert().Equal(testInstanceID, driftBlockedResp.ReconciliationResult.InstanceID)
	s.Assert().True(driftBlockedResp.ReconciliationResult.HasDrift)
	s.Assert().False(driftBlockedResp.ReconciliationResult.HasInterrupted)
	s.Require().Len(driftBlockedResp.ReconciliationResult.Resources, 1)
	s.Assert().Equal("resource-1", driftBlockedResp.ReconciliationResult.Resources[0].ResourceID)
	s.Assert().Equal("testResource", driftBlockedResp.ReconciliationResult.Resources[0].ResourceName)
	s.Assert().Equal(container.ReconciliationTypeDrift, driftBlockedResp.ReconciliationResult.Resources[0].Type)
}

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_force_bypasses_drift_check() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	// Create a changeset with DRIFT_DETECTED status
	driftChangesetID := "drift-changeset-force-destroy"
	err = s.changesetStore.Save(
		context.Background(),
		&manage.Changeset{
			ID:                driftChangesetID,
			InstanceID:        testInstanceID,
			Status:            manage.ChangesetStatusDriftDetected,
			BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
			Destroy:           true,
			Created:           testTime.Unix(),
		},
	)
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: driftChangesetID,
		Force:       true, // Force bypasses drift check
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	// Should return 202 Accepted (proceeds with destroy)
	s.Assert().Equal(http.StatusAccepted, result.StatusCode)

	wrappedResponse := &helpersv1.AsyncOperationResponse[state.InstanceState]{}
	err = json.Unmarshal(respData, wrappedResponse)
	s.Require().NoError(err)

	instance := wrappedResponse.Data

	s.Assert().Equal(testInstanceID, instance.InstanceID)
	s.Assert().Equal(core.InstanceStatusDestroying, instance.Status)
}

func (s *ControllerTestSuite) Test_destroy_blueprint_instance_handler_fails_for_deploy_changeset() {
	// Create the blueprint instance to be destroyed.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	// Save a deploy changeset (not a destroy changeset)
	err = s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqPayload := &BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testChangesetID,
	}

	reqBytes, err := json.Marshal(reqPayload)
	s.Require().NoError(err)

	path := fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID)
	req := httptest.NewRequest("POST", path, bytes.NewReader(reqBytes))
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
		"cannot destroy using a deploy changeset",
		responseError["message"],
	)
	s.Assert().Equal(
		"DEPLOY_CHANGESET",
		responseError["code"],
	)
}

// A destroy carries on after the request that asked for it has been answered, so it is
// traced as the root of a trace of its own, naming that request rather than hanging off
// a span that ended long before it.
func (s *ControllerTestSuite) Test_destroy_blueprint_instance_is_traced_as_an_operation() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)
	s.T().Cleanup(func() { telemetry.SetTracer(nil) })

	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	s.Require().NoError(s.saveDestroyChangeset())

	router := mux.NewRouter()
	// Stands in for the request instrumentation, which puts a span on the request's
	// context for the operation to name as having asked for it.
	var requestSpan telemetry.SpanContext
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := telemetry.CurrentTracer().Start(r.Context(), "request")
			defer span.End()
			requestSpan = span.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqBytes, err := json.Marshal(&BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testDestroyChangesetID,
	})
	s.Require().NoError(err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(
		"POST",
		fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID),
		bytes.NewReader(reqBytes),
	))
	result := w.Result()
	defer result.Body.Close()
	s.Require().Equal(http.StatusAccepted, result.StatusCode)

	// The destroy runs on a goroutine of its own, so the span appears after the
	// response rather than before it.
	var destroy *telemetry.RecordedSpan
	s.Require().Eventually(
		func() bool {
			recorded, found := recorder.Span(destroySpanName)
			if !found {
				return false
			}
			destroy = recorded

			return true
		},
		5*time.Second,
		5*time.Millisecond,
		"a destroy should record its own span, recorded %v",
		recorder.Names(),
	)

	s.Assert().True(destroy.Root, "a destroy begins a trace of its own")
	s.Assert().Empty(destroy.ParentID, "it does not hang off the request")
	s.Assert().Equal(
		requestSpan,
		destroy.LinkedTo,
		"it names the request that asked for it",
	)

	instanceID, _ := destroy.Attr("instanceId")
	s.Assert().Equal(testInstanceID, instanceID)
}

// An operation that fails before its work begins never reaches the goroutine that ends
// its span, so it has to be ended where it failed. An unended span is one the backend
// is never given, and these are the failures that need to be surfaced for diagnosis.
func (s *ControllerTestSuite) Test_destroy_that_cannot_load_its_blueprint_still_ends_its_span() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)
	s.T().Cleanup(func() { telemetry.SetTracer(nil) })

	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	s.Require().NoError(s.saveDestroyChangeset())

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrlLoadErrors.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	reqBytes, err := json.Marshal(&BlueprintInstanceDestroyRequestPayload{
		ChangeSetID: testDestroyChangesetID,
	})
	s.Require().NoError(err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(
		"POST",
		fmt.Sprintf("/deployments/instances/%s/destroy", testInstanceID),
		bytes.NewReader(reqBytes),
	))
	result := w.Result()
	defer result.Body.Close()
	s.Require().Equal(http.StatusAccepted, result.StatusCode)

	var destroy *telemetry.RecordedSpan
	s.Require().Eventually(
		func() bool {
			recorded, found := recorder.Span(destroySpanName)
			if !found || !recorded.Ended {
				return false
			}
			destroy = recorded

			return true
		},
		5*time.Second,
		5*time.Millisecond,
		"a destroy that failed to load should still end its span, recorded %v",
		recorder.Names(),
	)

	s.Assert().Error(
		destroy.Err,
		"the failure that stopped the operation should be on its span",
	)
}
