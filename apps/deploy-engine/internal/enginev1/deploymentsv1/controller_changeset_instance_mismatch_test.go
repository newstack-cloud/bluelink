package deploymentsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

const otherInstanceID = "other-blueprint-instance"

// The changes were computed against an existing instance, so creating from them would deploy
// a new instance as though it already held that instance's resources.
func (s *ControllerTestSuite) Test_create_rejects_a_changeset_staged_for_an_existing_instance() {
	s.Require().NoError(s.saveUpdateChangeset())

	result, body := s.sendDeployRequest("POST", "/deployments/instances", &BlueprintInstanceRequestPayload{
		BlueprintDocumentInfo: testDeployDocInfo(),
		InstanceName:          testNewInstanceName,
		ChangeSetID:           testChangesetID,
	})

	s.Assert().Equal(http.StatusConflict, result.StatusCode)
	s.Assert().Equal(map[string]any{
		"code":                "CHANGESET_INSTANCE_MISMATCH",
		"changesetInstanceId": testInstanceID,
		"message": fmt.Sprintf(
			"change set %q was staged for instance %q, not a new blueprint instance; "+
				"stage changes for a new instance and try again",
			testChangesetID,
			testInstanceID,
		),
	}, body)
}

func (s *ControllerTestSuite) Test_update_rejects_a_changeset_staged_for_a_new_instance() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	s.Require().NoError(s.saveTestChangeset())

	result, body := s.sendDeployRequest(
		"PATCH",
		"/deployments/instances/"+testInstanceID,
		&BlueprintInstanceRequestPayload{
			BlueprintDocumentInfo: testDeployDocInfo(),
			ChangeSetID:           testChangesetID,
		},
	)

	s.Assert().Equal(http.StatusConflict, result.StatusCode)
	s.Assert().Equal(map[string]any{
		"code":       "CHANGESET_INSTANCE_MISMATCH",
		"instanceId": testInstanceID,
		"message": fmt.Sprintf(
			"change set %q was staged for a new blueprint instance, not instance %q; "+
				"stage changes against the instance and try again",
			testChangesetID,
			testInstanceID,
		),
	}, body)
}

func (s *ControllerTestSuite) Test_update_rejects_a_changeset_staged_for_another_instance() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	s.Require().NoError(s.saveTestChangesetForInstance(otherInstanceID))

	result, body := s.sendDeployRequest(
		"PATCH",
		"/deployments/instances/"+testInstanceID,
		&BlueprintInstanceRequestPayload{
			BlueprintDocumentInfo: testDeployDocInfo(),
			ChangeSetID:           testChangesetID,
		},
	)

	s.Assert().Equal(http.StatusConflict, result.StatusCode)
	s.Assert().Equal(map[string]any{
		"code":                "CHANGESET_INSTANCE_MISMATCH",
		"changesetInstanceId": otherInstanceID,
		"instanceId":          testInstanceID,
		"message": fmt.Sprintf(
			"change set %q was staged for instance %q, not instance %q; "+
				"stage changes against the instance and try again",
			testChangesetID,
			otherInstanceID,
			testInstanceID,
		),
	}, body)
}

func (s *ControllerTestSuite) Test_destroy_rejects_a_changeset_staged_for_another_instance() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)
	s.Require().NoError(s.changesetStore.Save(context.Background(), &manage.Changeset{
		ID:                testDestroyChangesetID,
		InstanceID:        otherInstanceID,
		Destroy:           true,
		Status:            manage.ChangesetStatusChangesStaged,
		BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
		Created:           testTime.Unix(),
		Changes:           &changes.BlueprintChanges{RemovedResources: []string{"resource1"}},
	}))

	result, body := s.sendDeployRequest(
		"POST",
		"/deployments/instances/"+testInstanceID+"/destroy",
		&BlueprintInstanceDestroyRequestPayload{ChangeSetID: testDestroyChangesetID},
	)

	s.Assert().Equal(http.StatusConflict, result.StatusCode)
	s.Assert().Equal("CHANGESET_INSTANCE_MISMATCH", body["code"])
	s.Assert().Equal(otherInstanceID, body["changesetInstanceId"])
	s.Assert().Equal(testInstanceID, body["instanceId"])
}

func (s *ControllerTestSuite) sendDeployRequest(
	method string,
	path string,
	payload any,
) (*http.Response, map[string]any) {
	router := mux.NewRouter()
	router.HandleFunc("/deployments/instances", s.ctrl.CreateBlueprintInstanceHandler).Methods("POST")
	router.HandleFunc("/deployments/instances/{id}", s.ctrl.UpdateBlueprintInstanceHandler).Methods("PATCH")
	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		s.ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")

	payloadBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(method, path, bytes.NewReader(payloadBytes))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	body := map[string]any{}
	s.Require().NoError(json.Unmarshal(respData, &body))
	return result, body
}

func testDeployDocInfo() resolve.BlueprintDocumentInfo {
	return resolve.BlueprintDocumentInfo{
		FileSourceScheme: "file",
		Directory:        "/test/dir",
		BlueprintFile:    "test.blueprint.yaml",
	}
}
