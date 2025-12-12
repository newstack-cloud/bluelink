package deploymentsv1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (s *ControllerTestSuite) Test_get_blueprint_instance_exports_handler() {
	expectedExports, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/exports",
		s.ctrl.GetBlueprintInstanceExportsHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/instances/%s/exports", testInstanceID)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	actualExports := map[string]*state.ExportState{}
	err = json.Unmarshal(respData, &actualExports)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(
		expectedExports,
		actualExports,
	)
}

func (s *ControllerTestSuite) Test_get_blueprint_instance_exports_handler_by_name() {
	expectedExports, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/exports",
		s.ctrl.GetBlueprintInstanceExportsHandler,
	).Methods("GET")

	// Use the instance name instead of the ID
	path := fmt.Sprintf("/deployments/instances/%s/exports", testInstanceName)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	actualExports := map[string]*state.ExportState{}
	err = json.Unmarshal(respData, &actualExports)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(
		expectedExports,
		actualExports,
	)
}

func (s *ControllerTestSuite) Test_get_blueprint_instance_exports_handler_returns_404_not_found() {
	_, err := s.saveTestBlueprintInstance()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/exports",
		s.ctrl.GetBlueprintInstanceExportsHandler,
	).Methods("GET")

	path := fmt.Sprintf("/deployments/instances/%s/exports", nonExistentInstanceID)
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
