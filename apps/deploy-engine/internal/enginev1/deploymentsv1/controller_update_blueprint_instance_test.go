package deploymentsv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/celerity/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/celerity/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/celerity/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/celerity/libs/blueprint/core"
	"github.com/newstack-cloud/celerity/libs/blueprint/state"
)

func (s *ControllerTestSuite) Test_update_blueprint_instance() {
	// Create the blueprint instance to be updated.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	// Create the test change set to be used to start the deployment
	// process for the existing blueprint instance.
	err = s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		s.ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

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

	path := fmt.Sprintf(
		"/deployments/instances/%s",
		testInstanceID,
	)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(reqBytes))
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
	s.Assert().Equal(
		testInstanceID,
		instance.InstanceID,
	)
	s.Assert().Equal(
		core.InstanceStatusDeploying,
		instance.Status,
	)
	s.Assert().Equal(
		testTime.Unix(),
		int64(instance.LastStatusUpdateTimestamp),
	)
}

func (s *ControllerTestSuite) Test_update_blueprint_instance_handler_fails_for_invalid_plugin_config() {
	// Create the blueprint instance to be updated.
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	// Create the test change set to be used to start the deployment
	// process for the existing blueprint instance.
	err = s.saveTestChangeset()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		s.ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

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

	path := fmt.Sprintf(
		"/deployments/instances/%s",
		testInstanceID,
	)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(reqBytes))
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

func (s *ControllerTestSuite) Test_update_blueprint_instance_handler_returns_404_not_found() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		s.ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

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

	path := fmt.Sprintf("/deployments/instances/%s", nonExistentInstanceID)
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(reqBytes))
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
