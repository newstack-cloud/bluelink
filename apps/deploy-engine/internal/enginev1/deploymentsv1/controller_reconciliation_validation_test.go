package deploymentsv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// A check result returns precise statuses as integers, and the CLI passes them straight back,
// so apply takes the integer as well as the label.
func (s *ControllerTestSuite) Test_apply_reconciliation_accepts_statuses_as_labels_or_integers() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(testutils.WithReconciliationTracker(tracker))

	result, _ := s.postReconciliation(ctrl, "apply", &ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{
				ResourceID: "resource-1",
				Action:     "update_status",
				NewStatus:  strconv.Itoa(int(core.PreciseResourceStatusCreated)),
			},
		},
		LinkActions: []LinkReconcileActionPayload{
			{
				LinkID:    "link-1",
				Action:    "accept_external",
				NewStatus: "linked resources updated",
				IntermediaryActions: map[string]*IntermediaryReconcileActionPayload{
					"intermediary-1": {Action: "accept_external", NewStatus: "created"},
				},
			},
		},
		Config: &types.BlueprintOperationConfig{},
	})

	s.Require().Equal(http.StatusOK, result.StatusCode)
	applyCalls := tracker.GetApplyCalls()
	s.Require().Len(applyCalls, 1)
	s.Assert().Equal(core.PreciseResourceStatusCreated, applyCalls[0].ResourceActions[0].NewStatus)
	linkAction := applyCalls[0].LinkActions[0]
	s.Assert().Equal(core.PreciseLinkStatusLinkedResourcesUpdated, linkAction.NewStatus)
	s.Assert().Equal(
		core.PreciseResourceStatusCreated,
		linkAction.IntermediaryActions["intermediary-1"].NewStatus,
	)
}

func (s *ControllerTestSuite) Test_apply_reconciliation_rejects_invalid_actions() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(testutils.WithReconciliationTracker(tracker))

	result, respData := s.postReconciliation(ctrl, "apply", &ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		ResourceActions: []ResourceReconcileActionPayload{
			{Action: "accept", NewStatus: "stable"},
		},
		LinkActions: []LinkReconcileActionPayload{
			{
				LinkID:    "link-1",
				Action:    "update_status",
				NewStatus: "99",
				IntermediaryActions: map[string]*IntermediaryReconcileActionPayload{
					"intermediary-1": {Action: "update_status", NewStatus: "LINKED RESOURCES UPDATED"},
				},
			},
		},
		Config: &types.BlueprintOperationConfig{},
	})

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().ElementsMatch(
		[]inputvalidation.ValidationError{
			{
				Location: ".resourceActions[0].resourceId",
				Message:  "missing required value",
				Type:     "required",
			},
			{
				Location: ".resourceActions[0].action",
				Message: "the value must be one of the following: " +
					"accept_external update_status manual_cleanup_required",
				Type: "oneof",
			},
			{
				Location: ".resourceActions[0].newStatus",
				Message:  `the value must be a precise resource status label (e.g. "CREATED") or its integer`,
				Type:     inputvalidation.TagPreciseResourceStatus,
			},
			{
				Location: ".linkActions[0].newStatus",
				Message: `the value must be a precise link status label (e.g. "LINKED RESOURCES UPDATED") ` +
					"or its integer",
				Type: inputvalidation.TagPreciseLinkStatus,
			},
			{
				Location: ".linkActions[0].intermediaryActions[intermediary-1].newStatus",
				Message:  `the value must be a precise resource status label (e.g. "CREATED") or its integer`,
				Type:     inputvalidation.TagPreciseResourceStatus,
			},
		},
		s.validationErrors(respData),
	)
	s.Assert().False(tracker.WasApplyCalled())
}

func (s *ControllerTestSuite) Test_check_reconciliation_rejects_an_unknown_scope() {
	tracker := testutils.NewReconciliationTracker()
	ctrl := s.setupReconciliationTest(testutils.WithReconciliationTracker(tracker))

	result, respData := s.postReconciliation(ctrl, "check", &CheckReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Scope:                 "everything",
		Config:                &types.BlueprintOperationConfig{},
	})

	s.Assert().Equal(http.StatusUnprocessableEntity, result.StatusCode)
	s.Assert().Equal(
		[]inputvalidation.ValidationError{
			{
				Location: ".scope",
				Message:  "the value must be one of the following: all interrupted specific",
				Type:     "oneof",
			},
		},
		s.validationErrors(respData),
	)
	s.Assert().False(tracker.WasCheckCalled())
}

func (s *ControllerTestSuite) Test_apply_reconciliation_responds_with_camel_case_fields() {
	ctrl := s.setupReconciliationTest(
		testutils.WithApplyReconciliationResult(&container.ApplyReconciliationResult{
			InstanceID:       reconciliationTestInstanceID,
			ResourcesUpdated: 1,
			Errors: []container.ReconciliationError{
				{
					ElementID:   "link-1",
					ElementName: "resourceA::resourceB",
					ElementType: "link",
					Error:       "link not found",
				},
			},
		}),
	)

	result, respData := s.postReconciliation(ctrl, "apply", &ApplyReconciliationRequestPayload{
		BlueprintDocumentInfo: testBlueprintDocInfo(),
		Config:                &types.BlueprintOperationConfig{},
	})

	s.Require().Equal(http.StatusOK, result.StatusCode)
	s.Assert().JSONEq(
		fmt.Sprintf(`{
			"instanceId": %q,
			"resourcesUpdated": 1,
			"linksUpdated": 0,
			"errors": [{
				"elementId": "link-1",
				"elementName": "resourceA::resourceB",
				"elementType": "link",
				"error": "link not found"
			}]
		}`, reconciliationTestInstanceID),
		string(respData),
	)
}

func (s *ControllerTestSuite) postReconciliation(
	ctrl *Controller,
	operation string,
	payload any,
) (*http.Response, []byte) {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	path := fmt.Sprintf(
		"/deployments/instances/%s/reconciliation/%s",
		reconciliationTestInstanceID,
		operation,
	)
	req := httptest.NewRequest("POST", path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	return result, respData
}

func (s *ControllerTestSuite) validationErrors(respData []byte) []inputvalidation.ValidationError {
	validationError := &inputvalidation.FormattedValidationError{}
	s.Require().NoError(json.Unmarshal(respData, validationError))
	return validationError.Errors
}
