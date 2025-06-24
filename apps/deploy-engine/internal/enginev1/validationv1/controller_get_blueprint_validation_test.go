package validationv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

const (
	nonExistentBlueprintValidationID = "e301de45-5df3-4890-a91d-7a603c900acb"
)

func (s *ControllerTestSuite) Test_get_blueprint_validation_handler() {
	err := s.saveTestBlueprintValidation()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/validations/{id}",
		s.ctrl.GetBlueprintValidationHandler,
	).Methods("GET")

	path := fmt.Sprintf("/validations/%s", testBlueprintValidationID)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	respData, err := io.ReadAll(result.Body)
	s.Require().NoError(err)

	blueprintValidation := &manage.BlueprintValidation{}
	err = json.Unmarshal(respData, blueprintValidation)
	s.Require().NoError(err)

	s.Assert().Equal(http.StatusOK, result.StatusCode)
	s.Assert().Equal(
		testBlueprintValidationID,
		blueprintValidation.ID,
	)
	s.Assert().Equal(
		manage.BlueprintValidationStatusStarting,
		blueprintValidation.Status,
	)
	s.Assert().Equal(
		"file:///test/dir/test.blueprint.yaml",
		blueprintValidation.BlueprintLocation,
	)
	s.Assert().Equal(
		testTime.Unix(),
		blueprintValidation.Created,
	)
}

func (s *ControllerTestSuite) Test_get_blueprint_validation_handler_returns_404_not_found() {
	err := s.saveTestBlueprintValidation()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/validations/{id}",
		s.ctrl.GetBlueprintValidationHandler,
	).Methods("GET")

	path := fmt.Sprintf("/validations/%s", nonExistentBlueprintValidationID)
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
			"blueprint validation %q not found",
			nonExistentBlueprintValidationID,
		),
		responseError["message"],
	)
}

func (s *ControllerTestSuite) saveTestBlueprintValidation() error {
	blueprintValidation := &manage.BlueprintValidation{
		ID:                testBlueprintValidationID,
		Status:            manage.BlueprintValidationStatusStarting,
		BlueprintLocation: "file:///test/dir/test.blueprint.yaml",
		Created:           testTime.Unix(),
	}

	err := s.validationStore.Save(
		context.Background(),
		blueprintValidation,
	)
	if err != nil {
		return err
	}

	return nil
}
