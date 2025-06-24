package deploymentsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	testInstanceID        = "8582991a-9df3-4f7a-a649-344294aff656"
	testInstanceName      = "test-instance"
	nonExistentInstanceID = "2f16c75b-b397-498c-a501-61cb409e4fbb"
)

func (s *ControllerTestSuite) Test_get_blueprint_instance_handler() {
	expectedExports, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		s.ctrl.GetBlueprintInstanceHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/instances/%s", testInstanceID)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	instance := &state.InstanceState{}
	err = json.Unmarshal(respData, instance)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(
		testInstanceID,
		instance.InstanceID,
	)
	s.Assert().Equal(
		core.InstanceStatusDeploying,
		instance.Status,
	)
	s.Assert().Equal(
		expectedExports,
		instance.Exports,
	)
}

func (s *ControllerTestSuite) Test_get_blueprint_instance_handler_returns_404_not_found() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}",
		s.ctrl.GetBlueprintInstanceHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/instances/%s", nonExistentInstanceID)
	req := httptest.NewRequest("GET", path, nil)
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

func (s *ControllerTestSuite) saveTestBlueprintInstance() (map[string]*state.ExportState, error) {
	instance := state.InstanceState{
		InstanceID:   testInstanceID,
		InstanceName: testInstanceName,
		Status:       core.InstanceStatusDeploying,
		Exports: map[string]*state.ExportState{
			"exportedField": {
				Value: core.MappingNodeFromString("testValue"),
				Type:  schema.ExportTypeString,
				Field: "resources[\"testResource\"].spec.testField",
			},
		},
		LastStatusUpdateTimestamp: int(testTime.Unix()),
	}

	err := s.instances.Save(
		context.Background(),
		instance,
	)
	if err != nil {
		return nil, err
	}

	return instance.Exports, nil
}
