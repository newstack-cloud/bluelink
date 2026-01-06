package deploymentsv1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (s *ControllerTestSuite) Test_list_blueprint_instances_returns_all_instances() {
	s.saveTestInstances()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	req := httptest.NewRequest("GET", "/deployments/instances", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	listResult := &state.ListInstancesResult{}
	err = json.Unmarshal(respData, listResult)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(3, listResult.TotalCount)
	s.Assert().Len(listResult.Instances, 3)
}

func (s *ControllerTestSuite) Test_list_blueprint_instances_filters_by_search() {
	s.saveTestInstances()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	req := httptest.NewRequest("GET", "/deployments/instances?search=prod", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	listResult := &state.ListInstancesResult{}
	err = json.Unmarshal(respData, listResult)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(2, listResult.TotalCount)
	s.Assert().Len(listResult.Instances, 2)
	for _, inst := range listResult.Instances {
		s.Assert().Contains(inst.InstanceName, "prod")
	}
}

func (s *ControllerTestSuite) Test_list_blueprint_instances_paginates_with_limit_and_offset() {
	s.saveTestInstances()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	req := httptest.NewRequest("GET", "/deployments/instances?limit=2&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	listResult := &state.ListInstancesResult{}
	err = json.Unmarshal(respData, listResult)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(3, listResult.TotalCount)
	s.Assert().Len(listResult.Instances, 2)
}

func (s *ControllerTestSuite) Test_list_blueprint_instances_returns_empty_array_when_no_instances() {
	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	req := httptest.NewRequest("GET", "/deployments/instances", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	listResult := &state.ListInstancesResult{}
	err = json.Unmarshal(respData, listResult)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(0, listResult.TotalCount)
	s.Assert().Empty(listResult.Instances)
}

func (s *ControllerTestSuite) Test_list_blueprint_instances_search_is_case_insensitive() {
	s.saveTestInstances()

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances",
		s.ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	req := httptest.NewRequest("GET", "/deployments/instances?search=PROD", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	listResult := &state.ListInstancesResult{}
	err = json.Unmarshal(respData, listResult)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(2, listResult.TotalCount)
}

func (s *ControllerTestSuite) saveTestInstances() {
	instances := []state.InstanceState{
		{
			InstanceID:            "inst-1",
			InstanceName:          "my-app-prod",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1700000000,
		},
		{
			InstanceID:            "inst-2",
			InstanceName:          "my-app-staging",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1700000100,
		},
		{
			InstanceID:            "inst-3",
			InstanceName:          "another-prod",
			Status:                core.InstanceStatusDeploying,
			LastDeployedTimestamp: 1700000200,
		},
	}

	for _, inst := range instances {
		err := s.instances.Save(context.Background(), inst)
		s.Require().NoError(err)
	}
}
